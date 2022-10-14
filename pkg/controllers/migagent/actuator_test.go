package migagent

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/migagent/types"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	migtest "github.com/nebuly-ai/nebulnetes/pkg/test/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMigActuator_applyDeleteOp(t *testing.T) {
	testCases := []struct {
		name                string
		op                  types.DeleteOperation
		clientReturnedError error

		expectedDeleteCalls uint
		errorExpected       bool
	}{
		{
			name: "Empty delete operation",
			op: types.DeleteOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Resources: make(migtypes.MigDeviceResourceList, 0),
				Quantity:  0,
			},
			clientReturnedError: nil,
			expectedDeleteCalls: 0,
			errorExpected:       false,
		},
		{
			name: "Delete op does not have enough candidates",
			op: types.DeleteOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Resources: migtypes.MigDeviceResourceList{
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
				Quantity: 2,
			},
			clientReturnedError: nil,
			expectedDeleteCalls: 1,
			errorExpected:       true,
		},
		{
			name: "More candidates than required, the op should delete only Quantity resources",
			op: types.DeleteOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Resources: migtypes.MigDeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-2",
							Status:       resource.StatusFree,
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
				Quantity: 2,
			},
			clientReturnedError: nil,
			expectedDeleteCalls: 2,
			errorExpected:       false,
		},
		{
			name: "MIG client returns error",
			op: types.DeleteOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Resources: migtypes.MigDeviceResourceList{
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "uid-1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
				Quantity: 1,
			},
			clientReturnedError: fmt.Errorf("an error"),
			expectedDeleteCalls: 1,
			errorExpected:       true,
		},
	}

	var migClient = migtest.MockedMigClient{}
	var actuator = MigActuator{migClient: &migClient}

	for _, tt := range testCases {
		migClient.Reset()
		migClient.ReturnedError = tt.clientReturnedError
		t.Run(tt.name, func(t *testing.T) {
			err := actuator.applyDeleteOp(context.TODO(), tt.op)
			if tt.errorExpected {
				assert.Error(t, err)
			}
			if !tt.errorExpected {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedDeleteCalls, migClient.NumCallsDeleteMigResource)
		})
	}
}

func TestMigActuator_applyCreateOp(t *testing.T) {
	testCases := []struct {
		name                string
		op                  types.CreateOperation
		clientReturnedError error

		expectedCreateCalls uint
		errorExpected       bool
	}{
		{
			name: "Empty create operation",
			op: types.CreateOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Quantity: 0,
			},
			clientReturnedError: nil,
			expectedCreateCalls: 0,
			errorExpected:       false,
		},
		{
			name: "MIG client returns error",
			op: types.CreateOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Quantity: 1,
			},
			clientReturnedError: fmt.Errorf("an error"),
			expectedCreateCalls: 1,
			errorExpected:       true,
		},
		{
			name: "Create success, quantity > 1",
			op: types.CreateOperation{
				MigProfile: migtypes.MigProfile{
					GpuIndex: 0,
					Name:     "1g.10gb",
				},
				Quantity: 4,
			},
			clientReturnedError: nil,
			expectedCreateCalls: 4,
			errorExpected:       false,
		},
	}

	var migClient = migtest.MockedMigClient{}
	var actuator = MigActuator{migClient: &migClient}

	for _, tt := range testCases {
		migClient.Reset()
		migClient.ReturnedError = tt.clientReturnedError
		t.Run(tt.name, func(t *testing.T) {
			err := actuator.applyCreateOp(context.TODO(), tt.op)
			if tt.errorExpected {
				assert.Error(t, err)
			}
			if !tt.errorExpected {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCreateCalls, migClient.NumCallsCreateMigResource)
		})
	}
}
