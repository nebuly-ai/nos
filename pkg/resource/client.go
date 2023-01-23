/*
 * Copyright 2023 nebuly.com.
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

package resource

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
)

type Client interface {
	GetAllocatableDevices(ctx context.Context) ([]Device, error)
	GetUsedDevices(ctx context.Context) ([]Device, error)
}

type clientImpl struct {
	lister pdrv1.PodResourcesListerClient
}

func NewClient(lister pdrv1.PodResourcesListerClient) Client {
	return &clientImpl{lister: lister}
}

func (c clientImpl) GetAllocatableDevices(ctx context.Context) ([]Device, error) {
	// Fetch resources
	resp, err := c.lister.GetAllocatableResources(ctx, &pdrv1.AllocatableResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("unable to get allocatable resources from Kubelet gRPC socket: %s", err)
	}

	// Convert resp to devices
	devices := make([]Device, 0)
	for _, d := range resp.GetDevices() {
		for _, deviceId := range d.DeviceIds {
			device := Device{
				ResourceName: v1.ResourceName(d.GetResourceName()),
				DeviceId:     deviceId,
				Status:       StatusUnknown,
			}
			devices = append(devices, device)
		}
	}

	return devices, nil
}

func (c clientImpl) GetUsedDevices(ctx context.Context) ([]Device, error) {
	// List Pods Resources
	listResp, err := c.lister.List(ctx, &pdrv1.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("unable to list resources used by running Pods from Kubelet gRPC socket: %s", err)
	}

	// Convert resp to devices
	result := make([]Device, 0)
	for _, r := range listResp.PodResources {
		for _, cr := range r.Containers {
			for _, cd := range cr.GetDevices() {
				for _, cdId := range cd.DeviceIds {
					device := Device{
						ResourceName: v1.ResourceName(cd.GetResourceName()),
						DeviceId:     cdId,
						Status:       StatusUsed,
					}
					result = append(result, device)
				}
			}
		}
	}

	return result, nil
}
