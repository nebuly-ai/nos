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
	"github.com/google/go-cmp/cmp"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
)

// Geometry corresponds to the MIG Geometry of a GPU,
// namely the MIG profiles of the GPU with the respective quantity.
type Geometry map[ProfileName]int

func (g Geometry) AsResources() map[v1.ResourceName]int {
	res := make(map[v1.ResourceName]int)
	for p, v := range g {
		resourceName := v1.ResourceName(fmt.Sprintf("%s%s", constant.NvidiaMigResourcePrefix, p))
		res[resourceName] += v
	}
	return res
}

func (g Geometry) Id() string {
	var name string
	for profile, quantity := range g {
		name += fmt.Sprintf("%s:%d", profile, quantity)
	}
	return name
}

type GPUModel string

type GPU struct {
	index                int
	model                GPUModel
	allowedMigGeometries []Geometry
	usedMigDevices       map[ProfileName]int
	freeMigDevices       map[ProfileName]int
}

func NewGpuOrPanic(model GPUModel, index int, usedMigDevices, freeMigDevices map[ProfileName]int) GPU {
	gpu, err := NewGPU(model, index, usedMigDevices, freeMigDevices)
	if err != nil {
		panic(err)
	}
	return gpu
}

func NewGPU(model GPUModel, index int, usedMigDevices, freeMigDevices map[ProfileName]int) (GPU, error) {
	allowedGeometries, ok := GetAllowedGeometries(model)
	if !ok {
		return GPU{}, fmt.Errorf("model %q is not associated with any known GPU", model)
	}
	return GPU{
		index:                index,
		model:                model,
		allowedMigGeometries: allowedGeometries,
		usedMigDevices:       usedMigDevices,
		freeMigDevices:       freeMigDevices,
	}, nil
}

func (g *GPU) Clone() GPU {
	cloned := GPU{
		index:                g.index,
		model:                g.model,
		allowedMigGeometries: g.allowedMigGeometries,
		usedMigDevices:       make(map[ProfileName]int),
		freeMigDevices:       make(map[ProfileName]int),
	}
	for k, v := range g.freeMigDevices {
		cloned.freeMigDevices[k] = v
	}
	for k, v := range g.usedMigDevices {
		cloned.usedMigDevices[k] = v
	}
	return cloned
}

func (g *GPU) GetIndex() int {
	return g.index
}

func (g *GPU) GetModel() GPUModel {
	return g.model
}

func (g *GPU) GetGeometry() Geometry {
	res := make(Geometry)

	for profile, quantity := range g.usedMigDevices {
		res[profile] += quantity
	}
	for profile, quantity := range g.freeMigDevices {
		res[profile] += quantity
	}

	return res
}

// CanApplyGeometry returns true if the geometry provided as argument can be applied to the GPU, otherwise it
// returns false and the reason why the geometry cannot be applied.
func (g *GPU) CanApplyGeometry(geometry Geometry) (bool, string) {
	// Check if geometry is allowed
	if !g.AllowsGeometry(geometry) {
		return false, fmt.Sprintf("GPU model %s does not allow the provided MIG geometry", g.model)
	}
	// Check if new geometry deletes used devices
	for usedProfile, usedQuantity := range g.usedMigDevices {
		if geometry[usedProfile] < usedQuantity {
			return false, "cannot apply MIG geometry: cannot delete MIG devices being used"
		}
	}

	return true, ""
}

// ApplyGeometry applies the MIG geometry provided as argument by changing the free devices of the GPU.
// It returns an error if the provided geometry is not allowed or if applying it would require to delete any used
// device of the GPU.
func (g *GPU) ApplyGeometry(geometry Geometry) error {
	canApply, reason := g.CanApplyGeometry(geometry)
	if !canApply {
		return fmt.Errorf(reason)
	}
	// Apply geometry by changing free devices
	for profile, quantity := range geometry {
		g.freeMigDevices[profile] = quantity - g.usedMigDevices[profile]
	}
	// Delete all free devices not included in the new geometry
	for profile := range g.freeMigDevices {
		if _, ok := geometry[profile]; !ok {
			delete(g.freeMigDevices, profile)
		}
	}

	return nil
}

// UpdateGeometryFor updates the geometry of the GPU in order to create the highest possible number of required
// profiles provided as argument, without deleting any of the used profiles.
// The method returns the required profiles that have been created by applying the new geometry.
//
// If multiple geometries provides the required profiles, the method chooses the one that allows to create
// the highest number of new profiles.
//
// The GPU geometry gets updated only if it was possible to create at least one of the required profiles.
func (g *GPU) UpdateGeometryFor(requiredProfiles map[ProfileName]int) map[ProfileName]int {
	var createdProfiles = make(map[ProfileName]int)
	var geometryNumProvidedProfiles = make(map[string]int)
	var geometryLookup = make(map[string]Geometry)
	var bestGeometry *Geometry

	// For each allowed geometry, compute the number of required profiles that it can provide
	for _, candidate := range g.GetAllowedGeometries() {
		for requiredProfile, requiredQuantity := range requiredProfiles {
			// If Node already provides the profile resources then there's nothing to do
			if g.freeMigDevices[requiredProfile] >= requiredQuantity {
				continue
			}
			// If the candidate geometry does not provide the required profile, then skip it
			nFreeProfilesForGeometry := candidate[requiredProfile] - g.usedMigDevices[requiredProfile]
			if nFreeProfilesForGeometry <= 0 {
				continue
			}
			// If we cannot apply the geometry, then skip it
			if canApplyGeometry, _ := g.CanApplyGeometry(candidate); !canApplyGeometry {
				continue
			}
			candidateGeometryId := candidate.Id()
			geometryNumProvidedProfiles[candidateGeometryId] += nFreeProfilesForGeometry
			geometryLookup[candidateGeometryId] = candidate
		}
	}

	// Find, if any, the geometry that allows to create the highest number of required profiles
	maxFreeProfiles := 0
	for candidateId, nFreeProfiles := range geometryNumProvidedProfiles {
		if nFreeProfiles > maxFreeProfiles {
			maxFreeProfiles = nFreeProfiles
			candidate := geometryLookup[candidateId]
			bestGeometry = &candidate
		}
	}

	// No geometry can provide the required profiles, we're done
	if bestGeometry == nil {
		return createdProfiles
	}

	// Compute the required profiles that the best geometry provides
	currentGeometry := g.GetGeometry()
	for profile, quantity := range *bestGeometry {
		diff := quantity - currentGeometry[profile]
		if diff > 0 {
			createdProfiles[profile] = diff
		}
	}

	// Apply the new geometry
	if len(createdProfiles) > 0 {
		_ = g.ApplyGeometry(*bestGeometry)
	}

	return createdProfiles
}

// AllowsGeometry returns true if the geometry provided as argument is allowed by the GPU model
func (g *GPU) AllowsGeometry(geometry Geometry) bool {
	for _, allowedGeometry := range g.GetAllowedGeometries() {
		if cmp.Equal(geometry, allowedGeometry) {
			return true
		}
	}
	return false
}

// GetAllowedGeometries returns the MIG geometries allowed by the GPU model
func (g *GPU) GetAllowedGeometries() []Geometry {
	return g.allowedMigGeometries
}

// AddPod adds a Pod to the GPU by updating the free and used MIG devices according to the MIG resources
// requested by the Pod.
//
// AddPod returns an error if the GPU does not have enough free MIG resources for the Pod.
func (g *GPU) AddPod(pod v1.Pod) error {
	for r, q := range GetRequestedMigResources(pod) {
		if g.freeMigDevices[r] < q {
			return fmt.Errorf(
				"not enough free MIG devices (pod requests %d %s, but GPU only has %d)",
				q,
				r,
				g.freeMigDevices[r],
			)
		}
		g.freeMigDevices[r] -= q
		g.usedMigDevices[r] += q
	}
	return nil
}

func (g *GPU) HasFreeMigDevices() bool {
	return len(g.GetFreeMigDevices()) > 0
}

func (g *GPU) GetFreeMigDevices() map[ProfileName]int {
	return g.freeMigDevices
}

func (g *GPU) GetUsedMigDevices() map[ProfileName]int {
	return g.usedMigDevices
}
