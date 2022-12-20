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

package core_test

//import (
//	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
//	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig/migstate"
//	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
//	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
//	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
//	"github.com/stretchr/testify/assert"
//	v1 "k8s.io/api/core/v1"
//	"k8s.io/kubernetes/pkg/scheduler/framework"
//	"testing"
//)
//
//func newClusterSnapshotOrPanic(snapshot state.clusterSnapshot) migstate.MigClusterSnapshot {
//	s, err := migstate.NewClusterSnapshot(snapshot)
//	if err != nil {
//		panic(err)
//	}
//	return s
//}
//
//func TestLackingMigProfilesTracker_Remove(t *testing.T) {
//	testCases := []struct {
//		name                             string
//		snapshot                         migstate.MigClusterSnapshot
//		pods                             []v1.Pod
//		podToRemove                      v1.Pod
//		expectedRequestedMigProfiles     map[mig.ProfileName]int
//		expectedLackingMigProfiles       map[mig.ProfileName]int
//		expectedLackingMigProfilesLookup map[string]map[mig.ProfileName]int
//	}{
//		{
//			name:                             "Empty snapshot, empty tracker",
//			snapshot:                         newClusterSnapshotOrPanic(state.NewClusterSnapshot(map[string]framework.NodeInfo{})),
//			pods:                             []v1.Pod{},
//			podToRemove:                      v1.Pod{},
//			expectedRequestedMigProfiles:     map[mig.ProfileName]int{},
//			expectedLackingMigProfiles:       map[mig.ProfileName]int{},
//			expectedLackingMigProfilesLookup: map[string]map[mig.ProfileName]int{},
//		},
//		{
//			name:                             "Pod not tracked",
//			snapshot:                         newClusterSnapshotOrPanic(state.NewClusterSnapshot(map[string]framework.NodeInfo{})),
//			pods:                             []v1.Pod{},
//			podToRemove:                      factory.BuildPod("ns-1", "pd-1").Get(),
//			expectedRequestedMigProfiles:     map[mig.ProfileName]int{},
//			expectedLackingMigProfiles:       map[mig.ProfileName]int{},
//			expectedLackingMigProfilesLookup: map[string]map[mig.ProfileName]int{},
//		},
//		{
//			name:     "Quantities <= 0 should be removed",
//			snapshot: newClusterSnapshotOrPanic(state.NewClusterSnapshot(map[string]framework.NodeInfo{})),
//			pods: []v1.Pod{
//				factory.BuildPod("ns-1", "pd-1").WithContainer(
//					factory.BuildContainer("c1", "test").
//						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
//						WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 2).
//						Get(),
//				).Get(),
//				factory.BuildPod("ns-1", "pd-2").WithContainer(
//					factory.BuildContainer("c1", "test").
//						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
//						Get(),
//				).Get(),
//			},
//			podToRemove: factory.BuildPod("ns-1", "pd-1").WithContainer(
//				factory.BuildContainer("c1", "test").
//					WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
//					WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 2).
//					Get(),
//			).Get(),
//			expectedRequestedMigProfiles: map[mig.ProfileName]int{
//				mig.Profile1g10gb: 1,
//			},
//			expectedLackingMigProfiles: map[mig.ProfileName]int{
//				mig.Profile1g10gb: 1,
//			},
//			expectedLackingMigProfilesLookup: map[string]map[mig.ProfileName]int{
//				"ns-1/pd-1": {},
//				"ns-1/pd-2": {
//					mig.Profile1g10gb: 1,
//				},
//			},
//		},
//	}
//
//	for _, tt := range testCases {
//		t.Run(tt.name, func(t *testing.T) {
//			tracker := core.NewSliceTracker(
//				tt.snapshot,
//				tt.pods,
//			)
//			tracker.Remove(tt.podToRemove)
//			assert.Equal(t, tt.expectedLackingMigProfilesLookup, tracker.lackingMigProfilesLookup)
//			assert.Equal(t, tt.expectedRequestedMigProfiles, tracker.GetRequestedMigProfiles())
//			assert.Equal(t, tt.expectedLackingMigProfiles, tracker.GetLackingMigProfiles())
//		})
//	}
//}
