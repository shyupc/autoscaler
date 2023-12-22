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
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/klog/v2"
)

type autoScalingGroups struct {
	registeredAsgs           []*AutoScalingGroup
	instanceToAsg            map[string]*AutoScalingGroup
	cacheMutex               sync.Mutex
	instancesNotInManagedAsg map[string]struct{}
}

func newAutoScalingGroups() *autoScalingGroups {
	registry := &autoScalingGroups{
		instanceToAsg:            make(map[string]*AutoScalingGroup),
		instancesNotInManagedAsg: make(map[string]struct{}),
	}

	return registry
}

// Register asg in CuCloud Manager.
func (a *autoScalingGroups) Register(asg *AutoScalingGroup) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.registeredAsgs = append(a.registeredAsgs, asg)
}

// FindForInstance returns AsgConfig of the given Instance
func (a *autoScalingGroups) FindForInstance(instanceId string, manager *cucloudManager) (cloudprovider.NodeGroup, error) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	if config, found := a.instanceToAsg[instanceId]; found {
		return config, nil
	}
	if _, found := a.instancesNotInManagedAsg[instanceId]; found {
		return nil, nil
	}
	if err := a.regenerateCache(manager); err != nil {
		return nil, err
	}
	if config, found := a.instanceToAsg[instanceId]; found {
		return config, nil
	}
	// instance does not belong to any configured ASG
	a.instancesNotInManagedAsg[instanceId] = struct{}{}
	return nil, nil
}

func (a *autoScalingGroups) generateCache(manager *cucloudManager) {
	go wait.Forever(func() {
		a.cacheMutex.Lock()
		defer a.cacheMutex.Unlock()
		if err := a.regenerateCache(manager); err != nil {
			klog.Errorf("failed to do regenerating ASG cache,because of %s", err.Error())
		}
	}, time.Hour)
}

func (a *autoScalingGroups) regenerateCache(manager *cucloudManager) error {
	newCache := make(map[string]*AutoScalingGroup)

	for _, registeredAsg := range a.registeredAsgs {
		instances, err := manager.GetInstances(registeredAsg.groupID)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			newCache[instance.Id] = registeredAsg
		}
	}

	a.instanceToAsg = newCache
	return nil
}
