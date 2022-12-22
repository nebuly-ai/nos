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
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

var _ core.Partitioner = partitioner{}

func NewPartitioner(client client.Client) core.Partitioner {
	return partitioner{Client: client}
}

type partitioner struct {
	client.Client
}

func (p partitioner) ApplyPartitioning(ctx context.Context, node v1.Node, planId string, partitioning state.NodePartitioning) error {
	var err error
	logger := log.FromContext(ctx)

	// Compute GPU spec annotations
	gpuSpecAnnotationList, err := getGPUSpecAnnotationList(partitioning)
	if err != nil {
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

	// Patch node
	if err = p.Patch(ctx, &node, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("error patching node annotations: %v", err)
	}
	logger.V(1).Info("patched node annotations", "node", node.Name, "GPUSpecAnnotations", gpuSpecAnnotationList)

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
