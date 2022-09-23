package pod

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
)

// IsOverQuota foo
func IsOverQuota(pod v1.Pod) bool {
	if val, ok := pod.Labels[constant.LabelCapacityInfo]; ok {
		return val == string(constant.CapacityInfoOverQuota)
	}
	return false
}
