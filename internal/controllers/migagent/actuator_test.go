package migagent

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/migagent/plan"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	migtest "github.com/nebuly-ai/nebulnetes/pkg/test/mocks/mig"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMigActuator_applyDeleteOp(t *testing.T) {
	testCases := []struct {
		name                string
		op                  plan.DeleteOperation
		clientReturnedError gpu.Error

		expectedDeleteCalls uint
		errorExpected       bool
		restartExpected     bool
	}{
		{
			name: "Empty delete operation",
			op: plan.DeleteOperation{
				Resources: make(mig.DeviceResourceList, 0),
			},
			clientReturnedError: nil,
			expectedDeleteCalls: 0,
			errorExpected:       false,
			restartExpected:     false,
		},
		{
			name: "Delete op success with multiple resources",
			op: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-1",
							Status:       resource.StatusUnknown,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-2",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-3",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			clientReturnedError: nil,
			expectedDeleteCalls: 1,
			errorExpected:       false,
			restartExpected:     true,
		},
		{
			name: "MIG client returns error",
			op: plan.DeleteOperation{
				Resources: mig.DeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			clientReturnedError: gpu.GenericError.Errorf("an error"),
			expectedDeleteCalls: 1,
			errorExpected:       true,
			restartExpected:     false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var migClient = migtest.Client{}
			var actuator = MigActuator{migClient: &migClient}
			migClient.ReturnedError = tt.clientReturnedError
			status := actuator.applyDeleteOp(context.Background(), tt.op)
			if tt.errorExpected {
				assert.Error(t, status.Err)
			}
			if !tt.errorExpected {
				assert.NoError(t, status.Err)
			}
			assert.Equal(t, tt.restartExpected, status.PluginRestartRequired)
			assert.Equal(t, tt.expectedDeleteCalls, migClient.NumCallsDeleteMigResource)
		})
	}
}

//func TestMigActuator_applyCreateOps(t *testing.T) {
//	testCases := []struct {
//		name                string
//		ops                 plan.CreateOperationList
//		clientReturnedError gpu.Error
//
//		expectedCreateCalls uint
//		errorExpected       bool
//		restartExpected     bool
//	}{
//		{
//			name:                "Empty list",
//			ops:                 plan.CreateOperationList{},
//			clientReturnedError: nil,
//			expectedCreateCalls: 0,
//			errorExpected:       false,
//			restartExpected:     false,
//		},
//		{
//			name: "Empty create operation",
//			ops: plan.CreateOperationList{
//				{
//					MigProfile: mig.Profile{
//						GpuIndex: 0,
//						Name:     "1g.10gb",
//					},
//					Quantity: 0,
//				},
//			},
//			clientReturnedError: nil,
//			expectedCreateCalls: 0,
//			errorExpected:       false,
//			restartExpected:     false,
//		},
//		{
//			name: "MIG client returns error",
//			op: plan.CreateOperation{
//				MigProfile: mig.Profile{
//					GpuIndex: 0,
//					Name:     "1g.10gb",
//				},
//				Quantity: 1,
//			},
//			clientReturnedError: gpu.GenericError.Errorf("an error"),
//			expectedCreateCalls: 1,
//			errorExpected:       true,
//			restartExpected:     false,
//		},
//		{
//			name: "Create success, quantity > 1",
//			op: plan.CreateOperation{
//				MigProfile: mig.Profile{
//					GpuIndex: 0,
//					Name:     "1g.10gb",
//				},
//				Quantity: 4,
//			},
//			clientReturnedError: nil,
//			expectedCreateCalls: 4,
//			errorExpected:       false,
//			restartExpected:     true,
//		},
//	}
//
//	var migClient = migtest.Client{}
//	var actuator = MigActuator{migClient: &migClient}
//
//	for _, tt := range testCases {
//		migClient.Reset()
//		migClient.ReturnedError = tt.clientReturnedError
//		t.Run(tt.name, func(t *testing.T) {
//			status := actuator.applyCreateOps(context.TODO(), tt.op)
//			if tt.errorExpected {
//				assert.Error(t, status.Err)
//			}
//			if !tt.errorExpected {
//				assert.NoError(t, status.Err)
//			}
//			assert.Equal(t, tt.restartExpected, status.PluginRestartRequired)
//			assert.Equal(t, tt.expectedCreateCalls, migClient.NumCallsCreateMigResources)
//		})
//	}
//}
