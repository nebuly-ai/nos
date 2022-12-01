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
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"strings"
)

type resourceWithDeviceId struct {
	resourceName v1.ResourceName
	deviceId     string
}

func (r resourceWithDeviceId) isMigDevice() bool {
	return IsNvidiaMigDevice(r.resourceName)
}

type Client interface {
	GetMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	GetUsedMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	GetAllocatableMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error)
	CreateMigResources(ctx context.Context, profileList ProfileList) (ProfileList, error)
	DeleteMigResource(ctx context.Context, resource DeviceResource) gpu.Error
	DeleteAllExcept(ctx context.Context, resources DeviceResourceList) error
}

type clientImpl struct {
	lister     pdrv1.PodResourcesListerClient
	nvmlClient nvml.Client
}

func NewClient(lister pdrv1.PodResourcesListerClient, nvmlClient nvml.Client) Client {
	return &clientImpl{lister: lister, nvmlClient: nvmlClient}
}

// CreateMigResources creates the MIG resources provided as argument, which can span multiple GPUs, and returns
// the resources that were actually created.
//
// If any error happens, and it is not possible to create the required resources on a certain GPUs,
// CreateMigResources still tries to create the resources on the other GPUs and returns the ones that
// it possible to create. This means that if any error happens, the returned ProfileList will be a subset
// of the input list, otherwise the two lists will have the same length and items.
func (c clientImpl) CreateMigResources(ctx context.Context, profileList ProfileList) (ProfileList, error) {
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
	logger := klog.FromContext(ctx)

	// List Pods Resources
	listResp, err := c.lister.List(ctx, &pdrv1.ListPodResourcesRequest{})
	if err != nil {
		logger.Error(err, "unable to list resources used by running Pods from Kubelet gRPC socket")
		return nil, gpu.NewGenericError(err)
	}

	// Extract GPUs as resourceName + deviceId
	resources, err := fromListRespToGPUResourceWithDeviceId(listResp)
	if err != nil {
		logger.Error(err, "unable parse resources used by running pods")
		return nil, gpu.NewGenericError(err)
	}

	// Extract MIG devices
	return c.extractMigDevices(ctx, resources, resource.StatusUsed)
}

func (c clientImpl) GetAllocatableMigDeviceResources(ctx context.Context) (DeviceResourceList, gpu.Error) {
	logger := klog.FromContext(ctx)

	// List Allocatable Resources
	resp, err := c.lister.GetAllocatableResources(ctx, &pdrv1.AllocatableResourcesRequest{})
	if err != nil {
		logger.Error(err, "unable to get allocatable resources from Kubelet gRPC socket")
		return nil, gpu.NewGenericError(err)
	}

	// Extract GPUs as resourceName + deviceId
	resources := make([]resourceWithDeviceId, 0)
	for _, d := range resp.GetDevices() {
		// Consider only NVIDIA GPUs
		if !strings.HasPrefix(d.GetResourceName(), "nvidia.com/") {
			continue
		}
		// Check devices length
		if len(d.DeviceIds) != 1 {
			err := fmt.Errorf(
				"GPU resource %s should be associated with only 1 device, found %d: this should never happen",
				d.GetResourceName(),
				len(d.DeviceIds),
			)
			return nil, gpu.NewGenericError(err)
		}
		res := resourceWithDeviceId{
			resourceName: v1.ResourceName(d.GetResourceName()),
			deviceId:     d.DeviceIds[0],
		}
		resources = append(resources, res)
	}

	return c.extractMigDevices(ctx, resources, resource.StatusUnknown)
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

func (c clientImpl) extractMigDevices(ctx context.Context, resources []resourceWithDeviceId, devicesStatus resource.Status) ([]DeviceResource, gpu.Error) {
	logger := klog.FromContext(ctx)

	// Extract MIG devices
	migResources := make([]resourceWithDeviceId, 0)
	for _, r := range resources {
		if r.isMigDevice() {
			migResources = append(migResources, r)
		}
	}

	// Retrieve MIG device ID and GPU index
	migDevices := make([]DeviceResource, 0)
	for _, r := range migResources {
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.deviceId)
		if err.IsNotFound() {
			logger.V(1).Info("could not find GPU index of MIG device", "MIG device ID", r.deviceId)
			continue
		}
		if err != nil {
			logger.Error(
				err,
				"unable to fetch GPU index",
				"resourceName",
				r.resourceName,
				"MIG device ID",
				r.deviceId,
			)
			return nil, err
		}
		migDevice := DeviceResource{
			Device: resource.Device{
				ResourceName: r.resourceName,
				DeviceId:     r.deviceId,
				Status:       devicesStatus,
			},
			GpuIndex: gpuIndex,
		}
		migDevices = append(migDevices, migDevice)
	}

	return migDevices, nil
}

func fromListRespToGPUResourceWithDeviceId(listResp *pdrv1.ListPodResourcesResponse) ([]resourceWithDeviceId, error) {
	result := make([]resourceWithDeviceId, 0)
	for _, r := range listResp.PodResources {
		for _, cr := range r.Containers {
			for _, cd := range cr.GetDevices() {
				// Consider only NVIDIA GPUs
				if !strings.HasPrefix(cd.GetResourceName(), "nvidia.com/") {
					continue
				}
				// Check devices length
				if len(cd.DeviceIds) != 1 {
					err := fmt.Errorf(
						"GPU resource %s should be associated with only 1 device, found %d: this should never happen",
						cd.GetResourceName(),
						len(cd.DeviceIds),
					)
					return nil, err
				}
				resWithId := resourceWithDeviceId{
					deviceId:     cd.DeviceIds[0],
					resourceName: v1.ResourceName(cd.GetResourceName()),
				}
				result = append(result, resWithId)
			}
		}
	}
	return result, nil
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
