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
	"errors"
	"fmt"
	"strings"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/auth"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/services/v2/model"
)

const (
	listScalingGroupsUri           = "/csk-open/api/csk/{region_id}/clusters/{ckecluster_id}/scalingGroups"
	describeScalingGroupUri        = "/csk-open/api/csk/{region_id}/clusters/{ckecluster_id}/scalingGroups/{group_id}"
	deleteScalingGroupInstancesUri = "/csk-open/api/csk/{region_id}/clusters/{ckecluster_id}/scalingGroups/{group_id}/instances"
	createScalingGroupInstancesUri = "/csk-open/api/csk/{region_id}/clusters/{ckecluster_id}/scalingGroups/{group_id}/instances"
)

type ApiClient struct {
	credential   *auth.Credential
	regionId     string
	ckeClusterId string
}

func NewApiClient(credential *auth.Credential, ckeClusterId, regionId string) (*ApiClient, error) {
	if credential == nil {
		return nil, errors.New("authentication is not supplied for the client")
	}
	return &ApiClient{
		credential:   credential,
		regionId:     regionId,
		ckeClusterId: ckeClusterId,
	}, nil
}

func (api *ApiClient) Client() *core.HttpClient {
	return core.NewHttpClient(api.credential)
}

func (api *ApiClient) RenderUri(uri string, request model.CommonRequest) string {
	uri = strings.ReplaceAll(uri, "{region_id}", api.regionId)
	uri = strings.ReplaceAll(uri, "{ckecluster_id}", api.ckeClusterId)
	if request != nil {
		for k, v := range request.PathParam() {
			uri = strings.ReplaceAll(uri, fmt.Sprintf("{%s}", k), fmt.Sprintf("%v", v))
		}
		queryStr := ""
		for k, v := range request.QueryParam() {
			queryStr += fmt.Sprintf("&%s=%v", k, v)
		}
		if len(queryStr) > 0 {
			uri = fmt.Sprintf("%s?%s", uri, queryStr[1:])
		}
	}
	return uri
}
