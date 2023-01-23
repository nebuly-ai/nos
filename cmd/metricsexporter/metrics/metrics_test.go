package metrics

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
	"strconv"
	"testing"
)

func TestMetrics_YamlDeserialization(t *testing.T) {
	testCases := []struct {
		input          string
		expectedOutput Metrics
		err            bool
	}{
		{
			input:          "",
			expectedOutput: Metrics{},
			err:            false,
		},
		{
			input: `
installationUUID: feb0a960-ed22-4882-96cf-ef0b83deaeb1
nodes:
- name: node-1
  capacity:
    cpu: 5
    memory: 7111996Ki
  labels:
    nvidia.com/gpu: true
  nodeInfo:
    architecture: arm64
    containerRuntimeVersion: containerd://1.6.7
    kernelVersion: 5.15.49-linuxkit
    osImage: Ubuntu 22.04.1 LTS
    kubeletVersion: v1.24.4
- name: node-2
  capacity:
    cpu: 2
    memory: 7111996Ki
  labels:
  nodeInfo:
    architecture: arm64
    containerRuntimeVersion: containerd://1.6.7
    kernelVersion: 5.15.49-linuxkit
    osImage: Ubuntu 22.04.1 LTS
    kubeletVersion: v1.24.4
chartValues:
  allowDefaultNamespace: false
  global:
    nvidiaGpuResourceMemoryGB: 32
components:
  nos-gpu-partitioner: true
  nos-operator: true
  nos-scheduler: true
`,
			expectedOutput: Metrics{
				InstallationUUID: "feb0a960-ed22-4882-96cf-ef0b83deaeb1",
				Nodes: []Node{
					{
						Name: "node-1",
						Capacity: map[string]string{
							"cpu":    "5",
							"memory": "7111996Ki",
						},
						Labels: map[string]string{
							"nvidia.com/gpu": "true",
						},
						NodeInfo: v1.NodeSystemInfo{
							Architecture:            "arm64",
							ContainerRuntimeVersion: "containerd://1.6.7",
							KernelVersion:           "5.15.49-linuxkit",
							OSImage:                 "Ubuntu 22.04.1 LTS",
							KubeletVersion:          "v1.24.4",
						},
					},
					{
						Name: "node-2",
						Capacity: map[string]string{
							"cpu":    "2",
							"memory": "7111996Ki",
						},
						Labels: nil,
						NodeInfo: v1.NodeSystemInfo{
							Architecture:            "arm64",
							ContainerRuntimeVersion: "containerd://1.6.7",
							KernelVersion:           "5.15.49-linuxkit",
							OSImage:                 "Ubuntu 22.04.1 LTS",
							KubeletVersion:          "v1.24.4",
						},
					},
				},
				ChartValues: []byte(`{"allowDefaultNamespace":false,"global":{"nvidiaGpuResourceMemoryGB":32}}`),
				Components: ComponentToggle{
					GpuPartitioner: true,
					Scheduler:      true,
					Operator:       true,
				},
			},
			err: false,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var m = Metrics{}
			err := yaml.Unmarshal([]byte(tc.input), &m)
			if tc.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, m)
		})
	}
}
