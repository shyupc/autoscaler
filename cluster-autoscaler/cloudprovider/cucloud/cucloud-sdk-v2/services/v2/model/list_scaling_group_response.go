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

type ListScalingGroupResponse struct {
	Total int            `json:"total"`
	Items []ScalingGroup `json:"items"`
}

type ScalingGroup struct {
	ID           int                 `json:"id"`
	CkeClusterId string              `json:"cke_cluster_id"`
	RegionsId    string              `json:"regions_id"`
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	NodeNum      int                 `json:"node_num"`
	NodeRealNum  int                 `json:"node_real_num"`
	DiskConfig   *DiskConfig         `json:"disk"`
	NodeConfig   *NodeConfig         `json:"node_config"`
	AutoScale    *AutoScale          `json:"auto_scale"`
	ScaleNodes   []ScaleNodeInstance `json:"scale_node"`
}

type DiskConfig struct {
	SystemDisk Disk `json:"system_disk"`
	DataDisk   Disk `json:"data_disk"`
}

type Disk struct {
	DiskCapacity int    `json:"disk_capacity"`
	DiskType     string `json:"disk_type"`
	DiskId       string `json:"disk_id"`
	ProductId    string `json:"productId"`
}

type NodeConfig struct {
	Res       NodeRes     `json:"res"`
	ProductId string      `json:"productId"`
	Runtime   NodeRuntime `json:"runtime"`
	System    string      `json:"system"`
	Password  string      `json:"password"`
}

type NodeRes struct {
	TypeName   string            `json:"type_name"`
	TypeId     string            `json:"type_id"`
	ResourceId string            `json:"resource_id"`
	Resource   ResourceReference `json:"resource"`
}

type ResourceReference struct {
	Cpu int64 `json:"cpu"`
	Mem int64 `json:"mem"`
}

type NodeRuntime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type AutoScale struct {
	NodeMax     int `json:"node_max"`
	NodeMin     int `json:"node_min"`
	Priority    int `json:"priority"`
	CoolingTime int `json:"cooling_time"`
}

type NodeStatus string

const (
	NodeStatusRunning  NodeStatus = "running"
	NodeStatusDeleting NodeStatus = "deleting"
	NodeStatusDeleted  NodeStatus = "deleted"
	NodeStatusCreating NodeStatus = "creating"
	NodeStatusPending  NodeStatus = "pending"
	NodeStatusFailed   NodeStatus = "failed"
)

type ScaleNodeInstance struct {
	ProviderID string     `json:"providerID"`
	Status     NodeStatus `json:"status"`
}
