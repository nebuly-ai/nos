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

package plan

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestMigState_Matches(t *testing.T) {
	testCases := []struct {
		name           string
		stateResources []gpu.Device
		spec           map[string]string
		expected       bool
	}{
		{
			name:           "Empty",
			spec:           make(map[string]string),
			stateResources: make([]gpu.Device, 0),
			expected:       true,
		},
		{
			name: "Matches",
			stateResources: []gpu.Device{
				{
					Device: resource.Device{
						ResourceName: v1.ResourceName("nvidia.com/mig-1g.10gb"),
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: v1.ResourceName("nvidia.com/mig-1g.10gb"),
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: v1.ResourceName("nvidia.com/mig-2g.40gb"),
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: v1.ResourceName("nvidia.com/mig-1g.20gb"),
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: v1.ResourceName("nvidia.com/mig-1g.20gb"),
					},
					GpuIndex: 1,
				},
			},
			spec: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, "1g.10gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, "1g.20gb"): "2",
			},
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			specAnnotations := make(gpu.SpecAnnotationList[mig.ProfileName], 0)
			for k, v := range tt.spec {
				a, _ := mig.ParseSpecAnnotation(k, v)
				specAnnotations = append(specAnnotations, a)
			}
			state := NewMigState(tt.stateResources)
			matches := state.Matches(specAnnotations)
			assert.Equal(t, tt.expected, matches)
		})
	}
}
