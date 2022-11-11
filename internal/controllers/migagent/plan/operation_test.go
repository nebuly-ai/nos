package plan_test

import (
	"github.com/nebuly-ai/nebulnetes/internal/controllers/migagent/plan"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteOperation__Equal(t *testing.T) {
	testCases := []struct {
		name     string
		deleteOp plan.DeleteOperation
		other    plan.DeleteOperation
		expected bool
	}{
		{
			name:     "Empty op",
			deleteOp: plan.DeleteOperation{},
			other:    plan.DeleteOperation{},
			expected: true,
		},
		{
			name: "Op are equals, different order",
			deleteOp: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile2g20gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			other: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: mig.Profile2g20gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			expected: true,
		},
		{
			name: "Op are *not* equals",
			deleteOp: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "3",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			other: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.deleteOp.Equal(tt.other)
			swappedRes := tt.other.Equal(tt.deleteOp)
			assert.Equal(t, res, swappedRes, "equal function is not symmetric")
			assert.Equal(t, tt.expected, res)
		})
	}
}
