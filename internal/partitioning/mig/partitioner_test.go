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
	"context"
	"fmt"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	mig_partitioner "github.com/nebuly-ai/nos/internal/partitioning/mig"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestActuator__Apply(t *testing.T) {
	testCases := []struct {
		name          string
		snapshotNodes map[string]v1.Node
		desiredState  state.PartitioningState

		expectedAnnotations map[string]map[string]string
		expectedApplied     bool
		expectedErr         bool
	}{
		{
			name:                "Empty snapshot, empty desired state",
			snapshotNodes:       map[string]v1.Node{},
			desiredState:        map[string]state.NodePartitioning{},
			expectedAnnotations: map[string]map[string]string{},
			expectedApplied:     false,
			expectedErr:         false,
		},
		{
			name:          "Empty snapshot, desired state not empty: actuator should return node not found error",
			snapshotNodes: map[string]v1.Node{},
			desiredState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			expectedAnnotations: map[string]map[string]string{},
			expectedApplied:     false,
			expectedErr:         true,
		},
		{
			name: "Empty desired state: should do nothing",
			snapshotNodes: map[string]v1.Node{
				"node-1": factory.BuildNode("node-1").WithLabels(map[string]string{
					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					constant.LabelNvidiaCount:     "1",
					constant.LabelNvidiaProduct:   string(gpu.GPUModel_A100_SXM4_40GB),
				}).Get(),
			},
			desiredState:        map[string]state.NodePartitioning{},
			expectedAnnotations: map[string]map[string]string{},
			expectedApplied:     false,
			expectedErr:         false,
		},
		{
			name: "Desired state not empty: should update nodes only GPU Spec annotations according to it, deleting old annotations",
			snapshotNodes: map[string]v1.Node{
				"node-1": factory.BuildNode("node-1").WithAnnotations(map[string]string{
					"annotation-1": "foo",
				}).Get(),
				"node-2": factory.BuildNode("node-2").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, mig.Profile1g5gb):                         "4",
						fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, mig.Profile2g10gb):                        "1",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, mig.Profile2g10gb, resource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						fmt.Sprintf(constant.LabelNvidiaProduct): string(gpu.GPUModel_A100_SXM4_40GB),
						v1alpha1.LabelGpuPartitioning:            gpu.PartitioningKindMig.String(),
						constant.LabelNvidiaCount:                "2",
						constant.LabelNvidiaProduct:              string(gpu.GPUModel_A100_SXM4_40GB),
					}).
					Get(),
			},
			desiredState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g6gb.AsResourceName():  1,
								mig.Profile3g20gb.AsResourceName(): 2,
							},
						},
					},
				},
				"node-2": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile4g24gb.AsResourceName(): 1,
							},
						},
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile4g24gb.AsResourceName(): 2,
							},
						},
					},
				},
			},
			expectedApplied: true,
			expectedAnnotations: map[string]map[string]string{
				"node-1": {
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, mig.Profile1g6gb):  "1",
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, mig.Profile3g20gb): "2",
					"annotation-1": "foo",
				},
				"node-2": {
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, mig.Profile4g24gb):                        "1",
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, mig.Profile4g24gb):                        "2",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, mig.Profile2g10gb, resource.StatusFree): "1",
				},
			},
			expectedErr: false,
		},
		{
			name: "Desired state equals current state, should do nothing",
			snapshotNodes: map[string]v1.Node{
				"node-2": factory.BuildNode("node-2").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, mig.Profile2g10gb):                        "1",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g5gb, resource.StatusFree):  "4",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, mig.Profile2g10gb, resource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						fmt.Sprintf(constant.LabelNvidiaProduct): string(gpu.GPUModel_A100_SXM4_40GB),
						v1alpha1.LabelGpuPartitioning:            gpu.PartitioningKindMig.String(),
						constant.LabelNvidiaCount:                "2",
						constant.LabelNvidiaProduct:              string(gpu.GPUModel_A100_SXM4_40GB),
					}).
					Get(),
			},
			desiredState: map[string]state.NodePartitioning{
				"node-2": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 4,
							},
						},
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile2g10gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			expectedApplied: false,
			expectedAnnotations: map[string]map[string]string{
				"node-2": {
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 1, mig.Profile2g10gb):                        "1",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, mig.Profile2g10gb, resource.StatusFree): "1",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g5gb, resource.StatusFree):  "4",
				},
			},
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientBuilder := fake.NewClientBuilder()
			nodeInfos := make(map[string]framework.NodeInfo)
			for _, n := range tt.snapshotNodes {
				n := n
				fakeClientBuilder.WithObjects(&n)
				ni := framework.NewNodeInfo()
				ni.SetNode(&n)
				nodeInfos[n.Name] = *ni
			}

			s := state.NewClusterState(nodeInfos)
			snapshot, err := mig_partitioner.NewSnapshotTaker().TakeSnapshot(s)
			assert.NoError(t, err)

			fakeClient := fakeClientBuilder.Build()
			actuator := mig_partitioner.NewActuator(fakeClient)
			plan := core.NewPartitioningPlan(tt.desiredState)
			applied, err := actuator.Apply(context.Background(), snapshot, plan)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedApplied, applied)

			var updatedNode v1.Node
			for node, expectedNodeAnnotations := range tt.expectedAnnotations {
				assert.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: node}, &updatedNode))

				// Exclude plan ID annotation until we find a way to mock the ID generation
				annotationsWithoutPlanId := make(map[string]string)
				for k, v := range updatedNode.Annotations {
					if k != v1alpha1.AnnotationPartitioningPlan {
						annotationsWithoutPlanId[k] = v
					}
				}
				assert.Equal(t, expectedNodeAnnotations, annotationsWithoutPlanId)
			}
		})
	}
}
