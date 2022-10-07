//go:build nvml

package nvml

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type device struct {
	nvml.Device
}

type ClientImpl struct {
}

func NewClient() (ClientImpl, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return ClientImpl{}, fmt.Errorf("unable to initialize NVML: %s", nvml.ErrorString(ret))
	}
	return ClientImpl{}, nil
}

// GetGpuIndex returns the index of the GPU associated to the
// MIG device provided as arg. Returns err if the device
// is not found or any error occurs while retrieving it.
func (c ClientImpl) GetGpuIndex(migDeviceId string) (int, error) {
	var result int
	var found bool
	err := c.visitMIGDevices(func(gpuIndex, migDeviceIndex int, migDevice nvml.Device) (bool, error) {
		uuid, ret := migDevice.GetUUID()
		if ret != nvml.SUCCESS {
			return false, fmt.Errorf(
				"error getting UUID of MIG device with index %q on GPU %q: %s",
				migDeviceIndex,
				gpuIndex,
				nvml.ErrorString(ret),
			)
		}
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
		return 0, fmt.Errorf("error getting GPU index of MIG device %q: not found", migDeviceId)
	}

	return result, nil
}

func (c ClientImpl) visitMIGDevices(f func(gpuIndex, migDeviceIndex int, migDevice nvml.Device) (bool, error)) error {
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("error getting GPU device count: %v", nvml.ErrorString(ret))
	}

	for i := 0; i < count; i++ {
		d, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("error getting device handle for GPU with index %q: %v", i, nvml.ErrorString(ret))
		}
		continueVisiting, err := device{d}.visitMIGDevices(func(migDeviceIndex int, migDevice nvml.Device) (bool, error) {
			return f(i, migDeviceIndex, migDevice)
		})
		if err != nil {
			return fmt.Errorf("error visiting MIG devices of GPU with index %q: %v", i, err)
		}
		if !continueVisiting {
			return nil
		}
	}
	return nil
}

func (d device) visitMIGDevices(f func(migDeviceIndex int, migDevice nvml.Device) (bool, error)) (bool, error) {
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
			return false, fmt.Errorf("error getting MIG device handle at index %q: %v", i, nvml.ErrorString(ret))
		}
		return f(i, migDevice)
	}

	return true, nil
}
