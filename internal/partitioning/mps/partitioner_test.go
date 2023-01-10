/*
 * Copyright 2022 Nebuly.ai
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

package mps_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/mps"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestToPluginConfig(t *testing.T) {
	t.Run("Empty node partitioning", func(t *testing.T) {
		nodePartitioning := state.NodePartitioning{GPUs: []state.GPUPartitioning{}}
		config := mps.ToPluginConfig(nodePartitioning)
		assert.Empty(t, config.Sharing.TimeSlicing.Resources)
	})

	t.Run("Multiple GPUs, multiple resources with replicas", func(t *testing.T) {
		nodePartitioning := state.NodePartitioning{
			GPUs: []state.GPUPartitioning{
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						"nvidia.com/gpu-10gb": 2,
						"nvidia.com/gpu-5gb":  2,
					},
				},
				{
					GPUIndex: 1,
					Resources: map[v1.ResourceName]int{
						"nvidia.com/gpu-1gb": 3,
						"nvidia.com/gpu-2gb": 2,
					},
				},
			},
		}
		config := mps.ToPluginConfig(nodePartitioning)
		assert.Len(t, config.Sharing.TimeSlicing.Resources, 4)
	})
}

func TestPartitioner__ApplyPartitioning(t *testing.T) {
	t.Run("Device Plugin ConfigMap does not exist - should create it", func(t *testing.T) {
		node := factory.BuildNode("node-1").Get()
		k8sClient := fake.NewClientBuilder().WithObjects(&node).Build()
		devicePluginCm := types.NamespacedName{
			Namespace: "test",
			Name:      "test-cm",
		}
		devicePluginClient := mocks.NewDevicePluginClient(t)
		devicePluginClient.On("Restart", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(nil)
		partitioner := mps.NewPartitioner(
			k8sClient,
			devicePluginCm,
			devicePluginClient,
		)
		ctx := context.Background()

		err := partitioner.ApplyPartitioning(ctx, node, "plan", state.NodePartitioning{})
		assert.NoError(t, err)

		cm := &v1.ConfigMap{}
		err = k8sClient.Get(ctx, devicePluginCm, cm)
		assert.NoError(t, err)
		assert.Equal(t, cm.Namespace, devicePluginCm.Namespace)
		assert.Equal(t, cm.Name, devicePluginCm.Name)
	})

	t.Run("Device Plugin ConfigMap exists but its data is nil - should init it", func(t *testing.T) {
		node := factory.BuildNode("node-1").Get()
		devicePluginCM := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "test-name",
			},
		}
		cmNamespacedName := types.NamespacedName{
			Namespace: devicePluginCM.Namespace,
			Name:      devicePluginCM.Name,
		}
		devicePluginClient := mocks.NewDevicePluginClient(t)
		devicePluginClient.On("Restart", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(nil)
		k8sClient := fake.NewClientBuilder().
			WithObjects(&node).
			WithObjects(&devicePluginCM).
			Build()
		partitioner := mps.NewPartitioner(
			k8sClient,
			cmNamespacedName,
			devicePluginClient,
		)
		ctx := context.Background()

		err := partitioner.ApplyPartitioning(ctx, node, "plan", state.NodePartitioning{})
		assert.NoError(t, err)

		cm := &v1.ConfigMap{}
		err = k8sClient.Get(ctx, cmNamespacedName, cm)
		assert.NoError(t, err)
		assert.Equal(t, cm.Namespace, cmNamespacedName.Namespace)
		assert.Equal(t, cm.Name, cmNamespacedName.Name)
		assert.NotNil(t, cm.Data)
	})

	t.Run("Error restarting NVIDIA device plugin", func(t *testing.T) {
		node := factory.BuildNode("node-1").Get()
		k8sClient := fake.NewClientBuilder().WithObjects(&node).Build()
		devicePluginCm := types.NamespacedName{
			Namespace: "test",
			Name:      "test-cm",
		}
		devicePluginClient := mocks.NewDevicePluginClient(t)
		devicePluginClient.On("Restart", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(errors.New(""))
		partitioner := mps.NewPartitioner(
			k8sClient,
			devicePluginCm,
			devicePluginClient,
		)
		ctx := context.Background()

		err := partitioner.ApplyPartitioning(ctx, node, "plan", state.NodePartitioning{})
		assert.Error(t, err)
	})

	t.Run("Updating partitioning should delete previous node configs from device plugin CM", func(t *testing.T) {
		node := factory.BuildNode("node-1").Get()
		devicePluginCM := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "test-name",
			},
			Data: map[string]string{
				fmt.Sprintf(mps.DevicePluginConfigKeyFormat, node.Name, "old-plan-1"): "old-config",
				fmt.Sprintf(mps.DevicePluginConfigKeyFormat, node.Name, "old-plan-2"): "old-config",
				fmt.Sprintf(mps.DevicePluginConfigKeyFormat, "node-2", "old-plan-2"):  "config",
			},
		}
		cmNamespacedName := types.NamespacedName{
			Namespace: devicePluginCM.Namespace,
			Name:      devicePluginCM.Name,
		}

		k8sClient := fake.NewClientBuilder().
			WithObjects(&node).
			WithObjects(&devicePluginCM).
			Build()
		devicePluginClient := mocks.NewDevicePluginClient(t)
		devicePluginClient.On("Restart", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(nil)
		partitioner := mps.NewPartitioner(
			k8sClient,
			cmNamespacedName,
			devicePluginClient,
		)
		ctx := context.Background()

		nodePartitioning := state.NodePartitioning{
			GPUs: []state.GPUPartitioning{
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						"nvidia.com/gpu-10gb": 2,
						"nvidia.com/gpu-5gb":  2,
					},
				},
			},
		}
		planId := "plan-id"
		err := partitioner.ApplyPartitioning(ctx, node, planId, nodePartitioning)
		assert.NoError(t, err)

		// Fetch config map
		var updatedCm v1.ConfigMap
		assert.NoError(t, k8sClient.Get(ctx, cmNamespacedName, &updatedCm))

		// Check keys
		expectedKeys := []string{
			fmt.Sprintf(mps.DevicePluginConfigKeyFormat, node.Name, planId),
			fmt.Sprintf(mps.DevicePluginConfigKeyFormat, "node-2", "old-plan-2"),
		}
		updatedCmKeys := make([]string, 0, len(updatedCm.Data))
		for k := range updatedCm.Data {
			updatedCmKeys = append(updatedCmKeys, k)
		}
		assert.ElementsMatch(t, expectedKeys, updatedCmKeys)
	})
}
