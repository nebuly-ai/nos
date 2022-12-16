/*
 * Copyright 2022 Nebuly.ai
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

package timeslicing_test

import (
	deviceplugin "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestNewNode(t *testing.T) {
	testCases := []struct {
		name        string
		node        v1.Node
		config      deviceplugin.TimeSlicing
		expected    timeslicing.Node
		errExpected bool
	}{
		{
			name: "node without GPU count label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaMemory:  "10",
			}).Get(),
			errExpected: true,
		},
		{
			name: "node without GPU memory label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "1",
			}).Get(),
			errExpected: true,
		},
		{
			name: "node without GPU model label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaCount:  "1",
				constant.LabelNvidiaMemory: "1",
			}).Get(),
			errExpected: true,
		},
		{
			name: "config is empty, returned node should have all GPUs with just 1 replica",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "3", // Number of returned GPUs
				constant.LabelNvidiaMemory:  "2000",
			}).Get(),
			config:      deviceplugin.TimeSlicing{Resources: make([]deviceplugin.ReplicatedResource, 0)},
			errExpected: false,
			expected: timeslicing.Node{
				Name: "node-1",
				GPUs: []timeslicing.GPU{
					{
						Model:    "foo",
						Index:    0,
						Replicas: 1,
						MemoryGB: 2,
					},
					{
						Model:    "foo",
						Index:    1,
						Replicas: 1,
						MemoryGB: 2,
					},
					{
						Model:    "foo",
						Index:    2,
						Replicas: 1,
						MemoryGB: 2,
					},
				},
			},
		},
		{
			name: "GPUs not specified in config should be added with 1 replica",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "3", // Number of returned GPUs
				constant.LabelNvidiaMemory:  "2000",
			}).Get(),
			config: deviceplugin.TimeSlicing{Resources: []deviceplugin.ReplicatedResource{
				{
					Name:   "nvidia.com/gpu",
					Rename: "",
					Devices: deviceplugin.ReplicatedDevices{
						List: []deviceplugin.ReplicatedDeviceRef{"0", "1"},
					},
					Replicas: 2,
				},
			}},
			expected: timeslicing.Node{
				Name: "node-1",
				GPUs: []timeslicing.GPU{
					{
						Model:    "foo",
						Index:    0,
						Replicas: 2,
						MemoryGB: 2,
					},
					{
						Model:    "foo",
						Index:    1,
						Replicas: 2,
						MemoryGB: 2,
					},
					{
						Model:    "foo",
						Index:    2,
						Replicas: 1,
						MemoryGB: 2,
					},
				},
			},
		},
		{
			name: "config uses UUID instead of index",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "3", // Number of returned GPUs
				constant.LabelNvidiaMemory:  "2000",
			}).Get(),
			config: deviceplugin.TimeSlicing{Resources: []deviceplugin.ReplicatedResource{
				{
					Name:   "nvidia.com/gpu",
					Rename: "",
					Devices: deviceplugin.ReplicatedDevices{
						List: []deviceplugin.ReplicatedDeviceRef{"uuid-1"},
					},
					Replicas: 2,
				},
			}},
			errExpected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			node, err := timeslicing.NewNode(tt.node, tt.config)
			if tt.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, node.Name)
				assert.ElementsMatch(t, tt.expected.GPUs, node.GPUs)
			}
		})
	}
}
