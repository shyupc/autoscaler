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
	"io"
	"os"

	"gopkg.in/gcfg.v1"
	cucloudsdkv2auth "k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/auth"
	cucloudsdkv2 "k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/services/v2"
)

type CloudConfig struct {
	Global struct {
		Endpoint     string `gcfg:"endpoint"`
		AccessKey    string `gcfg:"access-key"`
		SecretKey    string `gcfg:"secret-key"`
		CkeClusterId string `gcfg:"ckecluster-id"`
		RegionId     string `gcfg:"region-id"`
	}
}

func (c *CloudConfig) validate() error {
	if len(c.Global.Endpoint) == 0 {
		return fmt.Errorf("endpoint missing from cloud configuration")
	}

	if len(c.Global.AccessKey) == 0 {
		return fmt.Errorf("access key missing from cloud coniguration")
	}

	if len(c.Global.SecretKey) == 0 {
		return fmt.Errorf("secret key missing from cloud configuration")
	}

	if len(c.Global.RegionId) == 0 {
		return fmt.Errorf("region id missing from cloud configuration")
	}

	if len(c.Global.CkeClusterId) == 0 {
		return fmt.Errorf("cke cluster id missing from cloud configuration")
	}

	return nil
}

func readConf(confFile string) (*CloudConfig, error) {
	var conf io.ReadCloser
	conf, err := os.Open(confFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %s, error: %v", confFile, err)
	}
	defer conf.Close()

	var cloudConfig CloudConfig
	if err := gcfg.ReadInto(&cloudConfig, conf); err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %s, error: %v", confFile, err)
	}

	return &cloudConfig, nil
}

func (c *CloudConfig) getApiClient() (*cucloudsdkv2.ApiClient, error) {
	credential := cucloudsdkv2auth.NewCredential(c.Global.AccessKey, c.Global.SecretKey, c.Global.Endpoint)
	return cucloudsdkv2.NewApiClient(credential, c.Global.CkeClusterId, c.Global.RegionId)
}
