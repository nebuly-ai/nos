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
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 1000,
					Memory:   200,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 2,
						constant.ResourceGPUMemory: 2 * constant.DefaultNvidiaGPUMemory,
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
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 4000,
					Memory:   350,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 5,
						constant.ResourceGPUMemory: 5 * constant.DefaultNvidiaGPUMemory,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				r := util.ComputePodResourceRequest(*pod)
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
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 4000,
					Memory:   200,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 5,
						constant.ResourceGPUMemory: 5 * constant.DefaultNvidiaGPUMemory,
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
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 1000,
					Memory:   50,
					ScalarResources: map[v1.ResourceName]int64{
						constant.ResourceNvidiaGPU: 2,
						constant.ResourceGPUMemory: 2 * constant.DefaultNvidiaGPUMemory,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				r := util.ComputePodResourceRequest(*pod)
				request := util.FromResourceListToFrameworkResource(r)
				elasticQuotaInfo.unreserveResource(request)
			}

			if !reflect.DeepEqual(elasticQuotaInfo, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected.Used, elasticQuotaInfo.Used)
			}
		})
	}
}

func TestElasticQuotaInfo_NewElasticQuotaInfo(t *testing.T) {
	t.Run("NewElasticQuotaInfo - max provided", func(t *testing.T) {
		eq := newElasticQuotaInfo("test", v1.ResourceList{}, v1.ResourceList{}, v1.ResourceList{})
		assert.True(t, eq.MaxEnforced)
	})

	t.Run("NewElasticQuotaInfo - max is nil", func(t *testing.T) {
		eq := newElasticQuotaInfo("test", v1.ResourceList{}, nil, v1.ResourceList{})
		assert.False(t, eq.MaxEnforced)
	})
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
