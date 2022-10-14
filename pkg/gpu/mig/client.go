package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
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
	GetMigDeviceResources(ctx context.Context) ([]types.MigDeviceResource, error)
	CreateMigResource(ctx context.Context, profile types.MigProfile) (types.MigDeviceResource, error)
	DeleteMigResource(ctx context.Context, resource types.MigDeviceResource) error
}

type nvmlMigClient struct {
	lister     pdrv1.PodResourcesListerClient
	nvmlClient nvml.Client
}

func NewNvmlMigClient(lister pdrv1.PodResourcesListerClient, nvmlClient nvml.Client) Client {
	return &nvmlMigClient{lister: lister, nvmlClient: nvmlClient}
}

func (c nvmlMigClient) CreateMigResource(ctx context.Context, profile types.MigProfile) (types.MigDeviceResource, error) {
	return types.MigDeviceResource{}, nil
}

func (c nvmlMigClient) DeleteMigResource(_ context.Context, resource types.MigDeviceResource) error {
	return c.nvmlClient.DeleteMigDevice(resource.DeviceId)
}

func (c nvmlMigClient) GetMigDeviceResources(ctx context.Context) ([]types.MigDeviceResource, error) {
	logger := klog.FromContext(ctx)

	// Get used
	used, err := c.getUsedMigDeviceResources(ctx)
	if err != nil {
		logger.Error(err, "unable to retrieve used MIG devices")
		return nil, err
	}
	// Get allocatable
	allocatable, err := c.getAllocatableMigDeviceResources(ctx)
	if err != nil {
		logger.Error(err, "unable to retrieve allocatable MIG devices")
		return nil, err
	}
	// Get free
	free := computeFreeDevicesAndUpdateStatus(used, allocatable)

	return append(used, free...), nil
}

func (c nvmlMigClient) getUsedMigDeviceResources(ctx context.Context) ([]types.MigDeviceResource, error) {
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
		if r.isMigDevice() {
			migResources = append(migResources, r)
		}
	}

	// Retrieve MIG device ID and GPU index
	migDevices := make([]types.MigDeviceResource, 0)
	for _, r := range migResources {
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.deviceId)
		if err != nil {
			logger.Error(
				err,
				"unable to fetch GPU index of MIG resource",
				"resourceName",
				r.resourceName,
			)
			return nil, err
		}
		migDevice := types.MigDeviceResource{
			Device: resource.Device{
				ResourceName: r.resourceName,
				DeviceId:     r.deviceId,
				Status:       resource.StatusUsed,
			},
			GpuIndex: gpuIndex,
		}
		migDevices = append(migDevices, migDevice)
	}

	return migDevices, nil
}

func (c nvmlMigClient) getAllocatableMigDeviceResources(ctx context.Context) ([]types.MigDeviceResource, error) {
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

	return c.extractMigDevices(ctx, resources)
}

func (c nvmlMigClient) extractMigDevices(ctx context.Context, resources []resourceWithDeviceId) ([]types.MigDeviceResource, error) {
	logger := klog.FromContext(ctx)

	// Extract MIG devices
	migResources := make([]resourceWithDeviceId, 0)
	for _, r := range resources {
		if r.isMigDevice() {
			migResources = append(migResources, r)
		}
	}

	// Retrieve MIG device ID and GPU index
	migDevices := make([]types.MigDeviceResource, 0)
	for _, r := range migResources {
		gpuIndex, err := c.nvmlClient.GetGpuIndex(r.deviceId)
		if err != nil {
			logger.Error(
				err,
				"unable to fetch GPU index",
				"migResourceName",
				r.resourceName,
				"migUUID",
				r.deviceId,
			)
			return nil, err
		}
		migDevice := types.MigDeviceResource{
			Device: resource.Device{
				ResourceName: r.resourceName,
				DeviceId:     r.deviceId,
				Status:       resource.StatusUnknown,
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

func computeFreeDevicesAndUpdateStatus(used []types.MigDeviceResource, allocatable []types.MigDeviceResource) []types.MigDeviceResource {
	usedLookup := make(map[string]types.MigDeviceResource)
	for _, u := range used {
		usedLookup[u.DeviceId] = u
	}

	// Compute (allocatable - used)
	res := make([]types.MigDeviceResource, 0)
	for _, a := range allocatable {
		if _, used := usedLookup[a.DeviceId]; !used {
			a.Status = resource.StatusFree
			res = append(res, a)
		}
	}
	return res
}
