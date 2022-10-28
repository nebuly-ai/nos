package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/test/gpu/nvml"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"testing"
)

type MockedPodResourcesListerClient struct {
	ListResp            pdrv1.ListPodResourcesResponse
	ListError           error
	GetAllocatableResp  pdrv1.AllocatableResourcesResponse
	GetAllocatableError error
}

func (c MockedPodResourcesListerClient) List(
	_ context.Context,
	_ *pdrv1.ListPodResourcesRequest,
	_ ...grpc.CallOption) (*pdrv1.ListPodResourcesResponse, error) {
	return &c.ListResp, c.ListError
}
func (c MockedPodResourcesListerClient) GetAllocatableResources(
	_ context.Context,
	_ *pdrv1.AllocatableResourcesRequest,
	_ ...grpc.CallOption) (*pdrv1.AllocatableResourcesResponse, error) {
	return &c.GetAllocatableResp, c.GetAllocatableError
}

func TestClient_GetUsedMigDevices(t *testing.T) {
	testCases := []struct {
		name                 string
		listPodResourcesResp pdrv1.ListPodResourcesResponse
		listPodResourcesErr  error
		getGpuIndexErr       error
		deviceIdToGPUIndex   map[string]int

		expectedError   bool
		expectedDevices []DeviceResource
	}{
		{
			name:                 "Empty list pod resources resp",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{},
			expectedError:        false,
			expectedDevices:      make([]DeviceResource, 0),
		},
		{
			name:                 "List pod resources returns error",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{},
			listPodResourcesErr:  fmt.Errorf("error"),
			expectedError:        true,
		},
		{
			name: "List pod resources returns a GPU associated with many device IDs",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{
				PodResources: []*pdrv1.PodResources{
					{
						Name:      "pod-1",
						Namespace: "ns-1",
						Containers: []*pdrv1.ContainerResources{
							{
								Name: "container-2",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nebuly.ai/custom-resource",
										DeviceIds:    []string{"1", "2"},
									},
								},
							},
							{
								Name: "container-1",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nvidia.com/gpu",
										DeviceIds:    []string{"1", "2"},
									},
									{
										ResourceName: "nvidia.com/another-gpu",
										DeviceIds:    []string{"1"},
									},
								},
							},
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "No GPU resources",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{
				PodResources: []*pdrv1.PodResources{
					{
						Name:      "pod-1",
						Namespace: "ns-1",
						Containers: []*pdrv1.ContainerResources{
							{
								Name: "container-2",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nebuly.ai/custom-resource",
										DeviceIds:    []string{"1", "2"},
									},
								},
							},
							{
								Name: "container-1",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "k8s.io/some-resource",
										DeviceIds:    []string{"1", "2"},
									},
									{
										ResourceName: "k8s.io/another-resource",
										DeviceIds:    []string{"1"},
									},
								},
							},
						},
					},
				},
			},
			expectedError:   false,
			expectedDevices: make([]DeviceResource, 0),
		},
		{
			name: "Error fetching Mig device GPU index",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{
				PodResources: []*pdrv1.PodResources{
					{
						Name:      "pod-1",
						Namespace: "ns-1",
						Containers: []*pdrv1.ContainerResources{
							{
								Name: "container-1",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nvidia.com/mig-2g.10gb",
										DeviceIds:    []string{"1"},
									},
								},
							},
						},
					},
				},
			},
			getGpuIndexErr: fmt.Errorf("error"),
			expectedError:  true,
		},
		{
			name: "Multiple GPUs, multiple MIG devices",
			listPodResourcesResp: pdrv1.ListPodResourcesResponse{
				PodResources: []*pdrv1.PodResources{
					{
						Name:      "pod-1",
						Namespace: "ns-1",
						Containers: []*pdrv1.ContainerResources{
							{
								Name: "container-1",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nvidia.com/gpu",
										DeviceIds:    []string{"gpu-1"},
									},
								},
							},
							{
								Name: "container-2",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nvidia.com/mig-2g.10gb",
										DeviceIds:    []string{"mig-device-1"},
									},
									{
										ResourceName: "nvidia.com/gpu",
										DeviceIds:    []string{"gpu-2"},
									},
								},
							},
						},
					},
					{
						Name:      "pod-2",
						Namespace: "ns-1",
						Containers: []*pdrv1.ContainerResources{
							{
								Name: "container-2",
								Devices: []*pdrv1.ContainerDevices{
									{
										ResourceName: "nvidia.com/mig-2g.20gb",
										DeviceIds:    []string{"mig-device-2"},
									},
									{
										ResourceName: "nvidia.com/mig-2g.20gb",
										DeviceIds:    []string{"mig-device-3"},
									},
									{
										ResourceName: "nvidia.com/gpu",
										DeviceIds:    []string{"gpu-3"},
									},
								},
							},
						},
					},
				},
			},
			deviceIdToGPUIndex: map[string]int{
				"mig-device-1": 1,
				"mig-device-2": 2,
				"mig-device-3": 2,
			},
			expectedDevices: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-2g.10gb",
						DeviceId:     "mig-device-1",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-2g.20gb",
						DeviceId:     "mig-device-2",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 2,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-2g.20gb",
						DeviceId:     "mig-device-3",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nvmlClient := nvml.MockedNvmlClient{
				MigDeviceIdToGPUIndex: tt.deviceIdToGPUIndex,
				ReturnedError:         tt.getGpuIndexErr,
			}
			podResourcesListerClient := MockedPodResourcesListerClient{
				ListResp:  tt.listPodResourcesResp,
				ListError: tt.listPodResourcesErr,
			}
			client := nvmlMigClient{
				lister:     podResourcesListerClient,
				nvmlClient: nvmlClient,
			}

			usedDevices, err := client.getUsedMigDeviceResources(context.TODO())
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.ElementsMatch(t, tt.expectedDevices, usedDevices)
				assert.Nil(t, err)
			}
		})
	}
}

func TestClient_GetAllocatableMigDevices(t *testing.T) {
	testCases := []struct {
		name                     string
		allocatableResourcesResp pdrv1.AllocatableResourcesResponse
		allocatableResourcesErr  error
		getGpuIndexErr           error
		deviceIdToGPUIndex       map[string]int

		expectedError   bool
		expectedDevices []DeviceResource
	}{
		{
			name:                     "Empty allocatable resources resp",
			allocatableResourcesResp: pdrv1.AllocatableResourcesResponse{},
			expectedError:            false,
			expectedDevices:          make([]DeviceResource, 0),
		},
		{
			name:                     "Allocatable resources returns error",
			allocatableResourcesResp: pdrv1.AllocatableResourcesResponse{},
			allocatableResourcesErr:  fmt.Errorf("error"),
			expectedError:            true,
		},
		{
			name: "List pod resources returns a GPU associated with many device IDs",
			allocatableResourcesResp: pdrv1.AllocatableResourcesResponse{
				Devices: []*pdrv1.ContainerDevices{
					{
						ResourceName: "nvidia.com/gpu",
						DeviceIds:    []string{"1", "2"},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Error fetching MIG resource GPU index",
			allocatableResourcesResp: pdrv1.AllocatableResourcesResponse{
				Devices: []*pdrv1.ContainerDevices{
					{
						ResourceName: "nvidia.com/mig-2g.10gb",
						DeviceIds:    []string{"1"},
					},
				},
			},
			getGpuIndexErr: fmt.Errorf("error"),
			expectedError:  true,
		},
		{
			name: "Multiple GPUs, multiple MIG devices",
			allocatableResourcesResp: pdrv1.AllocatableResourcesResponse{
				Devices: []*pdrv1.ContainerDevices{
					{
						ResourceName: "nvidia.com/gpu",
						DeviceIds:    []string{"1"},
					},
					{
						ResourceName: "nebuly.ai/custom-resource",
						DeviceIds:    []string{"9"},
					},
					{
						ResourceName: "nvidia.com/gpu",
						DeviceIds:    []string{"2"},
					},
					{
						ResourceName: "nvidia.com/mig-1g.20gb",
						DeviceIds:    []string{"mig-1"},
					},
					{
						ResourceName: "nvidia.com/mig-1g.20gb",
						DeviceIds:    []string{"mig-2"},
					},
					{
						ResourceName: "nvidia.com/mig-1g.10gb",
						DeviceIds:    []string{"mig-3"},
					},
				},
			},
			deviceIdToGPUIndex: map[string]int{
				"mig-1": 1,
				"mig-2": 1,
				"mig-3": 2,
			},
			expectedDevices: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-1g.20gb",
						DeviceId:     "mig-1",
						Status:       resource.StatusUnknown,
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-1g.20gb",
						DeviceId:     "mig-2",
						Status:       resource.StatusUnknown,
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-1g.10gb",
						DeviceId:     "mig-3",
						Status:       resource.StatusUnknown,
					},
					GpuIndex: 2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nvmlClient := nvml.MockedNvmlClient{
				MigDeviceIdToGPUIndex: tt.deviceIdToGPUIndex,
				ReturnedError:         tt.getGpuIndexErr,
			}
			podResourcesListerClient := MockedPodResourcesListerClient{
				GetAllocatableResp:  tt.allocatableResourcesResp,
				GetAllocatableError: tt.allocatableResourcesErr,
			}
			client := nvmlMigClient{
				lister:     podResourcesListerClient,
				nvmlClient: nvmlClient,
			}

			usedDevices, err := client.getAllocatableMigDeviceResources(context.TODO())
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.ElementsMatch(t, tt.expectedDevices, usedDevices)
				assert.Nil(t, err)
			}
		})
	}
}

func TestComputeFreeDevicesAndUpdateStatus(t *testing.T) {
	testCases := []struct {
		name        string
		used        []DeviceResource
		allocatable []DeviceResource
		expected    []DeviceResource
	}{
		{
			name:        "empty used, empty allocatable",
			used:        make([]DeviceResource, 0),
			allocatable: make([]DeviceResource, 0),
			expected:    make([]DeviceResource, 0),
		},
		{
			name: "Used devices, empty allocatable",
			used: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "0",
					},
					GpuIndex: 0,
				},
			},
			allocatable: make([]DeviceResource, 0),
			expected:    make([]DeviceResource, 0),
		},
		{
			name: "Allocatable devices, empty used",
			used: []DeviceResource{},
			allocatable: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "0",
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "1",
					},
					GpuIndex: 1,
				},
			},
			expected: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "0",
						Status:       resource.StatusFree,
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 1,
				},
			},
		},
		{
			name: "Multiple used, multiple allocatable",
			used: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "0",
					},
					GpuIndex: 0,
				},
			},
			allocatable: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "0",
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "1",
					},
					GpuIndex: 1,
				},
			},
			expected: []DeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 1,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			devices := computeFreeDevicesAndUpdateStatus(tt.used, tt.allocatable)
			assert.ElementsMatch(t, tt.expected, devices)
		})
	}
}
