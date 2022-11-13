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
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsNvidiaMigDevice(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expected     bool
	}{
		{
			name:         "Empty string",
			resourceName: "",
			expected:     false,
		},
		{
			name:         "Generic resource",
			resourceName: "nvidia.com/gpu",
			expected:     false,
		},
		{
			name:         "Malformed NVIDIA MIG",
			resourceName: "nvidia.com/mig-1ga1gb",
			expected:     false,
		},
		{
			name:         "Valid NVIDIA MIG",
			resourceName: "nvidia.com/mig-1g.1gb",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNvidiaMigDevice(v1.ResourceName(tt.resourceName)))
		})
	}
}

func TestExtractMemoryGBFromMigDevice(t *testing.T) {
	tests := []struct {
		name          string
		resourceName  string
		errorExpected bool
		expected      int64
	}{
		{
			name:          "Empty string",
			resourceName:  "",
			errorExpected: true,
		},
		{
			name:          "Generic resource",
			resourceName:  "nvidia.com/gpu",
			errorExpected: true,
		},
		{
			name:          "Malformed NVIDIA MIG",
			resourceName:  "nvidia.com/mig-1g12agb",
			errorExpected: true,
		},
		{
			name:          "Malformed NVIDIA MIG - multiple occurrences",
			resourceName:  "nvidia.com/mig-1g.1gb15gb",
			errorExpected: true,
		},
		{
			name:          "Valid NVIDIA MIG",
			resourceName:  "nvidia.com/mig-1g.16gb",
			errorExpected: false,
			expected:      16,
		},
		{
			name:          "Invalid NVIDIA MIG-like format",
			resourceName:  "foo/1g2gb",
			errorExpected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory, err := ExtractMemoryGBFromMigFormat(v1.ResourceName(tt.resourceName))
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, memory)
			}
		})
	}
}

func TestSpecMatchesStatusAnnotations(t *testing.T) {
	testCases := []struct {
		name     string
		status   map[string]string
		spec     map[string]string
		expected bool
	}{
		{
			name:     "Empty maps",
			status:   make(map[string]string),
			spec:     make(map[string]string),
			expected: true,
		},
		{
			name: "Matches",
			status: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 1, "1g.20gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 1, "1g.20gb"): "2",
			},
			spec: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "2g.40gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.20gb"): "4",
			},
			expected: true,
		},
		{
			name: "Do not matches",
			status: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 1, "1g.20gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 1, "1g.20gb"): "2",
			},
			spec: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "2g.40gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.20gb"): "4",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "4g.40gb"): "1",
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			specAnnotations := make([]GPUSpecAnnotation, 0)
			for k, v := range tt.spec {
				a, _ := NewGPUSpecAnnotationFromNodeAnnotation(k, v)
				specAnnotations = append(specAnnotations, a)
			}

			statusAnnotations := make([]GPUStatusAnnotation, 0)
			for k, v := range tt.status {
				a, _ := NewGPUStatusAnnotation(k, v)
				statusAnnotations = append(statusAnnotations, a)
			}

			matches := SpecMatchesStatus(specAnnotations, statusAnnotations)
			assert.Equal(t, tt.expected, matches)
		})
	}
}
