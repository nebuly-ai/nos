/*
Copyright 2020 The Kubernetes Authors.
Copyright 2022 Nebuly.ai.

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
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/util"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"
	"sort"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"

	"k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
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

var (
	lowPriority, midPriority, highPriority = int32(10), int32(100), int32(1000)
)

func TestPreFilter(t *testing.T) {
	type podInfo struct {
		podName      string
		podNamespace string
		memReq       int64
		cpuReq       int64
		gpuReq       int64
	}

	const nvidiaGPUResourceMemory = 8

	tests := []struct {
		name          string
		podInfos      []podInfo
		elasticQuotas map[string]*ElasticQuotaInfo
		expected      []framework.Code
	}{
		{
			name: "pods requesting resources not specified in ElasticQuota",
			podInfos: []podInfo{
				{podName: "ns1-p1", podNamespace: "ns1", cpuReq: 0, memReq: 500},
				{podName: "ns1-p2", podNamespace: "ns1", cpuReq: 0, memReq: 10},
				{podName: "ns1-p2", podNamespace: "ns1", cpuReq: 10, memReq: 10},          // request non-scalar resource not defined in quota
				{podName: "ns1-p2", podNamespace: "ns1", cpuReq: 0, memReq: 0, gpuReq: 1}, // request scalar resource not defined in quota
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
					Min: &framework.Resource{
						Memory: 1000,
					},
					Used: &framework.Resource{},
				},
			},
			expected: []framework.Code{
				framework.Success,
				framework.Success,
				framework.Unschedulable,
				framework.Success,
			},
		},
		{
			name: "pods subject to ElasticQuota",
			podInfos: []podInfo{
				{podName: "ns1-p1", podNamespace: "ns1", memReq: 500, gpuReq: 1},
				{podName: "ns1-p2", podNamespace: "ns1", memReq: 1800},
				{podName: "ns1-p2", podNamespace: "ns1", gpuReq: 2},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 5 * nvidiaGPUResourceMemory},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 6 * nvidiaGPUResourceMemory},
					},
					Used: &framework.Resource{
						Memory:          300,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 4 * nvidiaGPUResourceMemory},
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
				{podName: "ns1-p1", podNamespace: "ns1", memReq: 500},
				{podName: "ns1-p2", podNamespace: "ns1", memReq: 1800},
				{podName: "ns1-p2", podNamespace: "ns1", gpuReq: 6},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 5 * nvidiaGPUResourceMemory},
					},
					Used: &framework.Resource{
						Memory:          300,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 4 * nvidiaGPUResourceMemory},
					},
				},
				"ns2": {
					Namespaces: sets.NewString("ns2"),
					Min: &framework.Resource{
						Memory:          5000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 6 * nvidiaGPUResourceMemory},
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
				{podName: "ns2-p1", podNamespace: "ns2", gpuReq: 2},
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 5 * nvidiaGPUResourceMemory},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 100 * nvidiaGPUResourceMemory},
					},
					Used: &framework.Resource{
						Memory:          1800,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 4 * nvidiaGPUResourceMemory},
					},
				},
				"ns2": {
					Namespaces: sets.NewString("ns2"),
					Min: &framework.Resource{
						Memory:          1000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 1 * nvidiaGPUResourceMemory},
					},
					Max: &framework.Resource{
						Memory:          2000,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 100 * nvidiaGPUResourceMemory},
					},
					Used: &framework.Resource{
						Memory:          200,
						ScalarResources: map[v1.ResourceName]int64{v1alpha1.ResourceGPUMemory: 1 * nvidiaGPUResourceMemory},
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
				context.Background().Done(),
				frameworkruntime.WithPodNominator(testutil.NewPodNominator(nil)),
				frameworkruntime.WithSnapshotSharedLister(testutil.NewFakeSharedLister(make([]*v1.Pod, 0), make([]*v1.Node, 0))),
			)

			if err != nil {
				t.Fatal(err)
			}

			resourceCalculator := util.ResourceCalculator{NvidiaGPUDeviceMemoryGB: nvidiaGPUResourceMemory}
			cs := &CapacityScheduling{
				elasticQuotaInfos:  tt.elasticQuotas,
				fh:                 fwk,
				resourceCalculator: &resourceCalculator,
			}

			pods := make([]*v1.Pod, 0)
			for _, podInfo := range tt.podInfos {
				pod := makePod(podInfo.podName, podInfo.podNamespace, podInfo.memReq, podInfo.cpuReq, podInfo.gpuReq, 0, podInfo.podName, "", false)
				pods = append(pods, pod)
			}

			state := framework.NewCycleState()
			for i := range pods {
				if _, got := cs.PreFilter(context.TODO(), state, pods[i]); got.Code() != tt.expected[i] {
					t.Errorf("expected %v, got %v : %v", tt.expected[i], got.Code(), got.Message())
				}
			}
		})
	}
}

func TestDryRunPreemption(t *testing.T) {
	const nvidiaGPUResourceMemory = 8
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
			pod:  makePod("t1-p", "ns1", 50, 0, 0, highPriority, "", "t1-p", false),
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 50, 0, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 50, 0, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 50, 0, 0, midPriority, "t1-p3", "node-a", false),
			},
			nodes: []*v1.Node{
				st.MakeNode().Name("node-a").Capacity(map[v1.ResourceName]string{v1.ResourceMemory: "150"}).Obj(),
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
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
					Namespaces: sets.NewString("ns2"),
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
							makePod("t1-p1", "ns1", 50, 0, 0, midPriority, "t1-p1", "node-a", false),
						},
						NumPDBViolations: 0,
					},
					name: "node-a",
				},
			},
		},
		{
			name: "cross-namespace preemption - preemptor uses its Min quotas",
			pod:  makePod("t1-p", "ns1", 50, 0, 0, highPriority, "", "t1-p", false),
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 40, 0, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns2", 50, 0, 0, highPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 50, 0, 0, midPriority, "t1-p3", "node-a", true),
				makePod("t1-p4", "ns2", 10, 0, 0, lowPriority, "t1-p4", "node-a", false),
			},
			nodes: []*v1.Node{
				st.MakeNode().Name("node-a").Capacity(map[v1.ResourceName]string{v1.ResourceMemory: "150"}).Obj(),
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
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
					Namespaces: sets.NewString("ns2"),
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
							makePod("t1-p3", "ns2", 50, 0, 0, midPriority, "t1-p3", "node-a", false),
						},
						NumPDBViolations: 0,
					},
					name: "node-a",
				},
			},
		},
		{
			name: "cross-namespace preemption - guaranteed overquota limits",
			pod:  makePod("t1-p", "ns1", 70, 0, 0, highPriority, "", "t1-p", true),
			pods: []*v1.Pod{
				makePod("t1-p1", "ns1", 100, 100, 0, midPriority, "t1-p1", "node-a", false),
				makePod("t1-p2", "ns1", 150, 100, 0, midPriority, "t1-p2", "node-a", false),
				makePod("t1-p3", "ns2", 50, 0, 0, highPriority, "t1-p3", "node-a", false),
				makePod("t1-p4", "ns2", 50, 0, 0, midPriority, "t1-p4", "node-a", true),
				makePod("t1-p5", "ns2", 10, 0, 0, lowPriority, "t1-p5", "node-a", true),
			},
			nodes: []*v1.Node{
				st.MakeNode().
					Name("node-a").
					Capacity(map[v1.ResourceName]string{
						v1.ResourceMemory: "350",
						v1.ResourceCPU:    "200",
					}).
					Obj(),
			},
			elasticQuotas: map[string]*ElasticQuotaInfo{
				"ns1": {
					Namespaces: sets.NewString("ns1"),
					Max: &framework.Resource{
						Memory:   300,
						MilliCPU: 300,
					},
					Min: &framework.Resource{
						Memory:   150, // guaranteed overquota = 75 = 150 / (150 + 50 + 300) * ( (150 + 50 + 300) - (150 + 100 + 0) )
						MilliCPU: 200, // guaranteed overquota = 103 = 200 / (200 + 20 + 300) * ( (200 + 20 + 300) - (200 + 50 + 0) )
					},
					Used: &framework.Resource{
						Memory:   150,
						MilliCPU: 200,
					},
				},
				"ns2": {
					Namespaces: sets.NewString("ns2"),
					Max: &framework.Resource{
						Memory:   300,
						MilliCPU: 300,
					},
					Min: &framework.Resource{
						Memory:   50, // guaranteed overquota = 25 = 50 / (150 + 50 + 300) * ( (150 + 50 + 300) - (150 + 100 + 0) )
						MilliCPU: 20, // guaranteed overquota = 20 = 20 / (200 + 20 + 300) * ( (200 + 20 + 300) - (0 + 0 + 0) )
					},
					Used: &framework.Resource{
						Memory:   100, // used > (min + guaranteed overquota)
						MilliCPU: 50,  // used > (min + guaranteed overquota)
					},
				},
				"ns3": {
					Namespaces: sets.NewString("ns3"),
					Min: &framework.Resource{
						Memory:   300,
						MilliCPU: 300,
					},
					Used: &framework.Resource{
						Memory:   0,
						MilliCPU: 0,
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
							makePod("t1-p5", "ns2", 10, 0, 0, lowPriority, "t1-p5", "node-a", true),
						},
						NumPDBViolations: 0,
					},
					name: "node-a",
				},
			},
		},
	}

	resourceCalculator := util.ResourceCalculator{
		NvidiaGPUDeviceMemoryGB: nvidiaGPUResourceMemory,
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
			ctx := context.Background()
			cs := clientsetfake.NewSimpleClientset()
			fwk, err := st.NewFramework(
				registeredPlugins,
				"default-scheduler",
				ctx.Done(),
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

			// Some tests rely on PreFilter plugin to compute its CycleState.
			_, preFilterStatus := fwk.RunPreFilterPlugins(ctx, state, tt.pod)
			if !preFilterStatus.IsSuccess() {
				t.Errorf("Unexpected preFilterStatus: %v", preFilterStatus)
			}

			r := resourceCalculator.ComputePodRequest(*tt.pod)
			podReq := resource.FromListToFramework(r)
			elasticQuotaSnapshotState := &ElasticQuotaSnapshotState{
				elasticQuotaInfos: tt.elasticQuotas,
			}
			prefilterStatue := &PreFilterState{
				podReq:                         podReq,
				nominatedPodsReqWithPodReq:     podReq,
				nominatedPodsReqInEQWithPodReq: podReq,
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

func makePod(podName string, namespace string, memReq int64, cpuReq int64, gpuReq int64, priority int32, uid string, nodeName string, overquota bool) *v1.Pod {
	pause := imageutils.GetPauseImageName()
	pod := st.MakePod().Namespace(namespace).Name(podName).Container(pause).
		Priority(priority).Node(nodeName).UID(uid).ZeroTerminationGracePeriod().Obj()
	pod.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceMemory:          *k8sresource.NewQuantity(memReq, k8sresource.DecimalSI),
			v1.ResourceCPU:             *k8sresource.NewMilliQuantity(cpuReq, k8sresource.DecimalSI),
			constant.ResourceNvidiaGPU: *k8sresource.NewQuantity(gpuReq, k8sresource.DecimalSI),
		},
	}

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	if overquota == true {
		pod.Labels[v1alpha1.LabelCapacityInfo] = string(constant.CapacityInfoOverQuota)
	} else {
		pod.Labels[v1alpha1.LabelCapacityInfo] = string(constant.CapacityInfoInQuota)
	}

	return pod
}

func TestElasticQuotaInfos_Add(t *testing.T) {
	tests := []struct {
		name     string
		eqInfos  ElasticQuotaInfos
		eqInfo   ElasticQuotaInfo
		expected ElasticQuotaInfos
	}{
		{
			name:     "Empty ElasticQuotaInfos, empty ElasticQuotaInfo",
			eqInfos:  NewElasticQuotaInfos(),
			eqInfo:   ElasticQuotaInfo{},
			expected: NewElasticQuotaInfos(),
		},
		{
			name: "Empty ElasticQuotaInfo",
			eqInfos: ElasticQuotaInfos{
				"ns-1": &ElasticQuotaInfo{},
			},
			eqInfo: ElasticQuotaInfo{},
			expected: ElasticQuotaInfos{
				"ns-1": &ElasticQuotaInfo{},
			},
		},
		{
			name: "ElasticQuotaInfo with some namespaces not present",
			eqInfos: ElasticQuotaInfos{
				"ns-1": &ElasticQuotaInfo{
					ResourceName:      "test",
					ResourceNamespace: "test",
				},
			},
			eqInfo: ElasticQuotaInfo{
				ResourceName:      "updated",
				ResourceNamespace: "updated",
				Namespaces:        sets.NewString("ns-2", "ns-3", "ns-4"),
			},
			expected: ElasticQuotaInfos{
				"ns-1": &ElasticQuotaInfo{
					ResourceName:      "test",
					ResourceNamespace: "test",
				},
				"ns-2": &ElasticQuotaInfo{
					ResourceName:      "updated",
					ResourceNamespace: "updated",
					Namespaces:        sets.NewString("ns-2", "ns-3", "ns-4"),
				},
				"ns-3": &ElasticQuotaInfo{
					ResourceName:      "updated",
					ResourceNamespace: "updated",
					Namespaces:        sets.NewString("ns-2", "ns-3", "ns-4"),
				},
				"ns-4": &ElasticQuotaInfo{
					ResourceName:      "updated",
					ResourceNamespace: "updated",
					Namespaces:        sets.NewString("ns-2", "ns-3", "ns-4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.eqInfos.Add(&tt.eqInfo)
			assert.Len(t, tt.eqInfos, len(tt.expected))
			for ns, eqInfo := range tt.eqInfos {
				assert.Equal(t, tt.expected[ns].ResourceNamespace, eqInfo.ResourceNamespace)
				assert.Equal(t, tt.expected[ns].ResourceName, eqInfo.ResourceName)
				assert.Equal(t, tt.expected[ns].Namespaces, eqInfo.Namespaces)
			}
		})
	}
}

func TestElasticQuotaInfos_Update(t *testing.T) {
	tests := []struct {
		name      string
		eqInfos   ElasticQuotaInfos
		oldEqInfo ElasticQuotaInfo
		newEqInfo ElasticQuotaInfo
		expected  ElasticQuotaInfos
	}{
		{
			name:      "Empty ElasticQuotaInfos",
			eqInfos:   NewElasticQuotaInfos(),
			oldEqInfo: ElasticQuotaInfo{},
			newEqInfo: ElasticQuotaInfo{
				ResourceNamespace: "new-ns",
				ResourceName:      "new-name",
				Namespaces:        sets.NewString("ns-1", "ns-2"),
			},
			expected: ElasticQuotaInfos{
				"ns-1": &ElasticQuotaInfo{
					ResourceNamespace: "new-ns",
					ResourceName:      "new-name",
					Namespaces:        sets.NewString("ns-1", "ns-2"),
				},
				"ns-2": &ElasticQuotaInfo{
					ResourceNamespace: "new-ns",
					ResourceName:      "new-name",
					Namespaces:        sets.NewString("ns-1", "ns-2"),
				},
			},
		},
		{
			name:    "New EqInfo does not contain some namespaces present in old EqInfo",
			eqInfos: NewElasticQuotaInfos(),
			oldEqInfo: ElasticQuotaInfo{
				ResourceName:      "old-name",
				ResourceNamespace: "old-ns",
				Namespaces:        sets.NewString("ns-1", "ns-2"),
			},
			newEqInfo: ElasticQuotaInfo{
				ResourceNamespace: "new-ns",
				ResourceName:      "new-name",
				Namespaces:        sets.NewString("ns-2", "ns-3"),
			},
			expected: ElasticQuotaInfos{
				"ns-2": &ElasticQuotaInfo{
					ResourceNamespace: "new-ns",
					ResourceName:      "new-name",
					Namespaces:        sets.NewString("ns-2", "ns-3"),
				},
				"ns-3": &ElasticQuotaInfo{
					ResourceNamespace: "new-ns",
					ResourceName:      "new-name",
					Namespaces:        sets.NewString("ns-2", "ns-3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.eqInfos.Update(&tt.oldEqInfo, &tt.newEqInfo)
			assert.Len(t, tt.eqInfos, len(tt.expected))
			for ns, eqInfo := range tt.eqInfos {
				assert.Equal(t, tt.expected[ns].ResourceNamespace, eqInfo.ResourceNamespace)
				assert.Equal(t, tt.expected[ns].ResourceName, eqInfo.ResourceName)
				assert.Equal(t, tt.expected[ns].Namespaces, eqInfo.Namespaces)
			}
		})
	}
}
