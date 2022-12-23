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

package ts

import (
	"context"
	"fmt"
	nvidiav1 "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
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

	// Fetch nvidia-device-plugin config or create it if it doesn't exist
	if devicePluginCm, err = p.getOrCreateDevicePluginCM(ctx); err != nil {
		return err
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

func (p partitioner) getOrCreateDevicePluginCM(ctx context.Context) (v1.ConfigMap, error) {
	logger := log.FromContext(ctx)
	var res v1.ConfigMap

	cmObjectKey := client.ObjectKey{Name: p.devicePluginCM.Name, Namespace: p.devicePluginCM.Namespace}
	err := p.Client.Get(ctx, cmObjectKey, &res)

	// Error fetching CM
	if client.IgnoreNotFound(err) != nil {
		return res, fmt.Errorf("unable to get device plugin ConfigMap: %v", err)
	}

	// CM found, return it
	if err == nil {
		return res, nil
	}

	// CM does not exist, create it
	res.Name = p.devicePluginCM.Name
	res.Namespace = p.devicePluginCM.Namespace
	res.Data = map[string]string{}
	logger.Info(
		"device plugin ConfigMap not found, creating it",
		"name",
		res.Name,
		"namespace",
		res.Namespace,
	)
	if err = p.Create(ctx, &res); err != nil {
		return res, fmt.Errorf("unable to create device plugin ConfigMap: %v", err)
	}

	return res, nil
}

func ToPluginConfig(partitioning state.NodePartitioning) nvidiav1.Config {
	replicatedResources := make([]nvidiav1.ReplicatedResource, 0)
	for _, g := range partitioning.GPUs {
		for r, q := range g.Resources {
			nvidiaRes := nvidiav1.ReplicatedResource{
				Name:   nvidiav1.ResourceName(constant.ResourceNvidiaGPU),
				Rename: nvidiav1.ResourceName(strings.TrimPrefix(r.String(), constant.NvidiaResourcePrefix)),
				Devices: nvidiav1.ReplicatedDevices{
					List: []nvidiav1.ReplicatedDeviceRef{
						nvidiav1.ReplicatedDeviceRef(strconv.Itoa(g.GPUIndex)),
					},
				},
				Replicas: q,
			}
			replicatedResources = append(replicatedResources, nvidiaRes)
		}
	}
	return nvidiav1.Config{
		Version:   nvidiav1.Version,
		Flags:     nvidiav1.Flags{},
		Resources: nvidiav1.Resources{},
		Sharing: nvidiav1.Sharing{
			TimeSlicing: nvidiav1.TimeSlicing{Resources: replicatedResources},
		},
	}
}
