/*
 * Copyright 2023 nebuly.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
