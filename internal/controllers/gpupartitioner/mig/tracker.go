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
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig/migstate"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
)

// lackingMigProfilesTracker is a utility struct for tracking the lacking MIG resources of a list of pods
type lackingMigProfilesTracker struct {
	requestedMigProfiles     map[mig.ProfileName]int
	lackingMigProfiles       map[mig.ProfileName]int
	lackingMigProfilesLookup map[string]map[mig.ProfileName]int // Pod => lacking MIG profiles
}

func newLackingMigProfilesTracker(snapshot migstate.MigClusterSnapshot, pods []v1.Pod) lackingMigProfilesTracker {
	requestedMigProfiles := make(map[mig.ProfileName]int)
	lackingMigProfiles := make(map[mig.ProfileName]int)
	podsLackingMigProfiles := make(map[string]map[mig.ProfileName]int)
	for _, pod := range pods {
		podKey := util.GetNamespacedName(&pod).String()
		if podsLackingMigProfiles[podKey] == nil {
			podsLackingMigProfiles[podKey] = make(map[mig.ProfileName]int)
		}
		for profile, quantity := range snapshot.GetLackingMigProfiles(pod) {
			lackingMigProfiles[profile] += quantity
			podsLackingMigProfiles[podKey][profile] += quantity
		}
		for profile, quantity := range mig.GetRequestedMigResources(pod) {
			requestedMigProfiles[profile] += quantity
		}
	}
	return lackingMigProfilesTracker{
		requestedMigProfiles:     requestedMigProfiles,
		lackingMigProfiles:       lackingMigProfiles,
		lackingMigProfilesLookup: podsLackingMigProfiles,
	}
}

func (t lackingMigProfilesTracker) GetLackingMigProfiles() map[mig.ProfileName]int {
	return t.lackingMigProfiles
}

func (t lackingMigProfilesTracker) GetRequestedMigProfiles() map[mig.ProfileName]int {
	return t.requestedMigProfiles
}

func (t lackingMigProfilesTracker) Remove(pod v1.Pod) {
	// Update requested MIG profiles
	for profile, quantity := range mig.GetRequestedMigResources(pod) {
		t.requestedMigProfiles[profile] -= quantity
		if t.requestedMigProfiles[profile] <= 0 {
			delete(t.requestedMigProfiles, profile)
		}
	}
	// Update lacking MIG profiles
	if lackingMigProfiles, ok := t.lackingMigProfilesLookup[util.GetNamespacedName(&pod).String()]; ok {
		for profile, quantity := range lackingMigProfiles {
			t.lackingMigProfiles[profile] -= quantity
			lackingMigProfiles[profile] -= quantity
			if lackingMigProfiles[profile] <= 0 {
				delete(lackingMigProfiles, profile)
			}
			if t.lackingMigProfiles[profile] <= 0 {
				delete(t.lackingMigProfiles, profile)
			}
		}
	}
}
