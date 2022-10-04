package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
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
	schedulerFramework framework.Framework
}

func NewPlanner(kubeClient kubernetes.Interface) (*Planner, error) {
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

	return &Planner{schedulerFramework: f}, nil
}

func (p Planner) GetNodesPartitioningPlan(ctx context.Context, snapshot state.ClusterSnapshot, candidates []v1.Pod) (map[string]core.PartitioningPlan, error) {
	plan := make(map[string]core.PartitioningPlan)
	for _, pod := range candidates {
		lackingMIG, isLacking := p.getLackingMIGResource(snapshot, pod)
		if !isLacking {
			return plan, nil
		}
		candidateNodes := p.getCandidateNodes(snapshot, lackingMIG)
		for _, n := range candidateNodes {
			snapshot.Fork()
			_ = snapshot.UpdateAllocatableScalarResources(n.Name, n.GetAllocatableScalarResources())
			nodeInfo, _ := snapshot.GetNode(n.Name)
			podFits, err := p.podFitsNode(ctx, nodeInfo, pod)
			if err != nil {
				return nil, err
			}
			if !podFits {
				snapshot.Revert()
				continue
			}
			_ = snapshot.AddPod(n.Name, pod)
			snapshot.Commit()
			plan[n.Name] = n.GetGPUsGeometry()
		}
	}
	return plan, nil
}

// getLackingMIGResource returns, if any, a MIG resource requested by the Pod but currently not
// available in the ClusterSnapshot.
//
// As described in "Supporting MIG GPUs in Kubernetes" document, it is assumed that
// Pods request only one MIG device per time and with quantity 1, according to the
// idea that users should ask for a larger, single instance as opposed to multiple
// smaller instances.
func (p Planner) getLackingMIGResource(snapshot state.ClusterSnapshot, pod v1.Pod) (v1.ResourceName, bool) {
	for r := range snapshot.GetLackingScalarResources(pod) {
		if resource.IsNvidiaMigDevice(r) {
			return r, true
		}
	}
	return "", false
}

func (p Planner) getCandidateNodes(snapshot state.ClusterSnapshot, requiredMIGResource v1.ResourceName) []gpu.Node {
	result := make([]gpu.Node, 0)

	for _, n := range snapshot.GetNodes() {
		migNode := gpu.NewNode(n)
		if err := migNode.UpdateGeometryFor(requiredMIGResource); err != nil {
			result = append(result, migNode)
		}
	}

	return result
}

func (p Planner) podFitsNode(ctx context.Context, node framework.NodeInfo, pod v1.Pod) (bool, error) {
	cycleState := framework.NewCycleState()
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false, fmt.Errorf("error running pre filter plugins for pod %s; %s", pod.Name, preFilterStatus.Message())
	}
	filterStatus := p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge()
	return filterStatus.IsSuccess(), nil
}
