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

package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"k8s.io/klog/v2"
)

type Client interface {
	GetMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	GetUsedMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	GetAllocatableMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	CreateMigResources(ctx context.Context, profileList ProfileList) (ProfileList, error)
	DeleteMigResource(ctx context.Context, resource DeviceResource) gpu.Error
	DeleteAllExcept(ctx context.Context, resources DeviceResourceList) error
}

type clientImpl struct {
	resourceClient resource.Client
	nvmlClient     nvml.Client
}

func NewClient(resourceClient resource.Client, nvmlClient nvml.Client) Client {
	return &clientImpl{
		resourceClient: resourceClient,
		nvmlClient:     nvmlClient,
	}
}

// CreateMigResources creates the MIG resources provided as argument, which can span multiple GPUs, and returns
// the resources that were actually created.
//
// If any error happens, and it is not possible to create the required resources on a certain GPUs,
// CreateMigResources still tries to create the resources on the other GPUs and returns the ones that
// it possible to create. This means that if any error happens, the returned ProfileList will be a subset
// of the input list, otherwise the two lists will have the same length and items.
func (c clientImpl) CreateMigResources(_ context.Context, profileList ProfileList) (ProfileList, error) {
	var errors = make(gpu.ErrorList, 0)
	var createdProfiles = make(ProfileList, 0)
	for gpuIndex, profiles := range profileList.GroupByGPU() {
		profileNames := make([]string, 0)
		for _, p := range profiles {
			profileNames = append(profileNames, p.Name.AsString())
		}
		if err := c.nvmlClient.CreateMigDevices(profileNames, gpuIndex); err != nil {
			errors = append(errors, err)
			continue
		}
		createdProfiles = append(createdProfiles, profiles...)
	}
	if len(errors) > 0 {
		return createdProfiles, errors
	}
	return createdProfiles, nil
}

func (c clientImpl) DeleteMigResource(_ context.Context, resource DeviceResource) gpu.Error {
	return c.nvmlClient.DeleteMigDevice(resource.DeviceId)
}

func (c clientImpl) GetMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error) {
	// Get used
	used, err := c.GetUsedMigDeviceResources(ctx)
	if err != nil {
		return nil, err
	}
	// Get allocatable
	allocatable, err := c.GetAllocatableMigDeviceResources(ctx)
	if err != nil {
		return nil, err
	}
	// Get free
	free := computeFreeDevicesAndUpdateStatus(used, allocatable)

	return append(used, free...), nil
}

func (c clientImpl) GetUsedMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error) {
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

	// Extract MIG devices
	return c.extractMigDevices(ctx, usedGpus)
}

func (c clientImpl) GetAllocatableMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error) {
	// Fetch used devices
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
	return c.extractMigDevices(ctx, allocatableGPUs)
}

// DeleteAllExcept deletes all the devices that are not in the list of devices to keep.
func (c clientImpl) DeleteAllExcept(_ context.Context, resourcesToKeep DeviceResourceList) error {
	nResources := len(resourcesToKeep)
	idsToKeep := make([]string, nResources)
	for i, r := range resourcesToKeep {
		idsToKeep[i] = r.DeviceId
	}
	return c.nvmlClient.DeleteAllMigDevicesExcept(idsToKeep)
}

func (c clientImpl) extractMigDevices(ctx context.Context, devices []resource.Device) ([]DeviceResource, gpu.Error) {
	logger := klog.FromContext(ctx)

	// Retrieve MIG device ID and GPU index
	migDevices := make([]DeviceResource, 0)
	for _, r := range devices {
		if !IsNvidiaMigDevice(r.ResourceName) {
			continue
		}
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.DeviceId)
		if gpu.IgnoreNotFound(err) != nil {
			logger.Error(
				err,
				"unable to fetch GPU index",
				"resourceName",
				r.DeviceId,
				"MIG device ID",
				r.DeviceId,
			)
			return nil, err
		}
		if gpu.IsNotFound(err) {
			logger.V(1).Info("could not find GPU index of MIG device", "MIG device ID", r.DeviceId)
			continue
		}
		migDevice := DeviceResource{
			Device:   r,
			GpuIndex: gpuIndex,
		}
		migDevices = append(migDevices, migDevice)
	}

	return migDevices, nil
}

func computeFreeDevicesAndUpdateStatus(used []DeviceResource, allocatable []DeviceResource) []DeviceResource {
	usedLookup := make(map[string]DeviceResource)
	for _, u := range used {
		usedLookup[u.DeviceId] = u
	}

	// Compute (allocatable - used)
	res := make([]DeviceResource, 0)
	for _, a := range allocatable {
		if _, used := usedLookup[a.DeviceId]; !used {
			a.Status = resource.StatusFree
			res = append(res, a)
		}
	}
	return res
}
