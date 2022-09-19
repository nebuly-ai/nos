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
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"math"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func TestReserveResource(t *testing.T) {
	tests := []struct {
		before   *ElasticQuotaInfo
		name     string
		pods     []*v1.Pod
		expected *ElasticQuotaInfo
	}{
		{
			before: &ElasticQuotaInfo{
				Namespaces: sets.NewString("ns1"),
				Used: &framework.Resource{
					MilliCPU: 1000,
					Memory:   200,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 2,
						constant.ResourceGPUMemory: 2 * constant.DefaultNvidiaGPUResourceMemory,
					},
				},
			},
			name: "ElasticQuotaInfo ReserveResource",
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 1000, 1, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 100, 2000, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 0, 0, 2, midPriority, "t1-p3", "node-a", false),
			},
			expected: &ElasticQuotaInfo{
				Namespaces: sets.NewString("ns1"),
				Used: &framework.Resource{
					MilliCPU: 4000,
					Memory:   350,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 5,
						constant.ResourceGPUMemory: 5 * constant.DefaultNvidiaGPUResourceMemory,
					},
				},
			},
		},
	}

	resourceCalculator := util.ResourceCalculator{NvidiaGPUDeviceMemoryGB: constant.DefaultNvidiaGPUResourceMemory}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				r := resourceCalculator.ComputePodResourceRequest(*pod)
				request := util.FromResourceListToFrameworkResource(r)
				elasticQuotaInfo.reserveResource(request)
			}

			if !reflect.DeepEqual(elasticQuotaInfo, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected.Used, elasticQuotaInfo.Used)
			}
		})
	}
}

func TestUnReserveResource(t *testing.T) {
	tests := []struct {
		before   *ElasticQuotaInfo
		name     string
		pods     []*v1.Pod
		expected *ElasticQuotaInfo
	}{
		{
			before: &ElasticQuotaInfo{
				Namespaces: sets.NewString("ns1"),
				Used: &framework.Resource{
					MilliCPU: 4000,
					Memory:   200,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 5,
						constant.ResourceGPUMemory: 5 * constant.DefaultNvidiaGPUResourceMemory,
					},
				},
			},
			name: "ElasticQuotaInfo UnReserveResource",
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 1000, 1, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 100, 2000, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 0, 0, 2, midPriority, "t1-p3", "node-a", false),
			},
			expected: &ElasticQuotaInfo{
				Namespaces: sets.NewString("ns1"),
				Used: &framework.Resource{
					MilliCPU: 1000,
					Memory:   50,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 2,
						constant.ResourceGPUMemory: 2 * constant.DefaultNvidiaGPUResourceMemory,
					},
				},
			},
		},
	}

	resourceCalculator := util.ResourceCalculator{NvidiaGPUDeviceMemoryGB: constant.DefaultNvidiaGPUResourceMemory}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				r := resourceCalculator.ComputePodResourceRequest(*pod)
				request := util.FromResourceListToFrameworkResource(r)
				elasticQuotaInfo.unreserveResource(request)
			}

			if !reflect.DeepEqual(elasticQuotaInfo, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected.Used, elasticQuotaInfo.Used)
			}
		})
	}
}

func TestElasticQuotaInfo_UsedOverMaxWith(t *testing.T) {
	tests := []struct {
		name     string
		eq       ElasticQuotaInfo
		resource framework.Resource
		expected bool
	}{
		{
			name:     "Max not enforced",
			eq:       ElasticQuotaInfo{MaxEnforced: false},
			resource: framework.Resource{MilliCPU: 100},
			expected: false,
		},
		{
			name: "Max enforced - used > max",
			eq: ElasticQuotaInfo{
				Used:        &framework.Resource{MilliCPU: 100},
				Max:         &framework.Resource{MilliCPU: 100},
				MaxEnforced: true,
			},
			resource: framework.Resource{MilliCPU: 100},
			expected: true,
		},
		{
			name: "Max enforced - used = max",
			eq: ElasticQuotaInfo{
				Used:        &framework.Resource{MilliCPU: 50},
				Max:         &framework.Resource{MilliCPU: 100},
				MaxEnforced: true,
			},
			resource: framework.Resource{MilliCPU: 50},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.eq.usedOverMaxWith(&tt.resource)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestElasticQuotaInfos_GetGuaranteedOverquotas(t *testing.T) {
	tests := []struct {
		name                         string
		elasticQuotaInfos            ElasticQuotaInfos
		elasticQuotaName             string
		expectedGuaranteedOverquotas *framework.Resource
		errorExpected                bool
	}{
		{
			name:                         "ElasticQuotaInfo not present",
			elasticQuotaInfos:            NewElasticQuotaInfos(),
			elasticQuotaName:             "not-present",
			expectedGuaranteedOverquotas: nil,
			errorExpected:                true,
		},
		{
			name: "ElasticQuota is empty",
			elasticQuotaInfos: map[string]*ElasticQuotaInfo{
				"eq-1": {
					Min:  &framework.Resource{},
					Max:  &framework.Resource{},
					Used: &framework.Resource{},
				},
				"eq-2": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         100,
						Memory:           1000,
						EphemeralStorage: 0,
						AllowedPodNumber: 10,
					},
					Max: &framework.Resource{
						MilliCPU:         200,
						Memory:           2000,
						EphemeralStorage: 0,
						AllowedPodNumber: 20,
					},
					Used: &framework.Resource{
						MilliCPU:         50,
						Memory:           50,
						EphemeralStorage: 0,
						AllowedPodNumber: 5,
					},
					MaxEnforced: false,
				},
			},
			elasticQuotaName:             "eq-1",
			expectedGuaranteedOverquotas: &framework.Resource{},
			errorExpected:                false,
		},
		{
			name: "All ElasticQuotas are empty",
			elasticQuotaInfos: map[string]*ElasticQuotaInfo{
				"eq-1": {
					Min:  &framework.Resource{},
					Max:  &framework.Resource{},
					Used: &framework.Resource{},
				},
				"eq-2": {
					Min:  &framework.Resource{},
					Max:  &framework.Resource{},
					Used: &framework.Resource{},
				},
			},
			elasticQuotaName:             "eq-1",
			expectedGuaranteedOverquotas: &framework.Resource{},
			errorExpected:                false,
		},
		{
			name: "ElasticQuota with scalar resources - guaranteed overquotas for each resource is proportional to Min",
			elasticQuotaInfos: map[string]*ElasticQuotaInfo{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         10,
						Memory:           10,
						EphemeralStorage: 0,
						AllowedPodNumber: 10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU:                5,
							constant.ResourceGPUMemory:                64,
							v1.ResourceName("nebuly.ai/new-resource"): 3, // resource present only in eq-1
						},
					},
					Used: &framework.Resource{
						MilliCPU:         5,
						Memory:           5,
						EphemeralStorage: 0,
						AllowedPodNumber: 5,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU:                0,
							constant.ResourceGPUMemory:                10,
							v1.ResourceName("nebuly.ai/new-resource"): 1,
						},
					},
					MaxEnforced: false,
				},
				"eq-2": {
					Namespaces: sets.NewString("ns-2"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         30,
						Memory:           30,
						EphemeralStorage: 30,
						AllowedPodNumber: 30,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 3,
							constant.ResourceGPUMemory: 24,
						},
					},
					Used: &framework.Resource{
						MilliCPU:         35,
						Memory:           35,
						EphemeralStorage: 0,
						AllowedPodNumber: 5,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 0,
							constant.ResourceGPUMemory: 10,
						},
					},
					MaxEnforced: false,
				},
				"eq-3": {
					Namespaces: sets.NewString("ns-3"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         20,
						Memory:           20,
						EphemeralStorage: 20,
						AllowedPodNumber: 0,
					},
					Used: &framework.Resource{
						MilliCPU:         10,
						Memory:           10,
						EphemeralStorage: 10,
						AllowedPodNumber: 0,
					},
					MaxEnforced: false,
				},
			},
			elasticQuotaName: "eq-1",
			expectedGuaranteedOverquotas: &framework.Resource{
				MilliCPU:         2, // math.Floor(10 / (10 + 30 + 20) * (Max(0, 10-5) + Max(0, 30-35) + Max(0, 20-10)))
				Memory:           2, // math.Floor(10 / (10 + 30 + 20) * (Max(0, 10-5) + Max(0, 30-35) + Max(0, 20-10)))
				EphemeralStorage: 0,
				AllowedPodNumber: 7, // math.Floor(10 / (10 + 30 + 0) * (Max(0, 10-5) + Max(0, 30-5) + Max(0, 0-0)))
				ScalarResources: map[v1.ResourceName]int64{
					v1.ResourceName("nebuly.ai/new-resource"): 2,  // tot. unused overquotas, since "new-resource" is defined only for eq-1
					constant.ResourceNvidiaGPU:                5,  // math.Floor(5 / (5 + 3) * (Max(0, 5-0) + Max(0, 3-0)))
					constant.ResourceGPUMemory:                49, // math.Floor(64 / (64 + 24) * (Max(0, 64-10) + Max(0, 24-10)))
				},
			},
			errorExpected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guaranteedOverquotas, err := tt.elasticQuotaInfos.GetGuaranteedOverquotas(tt.elasticQuotaName)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedGuaranteedOverquotas, guaranteedOverquotas)
		})
	}
}

func TestElasticQuotaInfos_getGuaranteedOverquotasPercentage(t *testing.T) {
	tests := []struct {
		name              string
		elasticQuotaInfos ElasticQuotaInfos
		elasticQuota      string
		expected          map[v1.ResourceName]float64
	}{
		{
			name: "Single empty elastic quota",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": {},
			},
			elasticQuota: "eq-1",
			expected:     map[v1.ResourceName]float64{},
		},
		{
			name: "Multiple elastic quotas, one is empty",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         30,
						Memory:           30,
						EphemeralStorage: 30,
						AllowedPodNumber: 30,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 3,
							constant.ResourceGPUMemory: 24,
						},
					},
				},
				"eq-2": {},
			},
			elasticQuota: "eq-1",
			expected: map[v1.ResourceName]float64{
				v1.ResourceCPU:              1,
				v1.ResourceMemory:           1,
				v1.ResourcePods:             1,
				v1.ResourceEphemeralStorage: 1,
				constant.ResourceGPUMemory:  1,
				constant.ResourceNvidiaGPU:  1,
			},
		},
		{
			name: "Single elastic quota, guaranteed overquotas percentage should be 100%",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         30,
						Memory:           30,
						EphemeralStorage: 30,
						AllowedPodNumber: 30,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 3,
							constant.ResourceGPUMemory: 24,
						},
					},
				},
			},
			elasticQuota: "eq-1",
			expected: map[v1.ResourceName]float64{
				v1.ResourceCPU:              1,
				v1.ResourceMemory:           1,
				v1.ResourcePods:             1,
				v1.ResourceEphemeralStorage: 1,
				constant.ResourceGPUMemory:  1,
				constant.ResourceNvidiaGPU:  1,
			},
		},
		{
			name: "Resource values are max",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         resource.MaxMilliValue,
						Memory:           math.MaxInt64,
						EphemeralStorage: math.MaxInt64,
						AllowedPodNumber: math.MaxInt64,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: math.MaxInt64,
							constant.ResourceGPUMemory: math.MaxInt64,
						},
					},
				},
			},
			elasticQuota: "eq-1",
			expected: map[v1.ResourceName]float64{
				v1.ResourceCPU:              1,
				v1.ResourceMemory:           1,
				v1.ResourcePods:             1,
				v1.ResourceEphemeralStorage: 1,
				constant.ResourceGPUMemory:  1,
				constant.ResourceNvidiaGPU:  1,
			},
		},
		{
			name: "ElasticQuotas do not specify a Min for some resources - Percentages include all non-scalar resources",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU: 10,
						Memory:   10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 10,
						},
					},
				},
				"eq-2": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         10,
						AllowedPodNumber: 10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceGPUMemory: 10,
						},
					},
				},
			},
			elasticQuota: "eq-1",
			expected: map[v1.ResourceName]float64{
				v1.ResourceCPU:              0.5,
				v1.ResourceMemory:           1,
				v1.ResourcePods:             0,
				v1.ResourceEphemeralStorage: 0,
				constant.ResourceNvidiaGPU:  1,
			},
		},
		{
			name: "Multiple elastic quota, elastic quota with scalar resources. Overquotas % should be proportional to Min.",
			elasticQuotaInfos: map[string]*ElasticQuotaInfo{
				"eq-1": {
					Namespaces: sets.NewString("ns-1"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         50,
						Memory:           10,
						EphemeralStorage: 0,
						AllowedPodNumber: 10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU:                5,
							constant.ResourceGPUMemory:                64,
							v1.ResourceName("nebuly.ai/new-resource"): 3, // resource present only in eq-1
						},
					},
				},
				"eq-2": {
					Namespaces: sets.NewString("ns-2"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         30,
						Memory:           30,
						EphemeralStorage: 30,
						AllowedPodNumber: 30,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 3,
							constant.ResourceGPUMemory: 24,
						},
					},
				},
				"eq-3": {
					Namespaces: sets.NewString("ns-3"),
					pods:       sets.NewString("pd-1", "pd-2"),
					Min: &framework.Resource{
						MilliCPU:         20,
						Memory:           60,
						EphemeralStorage: 20,
						AllowedPodNumber: 0,
					},
				},
			},
			elasticQuota: "eq-1",
			expected: map[v1.ResourceName]float64{
				v1.ResourceCPU:                            0.5,
				v1.ResourceMemory:                         0.1,
				v1.ResourcePods:                           0.25,
				v1.ResourceEphemeralStorage:               0,
				v1.ResourceName("nebuly.ai/new-resource"): 1,
				constant.ResourceGPUMemory:                float64(64) / float64(64+24),
				constant.ResourceNvidiaGPU:                float64(5) / float64(5+3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eqInfo := tt.elasticQuotaInfos[tt.elasticQuota]
			percentages := tt.elasticQuotaInfos.getGuaranteedOverquotasPercentages(eqInfo)
			assert.Equal(t, tt.expected, percentages)
		})
		t.Run("Sum of guaranteed overquotas percentages should be 1", func(t *testing.T) {
			var totalPercentages = make(map[v1.ResourceName]float64)
			for _, eqInfo := range tt.elasticQuotaInfos {
				for r, p := range tt.elasticQuotaInfos.getGuaranteedOverquotasPercentages(eqInfo) {
					totalPercentages[r] += p
				}
			}
			for r, p := range totalPercentages {
				if p != 0 {
					assert.Greater(t, p, float64(0))
					assert.InDeltaf(
						t,
						float64(1),
						p,
						1e-4,
						"Sum of all guaranteed overquata percentages should be approximately 1: got %f for resource %s",
						p,
						r,
					)
				}
			}
		})
	}
}

func TestElasticQuotaInfos_getAggregatedOverquotas(t *testing.T) {
	tests := []struct {
		name              string
		elasticQuotaInfos ElasticQuotaInfos
		expected          framework.Resource
	}{
		{
			name:              "Empty elastic quota infos",
			elasticQuotaInfos: ElasticQuotaInfos{},
			expected:          framework.Resource{},
		},
		{
			name: "Single elastic quota info",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq": {
					Namespaces: sets.NewString("ns"),
					Min: &framework.Resource{
						MilliCPU:         100,
						Memory:           200,
						EphemeralStorage: 5,
						AllowedPodNumber: 10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 5,
							constant.ResourceGPUMemory: 5,
						},
					},
					Max: nil,
					Used: &framework.Resource{
						MilliCPU:         0,
						Memory:           100,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 5,
							constant.ResourceGPUMemory: 0,
						},
					},
				},
			},
			expected: framework.Resource{
				MilliCPU:         100,
				Memory:           100,
				EphemeralStorage: 5,
				AllowedPodNumber: 10,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 0,
					constant.ResourceGPUMemory: 5,
				},
			},
		},
		{
			name: "Multiple ElasticQuotaInfos",
			elasticQuotaInfos: ElasticQuotaInfos{
				"eq-1": { // overquota
					Namespaces: sets.NewString("ns-1"),
					Min: &framework.Resource{
						MilliCPU:         100,
						Memory:           200,
						EphemeralStorage: 5,
						AllowedPodNumber: 5,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 5,
							constant.ResourceGPUMemory: 5,
						},
					},
					Used: &framework.Resource{
						MilliCPU:         150,
						Memory:           250,
						EphemeralStorage: 10,
						AllowedPodNumber: 10,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 10,
							constant.ResourceGPUMemory: 10,
						},
					},
				},
				"eq-2": {
					Namespaces: sets.NewString("ns-2"),
					Min: &framework.Resource{
						MilliCPU:         200,
						Memory:           200,
						EphemeralStorage: 5,
						AllowedPodNumber: 5,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 5,
							constant.ResourceGPUMemory: 5,
						},
					},
					Used: &framework.Resource{
						MilliCPU:         200,
						Memory:           0,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 0,
							constant.ResourceGPUMemory: 0,
						},
					},
				},
				"eq-3": {
					Namespaces: sets.NewString("ns-3"),
					Min: &framework.Resource{
						MilliCPU:         200,
						Memory:           200,
						EphemeralStorage: 5,
						AllowedPodNumber: 5,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 5,
						},
					},
					Used: &framework.Resource{
						MilliCPU:         0,
						Memory:           10,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 1,
						},
					},
				},
			},
			expected: framework.Resource{
				MilliCPU:         0 + 0 + 200,
				Memory:           0 + 200 + 190,
				EphemeralStorage: 0 + 5 + 5,
				AllowedPodNumber: 0 + 5 + 5,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 0 + 5 + 4,
					constant.ResourceGPUMemory: 0 + 5 + 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.elasticQuotaInfos.getAggregatedOverquotas()
			assert.Equal(t, tt.expected, actual)

			// aggregated_overquotas must be <= aggregated_min
			aggregatedMin := tt.elasticQuotaInfos.getAggregatedMin()
			assert.LessOrEqual(t, actual.MilliCPU, aggregatedMin.MilliCPU)
			assert.LessOrEqual(t, actual.Memory, aggregatedMin.Memory)
			assert.LessOrEqual(t, actual.EphemeralStorage, aggregatedMin.EphemeralStorage)
			assert.LessOrEqual(t, actual.AllowedPodNumber, aggregatedMin.AllowedPodNumber)
			for r, v := range actual.ScalarResources {
				assert.LessOrEqual(t, v, aggregatedMin.ScalarResources[r])
			}
		})
	}
}
