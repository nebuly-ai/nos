/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package capacityscheduling

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"math"
)

// ElasticQuotaInfos associates namespaces with the respective ElasticQuotaInfo that defines its quota
type ElasticQuotaInfos map[string]*ElasticQuotaInfo

func NewElasticQuotaInfos() ElasticQuotaInfos {
	return make(ElasticQuotaInfos)
}

func (e ElasticQuotaInfos) clone() ElasticQuotaInfos {
	elasticQuotas := make(ElasticQuotaInfos)
	for key, elasticQuotaInfo := range e {
		elasticQuotas[key] = elasticQuotaInfo.clone()
	}
	return elasticQuotas
}

func (e ElasticQuotaInfos) Delete(eqInfo *ElasticQuotaInfo) {
	for _, ns := range eqInfo.Namespaces.List() {
		delete(e, ns)
	}
}

func (e ElasticQuotaInfos) Update(oldEqInfo, newEqInfo *ElasticQuotaInfo) {
	// Set new EqInfo to specified namespaces
	for _, ns := range newEqInfo.Namespaces.List() {
		if old, ok := e[ns]; ok && old != nil {
			newEqInfo.pods = old.pods
			newEqInfo.Used = old.Used
		}
		e[ns] = newEqInfo
	}
	// Delete possible old namespaces not specified by new EqInfo
	for _, ns := range oldEqInfo.Namespaces.List() {
		if !util.InSlice(ns, newEqInfo.Namespaces.List()) {
			delete(e, ns)
		}
	}
}

func (e ElasticQuotaInfos) Add(eqInfo *ElasticQuotaInfo) {
	for _, ns := range eqInfo.Namespaces.List() {
		e[ns] = eqInfo
	}
}

func (e ElasticQuotaInfos) AggregatedUsedOverMinWith(podRequest framework.Resource) bool {
	min := e.getAggregatedMin()
	used := e.getAggregatedUsed()
	used.Add(resource.FromFrameworkToList(podRequest))
	return greaterThan(used, min)
}

func (e ElasticQuotaInfos) GetGuaranteedOverquotas(elasticQuota string) (*framework.Resource, error) {
	eqInfo, ok := e[elasticQuota]
	if !ok {
		return nil, fmt.Errorf("elastic quota %q not present in elastic quota infos", elasticQuota)
	}

	var result = framework.NewResource(nil)
	percentages := e.getGuaranteedOverquotasPercentages(eqInfo)
	aggregatedOverquotas := e.getAggregatedOverquotas()

	result.MilliCPU = int64(math.Floor(float64(aggregatedOverquotas.MilliCPU) * percentages[v1.ResourceCPU]))
	result.Memory = int64(math.Floor(float64(aggregatedOverquotas.Memory) * percentages[v1.ResourceMemory]))
	result.AllowedPodNumber = int(math.Floor(float64(aggregatedOverquotas.AllowedPodNumber) * percentages[v1.ResourcePods]))
	result.EphemeralStorage = int64(math.Floor(float64(aggregatedOverquotas.EphemeralStorage) * percentages[v1.ResourceEphemeralStorage]))

	for r, v := range aggregatedOverquotas.ScalarResources {
		result.SetScalar(r, int64(math.Floor(float64(v)*percentages[r])))
	}

	return result, nil
}

func (e ElasticQuotaInfos) getGuaranteedOverquotasPercentages(eqInfo *ElasticQuotaInfo) map[v1.ResourceName]float64 {
	var result = make(map[v1.ResourceName]float64)
	if eqInfo.Min == nil {
		return result
	}

	var totalMin = resource.FromFrameworkToList(*e.getAggregatedMin())
	for r, m := range resource.FromFrameworkToList(*eqInfo.Min) {
		t := totalMin[r]
		var p float64
		if t.Value() > 0 {
			p = m.AsApproximateFloat64() / t.AsApproximateFloat64()
		}
		result[r] = p
	}
	return result
}

// getAggregatedOverquotas returns the total amount of quotas that can be used as "over-quotas", namely
// the quotas that ElasticQuotas can use for hosting a Pod over their Min limits.
//
// Example:
//
//	ElasticQuota A:
//		min:
//			cpu: 100m
//		used:
//			cpu: 350m
//
//	ElasticQuota B:
//		min:
//			cpu: 50m
//		used:
//			cpu: 0m
//
//	ElasticQuota C:
//		min:
//			cpu: 200m
//		used:
//			cpu: 50m
//
// Tot. available overquotas = 50m + 150m = 200m (150m of these quotas are already being used by ElasticQuota A)
func (e ElasticQuotaInfos) getAggregatedOverquotas() framework.Resource {
	var result = framework.Resource{}
	for _, eqInfo := range e {
		unused := resource.SubtractNonNegative(*eqInfo.Min, *eqInfo.Used)
		result = resource.Sum(result, unused)
	}
	return result
}

func (e ElasticQuotaInfos) getAggregatedMin() *framework.Resource {
	var totalMin = framework.Resource{}
	for _, eqi := range e {
		if eqi.Min == nil {
			continue
		}
		totalMin = resource.Sum(totalMin, *eqi.Min)
	}
	return &totalMin
}

func (e ElasticQuotaInfos) getAggregatedUsed() *framework.Resource {
	var totalUsed = framework.Resource{}
	for _, eqi := range e {
		if eqi.Used == nil {
			continue
		}
		totalUsed = resource.Sum(totalUsed, *eqi.Used)
	}
	return &totalUsed
}

// ElasticQuotaInfo wraps ElasticQuotas and CompositeElasticQuotas adding additional information and utility methods.
type ElasticQuotaInfo struct {
	// ResourceName is the name of the resource (ElasticQuota or CompositeElasticQuota)
	// associated to the ElasticQuotaInfo
	ResourceName string
	// ResourceNamespace is the namespace to which the resource (ElasticQuota or CompositeElasticQuota)
	// associated to the ElasticQuotaInfo belongs to
	ResourceNamespace string

	Namespaces         sets.String
	pods               sets.String
	Min                *framework.Resource
	Max                *framework.Resource
	Used               *framework.Resource
	MaxEnforced        bool
	resourceCalculator *gpu.Calculator
}

func (e *ElasticQuotaInfo) reserveResource(request framework.Resource) {
	e.Used.Memory += request.Memory
	e.Used.MilliCPU += request.MilliCPU
	for name, value := range request.ScalarResources {
		e.Used.SetScalar(name, e.Used.ScalarResources[name]+value)
	}
}

func (e *ElasticQuotaInfo) unreserveResource(request framework.Resource) {
	e.Used.Memory -= request.Memory
	e.Used.MilliCPU -= request.MilliCPU
	for name, value := range request.ScalarResources {
		e.Used.SetScalar(name, e.Used.ScalarResources[name]-value)
	}
}

func (e *ElasticQuotaInfo) usedOverMinWith(podRequest *framework.Resource) bool {
	return e.usedOverWith(e.Min, podRequest)
}

func (e *ElasticQuotaInfo) usedOverMaxWith(podRequest *framework.Resource) bool {
	if e.MaxEnforced {
		return e.usedOverWith(e.Max, podRequest)
	}
	return false
}

// usedOver returns true if used > min
func (e *ElasticQuotaInfo) usedOverMin() bool {
	return e.usedOver(e.Min)
}

// usedOver returns true if used > resource
func (e *ElasticQuotaInfo) usedOver(resource *framework.Resource) bool {
	return greaterThan(e.Used, resource)
}

// usedOverWith returns true if used + podRequest > resource
func (e *ElasticQuotaInfo) usedOverWith(resource *framework.Resource, podRequest *framework.Resource) bool {
	return sumGreaterThan(podRequest, e.Used, resource)
}

// usedLteWith returns true if used + podRequest <= resource
func (e *ElasticQuotaInfo) usedLteWith(resource *framework.Resource, podRequest *framework.Resource) bool {
	return sumLessThanEqual(podRequest, e.Used, resource)
}

func (e *ElasticQuotaInfo) clone() *ElasticQuotaInfo {
	newEQInfo := &ElasticQuotaInfo{
		ResourceName:       e.ResourceName,
		ResourceNamespace:  e.ResourceNamespace,
		pods:               sets.NewString(),
		Namespaces:         sets.NewString(),
		MaxEnforced:        e.MaxEnforced,
		resourceCalculator: e.resourceCalculator,
	}

	if e.Min != nil {
		newEQInfo.Min = e.Min.Clone()
	}
	if e.Max != nil {
		newEQInfo.Max = e.Max.Clone()
	}
	if e.Used != nil {
		newEQInfo.Used = e.Used.Clone()
	}
	if len(e.pods) > 0 {
		pods := e.pods.List()
		for _, pod := range pods {
			newEQInfo.pods.Insert(pod)
		}
	}
	if len(e.Namespaces) > 0 {
		namespaces := e.Namespaces.List()
		for _, ns := range namespaces {
			newEQInfo.Namespaces.Insert(ns)
		}
	}

	return newEQInfo
}

func (e *ElasticQuotaInfo) addPodIfNotPresent(pod *v1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	if e.pods.Has(key) {
		return nil
	}

	e.pods.Insert(key)
	r := e.resourceCalculator.ComputePodRequest(*pod)
	podRequest := resource.FromListToFramework(r)
	e.reserveResource(podRequest)

	return nil
}

func (e *ElasticQuotaInfo) deletePodIfPresent(pod *v1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	if !e.pods.Has(key) {
		return nil
	}

	e.pods.Delete(key)
	r := e.resourceCalculator.ComputePodRequest(*pod)
	podRequest := resource.FromListToFramework(r)
	e.unreserveResource(podRequest)

	return nil
}

// greaterThan returns true if any resource x is > of the respective resource of y.
func greaterThan(x, y *framework.Resource) bool {
	return sumGreaterThan(x, &framework.Resource{}, y)
}

// sumGreaterThan returns true if any resource of (x1 + x2) that is also present in y is > of the
// respective resource of y.
func sumGreaterThan(x1, x2, y *framework.Resource) bool {
	if x1.MilliCPU+x2.MilliCPU > y.MilliCPU {
		return true
	}

	if x1.Memory+x2.Memory > y.Memory {
		return true
	}

	allScalars := util.GetKeys(x1.ScalarResources, x2.ScalarResources, y.ScalarResources)
	for _, rName := range allScalars {
		if _, ok := y.ScalarResources[rName]; ok {
			if x1.ScalarResources[rName]+x2.ScalarResources[rName] > y.ScalarResources[rName] {
				return true
			}
		}
	}

	return false
}

// sumLessThanEqual returns true if all the resources of (x1 + x2) are less than or equal than the respective resource
// of y, and returns false if any resource of (x1 + x2) that is also present in y is > of the respective resource of y.
func sumLessThanEqual(x1, x2, y *framework.Resource) bool {
	if x1.MilliCPU+x2.MilliCPU > y.MilliCPU {
		return false
	}

	if x1.Memory+x2.Memory > y.Memory {
		return false
	}

	allScalar := util.GetKeys(x1.ScalarResources, x2.ScalarResources, y.ScalarResources)
	for _, rName := range allScalar {
		if yVal, ok := y.ScalarResources[rName]; ok {
			if x1.ScalarResources[rName]+x2.ScalarResources[rName] > yVal {
				return false
			}
		}
	}

	return true
}
