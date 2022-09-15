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
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"math"
)

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

func (e ElasticQuotaInfos) AggregatedUsedOverMinWith(podRequest framework.Resource) bool {
	min := e.getAggregatedMin()
	used := e.getAggregatedUsed()
	used.Add(util.FromFrameworkResourceToResourceList(podRequest))
	return cmp(used, min)
}

func (e ElasticQuotaInfos) GetGuaranteedOverquotas(elasticQuota string) (*framework.Resource, error) {
	eqInfo, ok := e[elasticQuota]
	if !ok {
		return nil, fmt.Errorf("elastic quota %q not present in elastic quota infos", elasticQuota)
	}

	var result = framework.NewResource(nil)
	percentages := e.getGuaranteedOverquotasPercentages(eqInfo)
	aggregatedUnused := e.getAggregatedUnused()

	result.MilliCPU = int64(math.Floor(float64(aggregatedUnused.MilliCPU) * percentages[v1.ResourceCPU]))
	result.Memory = int64(math.Floor(float64(aggregatedUnused.Memory) * percentages[v1.ResourceMemory]))
	result.AllowedPodNumber = int(math.Floor(float64(aggregatedUnused.AllowedPodNumber) * percentages[v1.ResourcePods]))
	result.EphemeralStorage = int64(math.Floor(float64(aggregatedUnused.EphemeralStorage) * percentages[v1.ResourceEphemeralStorage]))

	for r, v := range aggregatedUnused.ScalarResources {
		result.SetScalar(r, int64(math.Floor(float64(v)*percentages[r])))
	}

	return result, nil
}

func (e ElasticQuotaInfos) getGuaranteedOverquotasPercentages(eqInfo *ElasticQuotaInfo) map[v1.ResourceName]float64 {
	var result = make(map[v1.ResourceName]float64)
	if eqInfo.Min == nil {
		return result
	}

	var totalMin = util.FromFrameworkResourceToResourceList(*e.getAggregatedMin())
	for r, m := range util.FromFrameworkResourceToResourceList(*eqInfo.Min) {
		t := totalMin[r]
		var p float64
		if t.Value() > 0 {
			p = m.AsApproximateFloat64() / t.AsApproximateFloat64()
		}
		result[r] = p
	}
	return result
}

func (e ElasticQuotaInfos) getAggregatedMin() *framework.Resource {
	var totalMin = framework.Resource{}
	for _, eqi := range e {
		if eqi.Min == nil {
			continue
		}
		totalMin = util.SumResources(totalMin, *eqi.Min)
	}
	return &totalMin
}

func (e ElasticQuotaInfos) getAggregatedUsed() *framework.Resource {
	var totalUsed = framework.Resource{}
	for _, eqi := range e {
		if eqi.Used == nil {
			continue
		}
		totalUsed = util.SumResources(totalUsed, *eqi.Used)
	}
	return &totalUsed
}

func (e ElasticQuotaInfos) getAggregatedUnused() framework.Resource {
	totalMin := e.getAggregatedMin()
	totalUsed := e.getAggregatedUsed()
	return util.SubtractResources(*totalMin, *totalUsed)
}

// ElasticQuotaInfo is a wrapper to a ElasticQuota with information.
// Each namespace can only have one ElasticQuota.
type ElasticQuotaInfo struct {
	Namespace          string
	pods               sets.String
	Min                *framework.Resource
	Max                *framework.Resource
	Used               *framework.Resource
	MaxEnforced        bool
	resourceCalculator util.ResourceCalculator
}

func newElasticQuotaInfo(namespace string, min, max, used v1.ResourceList, resourceCalculator util.ResourceCalculator) *ElasticQuotaInfo {
	elasticQuotaInfo := &ElasticQuotaInfo{
		Namespace:          namespace,
		pods:               sets.NewString(),
		Min:                framework.NewResource(min),
		Max:                framework.NewResource(max),
		Used:               framework.NewResource(used),
		MaxEnforced:        max != nil,
		resourceCalculator: resourceCalculator,
	}
	return elasticQuotaInfo
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
	return cmp(e.Used, resource)
}

// usedOverWith returns true if used > resource + podRequest
func (e *ElasticQuotaInfo) usedOverWith(resource *framework.Resource, podRequest *framework.Resource) bool {
	return cmp2(podRequest, e.Used, resource)
}

// usedLteWith returns true if used <= resource + podRequest
func (e *ElasticQuotaInfo) usedLteWith(resource *framework.Resource, podRequest *framework.Resource) bool {
	return !cmp2(podRequest, e.Used, resource)
}

func (e *ElasticQuotaInfo) clone() *ElasticQuotaInfo {
	newEQInfo := &ElasticQuotaInfo{
		Namespace: e.Namespace,
		pods:      sets.NewString(),
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
	r := e.resourceCalculator.ComputePodResourceRequest(*pod)
	podRequest := util.FromResourceListToFrameworkResource(r)
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
	r := e.resourceCalculator.ComputePodResourceRequest(*pod)
	podRequest := util.FromResourceListToFrameworkResource(r)
	e.unreserveResource(podRequest)

	return nil
}

func cmp(x, y *framework.Resource) bool {
	return cmp2(x, &framework.Resource{}, y)
}

func cmp2(x1, x2, y *framework.Resource) bool {
	if x1.MilliCPU+x2.MilliCPU > y.MilliCPU {
		return true
	}

	if x1.Memory+x2.Memory > y.Memory {
		return true
	}

	for rName, rQuant := range x1.ScalarResources {
		if _, ok := y.ScalarResources[rName]; ok {
			if rQuant+x2.ScalarResources[rName] > y.ScalarResources[rName] {
				return true
			}
		}
	}

	return false
}
