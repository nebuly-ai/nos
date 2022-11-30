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
	allLackingMigProfiles map[mig.ProfileName]int
	// podsLackingMigProfiles is a lookup table that associates each pod namespaced name with the MIG profiles it lacks
	podsLackingMigProfiles map[string]map[mig.ProfileName]int
}

func newLackingMigProfilesTracker(snapshot migstate.MigClusterSnapshot, pods []v1.Pod) lackingMigProfilesTracker {
	allLackingMigProfiles := make(map[mig.ProfileName]int)
	podsLackingMigProfiles := make(map[string]map[mig.ProfileName]int)
	for _, pod := range pods {
		podKey := util.GetNamespacedName(&pod).String()
		if podsLackingMigProfiles[podKey] == nil {
			podsLackingMigProfiles[podKey] = make(map[mig.ProfileName]int)
		}
		for profile, quantity := range snapshot.GetLackingMigProfiles(pod) {
			allLackingMigProfiles[profile] += quantity
			podsLackingMigProfiles[podKey][profile] += quantity
		}
	}
	return lackingMigProfilesTracker{
		allLackingMigProfiles:  allLackingMigProfiles,
		podsLackingMigProfiles: podsLackingMigProfiles,
	}
}

func (t lackingMigProfilesTracker) GetLackingMigProfiles() map[mig.ProfileName]int {
	return t.allLackingMigProfiles
}

func (t lackingMigProfilesTracker) Remove(pod v1.Pod) {
	lackingMigProfiles, ok := t.podsLackingMigProfiles[util.GetNamespacedName(&pod).String()]
	if !ok {
		return
	}

	for profile, quantity := range lackingMigProfiles {
		t.allLackingMigProfiles[profile] -= quantity
		lackingMigProfiles[profile] -= quantity
		if lackingMigProfiles[profile] <= 0 {
			delete(lackingMigProfiles, profile)
		}
		if t.allLackingMigProfiles[profile] <= 0 {
			delete(t.allLackingMigProfiles, profile)
		}
	}
}
