package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig/migstate"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Planner struct {
	schedulerFramework framework.Framework
}

func NewPlanner(scheduler framework.Framework) Planner {
	return Planner{
		schedulerFramework: scheduler,
	}
}

func (p Planner) Plan(ctx context.Context, s state.ClusterSnapshot, candidates []v1.Pod) (state.PartitioningState, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("planning desired GPU partitioning", "candidatePods", len(candidates))
	var err error
	var snapshot migstate.MigClusterSnapshot
	if snapshot, err = migstate.NewClusterSnapshot(s); err != nil {
		return state.PartitioningState{}, fmt.Errorf("error initializing MIG cluster snapshot: %v", err)
	}

	partitioningState := snapshot.GetPartitioningState()
	for _, pod := range candidates {
		candidateNodes := snapshot.GetCandidateNodes()
		logger.V(1).Info(
			fmt.Sprintf("found %d candidate nodes for pod", len(candidateNodes)),
			"namespace",
			pod.GetNamespace(),
			"pod",
			pod.GetName(),
		)
		for _, n := range candidateNodes {
			// If Pod already fits, move on to next pod
			if p.addPodToSnapshot(ctx, pod, n.Name, snapshot) {
				logger.V(3).Info(
					"pod fits node, cluster snapshot updated",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.Name,
				)
				break
			}

			// Check if any MIG resource is lacking
			lackingMig, isLacking := snapshot.GetLackingMigProfile(pod)
			if !isLacking {
				logger.V(3).Info(
					"no lacking MIG resources, skipping node",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.Name,
				)
				continue
			}

			// Fork the state
			if err = snapshot.Fork(); err != nil {
				return partitioningState, fmt.Errorf("error forking snapshot, this should never happen: %v", err)
			}

			// Try update the node MIG geometry
			if err = n.UpdateGeometryFor(lackingMig); err != nil {
				snapshot.Revert()
				logger.V(3).Info(
					"cannot update node MIG geometry",
					"reason",
					err,
					"node",
					n.Name,
					"lackingMig",
					lackingMig,
				)
				continue
			}

			// Update MIG nodes geometry and allocatable scalar resources according to the update MIG geometry
			if err = snapshot.SetNode(n); err != nil {
				return partitioningState, err
			}

			// Check if the Pod now fits the Node with the updated MIG geometry
			if p.addPodToSnapshot(ctx, pod, n.Name, snapshot) {
				snapshot.Commit()
				partitioningState[n.Name] = migstate.FromMigNodeToNodePartitioning(n)
				logger.V(3).Info(
					"pod fits node, state snapshot updated with new MIG geometry",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.Name,
				)
				break
			}

			// Could not make the Pod fit, revert changes and move on
			logger.V(3).Info("pod does not fit node", "namespace", pod.Namespace, "pod", pod.Name, "node", n.Name)
			snapshot.Revert()
		}
	}
	return partitioningState, nil
}

func (p Planner) addPodToSnapshot(ctx context.Context, pod v1.Pod, node string, snapshot migstate.MigClusterSnapshot) bool {
	// Run a scheduler cycle to check whether the Pod can be scheduled on the Node
	nodeInfo, _ := snapshot.GetNode(node)
	if p.podFitsNode(ctx, nodeInfo, pod) {
		// Try to add the Pod to the cluster
		if err := snapshot.AddPod(node, pod); err == nil {
			return true
		}
	}
	return false
}

func (p Planner) podFitsNode(ctx context.Context, node framework.NodeInfo, pod v1.Pod) bool {
	cycleState := framework.NewCycleState()
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false
	}
	return p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge().IsSuccess()
}
