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
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"sort"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/events"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	plfeature "k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	"k8s.io/kubernetes/pkg/scheduler/framework/preemption"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	imageutils "k8s.io/kubernetes/test/utils/image"

	testutil "github.com/nebuly-ai/nebulnetes/pkg/test/util"
)

const ResourceGPU v1.ResourceName = "nvidia.com/gpu"

var (
	lowPriority, midPriority, highPriority = int32(10), int32(100), int32(1000)
)

func TestPreFilter(t *testing.T) {
	type podInfo struct {
		podName      string
		podNamespace string
		memReq       int64
		gpuMemReq    int64
	}

	tests := []struct {
		name          string
		podInfos      []podInfo
		elasticQuotas map[string]*ElasticQuotaInfo
		expected      []framework.Code
	}{
		{
			name: "pods requesting resources not specified in ElasticQuota",
			podInfos: []podInfo{
				{podName: "ns1-p1", podNamespace: "ns1", memReq: 500, gpuMemReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", memReq: 10, gpuMemReq: 1},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Min: &framework.Resource{
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 5},
					},
					Used: &framework.Resource{},
				},
			},
			expected: []framework.Code{
				framework.Unschedulable,
				framework.Unschedulable,
			},
		},
		{
			name: "pods subject to ElasticQuota",
			podInfos: []podInfo{
				{podName: "ns1-p1", podNamespace: "ns1", memReq: 500, gpuMemReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", memReq: 1800, gpuMemReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", gpuMemReq: 3},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 5},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 10},
					},
					Used: &framework.Resource{
						Memory:          300,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 4},
					},
				},
			},
			expected: []framework.Code{
				framework.Success,
				framework.Unschedulable,
				framework.Unschedulable,
			},
		},
		{
			name: "ElasticQuota not enforcing Max",
			podInfos: []podInfo{
				{podName: "ns1-p1", podNamespace: "ns1", memReq: 500, gpuMemReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", memReq: 1800, gpuMemReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", gpuMemReq: 3},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 5},
					},
					Used: &framework.Resource{
						Memory:          300,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 4},
					},
				},
				"ns2": {
					Namespace: "ns2",
					Min: &framework.Resource{
						Memory:          5000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 6},
					},
					Used: &framework.Resource{},
				},
			},
			expected: []framework.Code{
				framework.Success,
				framework.Success,
				framework.Success,
			},
		},
		{
			name: "the sum of used is bigger than the sum of min",
			podInfos: []podInfo{
				{podName: "ns2-p1", podNamespace: "ns2", memReq: 500},
				{podName: "ns2-p1", podNamespace: "ns2", gpuMemReq: 2},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 5},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 100},
					},
					Used: &framework.Resource{
						Memory:          1800,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 4},
					},
				},
				"ns2": {
					Namespace: "ns2",
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 1},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 100},
					},
					Used: &framework.Resource{
						Memory:          200,
						ScalarResources: map[v1.ResourceName]int64{constant.ResourceGPUMemory: 1},
					},
				},
			},
			expected: []framework.Code{
				framework.Unschedulable,
				framework.Unschedulable,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var registerPlugins []st.RegisterPluginFunc
			registeredPlugins := append(
				registerPlugins,
				st.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
				st.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
			)

			fwk, err := st.NewFramework(
				registeredPlugins, "",
				frameworkruntime.WithPodNominator(testutil.NewPodNominator(nil)),
				frameworkruntime.WithSnapshotSharedLister(testutil.NewFakeSharedLister(make([]*v1.Pod, 0), make([]*v1.Node, 0))),
			)

			if err != nil {
				t.Fatal(err)
			}

			cs := &CapacityScheduling{
				elasticQuotaInfos: tt.elasticQuotas,
				fh:                fwk,
			}

			pods := make([]*v1.Pod, 0)
			for _, podInfo := range tt.podInfos {
				pod := makePod(podInfo.podName, podInfo.podNamespace, podInfo.memReq, 0, 0, podInfo.gpuMemReq, 0, podInfo.podName, "", false)
				pods = append(pods, pod)
			}

			state := framework.NewCycleState()
			for i := range pods {
				if _, got := cs.PreFilter(nil, state, pods[i]); got.Code() != tt.expected[i] {
					t.Errorf("expected %v, got %v : %v", tt.expected[i], got.Code(), got.Message())
				}
			}
		})
	}
}

func TestDryRunPreemption(t *testing.T) {
	res := map[v1.ResourceName]string{v1.ResourceMemory: "150"}
	tests := []struct {
		name          string
		pod           *v1.Pod
		pods          []*v1.Pod
		nodes         []*v1.Node
		nodesStatuses framework.NodeToStatusMap
		elasticQuotas map[string]*ElasticQuotaInfo
		want          []preemption.Candidate
	}{
		{
			name: "in-namespace preemption",
			pod:  makePod("t1-p", "ns1", 50, 0, 0, 0, highPriority, "", "t1-p", false),
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 0, 0, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 50, 0, 0, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 50, 0, 0, 0, midPriority, "t1-p3", "node-a", false),
			},
			nodes: []*v1.Node{
				st.MakeNode().Name("node-a").Capacity(res).Obj(),
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Max: &framework.Resource{
						Memory: 200,
					},
					Min: &framework.Resource{
						Memory: 50,
					},
					Used: &framework.Resource{
						Memory: 50,
					},
				},
				"ns2": {
					Namespace: "ns2",
					Max: &framework.Resource{
						Memory: 200,
					},
					Min: &framework.Resource{
						Memory: 200,
					},
					Used: &framework.Resource{
						Memory: 100,
					},
				},
			},
			nodesStatuses: framework.NodeToStatusMap{
				"node-a": framework.NewStatus(framework.Unschedulable),
			},
			want: []preemption.Candidate{
				&candidate{
					victims: &extenderv1.Victims{
						Pods: []*v1.Pod{
							makePod("t1-p1", "ns1", 50, 0, 0, 0, midPriority, "t1-p1", "node-a", false),
						},
						NumPDBViolations: 0,
					},
					name: "node-a",
				},
			},
		},
		{
			name: "cross-namespace preemption",
			pod:  makePod("t1-p", "ns1", 50, 0, 0, 0, highPriority, "", "t1-p", false),
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 40, 0, 0, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 50, 0, 0, 0, highPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 50, 0, 0, 0, midPriority, "t1-p3", "node-a", true),
				makePod("t1-p4", "ns2", 10, 0, 0, 0, lowPriority, "t1-p4", "node-a", false),
			},
			nodes: []*v1.Node{
				st.MakeNode().Name("node-a").Capacity(res).Obj(),
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespace: "ns1",
					Max: &framework.Resource{
						Memory: 200,
					},
					Min: &framework.Resource{
						Memory: 150,
					},
					Used: &framework.Resource{
						Memory: 50,
					},
				},
				"ns2": {
					Namespace: "ns2",
					Max: &framework.Resource{
						Memory: 200,
					},
					Min: &framework.Resource{
						Memory: 50,
					},
					Used: &framework.Resource{
						Memory: 100,
					},
				},
			},
			nodesStatuses: framework.NodeToStatusMap{
				"node-a": framework.NewStatus(framework.Unschedulable),
			},
			want: []preemption.Candidate{
				&candidate{
					victims: &extenderv1.Victims{
						Pods: []*v1.Pod{
							makePod("t1-p3", "ns2", 50, 0, 0, 0, midPriority, "t1-p3", "node-a", false),
						},
						NumPDBViolations: 0,
					},
					name: "node-a",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registeredPlugins := []st.RegisterPluginFunc{
				st.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
				st.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
				st.RegisterPluginAsExtensions(noderesources.Name, func(plArgs apiruntime.Object, fh framework.Handle) (framework.Plugin, error) {
					return noderesources.NewFit(plArgs, fh, plfeature.Features{})
				}, "Filter", "PreFilter"),
			}

			cs := clientsetfake.NewSimpleClientset()
			fwk, err := st.NewFramework(
				registeredPlugins,
				"default-scheduler",
				frameworkruntime.WithClientSet(cs),
				frameworkruntime.WithEventRecorder(&events.FakeRecorder{}),
				frameworkruntime.WithPodNominator(testutil.NewPodNominator(nil)),
				frameworkruntime.WithSnapshotSharedLister(testutil.NewFakeSharedLister(tt.pods, tt.nodes)),
				frameworkruntime.WithInformerFactory(informers.NewSharedInformerFactory(cs, 0)),
			)
			if err != nil {
				t.Fatal(err)
			}

			state := framework.NewCycleState()
			ctx := context.Background()

			// Some tests rely on PreFilter plugin to compute its CycleState.
			_, preFilterStatus := fwk.RunPreFilterPlugins(ctx, state, tt.pod)
			if !preFilterStatus.IsSuccess() {
				t.Errorf("Unexpected preFilterStatus: %v", preFilterStatus)
			}

			podReq := computePodResourceRequest(tt.pod)
			elasticQuotaSnapshotState := &ElasticQuotaSnapshotState{
				elasticQuotaInfos: tt.elasticQuotas,
			}
			prefilterStatue := &PreFilterState{
				podReq:                         *podReq,
				nominatedPodsReqWithPodReq:     *podReq,
				nominatedPodsReqInEQWithPodReq: *podReq,
			}
			state.Write(preFilterStateKey, prefilterStatue)
			state.Write(ElasticQuotaSnapshotKey, elasticQuotaSnapshotState)

			pe := preemption.Evaluator{
				PluginName: Name,
				Handler:    fwk,
				PodLister:  fwk.SharedInformerFactory().Core().V1().Pods().Lister(),
				PdbLister:  getPDBLister(fwk.SharedInformerFactory()),
				State:      state,
				Interface: &preemptor{
					fh:    fwk,
					state: state,
				},
			}

			nodeInfos, _ := fwk.SnapshotSharedLister().NodeInfos().List()
			got, _, err := pe.DryRunPreemption(ctx, tt.pod, nodeInfos, nil, 0, int32(len(nodeInfos)))
			if err != nil {
				t.Fatalf("unexpected error during DryRunPreemption(): %v", err)
			}

			// Sort the values (inner victims) and the candidate itself (by its NominatedNodeName).
			for i := range got {
				victims := got[i].Victims().Pods
				sort.Slice(victims, func(i, j int) bool {
					return victims[i].Name < victims[j].Name
				})
			}
			sort.Slice(got, func(i, j int) bool {
				return got[i].Name() < got[j].Name()
			})

			if len(got) != len(tt.want) {
				t.Fatalf("Unexpected candidate length: want %v, but got %v", len(tt.want), len(got))
			}
			for i, c := range got {
				if diff := gocmp.Diff(c.Victims(), got[i].Victims()); diff != "" {
					t.Errorf("Unexpected victims at index %v (-want, +got): %s", i, diff)
				}
				if diff := gocmp.Diff(c.Name(), got[i].Name()); diff != "" {
					t.Errorf("Unexpected victims at index %v (-want, +got): %s", i, diff)
				}
			}
		})
	}
}

func makePod(podName string, namespace string, memReq int64, cpuReq int64, gpuReq int64, gpuMemoryReq int64, priority int32, uid string, nodeName string, overquota bool) *v1.Pod {
	pause := imageutils.GetPauseImageName()
	pod := st.MakePod().Namespace(namespace).Name(podName).Container(pause).
		Priority(priority).Node(nodeName).UID(uid).ZeroTerminationGracePeriod().Obj()
	pod.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(memReq, resource.DecimalSI),
			v1.ResourceCPU:    *resource.NewMilliQuantity(cpuReq, resource.DecimalSI),
			ResourceGPU:       *resource.NewQuantity(gpuReq, resource.DecimalSI),
		},
	}

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[constant.LabelGPUMemory] = fmt.Sprintf("%d", gpuMemoryReq)

	if overquota == true {
		pod.Labels[constant.LabelCapacityInfo] = string(constant.CapacityInfoOverQuota)
	} else {
		pod.Labels[constant.LabelCapacityInfo] = string(constant.CapacityInfoInQuota)
	}

	return pod
}
