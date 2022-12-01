//go:build nvml

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

package nvml

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	nvlibdevice "gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	nvlibNvml "gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvml"
)

type clientImpl struct {
	nvmlClient  nvlibNvml.Interface
	nvlibClient nvlibdevice.Interface
	logger      logr.Logger
}

func NewClient(logger logr.Logger) Client {
	nvmlClient := nvlibNvml.New()
	return &clientImpl{
		nvmlClient:  nvmlClient,
		nvlibClient: nvlibdevice.New(nvlibdevice.WithNvml(nvmlClient)),
		logger:      logger,
	}
}

func (c *clientImpl) init() gpu.Error {
	if ret := c.nvmlClient.Init(); ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("unable to initialize NVML: %s", ret.Error())
	}
	return nil
}

func (c *clientImpl) shutdown() {
	if ret := c.nvmlClient.Shutdown(); ret != nvlibNvml.SUCCESS {
		c.logger.Error(gpu.GenericErr.Errorf(ret.Error()), "unable to shut down NVML")
	}
}

// GetGpuIndex returns the index of the GPU associated to the
// MIG device provided as arg. Returns err if the device
// is not found or any error occurs while retrieving it.
func (c *clientImpl) GetGpuIndex(migDeviceId string) (int, gpu.Error) {
	if err := c.init(); err != nil {
		return 0, err
	}
	defer c.shutdown()

	c.logger.V(3).Info("retrieving GPU index of MIG device", "MIGDeviceUUID", migDeviceId)
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
		c.logger.V(4).Info(
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
		return 0, gpu.NewGenericError(err)
	}
	if !found {
		return 0, gpu.NotFoundErr.Errorf("GPU index of MIG device %s not found", migDeviceId)
	}
	return result, nil
}

func (c *clientImpl) DeleteMigDevice(id string) gpu.Error {
	if err := c.init(); err != nil {
		return err
	}
	defer c.shutdown()

	// Fetch MIG device handle
	d, ret := c.nvmlClient.DeviceGetHandleByUUID(id)
	if ret == nvlibNvml.ERROR_NOT_FOUND {
		return gpu.NotFoundErr.Errorf("MIG device %s not found", id)
	}
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error getting MIG device with UUID %s: %s", id, ret.Error())
	}
	isMig, ret := d.IsMigDeviceHandle()
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf(
			"error determining whether the device with UUID %s is a MIG device: %s",
			id,
			ret.Error(),
		)
	}
	if !isMig {
		return gpu.GenericErr.Errorf("device with UUID %s is not a MIG device", id)
	}

	// Fetch GPU Instance and Compute Instances
	giId, ret := d.GetGpuInstanceId()
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error getting GPU Instance ID: %s", ret.Error())
	}
	parentGpu, ret := d.GetDeviceHandleFromMigDeviceHandle()
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error getting device handle from MIG device: %s", ret.Error())
	}
	gi, ret := parentGpu.GetGpuInstanceById(giId)
	if ret == nvlibNvml.ERROR_NOT_FOUND {
		return gpu.NotFoundErr.Errorf("GPU instance %s not found", giId)
	}
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error getting GPU Instance %d: %s", giId, ret.Error())
	}

	// Delete Compute Instances
	var numVisitedCi uint8
	err := visitComputeInstances(gi, func(ci nvlibNvml.ComputeInstance, ciProfileId int, ciEngProfileId int, ciProfileInfo nvlibNvml.ComputeInstanceProfileInfo) error {
		numVisitedCi++
		c.logger.V(1).Info(
			"deleting compute instance",
			"profileInfo",
			ciProfileInfo,
			"profileID",
			ciProfileId,
			"engProfileId",
			ciEngProfileId,
		)
		if r := ci.Destroy(); r != nvlibNvml.SUCCESS {
			return gpu.GenericErr.Errorf("error deleting compute instance: %s", r.Error())
		}
		return nil
	})
	if err != nil {
		return gpu.GenericErr.Errorf("error destroying compute instances: %s", err)
	}
	if numVisitedCi == 0 {
		return gpu.GenericErr.Errorf("cannot delete %s: the device does not have any compute instance associated", id)
	}

	// Delete GPU Instance
	c.logger.V(1).Info("deleting GPU instance")
	if ret = gi.Destroy(); ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error deleting GPU instance: %s", ret.Error())
	}

	return nil
}

func (c *clientImpl) CreateMigDevices(migProfileNames []string, gpuIndex int) gpu.Error {
	r := nvml.Init()
	if r != nvml.SUCCESS {
		return gpu.GenericErr.Errorf("error initializing nvml client: %s", nvml.ErrorString(r))
	}
	defer nvml.Shutdown()

	// Parse MIG profiles
	mps := make([]nvlibdevice.MigProfile, 0)
	for _, profileName := range migProfileNames {
		mp, err := c.nvlibClient.ParseMigProfile(profileName)
		if err != nil {
			return gpu.GenericErr.Errorf("invalid MIG profile: %s", err.Error())
		}
		mps = append(mps, mp)
	}

	// Check if GPU is MIG-enabled
	d, ret := c.nvmlClient.DeviceGetHandleByIndex(gpuIndex)
	if ret == nvlibNvml.ERROR_NOT_FOUND {
		return gpu.NotFoundErr.Errorf("GPU with index %d not found", gpuIndex)
	}
	if ret != nvlibNvml.SUCCESS {
		return gpu.GenericErr.Errorf("error getting GPU with index %d: %s", gpuIndex, ret.Error())
	}
	device, err := c.nvlibClient.NewDevice(d)
	if err != nil {
		return gpu.GenericErr.Errorf("error getting GPU with index %d: %s", gpuIndex, ret.Error())
	}
	isMigCapable, err := device.IsMigCapable()
	if err != nil {
		return gpu.NewGenericError(err)
	}
	if !isMigCapable {
		return gpu.GenericErr.Errorf("MIG is not enabled on GPU with index %d", gpuIndex)
	}

	// Function for destroying the CIs and GIs created while trying MIG profiles permutations
	cleanup := func(gis []nvlibNvml.GpuInstance, cis []nvlibNvml.ComputeInstance) error {
		c.logger.V(1).Info("cleaning up created resources")
		var anyErrorCleaningUp bool
		// cleanup compute instances
		for _, ci := range cis {
			if ret = ci.Destroy(); ret != nvlibNvml.SUCCESS {
				c.logger.Error(gpu.GenericErr.Errorf(ret.Error()), "error deleting compute instance")
				anyErrorCleaningUp = true
			}
		}
		// cleanup GPU instances
		for _, gi := range gis {
			if ret = gi.Destroy(); ret != nvlibNvml.SUCCESS {
				c.logger.Error(gpu.GenericErr.Errorf(ret.Error()), "error deleting GPU instance")
				anyErrorCleaningUp = true
			}
		}
		if anyErrorCleaningUp {
			return fmt.Errorf("error cleaning up created resources, some resources might not have been deleted")
		}
		return nil
	}

	// Iterate permutations until success
	// (MIG profile creation success depends on the order on which they are created)
	var anyPermutationApplied bool
	var nAttempts int
	var maxAttempts = 20
	err = util.IterPermutations(mps, func(mps []nvlibdevice.MigProfile) (bool, error) {
		// TODO: optimize permutation search instead of trying all of them and limiting the max attempts
		if nAttempts > maxAttempts {
			return false, fmt.Errorf("could not find a valid permutation for creating MIG profiles: too many attempts")
		}
		c.logger.V(1).Info("trying to create MIG profiles", "permutation", mps)
		nAttempts++
		createdGIs := make([]nvlibNvml.GpuInstance, 0)
		createdCIs := make([]nvlibNvml.ComputeInstance, 0)
		for _, mp := range mps {
			// Create GPU Instance
			giProfileInfo, ret := device.GetGpuInstanceProfileInfo(mp.GetInfo().GIProfileID)
			if ret != nvlibNvml.SUCCESS {
				return false, gpu.GenericErr.Errorf("error getting GPU instance profile info: %s", ret.Error())
			}
			gi, ret := device.CreateGpuInstance(&giProfileInfo)
			if ret != nvlibNvml.SUCCESS {
				c.logger.V(1).Info("could not create GPU instance", "error", ret.Error())
				return true, cleanup(createdGIs, createdCIs)
			}
			c.logger.V(1).Info("created GPU Instance", "GpuInstanceID", mp.GetInfo().GIProfileID)
			createdGIs = append(createdGIs, gi)

			// Create Compute Instance
			ciProfileInfo, ret := gi.GetComputeInstanceProfileInfo(mp.GetInfo().CIProfileID, mp.GetInfo().CIEngProfileID)
			if ret != nvlibNvml.SUCCESS {
				return false, gpu.GenericErr.Errorf("error getting compute instance profile info: %s", ret.Error())
			}
			ci, ret := gi.CreateComputeInstance(&ciProfileInfo)
			if ret != nvlibNvml.SUCCESS {
				c.logger.V(1).Info("could not create compute instance", "error", ret.Error())
				return true, cleanup(createdGIs, createdCIs)
			}
			c.logger.V(1).Info("created compute Instance", "ComputeInstanceId", mp.GetInfo().CIProfileID)
			createdCIs = append(createdCIs, ci)
		}
		// all MIG profiles of the permutation have been created, stop iterating
		anyPermutationApplied = true
		c.logger.V(1).Info("MIG profiles successfully created", "permutations", mps)
		return false, nil
	})

	if err != nil {
		return gpu.GenericErr.Errorf("error while applying permutations: %s", err)
	}
	if !anyPermutationApplied {
		return gpu.GenericErr.Errorf("could not create MIG profiles: could not find any valid permutation")
	}
	return nil
}

// GetMigEnabledGPUs returns the indexes of the GPUs that have MIG mode enabled
func (c *clientImpl) GetMigEnabledGPUs() ([]int, gpu.Error) {
	r := nvml.Init()
	if r != nvml.SUCCESS {
		return nil, gpu.GenericErr.Errorf("error initializing nvml client: %s", nvml.ErrorString(r))
	}
	defer nvml.Shutdown()

	devices, err := c.nvlibClient.GetDevices()
	if err != nil {
		return nil, gpu.NewGenericError(err)
	}

	indexes := make([]int, 0)
	for _, d := range devices {
		isEnabled, err := d.IsMigEnabled()
		if err != nil {
			return nil, gpu.NewGenericError(err)
		}
		if !isEnabled {
			continue
		}
		gpuIndex, ret := d.GetIndex()
		if ret != nvlibNvml.SUCCESS {
			return nil, gpu.NewGenericError(err)
		}
		indexes = append(indexes, gpuIndex)
	}

	return indexes, nil
}

// DeleteAllMigDevicesExcept deletes all the MIG resources (Compute Instances and GPU Instances) except the ones
// associated with the MIG devices with the provided IDs
func (c *clientImpl) DeleteAllMigDevicesExcept(migDeviceIds []string) error {
	r := nvml.Init()
	if r != nvml.SUCCESS {
		return gpu.GenericErr.Errorf("error initializing nvml client: %s", nvml.ErrorString(r))
	}
	defer nvml.Shutdown()

	err := c.nvlibClient.VisitDevices(func(i int, device nvlibdevice.Device) error {
		// Check if device is MIG-enabled
		isMig, err := device.IsMigEnabled()
		if err != nil {
			return gpu.GenericErr.Errorf("error checking if device is MIG enabled: %s", err)
		}
		if !isMig {
			return nil
		}

		err = visitGpuInstances(device, func(gi nvlibNvml.GpuInstance) error {
			// Check if GPU instance can be deleted
			giInfo, ret := gi.GetInfo()
			if ret != nvlibNvml.SUCCESS {
				return gpu.GenericErr.Errorf("error getting GPU instance info: %s", ret.Error())
			}
			giDeviceId, ret := giInfo.Device.GetUUID()
			if ret != nvlibNvml.SUCCESS {
				return gpu.GenericErr.Errorf("error getting GPU instance device UUID: %s", ret.Error())
			}
			if util.InSlice(giDeviceId, migDeviceIds) {
				return nil
			}

			// Delete compute instances
			err := visitComputeInstances(gi, func(ci nvlibNvml.ComputeInstance, _ int, _ int, _ nvlibNvml.ComputeInstanceProfileInfo) error {
				ciInfo, ret := ci.GetInfo()
				if ret != nvlibNvml.SUCCESS {
					return gpu.GenericErr.Errorf("error getting compute instance info: %s", ret.Error())
				}
				ciDeviceId, ret := ciInfo.Device.GetUUID()
				if ret != nvlibNvml.SUCCESS {
					return gpu.GenericErr.Errorf("error getting compute instance device UUID: %s", ret.Error())
				}
				if util.InSlice(ciDeviceId, migDeviceIds) {
					return nil
				}
				ret = ci.Destroy()
				if ret == nvlibNvml.ERROR_INVALID_ARGUMENT {
					return nil
				}
				if ret != nvlibNvml.SUCCESS {
					return gpu.GenericErr.Errorf("error destroying compute instance: %s", ret.Error())
				}
				c.logger.Info("deleted compute instance", "ComputeInstanceId", ciInfo.Id)
				return nil
			})
			if err != nil {
				return err
			}

			// Delete GPU instance
			ret = gi.Destroy()
			if ret == nvlibNvml.ERROR_INVALID_ARGUMENT {
				return nil
			}
			if ret != nvlibNvml.SUCCESS {
				return gpu.GenericErr.Errorf("error destroying GPU instance: %s", ret.Error())
			}
			c.logger.Info("deleted GPU instance", "GpuInstanceId", giInfo.Id)

			return nil
		})

		return err
	})

	if err != nil {
		return gpu.NewGenericError(err)
	}
	return nil
}

func visitGpuInstances(device nvlibdevice.Device, f func(ci nvlibNvml.GpuInstance) error) error {
	for i := 0; i < nvlibNvml.GPU_INSTANCE_PROFILE_COUNT; i++ {
		profile, ret := device.GetGpuInstanceProfileInfo(i)
		if ret == nvlibNvml.ERROR_NOT_FOUND {
			continue
		}
		if ret == nvlibNvml.ERROR_NOT_SUPPORTED {
			continue
		}
		if ret == nvlibNvml.ERROR_INVALID_ARGUMENT {
			continue
		}
		if ret != nvlibNvml.SUCCESS {
			return fmt.Errorf("error getting GPU profile info: %s", ret.Error())
		}

		gis, ret := device.GetGpuInstances(&profile)
		if ret != nvlibNvml.SUCCESS {
			return fmt.Errorf("error getting GPU instances: %s", ret.Error())
		}

		for _, gi := range gis {
			err := f(gi)
			if err != nil {
				return err
			}
		}
	}
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
				return gpu.GenericErr.Errorf("error getting Compute instance profile info for (%d, %d): %s", j, k, ret.Error())
			}

			cis, ret := gpuInstance.GetComputeInstances(&ciProfileInfo)
			if ret != nvlibNvml.SUCCESS {
				return gpu.GenericErr.Errorf("error getting Compute instances for profile (%d, %d): %s", j, k, ret.Error())
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
