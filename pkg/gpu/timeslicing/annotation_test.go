package timeslicing_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestAnnotationConversions(t *testing.T) {
	devices := gpu.DeviceList{
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-1",
				Status:       resource.StatusUsed,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-1",
				Status:       resource.StatusUsed,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-20gb",
				DeviceId:     "id-1",
				Status:       resource.StatusFree,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-2",
				Status:       resource.StatusFree,
			},
			GpuIndex: 1,
		},
	}

	// From devices to annotations
	timeSlicingAnnotations := timeslicing.ComputeStatusAnnotations(devices)
	stringAnnotations := make(map[string]string)
	for _, a := range timeSlicingAnnotations {
		stringAnnotations[a.String()] = a.GetValue()
	}

	// From annotations to devices
	node := v1.Node{}
	node.Annotations = stringAnnotations
	parsedStatusAnnotations, _ := timeslicing.ParseNodeAnnotations(node)

	// Check that the devices are the same
	assert.ElementsMatch(t, timeSlicingAnnotations, parsedStatusAnnotations)
}
