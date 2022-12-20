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

package ts_test

//func TestNewSnapshot(t *testing.T) {
//	testCases := []struct {
//		name          string
//		snapshotNodes []v1.Node
//		expectedNodes map[string]timeslicing.Node
//		expectedErr   bool
//	}{
//		{
//			name:          "Empty snapshot",
//			snapshotNodes: []v1.Node{},
//			expectedNodes: map[string]timeslicing.Node{},
//			expectedErr:   false,
//		},
//		{
//			name: "Snapshot should include only nodes with gpu-partitioning=TimeSlicing",
//			snapshotNodes: []v1.Node{
//				factory.BuildNode("node-1").WithLabels(map[string]string{
//					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
//					constant.LabelNvidiaMemory:    "10",
//					constant.LabelNvidiaProduct:   "foo",
//					constant.LabelNvidiaCount:     "1",
//				}).Get(),
//				factory.BuildNode("node-2").WithLabels(map[string]string{
//					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
//				}).Get(),
//				factory.BuildNode("node-3").Get(),
//			},
//			expectedNodes: map[string]timeslicing.Node{
//				"node-1": {
//					GetName: "node-1",
//					GPUs: []timeslicing.GPU{
//						timeslicing.NewFullGPU(
//							"foo",
//							0,
//							1,
//						),
//					},
//				},
//			},
//			expectedErr: false,
//		},
//		{
//			name: "Should return error if time-slicing enabled nodes do not have required GPU labels",
//			snapshotNodes: []v1.Node{
//				factory.BuildNode("node-1").WithLabels(map[string]string{
//					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
//				}).Get(),
//			},
//			expectedErr: true,
//		},
//	}
//
//	for _, tt := range testCases {
//		t.Run(tt.name, func(t *testing.T) {
//			// Init cluster snapshot
//			nodeInfos := make(map[string]framework.NodeInfo)
//			for _, n := range tt.snapshotNodes {
//				n := n
//				ni := framework.NewNodeInfo()
//				ni.SetNode(&n)
//				nodeInfos[n.GetName] = *ni
//			}
//			snapshot := core.NewClusterSnapshot(nodeInfos)
//
//			// Init MIG cluster snapshot
//			tsSnapshot, err := ts.NewSnapshot(snapshot)
//			if tt.expectedErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//				assert.Equal(t, tt.expectedNodes, tsSnapshot.GetNodes())
//			}
//		})
//	}
//}
//
////func TestNewSnapshot(t *testing.T) {
////	testCases := []struct {
////		name                 string
////		snapshotNodes        []v1.Node
////		nvidiaDevicePluginCm v1.ConfigMap
////		expectedNodes        map[string]timeslicing.Node
////		expectedErr          bool
////	}{
////		{
////			name:          "empty snapshot, config is not empty",
////			snapshotNodes: []v1.Node{},
////			nvidiaDevicePluginCm: v1.ConfigMap{
////				Data: map[string]string{
////					"node-1": "",
////				},
////			},
////			expectedNodes: map[string]timeslicing.Node{},
////			expectedErr:   false,
////		},
////		{
////			name:          "empty snapshot, empty config",
////			snapshotNodes: []v1.Node{},
////			nvidiaDevicePluginCm: v1.ConfigMap{
////				Data: map[string]string{},
////			},
////			expectedNodes: map[string]timeslicing.Node{},
////			expectedErr:   false,
////		},
////		{
////			name: "ConfigMap is empty, snapshot should include GPUs and nodes anyway",
////			snapshotNodes: []v1.Node{
////				factory.BuildNode("node-1").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "2",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "2000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////				factory.BuildNode("node-2").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "3",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "3000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////			},
////			nvidiaDevicePluginCm: v1.ConfigMap{Data: map[string]string{}},
////			expectedNodes: map[string]timeslicing.Node{
////				"node-1": {
////					GetName: "node-1",
////					GPUs: []timeslicing.GPU{
////						{
////							Model:    "foo",
////							Index:    0,
////							Replicas: 1,
////							MemoryGB: 2,
////						},
////						{
////							Model:    "foo",
////							Index:    1,
////							Replicas: 1,
////							MemoryGB: 2,
////						},
////					},
////				},
////				"node-2": {
////					GetName: "node-2",
////					GPUs: []timeslicing.GPU{
////						{
////							Model:    "foo",
////							Index:    0,
////							Replicas: 1,
////							MemoryGB: 3,
////						},
////						{
////							Model:    "foo",
////							Index:    1,
////							Replicas: 1,
////							MemoryGB: 3,
////						},
////						{
////							Model:    "foo",
////							Index:    2,
////							Replicas: 1,
////							MemoryGB: 3,
////						},
////					},
////				},
////			},
////			expectedErr: false,
////		},
////		{
////			name: "Snapshot should include only nodes with time-slicing enabled",
////			snapshotNodes: []v1.Node{
////				factory.BuildNode("node-1").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "2",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "2000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////				factory.BuildNode("node-2").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct: "foo",
////					constant.LabelNvidiaCount:   "3",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:  "3000", // Memory in Bytes
////				}).Get(),
////			},
////			nvidiaDevicePluginCm: v1.ConfigMap{Data: map[string]string{}},
////			expectedNodes: map[string]timeslicing.Node{
////				"node-1": {
////					GetName: "node-1",
////					GPUs: []timeslicing.GPU{
////						{
////							Model:    "foo",
////							Index:    0,
////							Replicas: 1,
////							MemoryGB: 2,
////						},
////						{
////							Model:    "foo",
////							Index:    1,
////							Replicas: 1,
////							MemoryGB: 2,
////						},
////					},
////				},
////			},
////			expectedErr: false,
////		},
////		{
////			name: "Node with time-slicing enabled does not have all required labels, should return error",
////			snapshotNodes: []v1.Node{
////				factory.BuildNode("node-1").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaMemory:    "2000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////				factory.BuildNode("node-2").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "3",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "3000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////			},
////			nvidiaDevicePluginCm: v1.ConfigMap{Data: map[string]string{}},
////			expectedErr:          true,
////		},
////		{
////			name: "CM not empty, should use it for configuring time-slicing nodes",
////			snapshotNodes: []v1.Node{
////				factory.BuildNode("node-1").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "3",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "3000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////			},
////			nvidiaDevicePluginCm: v1.ConfigMap{Data: map[string]string{
////				"node-1": `
////sharing:
////  timeSlicing:
////    resources:
////      - name: nvidia.com/gpu
////        replicas: 2
////        devices:
////        - 0
////      - name: nvidia.com/gpu
////        replicas: 3
////        devices:
////        - 1
////      - name: nvidia.com/gpu
////        replicas: 2
////        devices:
////        - 2
////`,
////			}},
////			expectedNodes: map[string]timeslicing.Node{
////				"node-1": {
////					GetName: "node-1",
////					GPUs: []timeslicing.GPU{
////						{
////							Model:    "foo",
////							Index:    0,
////							Replicas: 2,
////							MemoryGB: 3,
////						},
////						{
////							Model:    "foo",
////							Index:    1,
////							Replicas: 3,
////							MemoryGB: 3,
////						},
////						{
////							Model:    "foo",
////							Index:    2,
////							Replicas: 2,
////							MemoryGB: 3,
////						},
////					},
////				},
////			},
////		},
////		{
////			name: "CM contains malformed data for node, should return error",
////			snapshotNodes: []v1.Node{
////				factory.BuildNode("node-1").WithLabels(map[string]string{
////					constant.LabelNvidiaProduct:   "foo",
////					constant.LabelNvidiaCount:     "3",    // Number of returned GPUs
////					constant.LabelNvidiaMemory:    "3000", // Memory in Bytes
////					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
////				}).Get(),
////			},
////			nvidiaDevicePluginCm: v1.ConfigMap{Data: map[string]string{
////				"node-1": `malformed`,
////			}},
////			expectedErr: true,
////		},
////	}
////
////	for _, tt := range testCases {
////		t.Run(tt.name, func(t *testing.T) {
////			// Init cluster snapshot
////			nodeInfos := make(map[string]framework.NodeInfo)
////			for _, n := range tt.snapshotNodes {
////				n := n
////				ni := framework.NewNodeInfo()
////				ni.SetNode(&n)
////				nodeInfos[n.GetName] = *ni
////			}
////			snapshot := state.NewClusterSnapshot(nodeInfos)
////
////			// Init TimeSlicing cluster snapshot
////			timeSlicingSnapshot, err := tsstate.NewSnapshot(snapshot, tt.nvidiaDevicePluginCm)
////			if tt.expectedErr {
////				assert.Error(t, err)
////			} else {
////				assert.NoError(t, err)
////				assert.Equal(t, tt.expectedNodes, timeSlicingSnapshot.GetNodes())
////			}
////		})
////	}
////}