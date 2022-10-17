//go:build nvml

package nvml

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	nvlibdevice "gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	nvlibNvml "gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
	"k8s.io/klog/v2"
)

type clientImpl struct {
	nvmlClient  nvlibNvml.Interface
	nvlibClient nvlibdevice.Interface
}

func NewClient() Client {
	nvmlClient := nvlibNvml.New()
	return &clientImpl{
		nvmlClient:  nvmlClient,
		nvlibClient: nvlibdevice.New(nvlibdevice.WithNvml(nvmlClient)),
	}
}

func (c *clientImpl) init() error {
	if ret := c.nvmlClient.Init(); ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("unable to initialize NVML: %s", ret.Error())
	}
	return nil
}

func (c *clientImpl) shutdown() {
	if ret := c.nvmlClient.Shutdown(); ret != nvlibNvml.SUCCESS {
		klog.Errorf("unable to shut down NVML: %s", ret.Error())
	}
}

// GetGpuIndex returns the index of the GPU associated to the
// MIG device provided as arg. Returns err if the device
// is not found or any error occurs while retrieving it.
func (c *clientImpl) GetGpuIndex(migDeviceId string) (int, error) {
	if err := c.init(); err != nil {
		return 0, err
	}
	defer c.shutdown()

	klog.V(1).InfoS("retrieving GPU index of MIG device", "MIGDeviceUUID", migDeviceId)
	var result int
	var err error
	var found bool
	err = c.nvlibClient.VisitMigDevices(func(gpuIndex int, _ nvlibdevice.Device, migIndex int, m nvlibdevice.MigDevice) error {
		if found {
			return nil
		}
		uuid, ret := m.GetUUID()
		if ret != nvlibNvml.SUCCESS {
			return fmt.Errorf(
				"error getting UUID of MIG device with index %d on GPU %v: %s",
				migIndex,
				gpuIndex,
				ret.Error(),
			)
		}
		klog.V(3).InfoS(
			"visiting MIG device",
			"GPUIndex",
			gpuIndex,
			"MIGDeviceIndex",
			migIndex,
			"MIGDeviceUUID",
			uuid,
		)
		if uuid == migDeviceId {
			result = gpuIndex
			found = true
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("error getting GPU index of MIG device %s: not found", migDeviceId)
	}
	return result, err
}

func (c *clientImpl) DeleteMigDevice(id string) error {
	if err := c.init(); err != nil {
		return err
	}
	defer c.shutdown()

	// Fetch MIG device handle
	d, ret := c.nvmlClient.DeviceGetHandleByUUID(id)
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting MIG device with UUID %s: %s", id, ret.Error())
	}
	isMig, ret := d.IsMigDeviceHandle()
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf(
			"error determining whether the device with UUID %s is a MIG device: %s",
			id,
			ret.Error(),
		)
	}
	if !isMig {
		return fmt.Errorf("device with UUID %s is not a MIG device", id)
	}

	// Fetch GPU Instance and Compute Instances
	giId, ret := d.GetGpuInstanceId()
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting GPU Instance ID: %s", ret.Error())
	}
	parentGpu, ret := d.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting device handle from MIG device: %s", ret.Error())
	}
	gi, ret := parentGpu.GetGpuInstanceById(giId)
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting GPU Instance %d: %s", giId, ret.Error())
	}

	// Delete Compute Instances
	var numVisitedCi uint8
	err := visitComputeInstances(gi, func(ci nvlibNvml.ComputeInstance, ciProfileId int, ciEngProfileId int, ciProfileInfo nvlibNvml.ComputeInstanceProfileInfo) error {
		numVisitedCi++
		klog.V(1).InfoS(
			"deleting compute instance",
			"profileInfo",
			ciProfileInfo,
			"profileID",
			ciProfileId,
			"engProfileId",
			ciEngProfileId,
		)
		if r := ci.Destroy(); r != nvlibNvml.SUCCESS {
			return fmt.Errorf("error deleting compute instance: %s", r.Error())
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error destroying compute instances: %s", err)
	}
	if numVisitedCi == 0 {
		return fmt.Errorf("cannot delete %s: the device does not have any compute instance associated", id)
	}

	// Delete GPU Instance
	klog.V(1).InfoS("deleting GPU instance")
	if ret = gi.Destroy(); ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error deleting GPU instance: %s", ret.Error())
	}

	return nil
}

func (c *clientImpl) CreateMigDevice(migProfileName string, gpuIndex int) error {
	r := nvml.Init()
	if r != nvml.SUCCESS {
		return fmt.Errorf("error initializing nvml client: %s", nvml.ErrorString(r))
	}
	defer nvml.Shutdown()

	// Parse MIG profile
	mp, err := c.nvlibClient.ParseMigProfile(migProfileName)
	if err != nil {
		return fmt.Errorf("invalid MIG profile: %s", err.Error())
	}

	// Check if GPU is MIG-enabled
	d, ret := c.nvmlClient.DeviceGetHandleByIndex(gpuIndex)
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting GPU with index %d: %s", gpuIndex, ret.Error())
	}
	device, err := c.nvlibClient.NewDevice(d)
	if err != nil {
		return fmt.Errorf("error getting GPU with index %d: %s", gpuIndex, ret.Error())
	}
	isMigCapable, err := device.IsMigCapable()
	if err != nil {
		return err
	}
	if !isMigCapable {
		return fmt.Errorf("MIG is not enabled on GPU with index %d", gpuIndex)
	}

	// Fetch nvml Device
	// Todo: from now on we have to work with github.com/NVIDIA/go-nvml/pkg/nvml types because
	// at the moment the types from nvlib does not provide methods for creating GPU instances.
	// This won't be necessary anymore when the missing methods will be added to nvlib.
	nvmlDevice, r := nvml.DeviceGetHandleByIndex(gpuIndex)
	if r != nvml.SUCCESS {
		return fmt.Errorf(nvml.ErrorString(r))
	}

	// Create GPU Instance
	giProfileInfo, ret := device.GetGpuInstanceProfileInfo(mp.GetInfo().GIProfileID)
	if ret != nvlibNvml.SUCCESS {
		return fmt.Errorf("error getting GPU instance profile info: %s", err.Error())
	}
	gi, r := nvmlDevice.CreateGpuInstance((*nvml.GpuInstanceProfileInfo)(&giProfileInfo))
	if r != nvml.SUCCESS {
		return fmt.Errorf("error creating GPU instance: %s", nvml.ErrorString(r))
	}
	klog.V(1).InfoS("created GPU instance", "giProfileInfo", giProfileInfo)

	// Create Compute Instance
	ciProfileInfo, r := gi.GetComputeInstanceProfileInfo(mp.GetInfo().CIProfileID, mp.GetInfo().CIEngProfileID)
	if r != nvml.SUCCESS {
		// Cleanup created GPU instance
		klog.V(1).InfoS("error getting GPU instance profile info, destroying previously created GPU instance")
		r := gi.Destroy()
		if r != nvml.SUCCESS {
			klog.Errorf("error destroying GPU instance: %v", gi)
		}
		gi.Destroy()
		return fmt.Errorf("error getting Compute Instance profile info: %s", nvml.ErrorString(r))
	}
	_, r = gi.CreateComputeInstance(&ciProfileInfo)
	if r != nvml.SUCCESS {
		// Cleanup created GPU instance
		klog.V(1).InfoS("error creating Compute Instance, destroying previously created GPU instance")
		r := gi.Destroy()
		if r != nvml.SUCCESS {
			klog.Errorf("error destroying GPU instance: %v", gi)
		}
		return fmt.Errorf("error creating Compute Instance: %s", nvml.ErrorString(r))
	}
	klog.V(1).InfoS("created compute instance", "ciProfileInfo", ciProfileInfo)

	return nil
}

func visitComputeInstances(
	gpuInstance nvlibNvml.GpuInstance,
	f func(ci nvlibNvml.ComputeInstance, ciProfileId int, ciEngProfileId int, ciProfileInfo nvlibNvml.ComputeInstanceProfileInfo) error,
) error {
	for j := 0; j < nvlibNvml.COMPUTE_INSTANCE_PROFILE_COUNT; j++ {
		for k := 0; k < nvlibNvml.COMPUTE_INSTANCE_ENGINE_PROFILE_COUNT; k++ {
			ciProfileInfo, ret := gpuInstance.GetComputeInstanceProfileInfo(j, k)
			if ret == nvlibNvml.ERROR_NOT_SUPPORTED {
				continue
			}
			if ret == nvlibNvml.ERROR_INVALID_ARGUMENT {
				continue
			}
			if ret != nvlibNvml.SUCCESS {
				return fmt.Errorf("error getting Compute instance profile info for (%d, %d): %s", j, k, ret.Error())
			}

			cis, ret := gpuInstance.GetComputeInstances(&ciProfileInfo)
			if ret != nvlibNvml.SUCCESS {
				return fmt.Errorf("error getting Compute instances for profile (%d, %d): %s", j, k, ret.Error())
			}

			for _, ci := range cis {
				err := f(ci, j, k, ciProfileInfo)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
