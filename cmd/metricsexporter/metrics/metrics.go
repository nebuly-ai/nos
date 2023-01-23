package metrics

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
)

type Node struct {
	Name     string `json:"name"`
	Capacity map[string]string
	Labels   map[string]string
	NodeInfo v1.NodeSystemInfo
}

type ComponentToggle struct {
	GpuPartitioner bool `json:"nos-gpu-partitioner"`
	Scheduler      bool `json:"nos-scheduler"`
	Operator       bool `json:"nos-operator"`
}

type Metrics struct {
	InstallationUUID string          `json:"installationUUID"`
	Nodes            []Node          `json:"nodes"`
	ChartValues      json.RawMessage `json:"chartValues"`
	Components       ComponentToggle `json:"components"`
}
