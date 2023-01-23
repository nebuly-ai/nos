/*
 * Copyright 2023 nebuly.com.
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
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type actuator struct {
	Partitioner
	client.Client
}

func NewActuator(client client.Client, partitioner Partitioner) Actuator {
	return actuator{
		Client:      client,
		Partitioner: partitioner,
	}
}

func (a actuator) Apply(ctx context.Context, snapshot Snapshot, plan PartitioningPlan) (bool, error) {
	var err error
	logger := log.FromContext(ctx)
	logger.Info("applying desired partitioning")

	if snapshot.GetPartitioningState().Equal(plan.DesiredState) {
		logger.Info("current and desired partitioning states are equal, nothing to do")
		return false, nil
	}
	if plan.DesiredState.IsEmpty() {
		logger.Info("desired partitioning state is empty, nothing to do")
		return false, nil
	}

	for nodeName, partitioningState := range plan.DesiredState {
		node := v1.Node{}
		if err := a.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
			return false, fmt.Errorf("failed to get node %s: %w", nodeName, err)
		}
		logger.Info("partitioning node", "node", node.Name, "partitioning", partitioningState)
		if err = a.ApplyPartitioning(ctx, node, plan.GetId(), partitioningState); err != nil {
			return false, fmt.Errorf("error partitioning node %s: %w", nodeName, err)
		}
	}
	logger.Info("plan applied")

	return true, nil
}
