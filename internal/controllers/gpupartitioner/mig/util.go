package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/component-helpers/scheduling/corev1"
	"sort"
)

func SortCandidatePods(candidates []v1.Pod) []v1.Pod {
	sorted := make([]v1.Pod, len(candidates))
	copy(sorted, candidates)

	sort.SliceStable(sorted, func(i, j int) bool {
		// check priority first
		firstPodPriority := corev1.PodPriority(&sorted[i])
		secondPodPriority := corev1.PodPriority(&sorted[j])
		if firstPodPriority != secondPodPriority {
			return firstPodPriority > secondPodPriority
		}

		// if priority is equal, sort by requested MIG resources, placing first
		// the pods that require smaller MIG profiles in order to
		// maximize the number of pods that can be scheduled
		firstPodMigResources := mig.GetRequestedMigResources(sorted[i])
		if len(firstPodMigResources) == 0 {
			return false
		}
		secondPodMigResources := mig.GetRequestedMigResources(sorted[j])
		if len(secondPodMigResources) == 0 {
			return false
		}
		for firstPodProfile := range firstPodMigResources {
			for secondPodProfile := range secondPodMigResources {
				// we assume that a Pod requests at most one MIG profile
				return firstPodProfile.SmallerThan(secondPodProfile)
			}
		}

		return false
	})

	return sorted
}
