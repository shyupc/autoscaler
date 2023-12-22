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

package v2

import (
	"fmt"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/services/v2/model"
)

func (api *ApiClient) DescribeScalingGroup(groupID string) (*model.ScalingGroup, error) {
	// create request body
	request := model.DescribeScalingGroupRequest{
		GroupId: groupID,
	}
	// render uri
	uri := api.RenderUri(describeScalingGroupUri, &request)
	// do request
	resp, err := api.Client().Uri(uri).Get()
	if err != nil {
		return nil, err
	}
	// get result
	var r model.DescribeScalingGroupResponse
	if err = resp.Decode(&r); err != nil {
		return nil, fmt.Errorf("decoding response body failed: %v", err)
	}

	return &r.ScalingGroup, nil
}
