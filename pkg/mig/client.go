package mig

import (
	"context"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"strings"
)

type NvmlClient struct {
}

func NewClient() (NvmlClient, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return NvmlClient{}, fmt.Errorf("unable to initialize NVML: %s", nvml.ErrorString(ret))
	}
	return NvmlClient{}, nil
}

// GetGpuIndex returns the index of the GPU associated to the
// MIG device provided as arg. Returns err if the device
// is not found or any error occurs while retrieving it.
func (c NvmlClient) GetGpuIndex(migDeviceId string) (int, error) {
	migDevice, ret := nvml.DeviceGetHandleByUUID(migDeviceId)
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("unable to get MIG device with UUID %s: %s", migDeviceId, nvml.ErrorString(ret))
	}
	gpuDevice, ret := migDevice.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf(
			"unable to get GPU of MIG device with UUID %s: %s",
			migDeviceId, nvml.ErrorString(ret),
		)
	}
	gpuIndex, ret := gpuDevice.GetIndex()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf(
			"unable to get index of GPU of MIG device with UUID %s: %s",
			migDeviceId, nvml.ErrorString(ret),
		)
	}
	return gpuIndex, nil
}

type PodResourcesClient struct {
	lister     pdrv1.PodResourcesListerClient
	nvmlClient NvmlClient
}

func (c PodResourcesClient) GetUsedMIGDevices(ctx context.Context) ([]MIGDevice, error) {
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

func (c PodResourcesClient) GetFreeMIGDevices(ctx context.Context) ([]MIGDevice, error) {
	return nil, nil
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
						"GPU resource %s is associated with multiple devices (%d): this should never happen",
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
