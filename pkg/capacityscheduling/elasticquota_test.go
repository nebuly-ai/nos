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
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
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
						ResourceGPU:                2,
						v1alpha1.ResourceGPUMemory: 0,
					},
				},
			},
			name: "ElasticQuotaInfo ReserveResource",
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 1000, 1, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 100, 2000, 0, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 0, 0, 2, 0, midPriority, "t1-p3", "node-a", false),
			},
			expected: &ElasticQuotaInfo{
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 4000,
					Memory:   350,
					ScalarResources: map[v1.ResourceName]int64{
						ResourceGPU:                5,
						v1alpha1.ResourceGPUMemory: 0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				request := computePodResourceRequest(pod)
				elasticQuotaInfo.reserveResource(*request)
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
						ResourceGPU: 5,
					},
				},
			},
			name: "ElasticQuotaInfo UnReserveResource",
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 1000, 1, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 100, 2000, 0, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 0, 0, 2, 0, midPriority, "t1-p3", "node-a", false),
			},
			expected: &ElasticQuotaInfo{
				Namespace: "ns1",
				Used: &framework.Resource{
					MilliCPU: 1000,
					Memory:   50,
					ScalarResources: map[v1.ResourceName]int64{
						ResourceGPU:                2,
						v1alpha1.ResourceGPUMemory: 0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elasticQuotaInfo := tt.before
			for _, pod := range tt.pods {
				request := computePodResourceRequest(pod)
				elasticQuotaInfo.unreserveResource(*request)
			}

			if !reflect.DeepEqual(elasticQuotaInfo, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected.Used, elasticQuotaInfo.Used)
			}
		})
	}
}

func TestGetPodMemoryRequest(t *testing.T) {
	tests := []struct {
		name              string
		pod               v1.Pod
		expectedGpuMemory int64
		expectedFound     bool
		expectedErr       bool
	}{
		{
			name:              "Pod without labels",
			pod:               v1.Pod{},
			expectedGpuMemory: 0,
			expectedFound:     false,
			expectedErr:       false,
		},
		{
			name:              "Label with invalid value",
			pod:               factory.BuildPod("default", "test").WithLabel(constant.LabelGPUMemory, "invalid").Get(),
			expectedGpuMemory: 0,
			expectedFound:     false,
			expectedErr:       true,
		},
		{
			name:              "Label with valid value",
			pod:               factory.BuildPod("default", "test").WithLabel(constant.LabelGPUMemory, "10").Get(),
			expectedGpuMemory: 10,
			expectedFound:     true,
			expectedErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuMemory, found, err := getPodGPUMemoryRequest(&tt.pod)
			if found != tt.expectedFound {
				t.Errorf("expected found=%v, got found=%v", tt.expectedFound, found)
			}
			if err == nil && tt.expectedErr == true {
				t.Errorf("error was expected, got err=nil")
			}
			if err != nil && tt.expectedErr == false {
				t.Errorf("nil error was expected, got err=%v", err)
			}
			if gpuMemory != tt.expectedGpuMemory {
				t.Errorf("expected gpuMemory=%v, got gpuMemory=%v", tt.expectedGpuMemory, gpuMemory)
			}
		})
	}
}
