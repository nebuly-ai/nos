/*
 * Copyright 2023 nebuly.com
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
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type nodeInitializer struct {
	partitioner         core.Partitioner
	partitionCalculator core.PartitionCalculator
}

func NewNodeInitializer(client client.Client) core.NodeInitializer {
	p := NewPartitioner(client)
	return nodeInitializer{
		partitioner:         p,
		partitionCalculator: NewPartitionCalculator(),
	}
}

func (n nodeInitializer) InitNodePartitioning(ctx context.Context, node v1.Node) error {
	logger := log.FromContext(ctx)

	if !gpu.IsMigPartitioningEnabled(node) {
		return fmt.Errorf("MIG partitioning is not enabled on node %s", node.Name)
	}

	// Initialize node GPUs
	nodeInfo := framework.NewNodeInfo()
	nodeInfo.SetNode(&node)
	migNode, err := mig.NewNode(*nodeInfo)
	if err != nil {
		return err
	}
	var initializedGPUs int
	for _, g := range migNode.GPUs {
		if len(g.GetGeometry()) > 0 {
			continue
		}
		logger.Info("initializing MIG geometry", "node", node.Name, "gpu", g.GetIndex())
		if err = g.InitGeometry(); err != nil {
			return fmt.Errorf("error initializing GPU geometry: %v", err)
		}
		initializedGPUs++
	}

	// No GPUs were initialized, nothing to do
	if initializedGPUs == 0 {
		logger.V(1).Info("all MIG GPUs are already initialized", "node", node.Name)
		return nil
	}

	// Apply new partitioning
	nodePartitioning := n.partitionCalculator.GetPartitioning(&migNode)
	logger.Info("applying partitioning", "node", node.Name, "partitioning", nodePartitioning)
	if err = n.partitioner.ApplyPartitioning(ctx, node, core.NewPartitioningPlanId(), nodePartitioning); err != nil {
		return fmt.Errorf("error applying partitioning: %v", err)
	}
	return nil
}
