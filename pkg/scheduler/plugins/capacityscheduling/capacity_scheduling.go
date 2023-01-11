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
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	schedulerconfig "github.com/nebuly-ai/nos/pkg/api/scheduler"
	gpu_util "github.com/nebuly-ai/nos/pkg/gpu/util"
	"github.com/nebuly-ai/nos/pkg/resource"
	podutil "github.com/nebuly-ai/nos/pkg/util/pod"
	"sort"
	"sync"

	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	corelisters "k8s.io/client-go/listers/core/v1"
	policylisters "k8s.io/client-go/listers/policy/v1"
	"k8s.io/client-go/tools/cache"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/preemption"
	"k8s.io/kubernetes/pkg/scheduler/metrics"
	schedutil "k8s.io/kubernetes/pkg/scheduler/util"
)

// CapacityScheduling is a plugin that implements the mechanism of capacity scheduling.
type CapacityScheduling struct {
	sync.RWMutex
	fh                       framework.Handle
	podLister                corelisters.PodLister
	pdbLister                policylisters.PodDisruptionBudgetLister
	elasticQuotaInfos        ElasticQuotaInfos
	resourceCalculator       resource.Calculator
	elasticQuotaInfoInformer *ElasticQuotaInfoInformer
}

// PreFilterState computed at PreFilter and used at PostFilter or Reserve.
type PreFilterState struct {
	podReq framework.Resource

	// nominatedPodsReqInEQWithPodReq is the sum of podReq and the requested resources of the Nominated Pods
	// which subject to the same quota(namespace) and is more important than the preemptor.
	nominatedPodsReqInEQWithPodReq framework.Resource

	// nominatedPodsReqWithPodReq is the sum of podReq and the requested resources of the Nominated Pods
	// which subject to the all quota(namespace). Generated Nominated Pods consist of two kinds of pods:
	// 1. the pods subject to the same quota(namespace) and is more important than the preemptor.
	// 2. the pods subject to the different quota(namespace) and the usage of quota(namespace) does not exceed min.
	nominatedPodsReqWithPodReq framework.Resource
}

// Clone the preFilter state.
func (s *PreFilterState) Clone() framework.StateData {
	return s
}

// ElasticQuotaSnapshotState stores the snapshot of elasticQuotas.
type ElasticQuotaSnapshotState struct {
	elasticQuotaInfos ElasticQuotaInfos
}

// Clone the ElasticQuotaSnapshot state.
func (s *ElasticQuotaSnapshotState) Clone() framework.StateData {
	return &ElasticQuotaSnapshotState{
		elasticQuotaInfos: s.elasticQuotaInfos.clone(),
	}
}

var _ framework.PreFilterPlugin = &CapacityScheduling{}
var _ framework.PostFilterPlugin = &CapacityScheduling{}
var _ framework.ReservePlugin = &CapacityScheduling{}
var _ framework.EnqueueExtensions = &CapacityScheduling{}
var _ preemption.Interface = &preemptor{}

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name = "CapacityScheduling"

	// preFilterStateKey is the key in CycleState to NodeResourcesFit pre-computed data.
	preFilterStateKey       = "PreFilter" + Name
	ElasticQuotaSnapshotKey = "ElasticQuotaSnapshot"
)

// Name returns name of the plugin. It is used in logs, etc.
func (c *CapacityScheduling) Name() string {
	return Name
}

// New initializes a new plugin and returns it.
func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*schedulerconfig.CapacitySchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("[CapacityScheduling] want args to be of type CapacitySchedulingArgs, got %T", obj)
	}

	klog.Info("using nvidiaGpuResourceMemoryGB=", args.NvidiaGpuResourceMemoryGB)

	c := &CapacityScheduling{
		fh:                handle,
		elasticQuotaInfos: NewElasticQuotaInfos(),
		podLister:         handle.SharedInformerFactory().Core().V1().Pods().Lister(),
		pdbLister:         getPDBLister(handle.SharedInformerFactory()),
		resourceCalculator: &gpu_util.ResourceCalculator{
			NvidiaGPUDeviceMemoryGB: args.NvidiaGpuResourceMemoryGB,
		},
	}

	eqInformer, err := NewElasticQuotaInfoInformer(handle.KubeConfig(), c.resourceCalculator)
	if err != nil {
		return nil, err
	}
	eqInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addElasticQuotaInfo,
		UpdateFunc: c.updateElasticQuotaInfo,
		DeleteFunc: c.deleteElasticQuotaInfo,
	})
	eqInformer.Start(nil)
	if !cache.WaitForCacheSync(nil, eqInformer.HasSynced) {
		return nil, fmt.Errorf("timed out waiting for ElasticQuotaInformer caches to sync %v", Name)
	}
	c.elasticQuotaInfoInformer = eqInformer

	podInformer := handle.SharedInformerFactory().Core().V1().Pods().Informer()
	podInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return assignedPod(t)
				case cache.DeletedFinalStateUnknown:
					if pod, ok := t.Obj.(*v1.Pod); ok {
						return assignedPod(pod)
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    c.addPod,
				UpdateFunc: c.updatePod,
				DeleteFunc: c.deletePod,
			},
		},
	)
	handle.SharedInformerFactory().Start(nil)
	if !cache.WaitForCacheSync(nil, podInformer.HasSynced) {
		return nil, fmt.Errorf("timed out waiting for PodInformer caches to sync %v", Name)
	}
	klog.InfoS("[CapacityScheduling] started")
	return c, nil
}

func (c *CapacityScheduling) EventsToRegister() []framework.ClusterEvent {
	// To register a custom event, follow the naming convention at:
	// https://git.k8s.io/kubernetes/pkg/scheduler/eventhandlers.go#L403-L410
	eqGVK := fmt.Sprintf("elasticquotas.v1alpha1.%v", v1alpha1.GroupName)
	return []framework.ClusterEvent{
		{Resource: framework.Pod, ActionType: framework.Delete},
		{Resource: framework.GVK(eqGVK), ActionType: framework.All},
	}
}

// PreFilter performs the following validations.
// 1. Check if the (pod.request + eq.allocated) is less than eq.max.
// 2. Check if the sum(eq's usage) > sum(eq's min).
func (c *CapacityScheduling) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	// TODO improve the efficiency of taking snapshot
	// e.g. use a two-pointer data structure to only copy the updated EQs when necessary.
	snapshotElasticQuota := c.snapshotElasticQuota()
	req := c.resourceCalculator.ComputePodRequest(*pod)
	podReq := resource.FromListToFramework(req)

	state.Write(ElasticQuotaSnapshotKey, snapshotElasticQuota)

	elasticQuotaInfos := snapshotElasticQuota.elasticQuotaInfos
	eq := snapshotElasticQuota.elasticQuotaInfos[pod.Namespace]
	if eq == nil {
		klog.V(1).InfoS("pod's namespace is not subject to any quota", "namespace", pod.Namespace)
		preFilterState := &PreFilterState{
			podReq: podReq,
		}
		state.Write(preFilterStateKey, preFilterState)
		return nil, framework.NewStatus(framework.Success)
	}

	// nominatedPodsReqInEQWithPodReq is the sum of podReq and the requested resources of the Nominated Pods
	// which subject to the same quota(namespace) and is more important than the preemptor.
	nominatedPodsReqInEQWithPodReq := &framework.Resource{}
	// nominatedPodsReqWithPodReq is the sum of podReq and the requested resources of the Nominated Pods
	// which subject to the all quota(namespace). Generated Nominated Pods consist of two kinds of pods:
	// 1. the pods subject to the same quota(namespace) and is more important than the preemptor.
	// 2. the pods subject to the different quota(namespace) and the usage of quota(namespace) does not exceed min.
	nominatedPodsReqWithPodReq := &framework.Resource{}

	nodeList, err := c.fh.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		return nil, framework.NewStatus(framework.Error, fmt.Sprintf("Error getting the nodelist: %v", err))
	}

	for _, node := range nodeList {
		nominatedPods := c.fh.NominatedPodsForNode(node.Node().Name)
		for _, p := range nominatedPods {
			if p.Pod.UID == pod.UID {
				continue
			}
			ns := p.Pod.Namespace
			info := c.elasticQuotaInfos[ns]
			if info != nil {
				pResourceRequest := c.resourceCalculator.ComputePodRequest(*p.Pod)
				// If they are subject to the same quota(namespace) and p is more important than pod,
				// p will be added to the nominatedResource and totalNominatedResource.
				// If they aren't subject to the same quota(namespace) and the usage of quota(p's namespace) does not exceed min,
				// p will be added to the totalNominatedResource.
				if ns == pod.Namespace && corev1helpers.PodPriority(p.Pod) >= corev1helpers.PodPriority(pod) {
					nominatedPodsReqInEQWithPodReq.Add(pResourceRequest)
					nominatedPodsReqWithPodReq.Add(pResourceRequest)
				} else if ns != pod.Namespace && !info.usedOverMin() {
					nominatedPodsReqWithPodReq.Add(pResourceRequest)
				}
			}
		}
	}

	nominatedPodsReqInEQWithPodReq.Add(resource.FromFrameworkToList(podReq))
	nominatedPodsReqWithPodReq.Add(resource.FromFrameworkToList(podReq))
	preFilterState := &PreFilterState{
		podReq:                         podReq,
		nominatedPodsReqInEQWithPodReq: *nominatedPodsReqInEQWithPodReq,
		nominatedPodsReqWithPodReq:     *nominatedPodsReqWithPodReq,
	}
	state.Write(preFilterStateKey, preFilterState)

	if eq.usedOverMaxWith(nominatedPodsReqInEQWithPodReq) {
		msg := fmt.Sprintf(
			"Pod %v/%v is rejected in PreFilter because quota %v/%v is more than Max",
			pod.Namespace,
			pod.Name,
			eq.ResourceNamespace,
			eq.ResourceName,
		)
		return nil, framework.NewStatus(framework.Unschedulable, msg)
	}

	if elasticQuotaInfos.AggregatedUsedOverMinWith(*nominatedPodsReqWithPodReq) {
		msg := fmt.Sprintf(
			"Pod %v/%v is rejected in PreFilter because total quota used is more than min",
			pod.Namespace,
			pod.Name,
		)
		return nil, framework.NewStatus(framework.Unschedulable, msg)
	}

	return nil, framework.NewStatus(framework.Success, "")
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (c *CapacityScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return c
}

// AddPod from pre-computed data in cycleState.
func (c *CapacityScheduling) AddPod(ctx context.Context, cycleState *framework.CycleState, podToSchedule *v1.Pod, podToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	elasticQuotaSnapshotState, err := getElasticQuotaSnapshotState(cycleState)
	if err != nil {
		klog.ErrorS(err, "Failed to read elasticQuotaSnapshot from cycleState", "elasticQuotaSnapshotKey", ElasticQuotaSnapshotKey)
		return framework.NewStatus(framework.Error, err.Error())
	}

	elasticQuotaInfo := elasticQuotaSnapshotState.elasticQuotaInfos[podToAdd.Pod.Namespace]
	if elasticQuotaInfo != nil {
		err = elasticQuotaInfo.addPodIfNotPresent(podToAdd.Pod)
		if err != nil {
			klog.ErrorS(err, "Failed to add Pod to its associated elasticQuota", "pod", klog.KObj(podToAdd.Pod))
		}
	}

	return framework.NewStatus(framework.Success, "")
}

// RemovePod from pre-computed data in cycleState.
func (c *CapacityScheduling) RemovePod(ctx context.Context, cycleState *framework.CycleState, podToSchedule *v1.Pod, podToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	elasticQuotaSnapshotState, err := getElasticQuotaSnapshotState(cycleState)
	if err != nil {
		klog.ErrorS(err, "Failed to read elasticQuotaSnapshot from cycleState", "elasticQuotaSnapshotKey", ElasticQuotaSnapshotKey)
		return framework.NewStatus(framework.Error, err.Error())
	}

	elasticQuotaInfo := elasticQuotaSnapshotState.elasticQuotaInfos[podToRemove.Pod.Namespace]
	if elasticQuotaInfo != nil {
		err = elasticQuotaInfo.deletePodIfPresent(podToRemove.Pod)
		if err != nil {
			klog.ErrorS(err, "Failed to delete Pod from its associated elasticQuota", "pod", klog.KObj(podToRemove.Pod))
		}
	}

	return framework.NewStatus(framework.Success, "")
}

func (c *CapacityScheduling) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, m framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	defer func() {
		metrics.PreemptionAttempts.Inc()
	}()

	pe := preemption.Evaluator{
		PluginName: c.Name(),
		Handler:    c.fh,
		PodLister:  c.podLister,
		PdbLister:  c.pdbLister,
		State:      state,
		Interface: &preemptor{
			fh:    c.fh,
			state: state,
		},
	}

	return pe.Preempt(ctx, pod, m)
}

func (c *CapacityScheduling) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	c.Lock()
	defer c.Unlock()

	elasticQuotaInfo := c.elasticQuotaInfos[pod.Namespace]
	if elasticQuotaInfo != nil {
		err := elasticQuotaInfo.addPodIfNotPresent(pod)
		if err != nil {
			klog.ErrorS(err, "Failed to add Pod to its associated elasticQuota", "pod", klog.KObj(pod))
			return framework.NewStatus(framework.Error, err.Error())
		}
	}
	return framework.NewStatus(framework.Success, "")
}

func (c *CapacityScheduling) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	c.Lock()
	defer c.Unlock()

	elasticQuotaInfo := c.elasticQuotaInfos[pod.Namespace]
	if elasticQuotaInfo != nil {
		err := elasticQuotaInfo.deletePodIfPresent(pod)
		if err != nil {
			klog.ErrorS(err, "Failed to delete Pod from its associated elasticQuota", "pod", klog.KObj(pod))
		}
	}
}

type preemptor struct {
	fh    framework.Handle
	state *framework.CycleState
}

func (p *preemptor) GetOffsetAndNumCandidates(n int32) (int32, int32) {
	return 0, n
}

func (p *preemptor) CandidatesToVictimsMap(candidates []preemption.Candidate) map[string]*extenderv1.Victims {
	m := make(map[string]*extenderv1.Victims)
	for _, c := range candidates {
		m[c.Name()] = c.Victims()
	}
	return m
}

// PodEligibleToPreemptOthers determines whether this pod should be considered
// for preempting other pods or not. If this pod has already preempted other
// pods and those are in their graceful termination period, it shouldn't be
// considered for preemption.
// We look at the node that is nominated for this pod and as long as there are
// terminating pods on the node, we don't consider this for preempting more pods.
func (p *preemptor) PodEligibleToPreemptOthers(pod *v1.Pod, nominatedNodeStatus *framework.Status) (bool, string) {
	if pod.Spec.PreemptionPolicy != nil && *pod.Spec.PreemptionPolicy == v1.PreemptNever {
		klog.V(5).InfoS("Pod is not eligible for preemption because of its preemptionPolicy", "pod", klog.KObj(pod), "preemptionPolicy", v1.PreemptNever)
		return false, "not eligible due to preemptionPolicy=Never."
	}

	preFilterState, err := getPreFilterState(p.state)
	if err != nil {
		klog.ErrorS(err, "Failed to read preFilterState from cycleState", "preFilterStateKey", preFilterStateKey)
		return false, "not eligible due to failed to read from cycleState"
	}

	nomNodeName := pod.Status.NominatedNodeName
	nodeLister := p.fh.SnapshotSharedLister().NodeInfos()
	if len(nomNodeName) > 0 {
		// If the pod's nominated node is considered as UnschedulableAndUnresolvable by the filters,
		// then the pod should be considered for preempting again.
		if nominatedNodeStatus.Code() == framework.UnschedulableAndUnresolvable {
			return true, ""
		}

		elasticQuotaSnapshotState, err := getElasticQuotaSnapshotState(p.state)
		if err != nil {
			klog.ErrorS(err, "Failed to read elasticQuotaSnapshot from cycleState", "elasticQuotaSnapshotKey", ElasticQuotaSnapshotKey)
			return true, ""
		}

		nodeInfo, _ := nodeLister.Get(nomNodeName)
		if nodeInfo == nil {
			return true, ""
		}

		podPriority := corev1helpers.PodPriority(pod)
		preemptorEQInfo, preemptorWithEQ := elasticQuotaSnapshotState.elasticQuotaInfos[pod.Namespace]
		if preemptorWithEQ {
			moreThanMinWithPreemptor := preemptorEQInfo.usedOverMinWith(&preFilterState.nominatedPodsReqInEQWithPodReq)
			for _, p := range nodeInfo.Pods {
				if p.Pod.DeletionTimestamp != nil {
					eqInfo, withEQ := elasticQuotaSnapshotState.elasticQuotaInfos[p.Pod.Namespace]
					if !withEQ {
						continue
					}
					if p.Pod.Namespace == pod.Namespace && corev1helpers.PodPriority(p.Pod) < podPriority {
						// There is a terminating pod on the nominated node.
						// If the terminating pod is in the same namespace with preemptor
						// and it is less important than preemptor,
						// return false to avoid preempting more pods.
						return false, "not eligible due to a terminating pod on the nominated node."
					} else if p.Pod.Namespace != pod.Namespace && !moreThanMinWithPreemptor && eqInfo.usedOverMin() {
						// There is a terminating pod on the nominated node.
						// The terminating pod isn't in the same namespace with preemptor.
						// If moreThanMinWithPreemptor is false, it indicates that preemptor can preempt the pods in other EQs whose used is over min.
						// And if the used of terminating pod's quota is over min, so the room released by terminating pod on the nominated node can be used by the preemptor.
						// return false to avoid preempting more pods.
						return false, "not eligible due to a terminating pod on the nominated node."
					}
				}
			}
		} else {
			for _, p := range nodeInfo.Pods {
				_, withEQ := elasticQuotaSnapshotState.elasticQuotaInfos[p.Pod.Namespace]
				if withEQ {
					continue
				}
				if p.Pod.DeletionTimestamp != nil && corev1helpers.PodPriority(p.Pod) < podPriority {
					// There is a terminating pod on the nominated node.
					return false, "not eligible due to a terminating pod on the nominated node."
				}
			}
		}
	}
	return true, ""
}

func (p *preemptor) SelectVictimsOnNode(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
	pdbs []*policy.PodDisruptionBudget) ([]*v1.Pod, int, *framework.Status) {
	elasticQuotaSnapshotState, err := getElasticQuotaSnapshotState(state)
	if err != nil {
		msg := "Failed to read elasticQuotaSnapshot from cycleState"
		klog.ErrorS(err, msg, "elasticQuotaSnapshotKey", ElasticQuotaSnapshotKey)
		return nil, 0, framework.NewStatus(framework.Unschedulable, msg)
	}

	preFilterState, err := getPreFilterState(state)
	if err != nil {
		msg := "Failed to read preFilterState from cycleState"
		klog.ErrorS(err, msg, "preFilterStateKey", preFilterStateKey)
		return nil, 0, framework.NewStatus(framework.Unschedulable, msg)
	}

	var nominatedPodsReqInEQWithPodReq framework.Resource
	var nominatedPodsReqWithPodReq framework.Resource
	podReq := preFilterState.podReq

	removePod := func(rpi *framework.PodInfo) error {
		if err := nodeInfo.RemovePod(rpi.Pod); err != nil {
			return err
		}
		status := p.fh.RunPreFilterExtensionRemovePod(ctx, state, pod, rpi, nodeInfo)
		if !status.IsSuccess() {
			return status.AsError()
		}
		return nil
	}
	addPod := func(api *framework.PodInfo) error {
		nodeInfo.AddPodInfo(api)
		status := p.fh.RunPreFilterExtensionAddPod(ctx, state, pod, api, nodeInfo)
		if !status.IsSuccess() {
			return status.AsError()
		}
		return nil
	}

	elasticQuotaInfos := elasticQuotaSnapshotState.elasticQuotaInfos
	podPriority := corev1helpers.PodPriority(pod)
	preemptorElasticQuotaInfo, preemptorWithElasticQuota := elasticQuotaInfos[pod.Namespace]

	// sort the pods in node by the priority class
	sort.Slice(nodeInfo.Pods, func(i, j int) bool { return !schedutil.MoreImportantPod(nodeInfo.Pods[i].Pod, nodeInfo.Pods[j].Pod) })

	var potentialVictims []*framework.PodInfo
	if preemptorWithElasticQuota {
		nominatedPodsReqInEQWithPodReq = preFilterState.nominatedPodsReqInEQWithPodReq
		nominatedPodsReqWithPodReq = preFilterState.nominatedPodsReqWithPodReq
		moreThanMinWithPreemptor := preemptorElasticQuotaInfo.usedOverMinWith(&nominatedPodsReqInEQWithPodReq)
		for _, pvPi := range nodeInfo.Pods {
			pvEqInfo, withEQ := elasticQuotaInfos[pvPi.Pod.Namespace]
			if !withEQ {
				continue
			}
			// Preemptor.Request + Quota.Used > Quota.Min  => overquota
			if moreThanMinWithPreemptor {

				// If pod_namespace == potential_victim_namespace than we select the pods
				// subject to the same quota(namespace) with the lower priority than the
				// preemptor's priority as potential victims in a node.
				if pvPi.Pod.Namespace == pod.Namespace {
					if corev1helpers.PodPriority(pvPi.Pod) < podPriority {
						potentialVictims = append(potentialVictims, pvPi)
						if err := removePod(pvPi); err != nil {
							return nil, 0, framework.AsStatus(err)
						}
					}
				}

				// If pod_namespace != potential_victim_namespace than we check
				// whether the preemptor EQ has guaranteed overquotas available,
				// and we select as potential victims over-quota pods in other namespaces where
				// UsedOverquotas > GuaranteedOverquotas
				if pvPi.Pod.Namespace == pod.Namespace {
					continue
				}
				if !podutil.IsOverQuota(*pvPi.Pod) {
					continue
				}
				guaranteeedOverquotas, _ := elasticQuotaInfos.GetGuaranteedOverquotas(pod.Namespace)
				minPlusGuaranteeedOverquotas := resource.Sum(*guaranteeedOverquotas, *preemptorElasticQuotaInfo.Min)
				if preemptorElasticQuotaInfo.usedLteWith(&minPlusGuaranteeedOverquotas, &nominatedPodsReqInEQWithPodReq) {
					pvGuaranteedOverquotas, _ := elasticQuotaInfos.GetGuaranteedOverquotas(pvPi.Pod.Namespace)
					pvMinPlusGuaranteedOverquotas := resource.Sum(*pvGuaranteedOverquotas, *pvEqInfo.Min)
					if pvEqInfo.usedOver(&pvMinPlusGuaranteedOverquotas) {
						potentialVictims = append(potentialVictims, pvPi)
						if err := removePod(pvPi); err != nil {
							return nil, 0, framework.AsStatus(err)
						}
					}
				}

			} else {
				// If Preemptor.Request + Quota.allocated <= Quota.min: It
				// means that its min(guaranteed) resource is used or
				// `borrowed` by other Quota. Potential victims in a node
				// will be chosen from Quotas that allocates more resources
				// than its min, i.e., borrowing resources from other
				// Quotas. Only Pods marked as "overquota" can be preempted.
				if pvPi.Pod.Namespace != pod.Namespace && pvEqInfo.usedOverMin() {
					if podutil.IsOverQuota(*pvPi.Pod) {
						potentialVictims = append(potentialVictims, pvPi)
						if err := removePod(pvPi); err != nil {
							return nil, 0, framework.AsStatus(err)
						}
					}
				}
			}
		}
	} else {
		for _, pi := range nodeInfo.Pods {
			_, withEQ := elasticQuotaInfos[pi.Pod.Namespace]
			if withEQ {
				continue
			}
			if corev1helpers.PodPriority(pi.Pod) < podPriority {
				potentialVictims = append(potentialVictims, pi)
				if err := removePod(pi); err != nil {
					return nil, 0, framework.AsStatus(err)
				}
			}
		}
	}

	// No potential victims are found, and so we don't need to evaluate the node again since its state didn't change.
	if len(potentialVictims) == 0 {
		message := fmt.Sprintf("No victims found on node %v for preemptor pod %v", nodeInfo.Node().Name, pod.Name)
		return nil, 0, framework.NewStatus(framework.UnschedulableAndUnresolvable, message)
	}

	// If the new pod does not fit after removing all the lower priority pods,
	// we are almost done and this node is not suitable for preemption. The only
	// condition that we could check is if the "pod" is failing to schedule due to
	// inter-pod affinity to one or more victims, but we have decided not to
	// support this case for performance reasons. Having affinity to lower
	// priority pods is not a recommended configuration anyway.
	if s := p.fh.RunFilterPluginsWithNominatedPods(ctx, state, pod, nodeInfo); !s.IsSuccess() {
		return nil, 0, s
	}

	// If the quota.used + pod.request > quota.max or sum(quotas.used) + pod.request > sum(quotas.min)
	// after removing all the lower priority pods,
	// we are almost done and this node is not suitable for preemption.
	if preemptorWithElasticQuota {
		if preemptorElasticQuotaInfo.usedOverMaxWith(&podReq) {
			return nil, 0, framework.NewStatus(framework.Unschedulable, "max quota exceeded")
		}
		if elasticQuotaInfos.AggregatedUsedOverMinWith(podReq) {
			return nil, 0, framework.NewStatus(framework.Unschedulable, "total min quota exceeded")
		}
	}

	var victims []*v1.Pod
	numViolatingVictim := 0
	sort.Slice(potentialVictims, func(i, j int) bool {
		return schedutil.MoreImportantPod(potentialVictims[i].Pod, potentialVictims[j].Pod)
	})
	// Try to reprieve as many pods as possible. We first try to reprieve the PDB
	// violating victims and then other non-violating ones. In both cases, we start
	// from the highest priority victims.
	violatingVictims, nonViolatingVictims := filterPodsWithPDBViolation(potentialVictims, pdbs)
	reprievePod := func(pi *framework.PodInfo) (bool, error) {
		if err := addPod(pi); err != nil { // this updates elastic quota infos
			return false, err
		}
		s := p.fh.RunFilterPluginsWithNominatedPods(ctx, state, pod, nodeInfo)
		fits := s.IsSuccess()
		if !fits {
			if err := removePod(pi); err != nil {
				return false, err
			}
			victims = append(victims, pi.Pod)
			klog.V(5).InfoS("Found a potential preemption victim on node", "pod", klog.KObj(pi.Pod), "node", klog.KObj(nodeInfo.Node()))
		}

		if preemptorWithElasticQuota && (preemptorElasticQuotaInfo.usedOverMaxWith(&nominatedPodsReqInEQWithPodReq) || elasticQuotaInfos.AggregatedUsedOverMinWith(nominatedPodsReqWithPodReq)) {
			if err := removePod(pi); err != nil {
				return false, err
			}
			victims = append(victims, pi.Pod)
			klog.V(5).InfoS("Found a potential preemption victim on node", "pod", klog.KObj(pi.Pod), " node", klog.KObj(nodeInfo.Node()))
		}

		return fits, nil
	}
	for _, pi := range violatingVictims {
		if fits, err := reprievePod(pi); err != nil {
			klog.ErrorS(err, "Failed to reprieve pod", "pod", klog.KObj(pi.Pod))
			return nil, 0, framework.AsStatus(err)
		} else if !fits {
			numViolatingVictim++
		}
	}
	// Now we try to reprieve non-violating victims.
	for _, pi := range nonViolatingVictims {
		if _, err := reprievePod(pi); err != nil {
			klog.ErrorS(err, "Failed to reprieve pod", "pod", klog.KObj(pi.Pod))
			return nil, 0, framework.AsStatus(err)
		}
	}
	return victims, numViolatingVictim, framework.NewStatus(framework.Success)
}

func (c *CapacityScheduling) addElasticQuotaInfo(obj interface{}) {
	eqInfo := obj.(*ElasticQuotaInfo)
	klog.V(1).InfoS(
		"add ElasticQuotaInfo",
		"namespace",
		eqInfo.ResourceNamespace,
		"name",
		eqInfo.ResourceName,
		"namespaces",
		eqInfo.Namespaces.List(),
	)
	c.Lock()
	defer c.Unlock()
	c.elasticQuotaInfos.Add(eqInfo)
}

func (c *CapacityScheduling) updateElasticQuotaInfo(oldObj, newObj interface{}) {
	oldEqInfo := oldObj.(*ElasticQuotaInfo)
	newEqInfo := newObj.(*ElasticQuotaInfo)
	klog.V(1).InfoS(
		"update ElasticQuotaInfo",
		"namespace",
		oldEqInfo.ResourceNamespace,
		"name",
		oldEqInfo.ResourceName,
		"namespaces",
		oldEqInfo.Namespaces.List(),
	)
	c.Lock()
	defer c.Unlock()
	c.elasticQuotaInfos.Update(oldEqInfo, newEqInfo)
}

func (c *CapacityScheduling) deleteElasticQuotaInfo(obj interface{}) {
	eqInfo := obj.(*ElasticQuotaInfo)
	klog.V(1).InfoS(
		"delete ElasticQuotaInfo",
		"namespace",
		eqInfo.ResourceNamespace,
		"name",
		eqInfo.ResourceName,
		"namespaces",
		eqInfo.Namespaces.List(),
	)
	c.Lock()
	defer c.Unlock()
	c.elasticQuotaInfos.Delete(eqInfo)
}

func (c *CapacityScheduling) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)

	c.Lock()
	defer c.Unlock()

	elasticQuotaInfo := c.getElasticQuotaInfoForPod(pod)
	if elasticQuotaInfo != nil {
		err := elasticQuotaInfo.addPodIfNotPresent(pod)
		if err != nil {
			klog.ErrorS(err, "Failed to add Pod to its associated elasticQuota", "pod", klog.KObj(pod))
		}
	}
}

func (c *CapacityScheduling) getElasticQuotaInfoForPod(pod *v1.Pod) *ElasticQuotaInfo {
	elasticQuotaInfo := c.elasticQuotaInfos[pod.Namespace]
	if elasticQuotaInfo != nil {
		return elasticQuotaInfo
	}

	// If elasticQuotaInfo is nil, first try to fetch it from CompositeElasticQuotas
	compositeEq, err := c.elasticQuotaInfoInformer.GetAssociatedCompositeElasticQuota(pod.Namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to get associated CompositeElasticQuota", "namespace", pod.Namespace)
		return nil
	}
	if compositeEq != nil {
		return compositeEq
	}

	// If no CompositeElasticQuotas is defined on Pod's namespace, try to fetch ElasticQuotaInfo from ElasticQuotas
	eq, err := c.elasticQuotaInfoInformer.GetAssociatedElasticQuota(pod.Namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to get associated ElasticQuota", "namespace", pod.Namespace)
		return nil
	}
	return eq
}

func (c *CapacityScheduling) updatePod(oldObj, newObj interface{}) {
	oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)

	if oldPod.Status.Phase == v1.PodSucceeded || oldPod.Status.Phase == v1.PodFailed {
		return
	}

	if newPod.Status.Phase != v1.PodRunning && newPod.Status.Phase != v1.PodPending {
		c.Lock()
		defer c.Unlock()

		elasticQuotaInfo := c.elasticQuotaInfos[newPod.Namespace]
		if elasticQuotaInfo != nil {
			err := elasticQuotaInfo.deletePodIfPresent(newPod)
			if err != nil {
				klog.ErrorS(err, "Failed to delete Pod from its associated elasticQuota", "pod", klog.KObj(newPod))
			}
		}
	}
}

func (c *CapacityScheduling) deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	c.Lock()
	defer c.Unlock()

	elasticQuotaInfo := c.elasticQuotaInfos[pod.Namespace]
	if elasticQuotaInfo != nil {
		err := elasticQuotaInfo.deletePodIfPresent(pod)
		if err != nil {
			klog.ErrorS(err, "Failed to delete Pod from its associated elasticQuota", "pod", klog.KObj(pod))
		}
	}
}

// getElasticQuotasSnapshot will return the snapshot of elasticQuotas.
func (c *CapacityScheduling) snapshotElasticQuota() *ElasticQuotaSnapshotState {
	c.RLock()
	defer c.RUnlock()

	elasticQuotaInfosDeepCopy := c.elasticQuotaInfos.clone()
	return &ElasticQuotaSnapshotState{
		elasticQuotaInfos: elasticQuotaInfosDeepCopy,
	}
}

func getPreFilterState(cycleState *framework.CycleState) (*PreFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %v", preFilterStateKey, err)
	}

	s, ok := c.(*PreFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to NodeResourcesFit.preFilterState error", c)
	}
	return s, nil
}

func getElasticQuotaSnapshotState(cycleState *framework.CycleState) (*ElasticQuotaSnapshotState, error) {
	c, err := cycleState.Read(ElasticQuotaSnapshotKey)
	if err != nil {
		// ElasticQuotaSnapshotState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %v", ElasticQuotaSnapshotKey, err)
	}

	s, ok := c.(*ElasticQuotaSnapshotState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to CapacityScheduling ElasticQuotaSnapshotState error", c)
	}
	return s, nil
}

func getPDBLister(informerFactory informers.SharedInformerFactory) policylisters.PodDisruptionBudgetLister {
	return informerFactory.Policy().V1().PodDisruptionBudgets().Lister()
}

// filterPodsWithPDBViolation groups the given "pods" into two groups of "violatingPods"
// and "nonViolatingPods" based on whether their PDBs will be violated if they are
// preempted.
// This function is stable and does not change the order of received pods. So, if it
// receives a sorted list, grouping will preserve the order of the input list.
func filterPodsWithPDBViolation(podInfos []*framework.PodInfo, pdbs []*policy.PodDisruptionBudget) (violatingPods, nonViolatingPods []*framework.PodInfo) {
	pdbsAllowed := make([]int32, len(pdbs))
	for i, pdb := range pdbs {
		pdbsAllowed[i] = pdb.Status.DisruptionsAllowed
	}

	for _, podInfo := range podInfos {
		pod := podInfo.Pod
		pdbForPodIsViolated := false
		// A pod with no labels will not match any PDB. So, no need to check.
		if len(pod.Labels) != 0 {
			for i, pdb := range pdbs {
				if pdb.Namespace != pod.Namespace {
					continue
				}
				selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
				if err != nil {
					continue
				}
				// A PDB with a nil or empty selector matches nothing.
				if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
					continue
				}

				// Existing in DisruptedPods means it has been processed in API server,
				// we don't treat it as a violating case.
				if _, exist := pdb.Status.DisruptedPods[pod.Name]; exist {
					continue
				}
				// Only decrement the matched pdb when it's not in its <DisruptedPods>;
				// otherwise we may over-decrement the budget number.
				pdbsAllowed[i]--
				// We have found a matching PDB.
				if pdbsAllowed[i] < 0 {
					pdbForPodIsViolated = true
				}
			}
		}
		if pdbForPodIsViolated {
			violatingPods = append(violatingPods, podInfo)
		} else {
			nonViolatingPods = append(nonViolatingPods, podInfo)
		}
	}
	return violatingPods, nonViolatingPods
}

// assignedPod selects pods that are assigned (scheduled and running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}
