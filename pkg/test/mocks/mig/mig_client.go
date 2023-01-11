/*
 * Copyright 2023 Nebuly.ai.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mig

import (
	"context"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"sync"
)

// Todo: use some tool for auto-generating mocks
type Client struct {
	NumCallsDeleteMigResource     uint
	NumCallsCreateMigResources    uint
	NumCallsGetMigDeviceResources uint

	ReturnedMigDeviceResources gpu.DeviceList
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

func (m *Client) GetMigDevices(_ context.Context) (gpu.DeviceList, gpu.Error) {
	m.lockGetMigDeviceResources.Lock()
	defer m.lockGetMigDeviceResources.Unlock()
	m.NumCallsGetMigDeviceResources++
	return m.ReturnedMigDeviceResources, m.ReturnedError
}

func (m *Client) CreateMigDevices(_ context.Context, _ mig.ProfileList) (mig.ProfileList, error) {
	m.lockCreateMigResource.Lock()
	defer m.lockCreateMigResource.Unlock()
	m.NumCallsCreateMigResources++
	return nil, m.ReturnedError
}

func (m *Client) DeleteMigDevice(_ context.Context, _ gpu.Device) gpu.Error {
	m.lockDeleteMigResource.Lock()
	defer m.lockDeleteMigResource.Unlock()
	m.NumCallsDeleteMigResource++
	return m.ReturnedError
}

func (m *Client) GetUsedMigDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	return gpu.DeviceList{}, m.ReturnedError
}

func (m *Client) GetAllocatableMigDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	return gpu.DeviceList{}, m.ReturnedError
}

func (m *Client) DeleteAllExcept(_ context.Context, resources gpu.DeviceList) error {
	return m.ReturnedError
}
