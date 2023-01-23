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

package gpu_test

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestSpecAnnotation_GetGpuIndex(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   int
	}{
		{
			name:       "Get Index",
			annotation: fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 2, "1g.10gb"),
			expected:   2,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := gpu.ParseSpecAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.Index)
		})
	}
}

func TestSpecAnnotation_GetProfile(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   string
	}{
		{
			name:       "Get profile",
			annotation: fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 2, "1g.10gb"),
			expected:   "1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := gpu.ParseSpecAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.ProfileName)
		})
	}
}

func TestSpecAnnotation_GetIndexWithProfile(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   string
	}{
		{
			name:       "Get GPU index with profile",
			annotation: fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 2, "1g.10gb"),
			expected:   "2-1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := gpu.ParseSpecAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetIndexWithProfile())
		})
	}
}

func TestStatusAnnotationList_GetFree(t *testing.T) {
	testCases := []struct {
		name     string
		list     gpu.StatusAnnotationList
		expected gpu.StatusAnnotationList
	}{
		{
			name:     "Empty list",
			list:     gpu.StatusAnnotationList{},
			expected: gpu.StatusAnnotationList{},
		},
		{
			name: "Only used annotations",
			list: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					ProfileName: "1g10gb",
					Index:       0,
					Status:      resource.StatusUsed,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Index:       0,
					Status:      resource.StatusUsed,
					Quantity:    1,
				},
			},
			expected: gpu.StatusAnnotationList{},
		},
		{
			name: "Used and Free annotations, only Free are returned",
			list: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					ProfileName: "1g10gb",
					Index:       0,
					Status:      resource.StatusUsed,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Index:       0,
					Status:      resource.StatusUsed,
					Quantity:    1,
				},
				gpu.StatusAnnotation{
					ProfileName: "1g10gb",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
			},
			expected: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					ProfileName: "1g10gb",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
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

func TestStatusAnnotationList_GetUsed(t *testing.T) {
	testCases := []struct {
		name     string
		list     gpu.StatusAnnotationList
		expected gpu.StatusAnnotationList
	}{
		{
			name:     "Empty list",
			list:     gpu.StatusAnnotationList{},
			expected: gpu.StatusAnnotationList{},
		},
		{
			name: "Only free annotations",
			list: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					ProfileName: "1g10gb",
					Status:      resource.StatusFree,
					Index:       0,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					Index:       0,
					ProfileName: "2g20gb",
					Status:      resource.StatusFree,
					Quantity:    1,
				},
			},
			expected: gpu.StatusAnnotationList{},
		},
		{
			name: "Used and Free annotations, only Used are returned",
			list: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					Status:      resource.StatusUsed,
					Index:       0,
					ProfileName: "1g10gb",
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Status:      resource.StatusUsed,
					Index:       0,
					Quantity:    1,
				},
				gpu.StatusAnnotation{
					Index:       0,
					ProfileName: "1g10gb",
					Status:      resource.StatusFree,
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
			},
			expected: gpu.StatusAnnotationList{
				gpu.StatusAnnotation{
					Status:      resource.StatusUsed,
					Index:       0,
					ProfileName: "1g10gb",
					Quantity:    2,
				},
				gpu.StatusAnnotation{
					ProfileName: "2g20gb",
					Status:      resource.StatusUsed,
					Index:       0,
					Quantity:    1,
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
		expectedStatusAnnotations gpu.StatusAnnotationList
		expectedSpecAnnotations   gpu.SpecAnnotationList
	}{
		{
			name:                      "Node without annotations",
			node:                      v1.Node{},
			expectedStatusAnnotations: make(gpu.StatusAnnotationList, 0),
			expectedSpecAnnotations:   make(gpu.SpecAnnotationList, 0),
		},
		{
			name: "Node with annotations",
			node: factory.BuildNode("test").
				WithAnnotations(
					map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 2, "1g.10gb"): "1",
						fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, "2g.10gb"): "2",
						"nos.nebuly.com/status-gpu-0-1g.10gb-free":                  "3",
					},
				).
				Get(),
			expectedStatusAnnotations: gpu.StatusAnnotationList{
				{
					ProfileName: "1g.10gb",
					Status:      resource.StatusFree,
					Index:       0,
					Quantity:    3,
				},
			},
			expectedSpecAnnotations: gpu.SpecAnnotationList{
				{
					ProfileName: "1g.10gb",
					Index:       2,
					Quantity:    1,
				},
				{
					ProfileName: "2g.10gb",
					Index:       1,
					Quantity:    2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			status, spec := gpu.ParseNodeAnnotations(tt.node)
			assert.ElementsMatch(t, tt.expectedStatusAnnotations, status)
			assert.ElementsMatch(t, tt.expectedSpecAnnotations, spec)
		})
	}
}

func TestParseStatusAnnotation(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		value       string
		expected    gpu.StatusAnnotation
		expectedErr bool
	}{
		{
			name:        "Empty key and value",
			key:         "",
			value:       "",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key without prefix",
			key:         "nos.nebuly.com/foo",
			value:       "1",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key with prefix, but without status",
			key:         v1alpha1.AnnotationGpuStatusPrefix + "foo",
			value:       "1",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Quantity is not an integer",
			key:         fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "1g.10gb", resource.StatusFree),
			value:       "foo",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Index is not an integer",
			key:         "nos.nebuly.com/status-gpu-foo-1g.10gb-free",
			value:       "1",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Invalid status",
			key:         "nos.nebuly.com/status-gpu-0-1g.10gb-foo",
			value:       "1",
			expected:    gpu.StatusAnnotation{},
			expectedErr: true,
		},
		{
			name:  "Valid annotation",
			key:   "nos.nebuly.com/status-gpu-1-1g.10gb-used",
			value: "1",
			expected: gpu.StatusAnnotation{
				ProfileName: "1g.10gb",
				Status:      resource.StatusUsed,
				Index:       1,
				Quantity:    1,
			},
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := gpu.ParseStatusAnnotation(tt.key, tt.value)
			if tt.expectedErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.expected, annotation)
		})
	}
}

func TestParseSpecAnnotation(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		value       string
		expected    gpu.SpecAnnotation
		expectedErr bool
	}{
		{
			name:        "Empty key and value",
			key:         "",
			value:       "",
			expected:    gpu.SpecAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key without prefix",
			key:         "nos.nebuly.com/foo",
			value:       "1",
			expected:    gpu.SpecAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Key with prefix, but without spec",
			key:         v1alpha1.AnnotationGpuSpecPrefix + "foo",
			value:       "1",
			expected:    gpu.SpecAnnotation{},
			expectedErr: true,
		},
		{
			name:        "Quantity is not an integer",
			key:         fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, "1g.10gb"),
			value:       "foo",
			expected:    gpu.SpecAnnotation{},
			expectedErr: true,
		},
		{
			name:  "Valid annotation",
			key:   fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, "1g.10gb"),
			value: "1",
			expected: gpu.SpecAnnotation{
				ProfileName: "1g.10gb",
				Index:       1,
				Quantity:    1,
			},
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, value := gpu.ParseSpecAnnotation(tt.key, tt.value)
			if tt.expectedErr {
				assert.Error(t, value)
			}
			assert.Equal(t, tt.expected, annotation)
		})
	}
}
