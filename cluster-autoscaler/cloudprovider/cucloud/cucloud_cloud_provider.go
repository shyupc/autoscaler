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
	"k8s.io/autoscaler/cluster-autoscaler/utils/gpu"
	"strings"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/autoscaler/cluster-autoscaler/config/dynamic"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"
	"k8s.io/klog/v2"
)

const (
	// GPULabel is the label added to nodes with GPU resource.
	GPULabel             = "cucloud.cn/gpu"
	defaultPodLimitCount = 110
)

var (
	availableGPUTypes = map[string]struct{}{}
)

// cucloudCloudProvider implements CloudProvider interface defined in autoscaler/cluster-autoscaler/cloudprovider/cloud_provider.go
type cucloudCloudProvider struct {
	manager         CucloudManager
	asgs            []*AutoScalingGroup
	resourceLimiter *cloudprovider.ResourceLimiter
	lock            sync.RWMutex
}

func newCucloudProvider(opts config.AutoscalingOptions, do cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) *cucloudCloudProvider {
	if do.AutoDiscoverySpecified() {
		klog.Errorf("only support static discovery scaling group in cucloud for now")
		return nil
	}

	cloudConfig, err := readConf(opts.CloudConfig)
	if err != nil {
		klog.Errorf("failed to read cloud configuration. error: %v", err)
		return nil
	}
	if err = cloudConfig.validate(); err != nil {
		klog.Errorf("cloud configuration is invalid. error: %v", err)
		return nil
	}

	manager, err := newCucloudManager(cloudConfig)
	if err != nil {
		klog.Errorf("failed to init cucloud service manager. error: %v", err)
		return nil
	}

	ccp := &cucloudCloudProvider{
		manager:         manager,
		resourceLimiter: rl,
		asgs:            make([]*AutoScalingGroup, 0),
	}

	if len(do.NodeGroupSpecs) <= 0 {
		klog.Error("no auto scaling group specified")
		return nil
	}
	err = ccp.buildAsgs(do.NodeGroupSpecs)
	if err != nil {
		klog.Errorf("failed to build auto scaling groups. error: %v", err)
		return nil
	}

	return ccp
}

// Name returns the name of the cloud provider.
func (ccp *cucloudCloudProvider) Name() string {
	return cloudprovider.CucloudProviderName
}

// NodeGroups returns all node groups managed by this cloud provider.
func (ccp *cucloudCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	ccp.lock.RLock()
	defer ccp.lock.RUnlock()

	groups := make([]cloudprovider.NodeGroup, 0, len(ccp.asgs))
	for i := range ccp.asgs {
		groups = append(groups, ccp.asgs[i])
	}

	return groups
}

// NodeGroupForNode returns the node group for the given node, nil if the node
// should not be processed by cluster autoscaler, or non-nil error if such
// occurred. Must be implemented.
func (ccp *cucloudCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	if _, found := node.ObjectMeta.Labels["node-role.kubernetes.io/master"]; found {
		return nil, nil
	}

	instanceID := node.Spec.ProviderID
	if len(instanceID) == 0 {
		klog.Warningf("Node %v has no providerId, not added from autoscaling, ignored", node.Name)
		// 适配老集群，无providerID的问题
		instanceID = node.Name
		//return nil, fmt.Errorf("provider id missing from node: %s", node.Name)
	}

	return ccp.manager.GetAsgForInstance(instanceID)
}

// HasInstance returns whether the node has corresponding instance in cloud provider,
// true if the node has an instance, false if it no longer exists
func (ccp *cucloudCloudProvider) HasInstance(node *apiv1.Node) (bool, error) {
	return true, cloudprovider.ErrNotImplemented
}

// Pricing returns pricing model for this cloud provider or error if not available. Not implemented.
func (ccp *cucloudCloudProvider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetAvailableMachineTypes get all machine types that can be requested from the cloud provider. Not implemented.
func (ccp *cucloudCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return []string{}, nil
}

// NewNodeGroup builds a theoretical node group based on the node definition provided. The node group is not automatically
// created on the cloud provider side. The node group is not returned by NodeGroups() until it is created. Not implemented.
func (ccp *cucloudCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string,
	taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetResourceLimiter returns struct containing limits (max, min) for resources (cores, memory etc.).
func (ccp *cucloudCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return ccp.resourceLimiter, nil
}

// GPULabel returns the label added to nodes with GPU resource.
func (ccp *cucloudCloudProvider) GPULabel() string {
	return GPULabel
}

// GetAvailableGPUTypes returns all available GPU types cloud provider supports.
func (ccp *cucloudCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return availableGPUTypes
}

// GetNodeGpuConfig returns the label, type and resource name for the GPU added to node. If node doesn't have
// any GPUs, it returns nil.
func (ccp *cucloudCloudProvider) GetNodeGpuConfig(node *apiv1.Node) *cloudprovider.GpuConfig {
	return gpu.GetNodeGPUFromCloudProvider(ccp, node)
}

// Cleanup currently does nothing.
func (ccp *cucloudCloudProvider) Cleanup() error {
	return nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
// Currently does nothing.
func (ccp *cucloudCloudProvider) Refresh() error {
	return nil
}

func (ccp *cucloudCloudProvider) buildAsgs(specs []string) error {
	asgs, err := ccp.manager.ListScalingGroups()
	if err != nil {
		klog.Errorf("failed to list scaling groups, because of %s", err.Error())
		return err
	}

	for _, spec := range specs {
		if err := ccp.addNodeGroup(spec, asgs); err != nil {
			klog.Warningf("failed to add node group to cucloud provider with spec: %s", spec)
			return err
		}
	}

	return nil
}

func (ccp *cucloudCloudProvider) addNodeGroup(spec string, asgs []AutoScalingGroup) error {
	asg, err := buildAsgFromSpec(spec, asgs, ccp.manager)
	if err != nil {
		klog.Errorf("failed to build ASG from spec, because of %s", err.Error())
		return err
	}
	ccp.addAsg(asg)
	return nil
}

func (ccp *cucloudCloudProvider) addAsg(asg *AutoScalingGroup) {
	ccp.asgs = append(ccp.asgs, asg)
	ccp.manager.RegisterAsg(asg)
}

func buildAsgFromSpec(specStr string, asgs []AutoScalingGroup, manager CucloudManager) (*AutoScalingGroup, error) {
	spec, err := dynamic.SpecFromString(specStr, true)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node group spec: %v", err)
	}
	gr, err := groupFromString(spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group name: %v", err)
	}
	for _, asg := range asgs {
		if asg.groupName == gr.groupName {
			return &AutoScalingGroup{
				manager:           manager,
				ruleName:          gr.ruleName,
				groupName:         asg.groupName,
				groupID:           asg.groupID,
				minInstanceNumber: asg.minInstanceNumber,
				maxInstanceNumber: asg.maxInstanceNumber,
			}, nil
		}
	}
	return nil, fmt.Errorf("no auto scaling group found, spec: %s", spec.Name)
}

type groupRule struct {
	ruleName  string
	groupName string
}

func groupFromString(value string) (*groupRule, error) {
	if !strings.Contains(value, "---") {
		// 兼容老数据
		return &groupRule{
			groupName: value,
		}, nil
	}
	tokens := strings.Split(value, "---")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("wrong group configuration: %s", value)
	}
	return &groupRule{
		ruleName:  tokens[0],
		groupName: tokens[1],
	}, nil
}

// BuildCucloud is called by the autoscaler/cluster-autoscaler/builder to build a cucloud cloud provider.
func BuildCucloud(opts config.AutoscalingOptions, do cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	if len(opts.CloudConfig) == 0 {
		klog.Fatalf("cloud config is missing.")
	}

	return newCucloudProvider(opts, do, rl)
}
