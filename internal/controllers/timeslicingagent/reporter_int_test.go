//go:build integration

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

package timeslicingagent_test

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Time Slicing Agent Reporter", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		}
		Expect(k8sClient.Create(ctx, node)).To(Succeed())
	})

	AfterEach(func() {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		}
		Expect(k8sClient.Delete(ctx, node)).To(Succeed())
	})

	When("The node does not have any GPU", func() {
		It("Should not update the node annotations", func() {
			gpuClient.On("GetDevices", mock.Anything).Return(gpu.DeviceList{}, nil)
			Consistently(func() error {
				var node v1.Node
				if err := k8sClient.Get(
					ctx,
					types.NamespacedName{Name: nodeName, Namespace: ""},
					&node,
				); err != nil {
					return err
				}
				if len(node.Annotations) > 0 {
					return fmt.Errorf("unexpected node annotations: %v", node.Annotations)
				}
				return nil
			}, 5*time.Second).Should(Succeed())
		})
	})

	When("The node has multiple GPUs", func() {
		It("Should update the node annotations exposing the node GPUs", func() {
			gpus := gpu.DeviceList{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu",
						DeviceId:     "id-1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-10gb",
						DeviceId:     "id-2",
						Status:       resource.StatusFree,
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-10gb",
						DeviceId:     "id-2",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 1,
				},
			}
			gpuClient.On("GetDevices", mock.Anything).Return(gpus, nil)

			expectedAnnotations := gpu.StatusAnnotationList[timeslicing.ProfileName]{
				{
					ProfileName: "nvidia.com/gpu",
					Index:       0,
					Status:      "free",
					Quantity:    1,
				},
				{
					ProfileName: "nvidia.com/gpu-10gb",
					Index:       1,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
				{
					ProfileName: "nvidia.com/gpu-10gb",
					Index:       1,
					Status:      resource.StatusUsed,
					Quantity:    1,
				},
			}
			Eventually(func() error {
				var node v1.Node
				if err := k8sClient.Get(
					ctx,
					types.NamespacedName{Name: nodeName, Namespace: ""},
					&node,
				); err != nil {
					return err
				}
				if len(node.Annotations) == 0 {
					return fmt.Errorf("node annotations are empty")
				}
				statusAnnotations, _ := timeslicing.ParseNodeAnnotations(node)
				if len(statusAnnotations) != 3 {
					return fmt.Errorf(
						"expected %d status annotations, found %d",
						3,
						len(statusAnnotations),
					)
				}
				if !util.UnorderedEqual(statusAnnotations, expectedAnnotations) {
					return fmt.Errorf(
						"expected status annotations %v, found %v",
						expectedAnnotations,
						statusAnnotations,
					)
				}

				return nil
			}, timeout, interval).Should(Succeed())
		})
	})
})
