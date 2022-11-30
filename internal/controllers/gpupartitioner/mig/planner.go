/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
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

func (p Planner) Plan(ctx context.Context, s state.ClusterSnapshot, candidatePods []v1.Pod) (core.PartitioningPlan, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("planning desired GPU partitioning", "candidatePods", len(candidatePods))
	var err error
	var snapshot migstate.MigClusterSnapshot
	if snapshot, err = migstate.NewClusterSnapshot(s); err != nil {
		return core.PartitioningPlan{}, fmt.Errorf("error initializing MIG cluster snapshot: %v", err)
	}

	partitioningState := snapshot.GetPartitioningState()
	tracker := newLackingMigProfilesTracker(snapshot, candidatePods)

	// No lacking MIG profiles, nothing to do
	if len(tracker.GetLackingMigProfiles()) == 0 {
		logger.V(1).Info("no lacking MIG profiles, nothing to do")
		return core.NewPartitioningPlan(partitioningState), nil
	}

	// Sort candidate pods
	sortedCandidatePods := SortCandidatePods(candidatePods)

	// Get candidate nodes
	candidateNodes := snapshot.GetCandidateNodes()
	logger.V(1).Info(fmt.Sprintf("found %d candidate nodes", len(candidateNodes)))

	for _, n := range candidateNodes {
		// If there are no more lacking MIG profiles we can stop
		lackingMigProfiles := tracker.GetLackingMigProfiles()
		if len(lackingMigProfiles) == 0 {
			return core.NewPartitioningPlan(partitioningState), nil
		}

		// Get node info
		nodeInfo, ok := snapshot.GetNode(n.Name)
		if !ok {
			return core.PartitioningPlan{}, fmt.Errorf(
				"cluster snapshot is inconsistent: node %s not found, this should never happen",
				n.Name,
			)
		}

		// Fork the state
		if err = snapshot.Fork(); err != nil {
			return core.PartitioningPlan{}, fmt.Errorf("error forking snapshot, this should never happen: %v", err)
		}

		// Try to update MIG geometry
		nodeGeometryUpdated := n.UpdateGeometryFor(lackingMigProfiles)
		if nodeGeometryUpdated {
			logger.V(1).Info("updated node MIG geometry", "node", n.Name, "geometry", n.GetGeometry())
			if err = snapshot.SetNode(n); err != nil {
				return core.PartitioningPlan{}, err
			}
		}

		// Try to add candidate pods to the node with the updated geometry
		var addedPods int
		for _, pod := range sortedCandidatePods {
			if added := p.tryAddPod(ctx, pod, nodeInfo, &snapshot); !added {
				logger.V(1).Info(
					"pod does not fit node",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.Name,
				)
				continue
			}
			logger.V(1).Info(
				"pod fits node",
				"namespace",
				pod.Namespace,
				"pod",
				pod.Name,
				"node",
				n.Name,
			)
			partitioningState[n.Name] = migstate.FromMigNodeToNodePartitioning(n)
			tracker.Remove(pod)
			addedPods++
		}

		// If the new geometry allowed to add any pod then commit changes, otherwise revert
		if addedPods == 0 {
			snapshot.Revert()
		}
		if addedPods > 0 {
			snapshot.Commit()
		}
	}

	return core.NewPartitioningPlan(partitioningState), nil
}

func (p Planner) tryAddPod(ctx context.Context, pod v1.Pod, nodeInfo framework.NodeInfo, snapshot *migstate.MigClusterSnapshot) bool {
	// First we check if there are any lacking MIG profiles,
	// if so we avoid running a scheduler cycle
	// since we already know that it is going to fail
	if len(snapshot.GetLackingMigProfiles(pod)) > 0 {
		return false
	}
	// Simulate scheduling
	if !p.canSchedulePod(ctx, pod, nodeInfo) {
		return false
	}
	// Add Pod to snapshot
	if err := snapshot.AddPod(nodeInfo.Node().Name, pod); err != nil {
		return false
	}
	return true
}

// canSchedulePod runs a scheduler cycle to check whether the Pod can be scheduled on the specified Node
func (p Planner) canSchedulePod(ctx context.Context, pod v1.Pod, node framework.NodeInfo) bool {
	logger := log.FromContext(ctx)
	logger.V(1).Info("simulating pod scheduling", "pod", pod.Name, "namespace", pod.Namespace)
	cycleState := framework.NewCycleState()

	// Run PreFilter plugins
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	logger.V(1).Info(
		"scheduler PreFilter status",
		"statusCode",
		preFilterStatus.Code(),
		"status",
		preFilterStatus,
	)
	if !preFilterStatus.IsSuccess() {
		return false
	}

	// Run Filter plugins
	filterStatus := p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge()
	logger.V(1).Info(
		"scheduler Filter status",
		"statusCode",
		filterStatus.Code(),
		"status",
		filterStatus,
	)

	return filterStatus.IsSuccess()
}
