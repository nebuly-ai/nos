/*
 * Copyright 2023 Nebuly.ai.
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

package pod

import (
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsPodOverQuota(t *testing.T) {
	tests := []struct {
		name     string
		pod      v1.Pod
		expected bool
	}{
		{
			name: "Pod with label with value overquota",
			pod: factory.BuildPod("ns-1", "pd-1").
				WithLabel(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoOverQuota)).
				Get(),
			expected: true,
		},
		{
			name: "Pod with label with value inquota",
			pod: factory.BuildPod("ns-1", "pd-1").
				WithLabel(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoInQuota)).
				Get(),
			expected: false,
		},
		{
			name:     "Pod without labels",
			pod:      factory.BuildPod("ns-1", "pd-1").Get(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsOverQuota(tt.pod))
		})
	}
}
