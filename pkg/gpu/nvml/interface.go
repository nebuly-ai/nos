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
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
)

type Client interface {
	GetGpuIndex(gpuId string) (int, gpu.Error)

	GetMigDeviceGpuIndex(migDeviceId string) (int, gpu.Error)

	DeleteMigDevice(id string) gpu.Error

	CreateMigDevices(migProfileNames []string, gpuIndex int) gpu.Error

	GetMigEnabledGPUs() ([]int, gpu.Error)

	DeleteAllMigDevicesExcept(migDeviceIds []string) error
}
