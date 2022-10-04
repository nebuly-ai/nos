package gpu

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"strings"
)

type Client struct {
	lister     pdrv1.PodResourcesListerClient
	nvmlClient nvml.Client
}

func (c Client) GetUsedMIGDevices(ctx context.Context) ([]MIGDevice, error) {
	logger := klog.FromContext(ctx)

	// List Pods Resources
	listResp, err := c.lister.List(ctx, &pdrv1.ListPodResourcesRequest{})
	if err != nil {
		logger.Error(err, "unable to list resources used by running Pods from Kubelet gRPC socket")
		return nil, err
	}

	// Extract GPUs as resourceName + deviceId
	resources, err := fromListRespToGPUResourceWithDeviceId(listResp)
	if err != nil {
		logger.Error(err, "unable parse resources used by running pods")
		return nil, err
	}

	// Extract MIG devices
	migResources := make([]resourceWithDeviceId, 0)
	for _, r := range resources {
		if r.isMIGDevice() {
			migResources = append(migResources, r)
		}
	}

	// Retrieve MIG device ID and GPU index
	migDevices := make([]MIGDevice, 0)
	for _, r := range migResources {
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.deviceId)
		if err != nil {
			logger.Error(err, "unable to fetch GPU index of MIG resource %s", r.resourceName)
			return nil, err
		}
		migDevice := MIGDevice{
			Device: Device{
				ResourceName: r.resourceName,
				DeviceId:     r.deviceId,
			},
			GpuIndex: gpuIndex,
		}
		migDevices = append(migDevices, migDevice)
	}

	return migDevices, nil
}

func (c Client) GetFreeMIGDevices(ctx context.Context) ([]MIGDevice, error) {
	logger := klog.FromContext(ctx)

	// Get allocatable
	allocatable, err := c.GetAllocatableMIGDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to retrieve allocatable MIG devices")
		return nil, err
	}
	// Get used
	used, err := c.GetUsedMIGDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to retrieve used MIG devices")
		return nil, err
	}
	usedLookup := make(map[string]MIGDevice)
	for _, u := range used {
		usedLookup[u.DeviceId] = u
	}

	// Compute (allocatable - used)
	res := make([]MIGDevice, 0)
	for _, a := range allocatable {
		if _, used := usedLookup[a.DeviceId]; !used {
			res = append(res, a)
		}
	}

	return res, nil
}

func (c Client) GetAllocatableMIGDevices(ctx context.Context) ([]MIGDevice, error) {
	logger := klog.FromContext(ctx)

	// List Allocatable Resources
	resp, err := c.lister.GetAllocatableResources(ctx, &pdrv1.AllocatableResourcesRequest{})
	if err != nil {
		logger.Error(err, "unable to get allocatable resources from Kubelet gRPC socket")
		return nil, err
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
			return nil, err
		}
		res := resourceWithDeviceId{
			resourceName: v1.ResourceName(d.GetResourceName()),
			deviceId:     d.DeviceIds[0],
		}
		resources = append(resources, res)
	}

	return c.extractMIGDevices(ctx, resources)
}

func (c Client) extractMIGDevices(ctx context.Context, resources []resourceWithDeviceId) ([]MIGDevice, error) {
	logger := klog.FromContext(ctx)

	// Extract MIG devices
	migResources := make([]resourceWithDeviceId, 0)
	for _, r := range resources {
		if r.isMIGDevice() {
			migResources = append(migResources, r)
		}
	}

	// Retrieve MIG device ID and GPU index
	migDevices := make([]MIGDevice, 0)
	for _, r := range migResources {
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.deviceId)
		if err != nil {
			logger.Error(err, "unable to fetch GPU index of MIG resource %s", r.resourceName)
			return nil, err
		}
		migDevice := MIGDevice{
			Device: Device{
				ResourceName: r.resourceName,
				DeviceId:     r.deviceId,
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
