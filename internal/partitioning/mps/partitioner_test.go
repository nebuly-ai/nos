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

package mps_test

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nos/internal/partitioning/mps"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func TestToPluginConfig(t *testing.T) {
	t.Run("Empty node partitioning", func(t *testing.T) {
		nodePartitioning := state.NodePartitioning{GPUs: []state.GPUPartitioning{}}
		config := mps.ToPluginConfig(nodePartitioning)
		assert.Empty(t, config.Sharing.MPS.Resources)
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
		assert.Len(t, config.Sharing.MPS.Resources, 4)
	})
}

func TestPartitioner__ApplyPartitioning(t *testing.T) {
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
		k8sClient := fake.NewClientBuilder().
			WithObjects(&node).
			WithObjects(&devicePluginCM).
			Build()
		partitioner := mps.NewPartitioner(
			k8sClient,
			cmNamespacedName,
			1*time.Millisecond,
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

	t.Run("Should wait device-plugin-delay before updating node labels with new config", func(t *testing.T) {
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

		// config
		delay := 1 * time.Second
		planId := "plan-id"
		k8sClient := fake.NewClientBuilder().
			WithObjects(&node).
			WithObjects(&devicePluginCM).
			Build()
		partitioner := mps.NewPartitioner(
			k8sClient,
			cmNamespacedName,
			delay,
		)
		ctx := context.Background()

		// apply partitioning
		start := time.Now()
		err := partitioner.ApplyPartitioning(ctx, node, planId, state.NodePartitioning{GPUs: []state.GPUPartitioning{}})
		end := time.Now()

		// check no errors
		assert.NoError(t, err)
		// check delay is enforced
		assert.Greater(t, end.Sub(start), delay)
		// check node labels have been updated
		assert.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: node.Namespace, Name: node.Name}, &node))
		assert.Contains(t, node.Labels, constant.LabelNvidiaDevicePluginConfig)
		assert.Equal(t, fmt.Sprintf(mps.DevicePluginConfigKeyFormat, node.Name, planId), node.Labels[constant.LabelNvidiaDevicePluginConfig])
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
		partitioner := mps.NewPartitioner(
			k8sClient,
			cmNamespacedName,
			1*time.Millisecond,
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
