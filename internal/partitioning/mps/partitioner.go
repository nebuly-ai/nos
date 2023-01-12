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

package mps

import (
	"context"
	"fmt"
	nvidiav1 "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	"github.com/nebuly-ai/nos/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"time"
)

const DevicePluginConfigKeyFormat = "%s-%s"

var _ core.Partitioner = partitioner{}

type partitioner struct {
	client.Client
	devicePluginCM    types.NamespacedName
	devicePluginDelay time.Duration
}

func NewPartitioner(
	client client.Client,
	devicePluginCM types.NamespacedName,
	devicePluginDelay time.Duration,
) core.Partitioner {

	return partitioner{
		Client:            client,
		devicePluginCM:    devicePluginCM,
		devicePluginDelay: devicePluginDelay,
	}
}

func (p partitioner) ApplyPartitioning(ctx context.Context, node v1.Node, planId string, partitioning state.NodePartitioning) error {
	logger := log.FromContext(ctx)

	var devicePluginCm v1.ConfigMap
	var err error

	// Fetch nvidia-device-plugin config
	if devicePluginCm, err = p.getDevicePluginCM(ctx); err != nil {
		return err
	}
	if devicePluginCm.Data == nil {
		devicePluginCm.Data = map[string]string{}
	}
	originalCm := devicePluginCm.DeepCopy()

	// Delete old node config
	for k := range devicePluginCm.Data {
		if strings.HasPrefix(k, node.Name) {
			delete(devicePluginCm.Data, k)
		}
	}

	// Update ConfigMap with new node config
	key := fmt.Sprintf(DevicePluginConfigKeyFormat, node.Name, planId)
	pluginConfig, err := ToPluginConfig(partitioning)
	if err != nil {
		return fmt.Errorf("unable to convert node partitioning state to device plugin config: %v", err)
	}
	pluginConfigYaml, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return fmt.Errorf("unable to marshal nvidia device plugin config: %v", err)
	}
	devicePluginCm.Data[key] = string(pluginConfigYaml)
	if err = p.Patch(ctx, &devicePluginCm, client.MergeFrom(originalCm)); err != nil {
		return err
	}

	// Wait for CM propagation time
	logger.Info(fmt.Sprintf("waiting %f seconds for device plugin config config propagation...", p.devicePluginDelay.Seconds()))
	time.Sleep(p.devicePluginDelay)

	// Update node labels to apply new config
	originalNode := node.DeepCopy()
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	node.Labels[constant.LabelNvidiaDevicePluginConfig] = key
	if err = p.Patch(ctx, &node, client.MergeFrom(originalNode)); err != nil {
		return err
	}
	logger.Info("node partitioning config updated", "node", node.Name, "plan", planId)

	return nil
}

func (p partitioner) getDevicePluginCM(ctx context.Context) (v1.ConfigMap, error) {
	var res v1.ConfigMap
	cmObjectKey := client.ObjectKey{Name: p.devicePluginCM.Name, Namespace: p.devicePluginCM.Namespace}
	err := p.Client.Get(ctx, cmObjectKey, &res)
	return res, err
}

func ToPluginConfig(partitioning state.NodePartitioning) (nvidiav1.Config, error) {
	replicatedResources := make([]nvidiav1.MPSResource, 0)
	for _, g := range partitioning.GPUs {
		for r, q := range g.Resources {
			slicingProfile, err := slicing.ExtractProfileName(r)
			if err != nil {
				return nvidiav1.Config{}, err
			}
			mpsResource := nvidiav1.MPSResource{
				Name:     nvidiav1.ResourceName(constant.ResourceNvidiaGPU),
				Rename:   nvidiav1.ResourceName(strings.TrimPrefix(r.String(), constant.NvidiaResourcePrefix)),
				MemoryGB: slicingProfile.GetMemorySizeGB(),
				Devices: []nvidiav1.ReplicatedDeviceRef{
					nvidiav1.ReplicatedDeviceRef(strconv.Itoa(g.GPUIndex)),
				},
				Replicas: q,
			}
			replicatedResources = append(replicatedResources, mpsResource)
		}
	}
	return nvidiav1.Config{
		Version: nvidiav1.Version,
		Flags: nvidiav1.Flags{
			CommandLineFlags: nvidiav1.CommandLineFlags{
				MigStrategy: util.StringAddr("none"),
			},
		},
		Sharing: nvidiav1.Sharing{
			MPS: nvidiav1.MPS{Resources: replicatedResources},
		},
	}, nil
}
