package mig

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig/migstate"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
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

func (a Actuator) Apply(ctx context.Context, s state.ClusterSnapshot, desiredState state.PartitioningState) error {
	var err error
	var snapshot migstate.MigClusterSnapshot
	logger := log.FromContext(ctx)
	logger.Info("applying desired MIG partitioning")

	if snapshot, err = migstate.NewClusterSnapshot(s); err != nil {
		return fmt.Errorf("error initializing MIG cluster snapshot: %v", err)
	}
	if snapshot.GetPartitioningState().Equal(desiredState) {
		logger.Info("current and desired partitioning states are equal, nothing to do")
		return nil
	}
	if desiredState.IsEmpty() {
		logger.Info("desired partitioning state is empty, nothing to do")
	}

	for node, partitioningState := range desiredState {
		logger.V(1).Info("applying node partitioning", "node", node, "partitioning", partitioningState)
		if err = a.applyNodePartitioning(ctx, node, partitioningState); err != nil {
			return fmt.Errorf("error partitioning node %s: %v", node, err)
		}
	}

	return nil
}

func (a Actuator) applyNodePartitioning(ctx context.Context, nodeName string, partitioning state.NodePartitioning) error {
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
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUSpecPrefix) {
			delete(node.Annotations, k)
		}
	}
	for _, annotation := range gpuSpecAnnotationList {
		node.Annotations[annotation.Name] = annotation.GetValue()
	}
	if err = a.Patch(ctx, &node, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("error patching node annotations: %v", err)
	}
	logger.V(1).Info("patched node annotations", "node", nodeName, "GPUSpecAnnotations", gpuSpecAnnotationList)

	return nil
}

func getGPUSpecAnnotationList(nodePartitioning state.NodePartitioning) (mig.GPUSpecAnnotationList, error) {
	res := make(mig.GPUSpecAnnotationList, 0)
	for _, gpu := range nodePartitioning.GPUs {
		for r, q := range gpu.Resources {
			migProfile, err := mig.ExtractMigProfile(r)
			if err != nil {
				return res, err
			}
			annotation := mig.NewGpuSpecAnnotation(gpu.GPUIndex, migProfile, q)
			res = append(res, annotation)
		}
	}
	return res, nil
}
