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

package timeslicing

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
)

type tsClient struct {
	resourceClient resource.Client
	nvmlClient     nvml.Client
}

func NewClient(resourceClient resource.Client, nvmlClient nvml.Client) gpu.Client {
	return &tsClient{
		resourceClient: resourceClient,
		nvmlClient:     nvmlClient,
	}
}

func (t tsClient) GetDevices(ctx context.Context) (gpu.DeviceList, gpu.Error) {
	//TODO implement me
	panic("implement me")
}
