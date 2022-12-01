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

package gpu

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsMigPartitioningEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		node     v1.Node
		expected bool
	}{
		{
			name:     "Node without partitioning label",
			node:     factory.BuildNode("node-1").Get(),
			expected: false,
		},
		{
			name: "Node with partitioning label, but not MIG",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				v1alpha1.LabelGpuPartitioning: PartitioningKindTimeSlicing.String(),
			}).Get(),
			expected: false,
		},
		{
			name: "Noe with partitioning label, MIG",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				v1alpha1.LabelGpuPartitioning: PartitioningKindMig.String(),
			}).Get(),
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			enabled := IsMigPartitioningEnabled(tt.node)
			assert.Equal(t, tt.expected, enabled)
		})
	}
}
