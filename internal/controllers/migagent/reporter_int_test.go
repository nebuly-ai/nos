//go:build integration

/*
 * Copyright 2023 Nebuly.ai.
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

package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("MigAgent - Reporter", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Reset mig client
		reporterMigClient.Reset()

		// Fetch the node and clean-up annotations
		var node v1.Node
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &node)).To(Succeed())
		updated := node.DeepCopy()
		updated.Annotations = map[string]string{}
		Expect(k8sClient.Patch(ctx, updated, client.MergeFrom(&node))).To(Succeed())
	})

	AfterEach(func() {
	})

	When("New MIG resources are created on the node", func() {
		It("Should expose those resources on the node as annotations", func() {
			By("Fetching the node")
			var node v1.Node
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &node)).To(Succeed())

			By("Checking the node does not have any annotation")
			Expect(node.Annotations).To(BeEmpty())

			By("Checking that after some time the node will have the annotations exposing the new resources")
			reporterMigClient.ReturnedMigDeviceResources = []gpu.Device{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-1g.10gb",
						DeviceId:     "id-1",
						Status:       "free",
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-2g.20gb",
						DeviceId:     "id-2",
						Status:       "used",
					},
					GpuIndex: 1,
				},
			}
			expectedAnnotationOne := fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "1g.10gb", resource.StatusFree)
			expectedAnnotationTwo := fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, "2g.20gb", resource.StatusUsed)
			expectedAnnotations := map[string]string{
				expectedAnnotationOne:                       "1",
				expectedAnnotationTwo:                       "1",
				v1alpha1.AnnotationReportedPartitioningPlan: "", // we're not using a real shared state in tests, so it does not get updated
			}
			Eventually(func() map[string]string {
				var updatedNode v1.Node
				err := k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &updatedNode)
				if err != nil {
					return nil
				}
				return updatedNode.Annotations
			}, timeout, interval).Should(Equal(expectedAnnotations))
		})
	})
})
