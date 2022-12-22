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
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

var _ core.Partitioner = partitioner{}

type partitioner struct {
	client.Client
	devicePluginCM types.NamespacedName
}

func NewPartitioner(client client.Client, devicePluginCM types.NamespacedName) core.Partitioner {
	return partitioner{
		Client:         client,
		devicePluginCM: devicePluginCM,
	}
}

func (p partitioner) ApplyPartitioning(ctx context.Context, node v1.Node, planId string, partitioning state.NodePartitioning) error {
	logger := log.FromContext(ctx)

	// Fetch nvidia-device-plugin config or create it if it doesn't exist
	var devicePluginCm v1.ConfigMap
	if err := p.Client.Get(ctx, client.ObjectKey{}, &devicePluginCm); err != nil {
		if errors.IsNotFound(err) {
			devicePluginCm.Name = p.devicePluginCM.Name
			devicePluginCm.Namespace = p.devicePluginCM.Namespace
			logger.Info(
				"device plugin ConfigMap not found, creating it",
				"name",
				devicePluginCm.Name,
				"namespace",
				devicePluginCm.Namespace,
			)
			return p.Create(ctx, &devicePluginCm)
		}
		logger.Error(err, "unable to get device plugin ConfigMap")
		return err
	}

	// Delete old node config
	for k := range devicePluginCm.Data {
		if strings.HasPrefix(k, node.Name) {
			delete(devicePluginCm.Data, k)
		}
	}

	// Update ConfigMap with new node config
	originalCm := devicePluginCm.DeepCopy()
	key := fmt.Sprintf("%s-%s", node.Name, planId)
	devicePluginCm.Data[key] = "" // todo
	if err := p.Patch(ctx, &devicePluginCm, client.MergeFrom(originalCm)); err != nil {
		return err
	}

	// Update node labels to apply new config
	originalNode := node.DeepCopy()
	node.Labels[constant.LabelNvidiaDevicePluginConfig] = key
	if err := p.Patch(ctx, &node, client.MergeFrom(originalNode)); err != nil {
		return err
	}
	logger.Info("node partitioning config updated", "node", node.Name, "plan", planId)

	return nil
}
