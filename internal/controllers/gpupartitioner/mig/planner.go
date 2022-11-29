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

	//// Compute all lacking MIG profiles
	//lackingMigProfiles := make(map[mig.ProfileName]int)
	//for _, pod := range candidates {
	//	for profile, quantity := range snapshot.GetLackingMigProfiles(pod) {
	//		lackingMigProfiles[profile] += quantity
	//	}
	//}
	//
	//// No lacking MIG profiles, nothing to do
	//if len(lackingMigProfiles) == 0 {
	//	logger.V(1).Info("no lacking MIG profiles, nothing to do")
	//	return partitioningState, nil
	//}

	// Sort candidates
	candidates = SortCandidatePods(candidates)
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
				logger.V(1).Info(
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
			lackingMigProfiles := snapshot.GetLackingMigProfiles(pod) // TODO: we should the lacking MIG resources of all the Pods for more effective partitioning
			if len(lackingMigProfiles) == 0 {
				logger.V(1).Info(
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

			// Try to update the node MIG geometry: if the node geometry can't be updated,
			// revert the state and move on to next node
			if updated := n.UpdateGeometryFor(lackingMigProfiles); !updated {
				snapshot.Revert()
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
				logger.V(1).Info(
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
			logger.V(1).Info("pod does not fit node, reverting changes", "namespace", pod.Namespace, "pod", pod.Name, "node", n.Name)
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
