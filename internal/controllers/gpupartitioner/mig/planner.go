package mig

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
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

func (p Planner) Plan(ctx context.Context, snapshot state.ClusterSnapshot, candidates []v1.Pod) (state.PartitioningState, error) {
	res := p.getPartitioningState(snapshot)
	p.logger.V(3).Info("planning desired GPU partitioning", "candidatePods", len(candidates))
	for _, pod := range candidates {
		lackingMig, isLacking := p.getLackingMigProfile(snapshot, pod)
		if !isLacking {
			continue
		}
		candidateNodes := p.getCandidateNodes(snapshot)
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
			// Check if node can potentially host the Pod by updating its MIG geometry
			if err := n.UpdateGeometryFor(lackingMig); err != nil {
				continue
			}

			// Fork the state and update the nodes' allocatable scalar resources by taking into
			// account the new MIG geometry
			if err := snapshot.Fork(); err != nil {
				return res, fmt.Errorf("error forking cluster snapshot, this should never happen: %v", err)
			}
			nodeInfo, _ := snapshot.GetNode(n.Name)
			scalarResources := getUpdatedScalarResources(*nodeInfo, n)
			nodeInfo.Allocatable.ScalarResources = scalarResources
			snapshot.SetNode(nodeInfo)

			// Run a scheduler cycle to check whether the Pod can be scheduled on the Node
			podFits := p.podFitsNode(ctx, *nodeInfo, pod)

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
			if err := snapshot.AddPod(n.Name, pod); err != nil {
				return res, err
			}
			snapshot.Commit()
			res[n.Name] = fromMigNodeToNodePartitioning(n)
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
	return res, nil
}

// getUpdatedScalarResources returns the scalar resources of the nodeInfo provided as argument updated for
// matching the MIG resources defied by the specified mig.Node
func getUpdatedScalarResources(nodeInfo framework.NodeInfo, node mig.Node) map[v1.ResourceName]int64 {
	res := make(map[v1.ResourceName]int64)

	// Set all non-MIG scalar resources
	for r, v := range nodeInfo.Allocatable.ScalarResources {
		if !mig.IsNvidiaMigDevice(r) {
			res[r] = v
		}
	}
	// Set MIG scalar resources
	for r, v := range node.GetGeometry().AsResources() {
		res[r] = int64(v)
	}

	return res
}

// getLackingMigProfile returns (if any) the MIG profile requested by the Pod but currently not
// available in the ClusterSnapshot.
//
// As described in "Supporting MIG GPUs in Kubernetes" document, it is assumed that
// Pods request only one MIG device per time and with quantity 1, according to the
// idea that users should ask for a larger, single instance as opposed to multiple
// smaller instances.
func (p Planner) getLackingMigProfile(snapshot state.ClusterSnapshot, pod v1.Pod) (mig.ProfileName, bool) {
	for r := range snapshot.GetLackingResources(pod).ScalarResources {
		if mig.IsNvidiaMigDevice(r) {
			profileName, _ := mig.ExtractMigProfile(r)
			return profileName, true
		}
	}
	return "", false
}

// getCandidateNodes returns the Nodes of the ClusterSnapshot with free (e.g. not allocated) MIG resources
// candidate for a MIG geometry updated aimed to schedule a pending pod
func (p Planner) getCandidateNodes(snapshot state.ClusterSnapshot) []mig.Node {
	result := make([]mig.Node, 0)

	var migNode mig.Node
	var err error

	for k, n := range snapshot.GetNodes() {
		if migNode, err = mig.NewNode(*n.Node()); err != nil {
			p.logger.Error(
				err,
				"unable to create MIG node",
				"node",
				k,
			)
			continue
		}
		if migNode.HasFreeMigResources() {
			result = append(result, migNode)
		}
	}

	return result
}

func (p Planner) getPartitioningState(snapshot state.ClusterSnapshot) state.PartitioningState {
	migNodes := make([]mig.Node, 0)
	for k, v := range snapshot.GetNodes() {
		node, err := mig.NewNode(*v.Node())
		if err != nil {
			p.logger.Error(err, "unable to create MIG node", "node", k)
			continue
		}
		migNodes = append(migNodes, node)
	}
	return fromMigNodesToPartitioningState(migNodes)
}

func fromMigNodesToPartitioningState(nodes []mig.Node) state.PartitioningState {
	res := make(map[string]state.NodePartitioning)
	for _, node := range nodes {
		res[node.Name] = fromMigNodeToNodePartitioning(node)
	}
	return res
}

func fromMigNodeToNodePartitioning(node mig.Node) state.NodePartitioning {
	gpuPartitioning := make([]state.GPUPartitioning, 0)
	for _, gpu := range node.GPUs {
		gp := state.GPUPartitioning{
			GPUIndex:  gpu.GetIndex(),
			Resources: gpu.GetGeometry().AsResources(),
		}
		gpuPartitioning = append(gpuPartitioning, gp)
	}
	return state.NodePartitioning{GPUs: gpuPartitioning}
}

func (p Planner) podFitsNode(ctx context.Context, node framework.NodeInfo, pod v1.Pod) bool {
	cycleState := framework.NewCycleState()
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false
	}
	return p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge().IsSuccess()
}
