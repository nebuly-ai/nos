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
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

type Actuator struct {
	client.Client
}

func NewActuator(client client.Client) Actuator {
	return Actuator{
		Client: client,
	}
}

func (a Actuator) Apply(ctx context.Context, snapshot core.Snapshot, plan core.PartitioningPlan) (bool, error) {
	var err error
	logger := log.FromContext(ctx)
	logger.Info("applying desired MIG partitioning")

	if snapshot.GetPartitioningState().Equal(plan.DesiredState) {
		logger.Info("current and desired partitioning states are equal, nothing to do")
		return false, nil
	}
	if plan.DesiredState.IsEmpty() {
		logger.Info("desired partitioning state is empty, nothing to do")
		return false, nil
	}

	for node, partitioningState := range plan.DesiredState {
		logger.Info("updating node", "node", node, "partitioning", partitioningState)
		if err = a.applyNodePartitioning(ctx, node, plan.GetId(), partitioningState); err != nil {
			return false, fmt.Errorf("error partitioning node %s: %v", node, err)
		}
	}

	logger.Info("plan applied")

	return true, nil
}

func (a Actuator) applyNodePartitioning(ctx context.Context, nodeName, planId string, partitioning state.NodePartitioning) error {
	var err error
	logger := log.FromContext(ctx)

	// Compute GPU spec annotations
	gpuSpecAnnotationList, err := getGPUSpecAnnotationList(partitioning)
	if err != nil {
		return err
	}

	// Fetch Node
	var node v1.Node
	if err = a.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return err
	}

	// Update node annotations
	original := node.DeepCopy()
	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}
	for k := range node.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGpuSpecPrefix) {
			delete(node.Annotations, k)
		}
	}
	for _, annotation := range gpuSpecAnnotationList {
		node.Annotations[annotation.String()] = annotation.GetValue()
	}
	node.Annotations[v1alpha1.AnnotationPartitioningPlan] = planId

	if err = a.Patch(ctx, &node, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("error patching node annotations: %v", err)
	}
	logger.V(1).Info("patched node annotations", "node", nodeName, "GPUSpecAnnotations", gpuSpecAnnotationList)

	return nil
}

func getGPUSpecAnnotationList(nodePartitioning state.NodePartitioning) (gpu.SpecAnnotationList, error) {
	res := make(gpu.SpecAnnotationList, 0)
	for _, g := range nodePartitioning.GPUs {
		for r, q := range g.Resources {
			migProfile, err := mig.ExtractProfileName(r)
			if err != nil {
				return res, err
			}
			annotation := gpu.SpecAnnotation{
				ProfileName: migProfile.String(),
				Index:       g.GPUIndex,
				Quantity:    q,
			}
			res = append(res, annotation)
		}
	}
	return res, nil
}
