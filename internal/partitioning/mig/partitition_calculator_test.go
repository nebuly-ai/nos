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

package mig_test

import (
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	mig_partitioner "github.com/nebuly-ai/nos/internal/partitioning/mig"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/test/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPartitioner__GetPartitioning(t *testing.T) {
	testCases := []struct {
		name     string
		node     core.PartitionableNode
		expected state.NodePartitioning
	}{
		{
			name:     "Node is not MIG node, should return empty partitioning",
			node:     &mocks.PartitionableNode{},
			expected: state.NodePartitioning{GPUs: make([]state.GPUPartitioning, 0)},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			partitioning := mig_partitioner.NewPartitionCalculator().GetPartitioning(tt.node)
			assert.True(t, tt.expected.Equal(partitioning))
		})
	}
}
