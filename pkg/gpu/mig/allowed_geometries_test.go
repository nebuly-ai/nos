/*
 * Copyright 2023 Nebuly.ai
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
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestAllowedMigGeometriesList__UnmarshalYaml(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		output mig.AllowedMigGeometriesList
		error  bool
	}{
		{
			name:   "empty",
			input:  ``,
			output: make(mig.AllowedMigGeometriesList, 0),
			error:  false,
		},
		{
			name: "field 'models' is malformed",
			input: `
- models: "A30"
  allowedGeometries:
    - 1g.6gb: 2
      2g.12gb: 1
    - 4g.24gb: 1
`,
			error: true,
		},
		{
			name: "field 'allowedGeometries' is malformed",
			input: `
- models: [ "A30" ]
  allowedGeometries: foo
`,
			error: true,
		},
		{
			name: "missing field 'models'",
			input: `
- allowedGeometries:
    - 1g.6gb: 2
      2g.12gb: 1
    - 4g.24gb: 1
`,
			error: true,
		},
		{
			name: "missing field 'allowedGeometries'",
			input: `
- models: [ "A30" ]
`,
			error: true,
		},
		{
			name: "not empty",
			input: `
- models: [ "A30" ]
  allowedGeometries:
    - 1g.6gb: 2
      2g.12gb: 1
    - 4g.24gb: 1
- models: [ "A100-SXM4-40GB", "NVIDIA-A100-40GB-PCIe" ]
  allowedGeometries:
    - 1g.5gb: 7
    - 1g.5gb: 5
      2g.10gb: 1
`,
			output: mig.AllowedMigGeometriesList{
				{
					Models: []gpu.Model{"A30"},
					Geometries: []gpu.Geometry{
						{
							mig.ProfileName("1g.6gb"):  2,
							mig.ProfileName("2g.12gb"): 1,
						},
						{
							mig.ProfileName("4g.24gb"): 1,
						},
					},
				},
				{
					Models: []gpu.Model{"A100-SXM4-40GB", "NVIDIA-A100-40GB-PCIe"},
					Geometries: []gpu.Geometry{
						{
							mig.ProfileName("1g.5gb"): 7,
						},
						{
							mig.ProfileName("1g.5gb"):  5,
							mig.ProfileName("2g.10gb"): 1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var res = make(mig.AllowedMigGeometriesList, 0)
			err := yaml.Unmarshal([]byte(tc.input), &res)
			if tc.error {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.output, res)
		})
	}
}

func TestAllowedMigGeometriesList__GroupByModel(t *testing.T) {
	testCases := []struct {
		name     string
		input    mig.AllowedMigGeometriesList
		expected map[gpu.Model][]gpu.Geometry
	}{
		{
			name:     "empty list",
			input:    mig.AllowedMigGeometriesList{},
			expected: map[gpu.Model][]gpu.Geometry{},
		},
		{
			name: "not-empty",
			input: mig.AllowedMigGeometriesList{
				{
					Models: []gpu.Model{"m1", "m2"},
					Geometries: []gpu.Geometry{
						{
							mig.ProfileName("p1"): 1,
							mig.ProfileName("p2"): 2,
						},
						{
							mig.ProfileName("p1"): 3,
						},
					},
				},
				{
					Models: []gpu.Model{"m3"},
					Geometries: []gpu.Geometry{
						{
							mig.ProfileName("p3"): 2,
							mig.ProfileName("p2"): 2,
						},
					},
				},
			},
			expected: map[gpu.Model][]gpu.Geometry{
				"m1": {
					{
						mig.ProfileName("p1"): 1,
						mig.ProfileName("p2"): 2,
					},
					{
						mig.ProfileName("p1"): 3,
					},
				},
				"m2": {
					{
						mig.ProfileName("p1"): 1,
						mig.ProfileName("p2"): 2,
					},
					{
						mig.ProfileName("p1"): 3,
					},
				},
				"m3": {
					{
						mig.ProfileName("p3"): 2,
						mig.ProfileName("p2"): 2,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.input.GroupByModel())
		})
	}
}
