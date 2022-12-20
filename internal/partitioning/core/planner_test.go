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

package core_test

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	partitioner_mig "github.com/nebuly-ai/nebulnetes/internal/partitioning/mig"
	state2 "github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	n8sresource "github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	scheduler_mock "github.com/nebuly-ai/nebulnetes/pkg/test/mocks/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"strconv"
	"testing"
)

func TestPlanner__Plan(t *testing.T) {
	testCases := []struct {
		name                     string
		snapshotNodes            []v1.Node
		candidatePods            []v1.Pod
		schedulerPreFilterStatus *framework.Status
		schedulerFilterStatus    *framework.Status

		expectedOverallPartitioning []state2.GPUPartitioning
		expectedErr                 bool
	}{
		{
			name:                     "Empty snapshot, no candidates",
			snapshotNodes:            make([]v1.Node, 0),
			candidatePods:            make([]v1.Pod, 0),
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),

			expectedOverallPartitioning: make([]state2.GPUPartitioning, 0),
			expectedErr:                 false,
		},
		{
			name:          "Empty snapshot, many candidates",
			snapshotNodes: make([]v1.Node, 0),
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(),
				factory.BuildPod("ns-2", "pd-2").Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),

			expectedOverallPartitioning: make([]state2.GPUPartitioning, 0),
			expectedErr:                 false,
		},
		{
			name: "Cluster geometry cannot be changed for pending Pods",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile4g20gb, n8sresource.StatusUsed): "1", // node provides required MIG resource, but it's used
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile4g24gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
				factory.BuildNode("node-2").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g5gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile1g5gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").Get(), // not requesting any MIG resource
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile4g20gb.AsResourceName(), 1).
						WithCPUMilliRequest(100).
						Get(),
				).Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile4g20gb.AsResourceName(): 1,
					},
				},
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile1g5gb.AsResourceName(): 1,
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Cluster geometry can be changed, but pod scheduling PreFilter fails",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile4g24gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile4g24gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
				factory.BuildNode("node-2").
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-2").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						WithCPUMilliRequest(100).
						Get(),
				).Get(),
				factory.BuildPod("ns-2", "pd-1").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile2g12gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Error),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						mig.Profile4g24gb.AsResourceName(): 1,
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Cluster geometry can be changed, but pod scheduling Filter fails",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile4g24gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile4g24gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
				factory.BuildNode("node-2").
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-2").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
						WithCPUMilliRequest(100).
						Get(),
				).Get(),
				factory.BuildPod("ns-2", "pd-1").WithContainer(
					factory.BuildContainer("test", "test").
						WithScalarResourceRequest(mig.Profile2g12gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Error),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile4g24gb.AsResourceName(): 1,
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Pods with multiple containers - Cluster geometry gets changed by splitting up MIG profile and " +
				"creating new ones from spare MIG capacity",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile4g24gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						constant.LabelNvidiaCount:     strconv.Itoa(1),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile4g24gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
				factory.BuildNode("node-2").
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						constant.LabelNvidiaCount:     strconv.Itoa(1),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						constant.ResourceNvidiaGPU: *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-2").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-1", "pd-1").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-2", "pd-2").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g6gb.AsResourceName(), 1).
							Get(),
					).Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						mig.Profile1g6gb.AsResourceName(): 4,
					},
				},
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						mig.Profile1g6gb.AsResourceName(): 4,
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Cluster geometry gets updated by grouping small unused MIG profiles into a larger one",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g6gb, n8sresource.StatusFree): "4",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, mig.Profile1g6gb, n8sresource.StatusFree): "4",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A30),
						constant.LabelNvidiaCount:     strconv.Itoa(2),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile1g6gb.AsResourceName(): *resource.NewQuantity(8, resource.DecimalSI),
					}).
					Get(),
				factory.BuildNode("node-2").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g5gb, n8sresource.StatusFree):  "5",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile2g10gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A100_SXM4_40GB),
						constant.LabelNvidiaCount:     strconv.Itoa(1),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile1g5gb.AsResourceName():  *resource.NewQuantity(5, resource.DecimalSI),
						mig.Profile2g10gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile3g20gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile3g20gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-1", "pd-2").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile4g24gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-1", "pd-3").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile2g12gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-1", "pd-4").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile2g12gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile3g20gb.AsResourceName(): 2,
					},
				},
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile4g24gb.AsResourceName(): 1,
					},
				},
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile2g12gb.AsResourceName(): 2,
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Geometry change with some MIG profiles in common",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").
					WithAnnotations(map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, mig.Profile1g10gb, n8sresource.StatusFree): "1",
					}).
					WithLabels(map[string]string{
						constant.LabelNvidiaProduct:   string(gpu.GPUModel_A100_PCIe_80GB),
						constant.LabelNvidiaCount:     strconv.Itoa(1),
						v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
					}).
					WithAllocatableResources(v1.ResourceList{
						mig.Profile1g10gb.AsResourceName(): *resource.NewQuantity(1, resource.DecimalSI),
					}).
					Get(),
			},
			candidatePods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile2g20gb.AsResourceName(), 1).
							Get(),
					).
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
				factory.BuildPod("ns-1", "pd-2").
					WithContainer(
						factory.BuildContainer("test", "test").
							WithScalarResourceRequest(mig.Profile4g40gb.AsResourceName(), 1).
							Get(),
					).
					Get(),
			},
			schedulerPreFilterStatus: framework.NewStatus(framework.Success),
			schedulerFilterStatus:    framework.NewStatus(framework.Success),
			expectedOverallPartitioning: []state2.GPUPartitioning{
				{
					Resources: map[v1.ResourceName]int{
						mig.Profile4g40gb.AsResourceName(): 1,
						mig.Profile2g20gb.AsResourceName(): 1,
						mig.Profile1g10gb.AsResourceName(): 1,
					},
				},
			},
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockedScheduler := scheduler_mock.NewFramework(t)
			mockedScheduler.On(
				"RunPreFilterPlugins",
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(nil, tt.schedulerPreFilterStatus).Maybe()
			mockedScheduler.On(
				"RunFilterPlugins",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(framework.PluginToStatus{"": tt.schedulerFilterStatus}).Maybe()

			snapshot := newSnapshotFromNodes(tt.snapshotNodes)
			planner := partitioner_mig.NewPlanner(mockedScheduler)
			plan, err := planner.Plan(context.Background(), snapshot, tt.candidatePods)

			// Compute overall partitioning ignoring GPU index
			overallGpuPartitioning := make([]state2.GPUPartitioning, 0)
			for _, nodePartitioning := range plan.DesiredState {
				for _, g := range nodePartitioning.GPUs {
					g.GPUIndex = 0
					overallGpuPartitioning = append(overallGpuPartitioning, g)
				}
			}
			for i := range tt.expectedOverallPartitioning {
				gpuPartitioning := &tt.expectedOverallPartitioning[i]
				gpuPartitioning.GPUIndex = 0
			}

			// Run assertions
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedOverallPartitioning, overallGpuPartitioning)
			}
		})
	}
}

// TODO: benchmark Plan
//func BenchmarkPlanner_Plan(b *testing.B) {
//	benchmarks := []struct {
//		numSnapshotNodes int
//		numCandidatePods int
//	}{
//		{
//			numSnapshotNodes: 10,
//			numCandidatePods: 10,
//		},
//		{
//			numSnapshotNodes: 10,
//			numCandidatePods: 1000,
//		},
//		{
//			numSnapshotNodes: 100,
//			numCandidatePods: 100,
//		},
//		{
//			numSnapshotNodes: 1000,
//			numCandidatePods: 10000,
//		},
//	}
//
//	mockedScheduler := scheduler_mock.NewFramework(b)
//	mockedScheduler.On(
//		"RunPreFilterPlugins",
//		mock.Anything,
//		mock.Anything,
//		mock.Anything,
//	).Return(nil, framework.NewStatus(framework.Success)).Maybe()
//	mockedScheduler.On(
//		"RunFilterPlugins",
//		mock.Anything,
//		mock.Anything,
//		mock.Anything,
//		mock.Anything,
//	).Return(framework.PluginToStatus{"": framework.NewStatus(framework.Success)}).Maybe()
//	planner := partitioner_mig.NewPlanner(mockedScheduler)
//
//	for _, bb := range benchmarks {
//		ctx := context.Background()
//		b.Run(fmt.Sprintf("snapshotNodes=%d,candidatePods=%d", bb.numSnapshotNodes, bb.numCandidatePods), func(b *testing.B) {
//			for n := 0; n < b.N; n++ {
//				_, err := planner.Plan(ctx, snapshot, candidates)
//				assert.NoError(b, err)
//			}
//		})
//	}
//}
//
//func newRandomNode() framework.NodeInfo {
//	name := util.RandomStringLowercase(10)
//	node := factory.BuildNode(name).Get()
//	nodeInfo := *framework.NewNodeInfo()
//}

func newSnapshotFromNodes(nodes []v1.Node) core.Snapshot {
	nodeInfos := make(map[string]framework.NodeInfo)
	for _, node := range nodes {
		n := node
		ni := framework.NewNodeInfo()
		ni.Requested = framework.NewResource(v1.ResourceList{})
		ni.Allocatable = framework.NewResource(v1.ResourceList{})
		ni.SetNode(&n)
		nodeInfos[n.Name] = *ni
	}
	s := state2.NewClusterState(nodeInfos)
	snapshot, err := partitioner_mig.NewSnapshotTaker().TakeSnapshot(&s)
	if err != nil {
		panic(err)
	}
	return snapshot
}
