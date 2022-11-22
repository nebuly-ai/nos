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

const (
	GPUModel_A100_SXM4_40GB GPUModel = "NVIDIA-A100-40GB-SXM4"
	GPUModel_A100_PCIe_80GB GPUModel = "NVIDIA-A100-80GB-PCIe"
	GPUModel_A30            GPUModel = "A30"
)

// TODO: move to yaml config file
var (
	gpuModelToAllowedMigGeometries = map[GPUModel][]Geometry{
		GPUModel_A30: {
			{
				Profile4g24gb: 1,
			},
			{
				Profile2g12gb: 2,
			},
			{
				Profile2g12gb: 1,
				Profile1g6gb:  2,
			},
			{
				Profile1g6gb: 4,
			},
		},
		GPUModel_A100_SXM4_40GB: {
			{
				Profile7g40gb: 1,
			},
			{
				Profile4g20gb: 1,
				Profile2g10gb: 1,
				Profile1g5gb:  1,
			},
			{
				Profile4g20gb: 1,
				Profile1g5gb:  3,
			},
			{
				Profile3g20gb: 2,
			},
			{
				Profile3g20gb: 1,
				Profile2g10gb: 1,
				Profile1g5gb:  1,
			},
			{
				Profile3g20gb: 1,
				Profile1g5gb:  3,
			},
			{
				Profile2g10gb: 2,
				Profile3g20gb: 1,
			},
			{
				Profile2g10gb: 1,
				Profile1g5gb:  2,
				Profile3g20gb: 1,
			},
			{
				Profile2g10gb: 3,
				Profile1g5gb:  1,
			},
			{
				Profile2g10gb: 2,
				Profile1g5gb:  3,
			},
			{
				Profile2g10gb: 1,
				Profile1g5gb:  5,
			},
			{
				Profile1g5gb: 7,
			},
		},
		GPUModel_A100_PCIe_80GB: {
			{
				Profile7g80gb: 1,
			},
			{
				Profile4g40gb: 1,
				Profile2g20gb: 1,
				Profile1g10gb: 1,
			},
			{
				Profile4g40gb: 1,
				Profile1g10gb: 3,
			},
			{
				Profile3g40gb: 2,
			},
			{
				Profile3g40gb: 1,
				Profile2g20gb: 1,
				Profile1g10gb: 1,
			},
			{
				Profile3g40gb: 1,
				Profile1g10gb: 3,
			},
			{
				Profile2g20gb: 2,
				Profile3g20gb: 1,
			},
			{
				Profile2g10gb: 1,
				Profile1g10gb: 2,
				Profile3g40gb: 1,
			},
			{
				Profile2g20gb: 3,
				Profile1g10gb: 1,
			},
			{
				Profile2g20gb: 2,
				Profile1g10gb: 3,
			},
			{
				Profile2g20gb: 1,
				Profile1g10gb: 5,
			},
			{
				Profile1g10gb: 7,
			},
		},
	}
)
