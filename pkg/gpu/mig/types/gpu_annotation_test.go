package types

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/stretchr/testify/assert"
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
		expected   string
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
