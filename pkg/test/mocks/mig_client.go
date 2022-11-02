package mocks

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"sync"
)

// Todo: use some tool for auto-generating mocks
type MockedMigClient struct {
	NumCallsDeleteMigResource     uint
	NumCallsCreateMigResource     uint
	NumCallsGetMigDeviceResources uint

	ReturnedMigDeviceResources mig.DeviceResourceList
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

func (m *MockedMigClient) GetMigDeviceResources(_ context.Context) (mig.DeviceResourceList, error) {
	m.lockGetMigDeviceResources.Lock()
	defer m.lockGetMigDeviceResources.Unlock()
	m.NumCallsGetMigDeviceResources++
	return m.ReturnedMigDeviceResources, m.ReturnedError
}

func (m *MockedMigClient) CreateMigResource(_ context.Context, _ mig.Profile) error {
	m.lockCreateMigResource.Lock()
	defer m.lockCreateMigResource.Unlock()
	m.NumCallsCreateMigResource++
	return m.ReturnedError
}

func (m *MockedMigClient) DeleteMigResource(_ context.Context, _ mig.DeviceResource) error {
	m.lockDeleteMigResource.Lock()
	defer m.lockDeleteMigResource.Unlock()
	m.NumCallsDeleteMigResource++
	return m.ReturnedError
}

func (m *MockedMigClient) GetUsedMigDeviceResources(ctx context.Context) (mig.DeviceResourceList, error) {
	return mig.DeviceResourceList{}, m.ReturnedError
}

func (m *MockedMigClient) GetAllocatableMigDeviceResources(ctx context.Context) (mig.DeviceResourceList, error) {
	return mig.DeviceResourceList{}, m.ReturnedError
}
