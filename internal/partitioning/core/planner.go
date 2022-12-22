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

package core

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

type PartitioningPlan struct {
	DesiredState state.PartitioningState
	id           string
}

func NewPartitioningPlan(s state.PartitioningState) PartitioningPlan {
	safeId := strings.NewReplacer(
		" ", "-",
		":", "-",
		"+", "-",
	).Replace(time.Now().UTC().String())
	return PartitioningPlan{
		DesiredState: s,
		id:           safeId,
	}
}

func (p PartitioningPlan) GetId() string {
	return p.id
}

type planner struct {
	sliceCalculator    gpu.SliceCalculator
	schedulerFramework framework.Framework
	partitioner        PartitionCalculator
	sorter             Sorter
}

func NewPlanner(partitioner PartitionCalculator, sliceCalculator gpu.SliceCalculator, schedulerFramework framework.Framework) Planner {
	return planner{
		partitioner:        partitioner,
		sliceCalculator:    sliceCalculator,
		schedulerFramework: schedulerFramework,
		sorter:             NewPodSorter(sliceCalculator),
	}
}

func (p planner) Plan(ctx context.Context, snapshot Snapshot, candidatePods []v1.Pod) (PartitioningPlan, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("planning desired GPU partitioning", "candidatePods", len(candidatePods))
	var err error

	partitioningState := snapshot.GetPartitioningState()
	tracker := NewSliceTracker(
		snapshot,
		p.sliceCalculator,
		candidatePods,
	)

	// No lacking slices, nothing to do
	if len(tracker.GetLackingSlices()) == 0 {
		logger.V(1).Info("no lacking profiles, nothing to do")
		return NewPartitioningPlan(partitioningState), nil
	}

	// Sort candidate pods
	sortedCandidatePods := p.sorter.Sort(candidatePods)

	// Get candidate nodes
	candidateNodes := snapshot.GetCandidateNodes()
	logger.V(1).Info(fmt.Sprintf("found %d candidate nodes", len(candidateNodes)))

	for _, n := range candidateNodes {
		// If there are no more lacking slices we can stop
		lackingSlices := tracker.GetLackingSlices()
		if len(lackingSlices) == 0 {
			return NewPartitioningPlan(partitioningState), nil
		}

		// Fork the state
		if err = snapshot.Fork(); err != nil {
			return PartitioningPlan{}, fmt.Errorf("error forking snapshot, this should never happen: %v", err)
		}

		// Try to update geometry
		nodeGeometryUpdated, err := n.UpdateGeometryFor(tracker.GetLackingSlices())
		if err != nil {
			return PartitioningPlan{}, err
		}
		if nodeGeometryUpdated {
			logger.V(1).Info("updated node geometry", "node", n.GetName(), "geometry", n.Geometry())
			snapshot.SetNode(n)
		}

		// Try to add candidate pods to the node with the updated geometry
		var addedPods int
		for _, pod := range sortedCandidatePods {
			if added := p.tryAddPod(ctx, pod, n.GetName(), snapshot); !added {
				logger.V(1).Info(
					"pod does not fit node",
					"namespace",
					pod.Namespace,
					"pod",
					pod.Name,
					"node",
					n.GetName,
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
				n.GetName,
			)
			partitioningState[n.GetName()] = p.partitioner.GetPartitioning(n)
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

	return NewPartitioningPlan(partitioningState), nil
}

func (p planner) tryAddPod(ctx context.Context, pod v1.Pod, nodeName string, snapshot Snapshot) bool {
	// First we check if there are any lacking slices,
	// if so we avoid running a scheduler cycle
	// since we already know that it is going to fail
	if len(snapshot.GetLackingSlices(pod)) > 0 {
		return false
	}
	// Simulate scheduling
	nodeInfo, ok := snapshot.GetNode(nodeName)
	if !ok {
		return false
	}
	if !p.canSchedulePod(ctx, pod, nodeInfo.NodeInfo()) {
		return false
	}
	// Add Pod to snapshot
	if err := snapshot.AddPod(nodeName, pod); err != nil {
		return false
	}
	return true
}

// canSchedulePod runs a scheduler cycle to check whether the Pod can be scheduled on the specified Node
func (p planner) canSchedulePod(ctx context.Context, pod v1.Pod, node framework.NodeInfo) bool {
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
