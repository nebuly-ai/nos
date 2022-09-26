package pod

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/component-helpers/scheduling/corev1"
)

// IsOverQuota returns true if the pod is "over-quota", false otherwise.
//
// A pod is considered over-quota if it is subject to an ElasticQuota, and it is using resources borrowed from another
// ElasticQuota.
func IsOverQuota(pod v1.Pod) bool {
	if val, ok := pod.Labels[constant.LabelCapacityInfo]; ok {
		return val == string(constant.CapacityInfoOverQuota)
	}
	return false
}

// ExtraResourcesCouldHelpScheduling returns true if the Pod is unschedulable
// and there a possibility that adding to the cluster additional resources
// could allow the Pod to be scheduled. Returns false otherwise.
func ExtraResourcesCouldHelpScheduling(pod v1.Pod) bool {
	return !IsScheduled(pod) &&
		IsUnschedulable(pod) &&
		!IsPreempting(pod) &&
		!IsOwnedByDaemonSet(pod) &&
		!IsOwnedByNode(pod)
}

func IsScheduled(pod v1.Pod) bool {
	return pod.Spec.NodeName != ""
}

func IsPreempting(pod v1.Pod) bool {
	return pod.Status.NominatedNodeName != ""
}

func IsUnschedulable(pod v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodScheduled && condition.Reason == v1.PodReasonUnschedulable {
			return true
		}
	}
	return false
}

func IsOwnedByDaemonSet(pod v1.Pod) bool {
	return IsOwnedBy(pod, schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "DaemonSet",
	})
}

func IsOwnedByNode(pod v1.Pod) bool {
	return IsOwnedBy(pod, schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Node",
	})
}

func IsOwnedBy(pod v1.Pod, gvk schema.GroupVersionKind) bool {
	for _, owner := range pod.ObjectMeta.OwnerReferences {
		if owner.APIVersion == gvk.GroupVersion().String() && owner.Kind == gvk.Kind {
			return true
		}
	}
	return false
}

// IsMoreImportant returns true if p1 is more important than p2, false otherwise
func IsMoreImportant(p1 v1.Pod, p2 v1.Pod) bool {
	p1Priority := corev1.PodPriority(&p1)
	p2Priority := corev1.PodPriority(&p2)
	return p1Priority > p2Priority
}
