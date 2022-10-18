package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"sync"
)

type MockedMigClient struct {
	NumCallsDeleteMigResource     uint
	NumCallsCreateMigResource     uint
	NumCallsGetMigDeviceResources uint

	ReturnedMigDeviceResources types.MigDeviceResourceList
	ReturnedError              error

	lockReset                 sync.Mutex
	lockGetMigDeviceResources sync.Mutex
	lockCreateMigResource     sync.Mutex
	lockDeleteMigResource     sync.Mutex
}

func (m *MockedMigClient) Reset() {
	m.lockReset.Lock()
	defer m.lockReset.Unlock()
	m.NumCallsDeleteMigResource = 0
	m.NumCallsCreateMigResource = 0
	m.NumCallsGetMigDeviceResources = 0
}

func (m *MockedMigClient) GetMigDeviceResources(_ context.Context) (types.MigDeviceResourceList, error) {
	m.lockGetMigDeviceResources.Lock()
	defer m.lockGetMigDeviceResources.Unlock()
	m.NumCallsGetMigDeviceResources++
	return m.ReturnedMigDeviceResources, m.ReturnedError
}

func (m *MockedMigClient) CreateMigResource(_ context.Context, _ types.MigProfile) error {
	m.lockCreateMigResource.Lock()
	defer m.lockCreateMigResource.Unlock()
	m.NumCallsCreateMigResource++
	return m.ReturnedError
}

func (m *MockedMigClient) DeleteMigResource(_ context.Context, _ types.MigDeviceResource) error {
	m.lockDeleteMigResource.Lock()
	defer m.lockDeleteMigResource.Unlock()
	m.NumCallsDeleteMigResource++
	return m.ReturnedError
}
