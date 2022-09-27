package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	scheduler_config "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	scheduler_plugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	"k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

type Planner struct {
	snapshot           state.ClusterSnapshot
	schedulerFramework framework.Framework
}

func NewPlanner(kubeClient kubernetes.Interface, clusterSnapshot state.ClusterSnapshot) (*Planner, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	config, err := scheduler_config.Default()
	if err != nil {
		return nil, fmt.Errorf("couldn't create scheduler config: %v", err)
	}
	if len(config.Profiles) != 1 || config.Profiles[0].SchedulerName != v1.DefaultSchedulerName {
		return nil, fmt.Errorf("unexpected scheduler config: expected default scheduler profile only (found %d profiles)", len(config.Profiles))
	}

	f, err := runtime.NewFramework(
		scheduler_plugins.NewInTreeRegistry(),
		&config.Profiles[0],
		runtime.WithInformerFactory(informerFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't create scheduler framework; %v", err)
	}

	return &Planner{schedulerFramework: f, snapshot: clusterSnapshot}, nil
}

func (p Planner) GetNodesPartitioningPlan(ctx context.Context, pendingCandidates []v1.Pod) (map[string]v1.ResourceList, error) {
	plan := make(map[string]v1.ResourceList)
	for _, pod := range pendingCandidates {
		podLackingMIGs := p.getLackingMIGResources(pod)
		if len(podLackingMIGs) == 0 {
			continue
		}
		nodesWithPartitionedResources := p.getCandidateNodesForPartitioning(podLackingMIGs)
		for n, scalarResources := range nodesWithPartitionedResources {
			p.snapshot.Fork()
			_ = p.snapshot.UpdateAllocatableScalarResources(n, scalarResources)
			nodeFits, err := p.nodeFits(ctx, n, pod)
			if err != nil {
				return nil, err
			}
			if !nodeFits {
				p.snapshot.Revert()
				continue
			}
			_ = p.snapshot.AddPod(n, pod)
			p.snapshot.Commit()
		}
	}
	return plan, nil
}

func (p Planner) getLackingMIGResources(pod v1.Pod) v1.ResourceList {
	result := make(v1.ResourceList)
	for r, q := range p.snapshot.GetLackingScalarResources(pod) {
		if resource.IsNvidiaMigDevice(r) {
			result[r] = q
		}
	}
	return result
}

func (p Planner) getCandidateNodesForPartitioning(requiredMigResources v1.ResourceList) map[string]v1.ResourceList {

	//for _, n := range p.snapshot.GetNodes() {
	//	for _, r := range n.Allocatable.ScalarResources {
	//	}
	//}

	return nil
}

func (p Planner) nodeFits(ctx context.Context, nodeName string, pod v1.Pod) (bool, error) {
	cycleState := framework.NewCycleState()
	node, found := p.snapshot.GetNode(nodeName)
	if !found {
		return false, nil
	}
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false, fmt.Errorf("error running pre filter plugins for pod %s; %s", pod.Name, preFilterStatus.Message())
	}
	filterStatus := p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge()
	return filterStatus.IsSuccess(), nil
}
