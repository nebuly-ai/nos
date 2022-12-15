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

package mig_test

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestGPUSpecAnnotation_GetGPUIndex(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   int
	}{
		{
			name:       "Get Index",
			annotation: fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   2,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := mig.NewGPUSpecAnnotationFromNodeAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetGPUIndex())
		})
	}
}

func TestGPUSpecAnnotation_GetMigProfile(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   mig.ProfileName
	}{
		{
			name:       "Get MIG profile",
			annotation: fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   "1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := mig.NewGPUSpecAnnotationFromNodeAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetMigProfileName())
		})
	}
}

func TestGPUSpecAnnotation_GetGpuIndexWithMigProfile(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   string
	}{
		{
			name:       "Get GPU index with MIG profile",
			annotation: fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   "2-1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := mig.NewGPUSpecAnnotationFromNodeAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetGPUIndexWithMigProfile())
		})
	}
}

func TestGPUStatusAnnotationList_GetFree(t *testing.T) {
	testCases := []struct {
		name     string
		list     mig.GPUStatusAnnotationList
		expected mig.GPUStatusAnnotationList
	}{
		{
			name:     "Empty list",
			list:     mig.GPUStatusAnnotationList{},
			expected: mig.GPUStatusAnnotationList{},
		},
		{
			name: "Only used annotations",
			list: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile1g10gb,
					Index:    0,
					Status:   resource.StatusUsed,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Index:    0,
					Status:   resource.StatusUsed,
					Quantity: 1,
				},
			},
			expected: mig.GPUStatusAnnotationList{},
		},
		{
			name: "Used and Free annotations, only Free are returned",
			list: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile1g10gb,
					Index:    0,
					Status:   resource.StatusUsed,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Index:    0,
					Status:   resource.StatusUsed,
					Quantity: 1,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile1g10gb,
					Index:    0,
					Status:   resource.StatusFree,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Index:    0,
					Status:   resource.StatusFree,
					Quantity: 1,
				},
			},
			expected: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile1g10gb,
					Index:    0,
					Status:   resource.StatusFree,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Index:    0,
					Status:   resource.StatusFree,
					Quantity: 1,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			free := tt.list.GetFree()
			assert.ElementsMatch(t, tt.expected, free)
		})
	}
}

func TestGPUStatusAnnotationList_GetUsed(t *testing.T) {
	testCases := []struct {
		name     string
		list     mig.GPUStatusAnnotationList
		expected mig.GPUStatusAnnotationList
	}{
		{
			name:     "Empty list",
			list:     mig.GPUStatusAnnotationList{},
			expected: mig.GPUStatusAnnotationList{},
		},
		{
			name: "Only free annotations",
			list: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile1g10gb,
					Status:   resource.StatusFree,
					Index:    0,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Index:    0,
					Profile:  mig.Profile2g20gb,
					Status:   resource.StatusFree,
					Quantity: 1,
				},
			},
			expected: mig.GPUStatusAnnotationList{},
		},
		{
			name: "Used and Free annotations, only Used are returned",
			list: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Status:   resource.StatusUsed,
					Index:    0,
					Profile:  mig.Profile1g10gb,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Status:   resource.StatusUsed,
					Index:    0,
					Quantity: 1,
				},
				mig.GPUStatusAnnotation{
					Index:    0,
					Profile:  mig.Profile1g10gb,
					Status:   resource.StatusFree,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Index:    0,
					Status:   resource.StatusFree,
					Quantity: 1,
				},
			},
			expected: mig.GPUStatusAnnotationList{
				mig.GPUStatusAnnotation{
					Status:   resource.StatusUsed,
					Index:    0,
					Profile:  mig.Profile1g10gb,
					Quantity: 2,
				},
				mig.GPUStatusAnnotation{
					Profile:  mig.Profile2g20gb,
					Status:   resource.StatusUsed,
					Index:    0,
					Quantity: 1,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			used := tt.list.GetUsed()
			assert.ElementsMatch(t, tt.expected, used)
		})
	}
}

func TestGetGPUAnnotationsFromNode(t *testing.T) {
	testCases := []struct {
		name                      string
		node                      v1.Node
		expectedStatusAnnotations []mig.GPUStatusAnnotation
		expectedSpecAnnotations   []mig.GPUSpecAnnotation
	}{
		{
			name:                      "Node without annotations",
			node:                      v1.Node{},
			expectedStatusAnnotations: make([]mig.GPUStatusAnnotation, 0),
			expectedSpecAnnotations:   make([]mig.GPUSpecAnnotation, 0),
		},
		{
			name: "Node with annotations",
			node: factory.BuildNode("test").
				WithAnnotations(
					map[string]string{
						fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 2, "1g.10gb"): "1",
						fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 1, "2g.10gb"): "2",
						"n8s.nebuly.ai/status-gpu-0-1g.10gb-free":                 "3",
					},
				).
				Get(),
			expectedStatusAnnotations: []mig.GPUStatusAnnotation{
				{
					Profile:  mig.Profile1g10gb,
					Status:   resource.StatusFree,
					Index:    0,
					Quantity: 3,
				},
			},
			expectedSpecAnnotations: []mig.GPUSpecAnnotation{
				{
					Name:     fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
					Quantity: 1,
				},
				{
					Name:     fmt.Sprintf(mig.AnnotationGPUMigSpecFormat, 1, "2g.10gb"),
					Quantity: 2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			status, spec := mig.GetGPUAnnotationsFromNode(tt.node)
			assert.ElementsMatch(t, tt.expectedStatusAnnotations, status)
			assert.ElementsMatch(t, tt.expectedSpecAnnotations, spec)
		})
	}
}

func TestParseGPUStatusAnnotation(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		value       string
		expected    mig.GPUStatusAnnotation
		expectedErr bool
	}{
		{
			name:        "Empty key and value",
			key:         "",
			value:       "",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key without prefix",
			key:         "n8s.nebuly.ai/foo",
			value:       "1",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key with prefix, but without status",
			key:         v1alpha1.AnnotationGPUStatusPrefix + "foo",
			value:       "1",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Quantity is not an integer",
			key:         fmt.Sprintf(mig.AnnotationMigStatusFormat, 0, "1g.10gb", resource.StatusFree),
			value:       "foo",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Index is not an integer",
			key:         "n8s.nebuly.ai/status-gpu-foo-1g.10gb-free",
			value:       "1",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Invalid status",
			key:         "n8s.nebuly.ai/status-gpu-0-1g.10gb-foo",
			value:       "1",
			expected:    mig.GPUStatusAnnotation{},
			expectedErr: true,
		},
		{
			name:  "Valid annotation",
			key:   "n8s.nebuly.ai/status-gpu-1-1g.10gb-used",
			value: "1",
			expected: mig.GPUStatusAnnotation{
				Profile:  mig.Profile1g10gb,
				Status:   resource.StatusUsed,
				Index:    1,
				Quantity: 1,
			},
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := mig.ParseGPUStatusAnnotation(tt.key, tt.value)
			if tt.expectedErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.expected, annotation)
		})
	}
}
