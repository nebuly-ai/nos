package migstate

import (
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
)

func FromMigNodeToNodePartitioning(node mig.Node) state.NodePartitioning {
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
