package mig

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig/migstate"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Planner struct {
	schedulerFramework framework.Framework
	logger             logr.Logger
}

func NewPlanner(scheduler framework.Framework, logger logr.Logger) Planner {
	return Planner{
		schedulerFramework: scheduler,
		logger:             logger,
	}
}

func (p Planner) Plan(ctx context.Context, s state.ClusterSnapshot, candidates []v1.Pod) (state.PartitioningState, error) {
	p.logger.V(3).Info("planning desired GPU partitioning", "candidatePods", len(candidates))
	var err error
	var snapshot migstate.MigClusterSnapshot
	if snapshot, err = migstate.NewClusterSnapshot(s); err != nil {
		return state.PartitioningState{}, fmt.Errorf("error initializing MIG cluster snapshot: %v", err)
	}

	partitioningState := snapshot.GetPartitioningState()
	for _, pod := range candidates {
		lackingMig, isLacking := snapshot.GetLackingMigProfile(pod)
		if !isLacking {
			continue
		}
		candidateNodes := snapshot.GetCandidateNodes()
		p.logger.V(1).Info(
			fmt.Sprintf("found %d candidate nodes for pod", len(candidateNodes)),
			"namespace",
			pod.GetNamespace(),
			"pod",
			pod.GetName(),
			"lackingResource",
			lackingMig,
		)
		for _, n := range candidateNodes {
			// Fork the state
			if err = snapshot.Fork(); err != nil {
				return partitioningState, fmt.Errorf("error forking snapshot, this should never happen: %v", err)
			}

			// Update the node MIG geometry (if possible)
			if err = n.UpdateGeometryFor(lackingMig); err != nil {
				snapshot.Revert()
				continue
			}

			// Update MIG nodes geometry and allocatable scalar resources according to the update MIG geometry
			if err = snapshot.SetNode(n); err != nil {
				return partitioningState, err
			}

			// Run a scheduler cycle to check whether the Pod can be scheduled on the Node
			nodeInfo, _ := snapshot.GetNode(n.Name)
			podFits := p.podFitsNode(ctx, nodeInfo, pod)

			// The Pod cannot be scheduled, revert the changes on the snapshot
			if !podFits {
				p.logger.V(3).Info(
					"pod does not fit node",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.Name,
				)
				snapshot.Revert()
				continue
			}

			// The Pod can be scheduled: commit changes, update desired partitioning and stop iterating over nodes
			if err = snapshot.AddPod(n.Name, pod); err != nil {
				return partitioningState, err
			}
			snapshot.Commit()
			partitioningState[n.Name] = migstate.FromMigNodeToNodePartitioning(n)
			p.logger.V(3).Info(
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
	}
	return partitioningState, nil
}

func (p Planner) podFitsNode(ctx context.Context, node framework.NodeInfo, pod v1.Pod) bool {
	cycleState := framework.NewCycleState()
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false
	}
	return p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge().IsSuccess()
}
