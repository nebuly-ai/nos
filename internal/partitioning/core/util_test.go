/*
 * Copyright 2023 nebuly.com.
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

import (
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	mig_partitioner "github.com/nebuly-ai/nos/internal/partitioning/mig"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestPodSorter(t *testing.T) {
	testCases := []struct {
		name     string
		pods     []v1.Pod
		expected []v1.Pod
	}{
		{
			name:     "Empty list",
			pods:     make([]v1.Pod, 0),
			expected: make([]v1.Pod, 0),
		},
		{
			name: "Single pod",
			pods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(),
			},
			expected: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(),
			},
		},
		{
			name: "Pod with same priority not requesting MIG resources, order should not change",
			pods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(),
				factory.BuildPod("ns-1", "pd-2").Get(),
				factory.BuildPod("ns-1", "pd-3").Get(),
			},
			expected: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(),
				factory.BuildPod("ns-1", "pd-2").Get(),
				factory.BuildPod("ns-1", "pd-3").Get(),
			},
		},
		{
			name: "Pod with different priorities: Pod with higher priority should be first",
			pods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").WithPriority(1).WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-2").WithPriority(2).WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile7g79gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
			expected: []v1.Pod{
				factory.BuildPod("ns-1", "pd-2").WithPriority(2).WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile7g79gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-1").WithPriority(1).WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
		},
		{
			name: "Pod with MIG Resources: Pod requesting smaller MIG profiles should be first",
			pods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-2").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-3").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile4g20gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
			expected: []v1.Pod{
				factory.BuildPod("ns-1", "pd-2").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-3").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile4g20gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sliceCalculator := mig_partitioner.NewSliceCalculator()
			res := core.NewPodSorter(sliceCalculator).Sort(tt.pods)
			assert.Equal(t, tt.expected, res)
		})
	}
}
