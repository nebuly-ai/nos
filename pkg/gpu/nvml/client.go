//go:build nvml

package nvml

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
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
