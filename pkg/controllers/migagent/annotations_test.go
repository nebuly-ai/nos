package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 1, "1g.20gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 1, "1g.20gb"): "2",
			},
			spec: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 0, "1g.10gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 0, "2g.40gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 1, "1g.20gb"): "4",
			},
			expected: true,
		},
		{
			name: "Do not matches",
			status: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 0, "1g.10gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 0, "2g.40gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, 1, "1g.20gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, 1, "1g.20gb"): "2",
			},
			spec: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 0, "1g.10gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 0, "2g.40gb"): "2",
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 1, "1g.20gb"): "4",
				fmt.Sprintf(v1alpha1.AnnotationGPUSpecFormat, 1, "4g.40gb"): "1",
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotations := make(map[string]string)
			for k, v := range tt.spec {
				annotations[k] = v
			}
			for k, v := range tt.status {
				annotations[k] = v
			}
			node := factory.BuildNode("test").WithAnnotations(annotations).Get()
			matches := specMatchesStatusAnnotations(node)
			assert.Equal(t, tt.expected, matches)
		})
	}
}
