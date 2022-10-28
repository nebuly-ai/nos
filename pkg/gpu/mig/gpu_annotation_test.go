package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
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
			annotation: fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   2,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := NewGPUSpecAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetGPUIndex())
		})
	}
}

func TestGPUSpecAnnotation_GetMigProfile(t *testing.T) {
	testCases := []struct {
		name       string
		annotation string
		expected   ProfileName
	}{
		{
			name:       "Get MIG profile",
			annotation: fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   "1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := NewGPUSpecAnnotation(tt.annotation, "1")
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
			annotation: fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
			expected:   "2-1g.10gb",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := NewGPUSpecAnnotation(tt.annotation, "1")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, annotation.GetGPUIndexWithMigProfile())
		})
	}
}

func TestGetGPUAnnotationsFromNode(t *testing.T) {
	testCases := []struct {
		name                      string
		node                      v1.Node
		expectedStatusAnnotations []GPUStatusAnnotation
		expectedSpecAnnotations   []GPUSpecAnnotation
	}{
		{
			name:                      "Node without annotations",
			node:                      v1.Node{},
			expectedStatusAnnotations: make([]GPUStatusAnnotation, 0),
			expectedSpecAnnotations:   make([]GPUSpecAnnotation, 0),
		},
		{
			name: "Node with annotations",
			node: factory.BuildNode("test").
				WithAnnotations(
					map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 2, "1g.10gb"): "1",
						fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "2g.10gb"): "2",
						"n8s.nebuly.ai/status-gpu-0-1g.10gb-free":                      "3",
					},
				).
				Get(),
			expectedStatusAnnotations: []GPUStatusAnnotation{
				{
					Name:     "n8s.nebuly.ai/status-gpu-0-1g.10gb-free",
					Quantity: 3,
				},
			},
			expectedSpecAnnotations: []GPUSpecAnnotation{
				{
					Name:     fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 2, "1g.10gb"),
					Quantity: 1,
				},
				{
					Name:     fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "2g.10gb"),
					Quantity: 2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			status, spec := GetGPUAnnotationsFromNode(tt.node)
			assert.ElementsMatch(t, tt.expectedStatusAnnotations, status)
			assert.ElementsMatch(t, tt.expectedSpecAnnotations, spec)
		})
	}
}
