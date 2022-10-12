//go:build nvml

package nvml

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"k8s.io/klog/v2"
)

type device struct {
	nvml.Device
}

type clientImpl struct {
}

func NewClient() Client {
	return clientImpl{}
}

// GetGpuIndex returns the index of the GPU associated to the
// MIG device provided as arg. Returns err if the device
// is not found or any error occurs while retrieving it.
func (c clientImpl) GetGpuIndex(migDeviceId string) (int, error) {
	klog.V(1).InfoS("retrieving GPU index of MIG device", "MIGDeviceUUID", migDeviceId)
	var result int
	var found bool
	err := c.visitMigDevices(func(gpuIndex, migDeviceIndex int, migDevice nvml.Device) (bool, error) {
		uuid, ret := migDevice.GetUUID()
		if ret != nvml.SUCCESS {
			return false, fmt.Errorf(
				"error getting UUID of MIG device with index %d on GPU %v: %s",
				migDeviceIndex,
				gpuIndex,
				nvml.ErrorString(ret),
			)
		}
		klog.V(3).InfoS(
			"visiting MIG device",
			"GPUIndex",
			gpuIndex,
			"MIGDeviceIndex",
			migDeviceIndex,
			"MIGDeviceUUID",
			uuid,
		)
		if uuid == migDeviceId {
			result = gpuIndex
			found = true
			return false, nil // found, stop iterating
		}
		return true, nil // continue iterating
	})

	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("error getting GPU index of MIG device %s: not found", migDeviceId)
	}

	return result, nil
}

func (c clientImpl) visitMigDevices(visit func(gpuIndex, migDeviceIndex int, migDevice nvml.Device) (bool, error)) error {
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("error getting GPU device count: %v", nvml.ErrorString(ret))
	}

	for i := 0; i < count; i++ {
		d, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("error getting device handle for GPU with index %d: %v", i, nvml.ErrorString(ret))
		}
		continueVisiting, err := device{d}.visitMigDevices(func(migDeviceIndex int, migDevice nvml.Device) (bool, error) {
			return visit(i, migDeviceIndex, migDevice)
		})
		if err != nil {
			return fmt.Errorf("error visiting MIG devices of GPU with index %d: %v", i, err)
		}
		if !continueVisiting {
			return nil
		}
	}
	return nil
}

func (d device) visitMigDevices(visit func(migDeviceIndex int, migDevice nvml.Device) (bool, error)) (bool, error) {
	count, ret := d.GetMaxMigDeviceCount()
	if ret != nvml.SUCCESS {
		return false, fmt.Errorf("error getting max MIG device count: %v", nvml.ErrorString(ret))
	}

	for i := 0; i < count; i++ {
		migDevice, ret := d.GetMigDeviceHandleByIndex(i)
		if ret == nvml.ERROR_NOT_FOUND {
			continue
		}
		if ret == nvml.ERROR_INVALID_ARGUMENT {
			continue
		}
		if ret != nvml.SUCCESS {
			return false, fmt.Errorf("error getting MIG device handle at index %d: %v", i, nvml.ErrorString(ret))
		}
		continueVisiting, err := visit(i, migDevice)
		if err != nil {
			return false, err
		}
		if !continueVisiting {
			return false, nil
		}
	}

	return true, nil
}
