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

package gpu

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nos/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type Client interface {
	GetDevices(ctx context.Context) (DeviceList, Error)

	GetUsedDevices(ctx context.Context) (DeviceList, Error)

	GetAllocatableDevices(ctx context.Context) (DeviceList, Error)
}

type DevicePluginClient interface {
	// Restart restarts the NVIDIA device plugin pod on the specified node, waiting until the
	// pod is again in state "Running" or the timeout is reached.
	Restart(ctx context.Context, nodeName string, timeout time.Duration) error
}

func NewDevicePluginClient(k8sClient client.Client) DevicePluginClient {
	return devicePluginClient{Client: k8sClient}
}

type devicePluginClient struct {
	client.Client
}

func (d devicePluginClient) Restart(ctx context.Context, nodeName string, timeout time.Duration) error {
	logger := log.FromContext(ctx)

	// Get pod
	var podList v1.PodList
	if err := d.List(
		ctx,
		&podList,
		client.MatchingLabels{"app": "nvidia-device-plugin-daemonset"},
		client.MatchingFields{constant.PodNodeNameKey: nodeName},
	); err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		return fmt.Errorf(
			"error getting nvidia device plugin pod on node %s: expected exactly 1 but got %d",
			nodeName,
			len(podList.Items),
		)
	}
	// Delete pod
	logger.V(1).Info(
		"deleting NVIDIA device plugin Pod",
		"pod",
		podList.Items[0].Name,
		"namespace",
		podList.Items[0].Namespace,
	)
	if err := d.Delete(ctx, &podList.Items[0]); err != nil {
		return fmt.Errorf("error deleting nvidia device plugin pod: %s", err.Error())
	}
	// Wait until the Pods gets recreated
	return d.WaitUntilRunning(ctx, nodeName, podList.Items[0].Name, timeout)
}

func (d devicePluginClient) WaitUntilRunning(ctx context.Context, nodeName string, oldPodName string, timeout time.Duration) error {
	logger := log.FromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var podList v1.PodList
	checkPodRecreated := func() (bool, error) {
		if err := d.List(
			ctx,
			&podList,
			client.MatchingLabels{"app": "nvidia-device-plugin-daemonset"},
			client.MatchingFields{constant.PodNodeNameKey: nodeName},
		); err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}
		pod := podList.Items[0]
		if pod.Name == oldPodName {
			return false, nil
		}
		if pod.DeletionTimestamp != nil {
			return false, nil
		}
		if pod.Status.Phase != v1.PodRunning {
			return false, nil
		}
		return true, nil
	}

	for {
		logger.V(1).Info("waiting for NVIDIA device plugin Pod to be recreated")
		recreated, err := checkPodRecreated()
		if err != nil {
			return err
		}
		if recreated {
			logger.V(1).Info("NVIDIA device plugin Pod recreated")
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("error waiting for NVIDIA device plugin Pod on node %s: timeout", nodeName)
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}
