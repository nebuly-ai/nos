//go:build integration

/*
 * Copyright 2023 nebuly.com
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

package gpupartitioner_test

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"time"
)

var _ = Describe("Node Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	migNodeInitializer.On("InitNodePartitioning", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	BeforeEach(func() {
	})

	AfterEach(func() {
	})

	When("A node does not have GPU Count label", func() {
		It("Should not be added to the Cluster State", func() {
			By("By creating a node without GPU Count label")
			nodeName := "node-without-gpu-count-label"
			node := factory.BuildNode(nodeName).WithLabels(map[string]string{
				constant.LabelNvidiaProduct:   gpu.GPUModel_A100_PCIe_80GB.String(),
				v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMps.String(),
			}).Get()
			Expect(k8sClient.Create(ctx, &node)).To(Succeed())

			By("Checking that the node is not added to the Cluster State")
			Consistently(func() bool {
				_, ok := clusterState.GetNode(nodeName)
				return ok
			}, 3, interval).Should(BeFalse())
		})
	})

	When("A node does not have GPU Model label", func() {
		It("Should not be added to the Cluster State", func() {
			By("By creating a node without GPU Model label")
			nodeName := "node-without-gpu-model-label"
			node := factory.BuildNode(nodeName).WithLabels(map[string]string{
				constant.LabelNvidiaCount:     "1",
				v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMps.String(),
			}).Get()
			Expect(k8sClient.Create(ctx, &node)).To(Succeed())

			By("Checking that the node is not added to the Cluster State")
			Consistently(func() bool {
				_, ok := clusterState.GetNode(nodeName)
				return ok
			}, 3, interval).Should(BeFalse())
		})
	})

	When("A node with GPU labels has MPS partitioning enabled", func() {
		It("Should always be added to the Cluster State", func() {
			By("By creating a node with MPS partitioning enabled")
			nodeName := "node-mps"
			node := factory.BuildNode(nodeName).WithLabels(map[string]string{
				constant.LabelNvidiaProduct:   gpu.GPUModel_A100_PCIe_80GB.String(),
				constant.LabelNvidiaCount:     "1",
				v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMps.String(),
			}).Get()
			Expect(k8sClient.Create(ctx, &node)).To(Succeed())

			By("Checking that the node is added to the Cluster State")
			Eventually(func() bool {
				_, ok := clusterState.GetNode(nodeName)
				return ok
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("A node with GPU labels has MIG partitioning enabled", func() {
		It("Should *not* be added to the Cluster State it is not initialized", func() {
			By("By creating a node with MIG partitioning enabled, but not initialized")
			nodeName := "node-mig-not-initialized"
			node := factory.BuildNode(nodeName).WithLabels(map[string]string{
				constant.LabelNvidiaProduct:   gpu.GPUModel_A100_PCIe_80GB.String(),
				constant.LabelNvidiaCount:     "1",
				v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
			}).Get()
			Expect(k8sClient.Create(ctx, &node)).To(Succeed())

			By("Checking that the node is *not* added to the Cluster State")
			Consistently(func() bool {
				_, ok := clusterState.GetNode(nodeName)
				return ok
			}, 5, interval).Should(BeFalse())
		})

		It("Should be added to the Cluster State it is initialized", func() {
			By("By creating an initialized node with MIG partitioning enabled")
			nodeName := "node-mig-initialized"
			node := factory.BuildNode(nodeName).
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct:   gpu.GPUModel_A100_PCIe_80GB.String(),
					constant.LabelNvidiaCount:     "1",
					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
				}).
				WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, 0, "10gb"): "1",
				}).
				Get()
			Expect(k8sClient.Create(ctx, &node)).To(Succeed())

			By("Checking that the node is added to the Cluster State")
			Eventually(func() bool {
				_, ok := clusterState.GetNode(nodeName)
				return ok
			}, timeout, interval).Should(BeTrue())
		})
	})
})
