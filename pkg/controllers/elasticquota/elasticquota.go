package elasticquota

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	resource2 "github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	quota "k8s.io/apiserver/pkg/quota/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
)

type elasticQuotaPodsReconciler struct {
	c                  client.Client
	resourceCalculator *resource2.Calculator
}

func (r *elasticQuotaPodsReconciler) PatchPodsAndComputeUsedQuota(ctx context.Context,
	pods []v1.Pod,
	quotaMin v1.ResourceList,
	quotaMax v1.ResourceList) (v1.ResourceList, error) {

	// Sort pods for finding overquotas
	r.sortPodListForFindingOverQuotaPods(pods)

	used := newZeroUsed(quotaMin, quotaMax)
	var err error
	for _, pod := range pods {
		request := r.resourceCalculator.ComputePodRequest(pod)
		used = quota.Add(used, request)

		var desiredCapacityInfo constant.CapacityInfo
		if less, _ := quota.LessThanOrEqual(used, quotaMin); less {
			desiredCapacityInfo = constant.CapacityInfoInQuota
		} else {
			desiredCapacityInfo = constant.CapacityInfoOverQuota
		}

		if _, err = r.patchCapacityInfoIfDifferent(ctx, &pod, desiredCapacityInfo); err != nil {
			return nil, err
		}
	}

	// Remove resources that are not enforced by ElasticQuota limits
	for r := range used {
		if _, ok := quotaMin[r]; !ok {
			delete(used, r)
		}
	}

	return used, nil
}

// sortPodListForFindingOverQuotaPods sorts the input list so that it can be used for finding the Pods that are
// "over-quota" (e.g. they are borrowing quotas from another namespace) and the ones that are "in-quota" (e.g.
// in their respective ElasticQuota used <= min)
func (r *elasticQuotaPodsReconciler) sortPodListForFindingOverQuotaPods(pods []v1.Pod) {
	sort.Slice(pods, func(i, j int) bool {
		// If creation timestamp is not the same, sort by creation timestamp
		firstPodCT := pods[i].ObjectMeta.CreationTimestamp
		secondPodCT := pods[j].ObjectMeta.CreationTimestamp
		if !firstPodCT.Equal(&secondPodCT) {
			return firstPodCT.Before(&secondPodCT)
		}

		// If priority is not the same, sort by priority
		firstPodPriority := *pods[i].Spec.Priority
		secondPodPriority := *pods[j].Spec.Priority
		if firstPodPriority != secondPodPriority {
			return firstPodPriority < secondPodPriority
		}

		// If resource request is not the same, sort by resource request
		firstPodRequest := r.resourceCalculator.ComputePodRequest(pods[i])
		secondPodRequest := r.resourceCalculator.ComputePodRequest(pods[j])
		if !quota.Equals(firstPodRequest, secondPodRequest) {
			less, _ := quota.LessThanOrEqual(firstPodRequest, secondPodRequest)
			return less
		}

		// As last resort, sort by name alphabetically
		return pods[i].Name < pods[j].Name
	})
}

// newZeroUsed will return the zero value of the union of min and max
func newZeroUsed(min v1.ResourceList, max v1.ResourceList) v1.ResourceList {
	minResources := quota.ResourceNames(min)
	maxResources := quota.ResourceNames(max)
	res := v1.ResourceList{}
	for _, v := range minResources {
		res[v] = *resource.NewQuantity(0, resource.DecimalSI)
	}
	for _, v := range maxResources {
		res[v] = *resource.NewQuantity(0, resource.DecimalSI)
	}
	return res
}

func (r *elasticQuotaPodsReconciler) patchCapacityInfoIfDifferent(ctx context.Context,
	pod *v1.Pod,
	desired constant.CapacityInfo) (bool, error) {

	logger := log.FromContext(ctx)
	desiredAsString := string(desired)
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	if pod.Labels[constant.LabelCapacityInfo] != desiredAsString {
		logger.V(1).Info(
			"updating Pod capacity info label",
			"currentValue",
			pod.Labels[constant.LabelCapacityInfo],
			"desiredValue",
			desiredAsString,
			"Pod",
			pod,
		)
		original := pod.DeepCopy()
		pod.Labels[constant.LabelCapacityInfo] = desiredAsString
		if err := r.c.Patch(ctx, pod, client.MergeFrom(original)); err != nil {
			msg := fmt.Sprintf("unable to update label %q with value %q", constant.LabelCapacityInfo, desiredAsString)
			logger.Error(err, msg, "pod", original)
			return false, err
		}
		return true, nil
	}
	return false, nil
}
