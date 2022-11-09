package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"sync"
)

// Todo: use some tool for auto-generating mocks
type Client struct {
	NumCallsDeleteMigResource     uint
	NumCallsCreateMigResources    uint
	NumCallsGetMigDeviceResources uint

	ReturnedMigDeviceResources mig.DeviceResourceList
	ReturnedError              gpu.Error

	lockReset                 sync.Mutex
	lockGetMigDeviceResources sync.Mutex
	lockCreateMigResource     sync.Mutex
	lockDeleteMigResource     sync.Mutex
}

func (m *Client) Reset() {
	m.lockReset.Lock()
	defer m.lockReset.Unlock()
	m.NumCallsDeleteMigResource = 0
	m.NumCallsCreateMigResources = 0
	m.NumCallsGetMigDeviceResources = 0
}

func (m *Client) GetMigDeviceResources(_ context.Context) (mig.DeviceResourceList, gpu.Error) {
	m.lockGetMigDeviceResources.Lock()
	defer m.lockGetMigDeviceResources.Unlock()
	m.NumCallsGetMigDeviceResources++
	return m.ReturnedMigDeviceResources, m.ReturnedError
}

func (m *Client) CreateMigResources(_ context.Context, _ mig.ProfileList) (mig.ProfileList, gpu.Error) {
	m.lockCreateMigResource.Lock()
	defer m.lockCreateMigResource.Unlock()
	m.NumCallsCreateMigResources++
	return nil, m.ReturnedError
}

func (m *Client) DeleteMigResource(_ context.Context, _ mig.DeviceResource) gpu.Error {
	m.lockDeleteMigResource.Lock()
	defer m.lockDeleteMigResource.Unlock()
	m.NumCallsDeleteMigResource++
	return m.ReturnedError
}

func (m *Client) GetUsedMigDeviceResources(ctx context.Context) (mig.DeviceResourceList, gpu.Error) {
	return mig.DeviceResourceList{}, m.ReturnedError
}

func (m *Client) GetAllocatableMigDeviceResources(ctx context.Context) (mig.DeviceResourceList, gpu.Error) {
	return mig.DeviceResourceList{}, m.ReturnedError
}
