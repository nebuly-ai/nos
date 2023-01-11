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
	"github.com/nebuly-ai/nos/pkg/gpu"
)

var (
	defaultKnownMigGeometries = map[gpu.Model][]gpu.Geometry{
		gpu.GPUModel_A30: {
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
		gpu.GPUModel_A100_SXM4_40GB: {
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
		gpu.GPUModel_A100_PCIe_80GB: {
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

func SetKnownGeometries(configs map[gpu.Model][]gpu.Geometry) error {
	if err := ValidateConfigs(configs); err != nil {
		return err
	}
	defaultKnownMigGeometries = configs
	return nil
}

func GetKnownGeometries() map[gpu.Model][]gpu.Geometry {
	if defaultKnownMigGeometries == nil {
		return map[gpu.Model][]gpu.Geometry{}
	}
	return defaultKnownMigGeometries
}

func GetAllowedGeometries(model gpu.Model) ([]gpu.Geometry, bool) {
	configs, ok := GetKnownGeometries()[model]
	return configs, ok
}

func ValidateConfigs(configs map[gpu.Model][]gpu.Geometry) error {
	if len(configs) == 0 {
		return fmt.Errorf("no known configs provided")
	}
	for _, geometryList := range configs {
		for _, geometry := range geometryList {
			for profile, quantity := range geometry {
				migProfile, ok := profile.(ProfileName)
				if !ok {
					return fmt.Errorf("invalid profile type %T, expected MIG profile name", profile)
				}
				if !migProfile.isValid() {
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
