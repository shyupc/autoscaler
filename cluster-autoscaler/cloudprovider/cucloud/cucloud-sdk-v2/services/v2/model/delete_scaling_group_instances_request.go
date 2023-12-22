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

package model

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/util"
)

type DeleteScalingGroupInstancesRequest struct {
	GroupId     string    `path:"group_id"`
	InstanceIds []string  `body:"instanceIds"`
	RuleName    string    `body:"rule_name"`
	ScaleType   ScaleType `body:"scale_type"`
}

func (r *DeleteScalingGroupInstancesRequest) BodyParam() RequestParam {
	return util.TransferToMapWithTag(r, tagBody)
}

func (r *DeleteScalingGroupInstancesRequest) PathParam() RequestParam {
	return util.TransferToMapWithTag(r, tagPath)
}

func (r *DeleteScalingGroupInstancesRequest) QueryParam() RequestParam {
	return nil
}
