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

import (
	"fmt"
)

const (
	GPUModel_A100_SXM4_40GB GPUModel = "NVIDIA-A100-40GB-SXM4"
	GPUModel_A100_PCIe_80GB GPUModel = "NVIDIA-A100-80GB-PCIe"
	GPUModel_A30            GPUModel = "A30"
)

var (
	defaultKnownMigGeometries = map[GPUModel][]Geometry{
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
				Profile7g79gb: 1,
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

func SetKnownGeometries(configs map[GPUModel][]Geometry) error {
	if err := validateConfigs(configs); err != nil {
		return err
	}
	defaultKnownMigGeometries = configs
	return nil
}

func GetKnownGeometries() map[GPUModel][]Geometry {
	if defaultKnownMigGeometries == nil {
		return map[GPUModel][]Geometry{}
	}
	return defaultKnownMigGeometries
}

func GetAllowedGeometries(model GPUModel) ([]Geometry, bool) {
	configs, ok := GetKnownGeometries()[model]
	return configs, ok
}

func validateConfigs(configs map[GPUModel][]Geometry) error {
	if len(configs) == 0 {
		return fmt.Errorf("no known configs provided")
	}
	for _, geometryList := range configs {
		for _, geometry := range geometryList {
			for profile, quantity := range geometry {
				if !profile.isValid() {
					return fmt.Errorf("invalid profile %s", profile)
				}
				if quantity < 1 {
					return fmt.Errorf("invalid quantity %d for profile %s", quantity, profile)
				}
			}
		}
	}
	return nil
}
