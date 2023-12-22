/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cucloud

import (
	"fmt"
	"math/rand"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	cucloudsdkv2 "k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/services/v2"
	cucloudmodel "k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/services/v2/model"
	"k8s.io/autoscaler/cluster-autoscaler/utils/units"
	"k8s.io/klog/v2"
)

type CucloudManager interface {
	// GetAsgForInstance returns auto scaling group for the given instance.
	GetAsgForInstance(instanceId string) (cloudprovider.NodeGroup, error)

	// ListScalingGroups list all scaling groups.
	ListScalingGroups() ([]AutoScalingGroup, error)

	// GetInstances gets the instances in an auto scaling group.
	GetInstances(groupID string) ([]cloudprovider.Instance, error)

	// RegisterAsg registers auto scaling group to manager
	RegisterAsg(asg *AutoScalingGroup)

	// getAsgTemplate Get auto scaling group template
	getAsgTemplate(groupID string) (*asgTemplate, error)

	// buildNodeFromTemplate returns template from instance flavor
	buildNodeFromTemplate(groupName string, template *asgTemplate) (*corev1.Node, error)

	// DeleteInstances is used to delete instances from auto scaling group by instanceIDs.
	DeleteInstances(groupID string, instanceIds []string, ruleName string) error

	// GetDesiredAsgInstanceSize gets the desire instance number of specific auto scaling group.
	GetDesiredAsgInstanceSize(groupID string) (int, error)

	// IncreaseAsgInstanceSize increases the instance number of specific auto scaling group.
	// the delta should be non-negative.
	IncreaseAsgInstanceSize(groupID string, delta int, ruleName string) error
}

type cucloudManager struct {
	asgs   *autoScalingGroups
	client *cucloudsdkv2.ApiClient
}

func newCucloudManager(c *CloudConfig) (*cucloudManager, error) {
	manager := &cucloudManager{
		asgs: newAutoScalingGroups(),
	}
	client, err := c.getApiClient()
	if err != nil {
		return nil, err
	}
	manager.client = client

	manager.asgs.generateCache(manager)

	return manager, nil
}

func (c *cucloudManager) GetAsgForInstance(instanceId string) (cloudprovider.NodeGroup, error) {
	return c.asgs.FindForInstance(instanceId, c)
}

func (c *cucloudManager) ListScalingGroups() ([]AutoScalingGroup, error) {
	r, err := c.client.ListScalingGroups()
	if err != nil {
		klog.Errorf("failed to list scaling groups. error: %v", err)
		return nil, err
	}

	if r == nil {
		klog.Info("no scaling group yet.")
		return nil, nil
	}

	asgs := make([]AutoScalingGroup, 0, len(r.Items))
	for _, sg := range r.Items {
		if sg.AutoScale == nil || sg.NodeConfig == nil || sg.DiskConfig == nil {
			// 过滤掉不可用的默认节点池
			continue
		}
		autoScalingGroup := newAutoScalingGroup(c, sg)
		asgs = append(asgs, autoScalingGroup)
		klog.Infof("found autoscaling group: %s", autoScalingGroup.groupName)
	}

	return asgs, nil
}

func (c *cucloudManager) GetInstances(groupID string) ([]cloudprovider.Instance, error) {
	sg, err := c.client.DescribeScalingGroup(groupID)
	if err != nil {
		return nil, err
	}

	instances := make([]cloudprovider.Instance, 0, len(sg.ScaleNodes))
	for _, sgi := range sg.ScaleNodes {
		// When a new instance joining to the scaling group, the instance id maybe empty(nil).
		if len(sgi.ProviderID) == 0 {
			klog.Infof("ignore instance without instance id, maybe instance is joining.")
			continue
		}
		if sgi.Status == cucloudmodel.NodeStatusDeleted || sgi.Status == cucloudmodel.NodeStatusFailed {
			//klog.Infof("instance has be removed, ignored.")
			continue
		}
		instance := cloudprovider.Instance{
			Id:     sgi.ProviderID,
			Status: c.transformInstanceState(sgi.Status),
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

func (c *cucloudManager) transformInstanceState(status cucloudmodel.NodeStatus) *cloudprovider.InstanceStatus {
	instanceStatus := &cloudprovider.InstanceStatus{}

	switch status {
	case cucloudmodel.NodeStatusRunning:
		instanceStatus.State = cloudprovider.InstanceRunning
	case cucloudmodel.NodeStatusCreating, cucloudmodel.NodeStatusPending:
		instanceStatus.State = cloudprovider.InstanceCreating
	case cucloudmodel.NodeStatusDeleting:
		instanceStatus.State = cloudprovider.InstanceDeleting
	default:
		instanceStatus.ErrorInfo = &cloudprovider.InstanceErrorInfo{
			ErrorClass:   cloudprovider.OtherErrorClass,
			ErrorMessage: fmt.Sprintf("invalid instance state: %v", status),
		}
		return instanceStatus
	}

	return instanceStatus
}

func (c *cucloudManager) RegisterAsg(asg *AutoScalingGroup) {
	c.asgs.Register(asg)
}

type asgTemplate struct {
	name       string
	cpuNum     int64
	memorySize int64
}

func (c *cucloudManager) getAsgTemplate(groupID string) (*asgTemplate, error) {
	sg, err := c.client.DescribeScalingGroup(groupID)
	if err != nil {
		klog.Errorf("failed to get ASG by id:%s,because of %s", groupID, err.Error())
		return nil, err
	}

	return &asgTemplate{
		name:       sg.Name,
		cpuNum:     sg.NodeConfig.Res.Resource.Cpu,
		memorySize: sg.NodeConfig.Res.Resource.Mem,
	}, nil
}

func (c *cucloudManager) buildNodeFromTemplate(groupName string, template *asgTemplate) (*corev1.Node, error) {
	node := corev1.Node{}
	nodeName := fmt.Sprintf("%s-asg-%d", groupName, rand.Int63())

	node.ObjectMeta = metav1.ObjectMeta{
		Name:   nodeName,
		Labels: map[string]string{},
	}

	node.Status = corev1.NodeStatus{
		Capacity: corev1.ResourceList{},
	}

	node.Status.Capacity[corev1.ResourcePods] = *resource.NewQuantity(defaultPodLimitCount, resource.DecimalSI)
	node.Status.Capacity[corev1.ResourceCPU] = *resource.NewQuantity(template.cpuNum, resource.DecimalSI)
	node.Status.Capacity[corev1.ResourceMemory] = *resource.NewQuantity(template.memorySize*units.GiB, resource.DecimalSI)

	node.Status.Allocatable = node.Status.Capacity

	node.Labels = cloudprovider.JoinStringMaps(node.Labels, buildGenericLabels(template, nodeName))

	node.Status.Conditions = cloudprovider.BuildReadyConditions()
	return &node, nil
}

func buildGenericLabels(template *asgTemplate, nodeName string) map[string]string {
	result := make(map[string]string)
	result[corev1.LabelArchStable] = cloudprovider.DefaultArch
	result[corev1.LabelOSStable] = cloudprovider.DefaultOS

	result[corev1.LabelInstanceTypeStable] = template.name
	result[corev1.LabelHostname] = nodeName

	return result
}

func (c *cucloudManager) DeleteInstances(groupID string, instanceIds []string, ruleName string) error {
	err := c.client.DeleteScalingGroupInstances(groupID, instanceIds, ruleName, cucloudmodel.ScaleTypeAuto)

	if err != nil {
		klog.Errorf("failed to delete scaling instances. group: %s, error: %v", groupID, err)
		return err
	}

	return nil
}

func (c *cucloudManager) GetDesiredAsgInstanceSize(groupID string) (int, error) {
	instances, err := c.GetInstances(groupID)
	if err != nil {
		klog.Errorf("failed to list scaling group instances. group: %s, error: %v", groupID, err)
		return 0, fmt.Errorf("failed to get instance list")
	}

	// Get desire instance number by total instance number minus the one which is in deleting state.
	desiredInstanceSize := len(instances)
	for _, instance := range instances {
		if instance.Status.State == cloudprovider.InstanceDeleting {
			desiredInstanceSize--
		}
	}

	return desiredInstanceSize, nil
}

func (c *cucloudManager) IncreaseAsgInstanceSize(groupID string, delta int, ruleName string) error {
	err := c.client.CreateScalingGroupInstances(groupID, delta, cucloudmodel.ScaleTypeAuto, ruleName)

	if err != nil {
		klog.Errorf("failed to create scaling instances. group: %s, error: %v", groupID, err)
		return err
	}

	return nil
}
