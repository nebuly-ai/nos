package mig

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
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
	logger             logr.Logger
}

func NewPlanner(kubeClient kubernetes.Interface, logger logr.Logger) (*Planner, error) {
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

	return &Planner{schedulerFramework: f, logger: logger}, nil
}

func (p Planner) GetNodesPartitioningPlan(ctx context.Context, snapshot state.ClusterSnapshot, candidates []v1.Pod) (map[string]core.PartitioningPlan, error) {
	plan := make(map[string]core.PartitioningPlan)
	for _, pod := range candidates {
		lackingMig, isLacking := p.getLackingMigResource(snapshot, pod)
		if !isLacking {
			return plan, nil
		}
		candidateNodes := p.getCandidateNodes(snapshot, lackingMig)
		for _, n := range candidateNodes {
			snapshot.Fork()
			//_ = snapshot.UpdateAllocatableScalarResources(n.Name, n.GetAllocatableScalarResources())
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

// getLackingMigResource returns, if any, a Mig resource requested by the Pod but currently not
// available in the ClusterSnapshot.
//
// As described in "Supporting MIG GPUs in Kubernetes" document, it is assumed that
// Pods request only one MIG device per time and with quantity 1, according to the
// idea that users should ask for a larger, single instance as opposed to multiple
// smaller instances.
func (p Planner) getLackingMigResource(snapshot state.ClusterSnapshot, pod v1.Pod) (v1.ResourceName, bool) {
	for r := range snapshot.GetLackingScalarResources(pod) {
		if mig.IsNvidiaMigDevice(r) {
			return r, true
		}
	}
	return "", false
}

func (p Planner) getCandidateNodes(snapshot state.ClusterSnapshot, requiredMigResource v1.ResourceName) []mig.Node {
	result := make([]mig.Node, 0)

	var migNode mig.Node
	var err error

	for _, n := range snapshot.GetNodes() {
		if migNode, err = mig.NewNode(n); err != nil {
			p.logger.V(1).Info(
				"node is not a valid candidate",
				"node",
				n.Node().Name,
				"reason",
				err,
			)
			continue
		}
		if err = migNode.UpdateGeometryFor(requiredMigResource); err != nil {
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
