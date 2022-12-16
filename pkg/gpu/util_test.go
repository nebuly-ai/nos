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

package gpu_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestGetMemoryGB(t *testing.T) {
	testCases := []struct {
		name        string
		node        v1.Node
		expectedVal int
		expectedErr bool
	}{
		{
			name:        "no label",
			node:        factory.BuildNode("node-1").Get(),
			expectedVal: 0,
			expectedErr: true,
		},
		{
			name: "label with invalid value",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaMemory: "invalid",
			}).Get(),
			expectedVal: 0,
			expectedErr: true,
		},
		{
			name: "memory value gets rounded up to the nearest GB",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaMemory: "1100",
			}).Get(),
			expectedVal: 2,
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			gb, err := gpu.GetMemoryGB(tt.node)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.expectedVal, gb)
			}
		})
	}
}
