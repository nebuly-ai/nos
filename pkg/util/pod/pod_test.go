package pod

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
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
