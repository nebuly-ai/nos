package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
)

type MockedMigClient struct {
	NumCallsDeleteMigResource     uint
	NumCallsCreateMigResource     uint
	NumCallsGetMigDeviceResources uint

	ReturnedMigDeviceResources types.MigDeviceResourceList
	ReturnedError              error
	ReturnedMigDeviceResource  types.MigDeviceResource
}

func (m *MockedMigClient) Reset() {
	m.NumCallsDeleteMigResource = 0
	m.NumCallsCreateMigResource = 0
	m.NumCallsGetMigDeviceResources = 0
}

func (m *MockedMigClient) GetMigDeviceResources(_ context.Context) (types.MigDeviceResourceList, error) {
	m.NumCallsGetMigDeviceResources++
	return m.ReturnedMigDeviceResources, m.ReturnedError
}

func (m *MockedMigClient) CreateMigResource(_ context.Context, _ types.MigProfile) error {
	m.NumCallsCreateMigResource++
	return m.ReturnedError
}

func (m *MockedMigClient) DeleteMigResource(_ context.Context, _ types.MigDeviceResource) error {
	m.NumCallsDeleteMigResource++
	return m.ReturnedError
}
