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

package slicing

import (
	"context"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/nvml"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/util"
)

type tsClient struct {
	resourceClient resource.Client
	nvmlClient     nvml.Client
}

func NewClient(resourceClient resource.Client, nvmlClient nvml.Client) gpu.Client {
	return &tsClient{
		resourceClient: resourceClient,
		nvmlClient:     nvmlClient,
	}
}

func (c tsClient) GetDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	// Get used
	used, err := c.GetUsedDevices(ctx)
	if err != nil {
		return nil, err
	}
	// Get allocatable
	allocatable, err := c.GetAllocatableDevices(ctx)
	if err != nil {
		return nil, err
	}
	// Get free
	free := gpu.ComputeFreeDevicesAndUpdateStatus(used, allocatable)

	return append(used, free...), nil
}

func (c tsClient) GetUsedDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	// Fetch used devices
	usedResources, err := c.resourceClient.GetUsedDevices(ctx)
	if err != nil {
		return nil, gpu.NewGenericError(err)
	}
	// Consider only NVIDIA GPUs
	var isNvidiaResource = func(d resource.Device) bool {
		return d.IsNvidiaResource()
	}
	usedGpus := util.Filter(usedResources, isNvidiaResource)
	// Convert to gpu.DeviceList
	return c.toGpuDeviceList(usedGpus)
}

func (c tsClient) GetAllocatableDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	// Fetch allocatable devices
	allocatableResources, err := c.resourceClient.GetAllocatableDevices(ctx)
	if err != nil {
		return nil, gpu.NewGenericError(err)
	}
	// Consider only NVIDIA GPUs
	var isNvidiaResource = func(d resource.Device) bool {
		return d.IsNvidiaResource()
	}
	allocatableGPUs := util.Filter(allocatableResources, isNvidiaResource)
	// Extract MIG devices
	return c.toGpuDeviceList(allocatableGPUs)
}

func (c tsClient) toGpuDeviceList(resources []resource.Device) (gpu.DeviceList, gpu.Error) {
	var res = make(gpu.DeviceList, len(resources))
	for i, r := range resources {
		id := ExtractGpuId(r.DeviceId)
		index, err := c.nvmlClient.GetGpuIndex(id)
		if err != nil {
			return nil, err
		}
		device := gpu.Device{
			Device: resource.Device{
				ResourceName: r.ResourceName,
				DeviceId:     r.DeviceId,
				Status:       r.Status,
			},
			GpuIndex: index,
		}
		res[i] = device
	}
	return res, nil
}
