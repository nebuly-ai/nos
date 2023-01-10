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

package mps

import (
	"context"
	"fmt"
	nvidiav1 "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/slicing"
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
	devicePluginCM     types.NamespacedName
	devicePluginClient gpu.DevicePluginClient
}

func NewPartitioner(
	client client.Client,
	devicePluginCM types.NamespacedName,
	devicePluginClient gpu.DevicePluginClient,
) core.Partitioner {

	return partitioner{
		Client:             client,
		devicePluginCM:     devicePluginCM,
		devicePluginClient: devicePluginClient,
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
	pluginConfig := ToPluginConfig(partitioning)
	pluginConfigYaml, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return fmt.Errorf("unable to marshal nvidia device plugin config: %v", err)
	}
	devicePluginCm.Data[key] = string(pluginConfigYaml)
	if err = p.Patch(ctx, &devicePluginCm, client.MergeFrom(originalCm)); err != nil {
		return err
	}

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

	// Restart the NVIDIA device plugin on the node
	logger.Info("restarting NVIDIA device plugin", "node", node.Name)
	if err = p.devicePluginClient.Restart(ctx, node.Name, 1*time.Minute); err != nil {
		logger.Error(err, "unable to restart NVIDIA device plugin")
		return err
	}

	return nil
}

func (p partitioner) getDevicePluginCM(ctx context.Context) (v1.ConfigMap, error) {
	var res v1.ConfigMap
	cmObjectKey := client.ObjectKey{Name: p.devicePluginCM.Name, Namespace: p.devicePluginCM.Namespace}
	err := p.Client.Get(ctx, cmObjectKey, &res)
	return res, err
}

func ToPluginConfig(partitioning state.NodePartitioning) nvidiav1.Config {
	replicatedResources := make([]nvidiav1.MPSResource, 0)
	for _, g := range partitioning.GPUs {
		for r, q := range g.Resources {
			slicingProfile, err := slicing.ExtractProfileName(r) // TODO: move size info into state.NodePartitioning
			if err != nil {
				continue
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
		Version:   nvidiav1.Version,
		Flags:     nvidiav1.Flags{},
		Resources: nvidiav1.Resources{},
		Sharing: nvidiav1.Sharing{
			MPS: nvidiav1.MPS{Resources: replicatedResources},
		},
	}
}
