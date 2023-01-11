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

package elasticquota

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/nebuly-ai/nos/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

var _ = Describe("CompositeElasticQuota controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any setup steps that needs to be executed after each test
	})

	When("New pods belonging to CompositeElasticQuota namespaces are in running status", func() {
		It("Should update the CompositeElasticQuota status", func() {
			const (
				namespaceOneName   = "ns-1"
				namespaceTwoName   = "ns-2"
				namespaceThreeName = "ns-3"
				compositeEqName    = "composite-eq-1"

				elasticQuotaMinCPUMilli  = 4000
				elasticQuotaMinGPUMemory = 4 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaMaxCPUMilli  = 6000
				elasticQuotaMaxGPUMemory = 5 * constant.DefaultNvidiaGPUResourceMemory

				containerOneCPUMilli        = 500
				containerOneNvidiaGpu       = 1
				containerTwoCPUMilli        = 500
				containerTwoNvidiaGpu       = 2
				containerTwoNvidiaMigMemory = 1
			)
			var containerTwoNvidiaMigResource = v1.ResourceName(
				fmt.Sprintf("nvidia.com/mig-1g.%dgb", containerTwoNvidiaMigMemory),
			)

			By("Creating namespaces successfully")
			namespaceOne := factory.BuildNamespace(namespaceOneName).Get()
			namespaceTwo := factory.BuildNamespace(namespaceTwoName).Get()
			namespaceThree := factory.BuildNamespace(namespaceThreeName).Get()
			Expect(k8sClient.Create(ctx, &namespaceOne)).To(Succeed())
			Expect(k8sClient.Create(ctx, &namespaceTwo)).To(Succeed())
			Expect(k8sClient.Create(ctx, &namespaceThree)).To(Succeed())

			By("Creating a CompositeElasticQuota successfully")
			compositeElasticQuota := v1alpha1.BuildCompositeEq(namespaceThreeName, compositeEqName).
				WithNamespaces(namespaceOneName, namespaceTwoName).
				WithMinCPUMilli(elasticQuotaMinCPUMilli).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxCPUMilli(elasticQuotaMaxCPUMilli).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
			Expect(k8sClient.Create(ctx, &compositeElasticQuota)).To(Succeed())

			By("Checking the created ElasticQuota matches the specs")
			var instance = v1alpha1.CompositeElasticQuota{}
			Eventually(func() v1alpha1.CompositeElasticQuotaSpec {
				lookupKey := types.NamespacedName{
					Name:      compositeElasticQuota.Name,
					Namespace: compositeElasticQuota.Namespace,
				}
				_ = k8sClient.Get(ctx, lookupKey, &instance)
				return instance.Spec
			}, timeout, interval).Should(Equal(compositeElasticQuota.Spec))

			By("Creating new Pods within the CompositeElasticQuota namespaces")
			podOne := factory.BuildPod(namespaceOneName, "pod-1").
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithCPUMilliLimit(containerOneCPUMilli).
						WithNvidiaGPULimit(containerOneNvidiaGpu).
						Get(),
				).
				Get()
			podTwo := factory.BuildPod(namespaceTwoName, "pod-1").
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithCPUMilliLimit(containerTwoCPUMilli).
						WithNvidiaGPULimit(containerTwoNvidiaGpu).
						WithScalarResourceLimit(containerTwoNvidiaMigResource, 1).
						Get(),
				).
				Get()
			Expect(k8sClient.Create(ctx, &podOne)).To(Succeed())
			Expect(k8sClient.Create(ctx, &podTwo)).To(Succeed())

			By("Not updating the CompositeElasticQuota status until the Pods are in running state")
			Eventually(func() string {
				lookupKey := types.NamespacedName{Name: compositeElasticQuota.Name, Namespace: compositeElasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return ""
				}
				return instance.Status.Used.Cpu().String()
			}, timeout, interval).Should(Equal("0"))

			By("Updating the Pods status to running")
			podOne.Status.Phase = v1.PodRunning
			podTwo.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &podOne)).To(Succeed())
			Expect(k8sClient.Status().Update(ctx, &podTwo)).To(Succeed())

			By("Checking that the ElasticQuota status gets updated considering the Pods in its namespaces")
			expectedCPUQuantity, _ := resource.ParseQuantity(
				strconv.Itoa((containerOneCPUMilli + containerTwoCPUMilli) / 1000),
			)
			expectedGPUMemoryQuantity, _ := resource.ParseQuantity(
				strconv.Itoa((containerOneNvidiaGpu+containerTwoNvidiaGpu)*constant.DefaultNvidiaGPUResourceMemory + containerTwoNvidiaMigMemory),
			)
			expectedUsedResourceList := v1.ResourceList{
				v1.ResourceCPU:             expectedCPUQuantity,
				v1alpha1.ResourceGPUMemory: expectedGPUMemoryQuantity,
			}
			Eventually(func() v1.ResourceList {
				lookupKey := types.NamespacedName{
					Name:      compositeElasticQuota.Name,
					Namespace: compositeElasticQuota.Namespace,
				}
				_ = k8sClient.Get(ctx, lookupKey, &instance)
				return instance.Status.Used
			}, timeout, interval).ShouldNot(Equal(expectedUsedResourceList))

			By("Checking the Pods' capacity-info label shows that the Pods are in-quota")
			var podInstance v1.Pod
			Eventually(func(g Gomega) {
				lookupKey := types.NamespacedName{Name: podOne.Name, Namespace: podOne.Namespace}
				g.Expect(k8sClient.Get(ctx, lookupKey, &podInstance)).To(Succeed())
				g.Expect(podInstance.Labels).To(HaveKeyWithValue(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoInQuota)))
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				lookupKey := types.NamespacedName{Name: podTwo.Name, Namespace: podTwo.Namespace}
				g.Expect(k8sClient.Get(ctx, lookupKey, &podInstance)).To(Succeed())
				g.Expect(podInstance.Labels).To(HaveKeyWithValue(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoInQuota)))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("A Pod scheduled in over-quota, borrowing quotas from an ElasticQuota", func() {
		It("Should add a label specifying that the Pod is using over-quotas and can thus be preempted", func() {
			const (
				elasticQuotaMinGPUMemory = 4 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaMaxGPUMemory = 6 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaName         = "composite-eq-2"
			)

			By("Creating a namespace successfully")
			namespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			Expect(k8sClient.Create(ctx, &namespace)).To(Succeed())

			By("Creating a CompositeElasticQuota successfully")
			compositeElasticQuota := v1alpha1.BuildCompositeEq(namespace.Name, elasticQuotaName).
				WithNamespaces(namespace.Name).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
			Expect(k8sClient.Create(ctx, &compositeElasticQuota)).To(Succeed())

			By("Creating an ElasticQuota with high GPUMemory min successfully")
			anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			eq := v1alpha1.BuildEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
				WithMinGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				WithMaxGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				Get()
			Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
			Expect(k8sClient.Create(ctx, &eq)).To(Succeed())

			By("Creating a new Pod within one of the CompositeElasticQuota namespaces, requesting GPUMemory > EQ min")
			pod := factory.BuildPod(compositeElasticQuota.Namespace, "pod-2").
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithNvidiaGPULimit(5).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithNvidiaGPULimit(5).
						Get(),
				).
				Get()
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())

			By("Updating the Pod status to running")
			pod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			By("Checking the Pod's capacity-info label shows that the Pod is in over-quota")
			var podInstance v1.Pod
			Eventually(func(g Gomega) {
				lookupKey := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
				g.Expect(k8sClient.Get(ctx, lookupKey, &podInstance)).To(Succeed())
				g.Expect(podInstance.Labels).To(HaveKeyWithValue(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoOverQuota)))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("A Pod scheduled in over-quota, borrowing quotas from another CompositeElasticQuota", func() {
		It("Should add a label specifying that the Pod is using over-quotas and can thus be preempted", func() {
			const (
				elasticQuotaMinGPUMemory  = 4 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaMaxGPUMemory  = 6 * constant.DefaultNvidiaGPUResourceMemory
				compositeElasticQuotaName = "composite-eq-3"
			)

			By("Creating a namespace successfully")
			namespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			Expect(k8sClient.Create(ctx, &namespace)).To(Succeed())

			By("Creating a CompositeElasticQuota successfully")
			compositeElasticQuota := v1alpha1.BuildCompositeEq(namespace.Name, compositeElasticQuotaName).
				WithNamespaces(namespace.Name).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
			Expect(k8sClient.Create(ctx, &compositeElasticQuota)).To(Succeed())

			By("Creating another CompositeElasticQuota with high GPUMemory min successfully")
			anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			anotherCompositeEq := v1alpha1.BuildCompositeEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
				WithNamespaces(anotherNamespace.Name).
				WithMinGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				WithMaxGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				Get()
			Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
			Expect(k8sClient.Create(ctx, &anotherCompositeEq)).To(Succeed())

			By("Creating a new Pod within one of the CompositeElasticQuota namespaces, requesting GPUMemory > EQ min")
			pod := factory.BuildPod(compositeElasticQuota.Namespace, "pod-2").
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithNvidiaGPULimit(5).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithNvidiaGPULimit(5).
						Get(),
				).
				Get()
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())

			By("Updating the Pod status to running")
			pod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			By("Checking the Pod's capacity-info label shows that the Pod is in over-quota")
			var podInstance v1.Pod
			Eventually(func(g Gomega) {
				lookupKey := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
				g.Expect(k8sClient.Get(ctx, lookupKey, &podInstance)).To(Succeed())
				g.Expect(podInstance.Labels).To(HaveKeyWithValue(v1alpha1.LabelCapacityInfo, string(constant.CapacityInfoOverQuota)))
			}, timeout, interval).Should(Succeed())
		})
	})

	When("A CompositeElasticQuota is created", func() {
		It("Should delete any overlapping ElasticQuota existing in one of the namespaces specified by the CompositeElasticQuota", func() {
			By("Creating namespaces successfully")
			namespaceOne := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			namespaceTwo := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			namespaceThree := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			Expect(k8sClient.Create(ctx, &namespaceOne)).To(Succeed())
			Expect(k8sClient.Create(ctx, &namespaceTwo)).To(Succeed())
			Expect(k8sClient.Create(ctx, &namespaceThree)).To(Succeed())

			By("Creating three ElasticQuotas successfully")
			eqOne := v1alpha1.BuildEq(namespaceOne.Name, "eq-1").
				WithMinCPUMilli(100).
				Get()
			eqTwo := v1alpha1.BuildEq(namespaceTwo.Name, "eq-2").
				WithMinCPUMilli(200).
				Get()
			eqThree := v1alpha1.BuildEq(namespaceThree.Name, "eq-3").
				WithMinCPUMilli(200).
				Get()
			Expect(k8sClient.Create(ctx, &eqOne)).To(Succeed())
			Expect(k8sClient.Create(ctx, &eqTwo)).To(Succeed())
			Expect(k8sClient.Create(ctx, &eqThree)).To(Succeed())

			By("Creating a CompositeElasticQuota successfully")
			compositeEq := v1alpha1.BuildCompositeEq(namespaceOne.Name, "composite-eq").
				WithMinCPUMilli(100).
				WithNamespaces(namespaceOne.Name, namespaceTwo.Name).
				Get()
			Expect(k8sClient.Create(ctx, &compositeEq)).To(Succeed())

			By("Checking that the ElasticQuotas existing in the two namespaces defined by the " +
				"CompositeElasticQuota get deleted")
			Eventually(func() bool {
				var nsOneEqList v1alpha1.ElasticQuotaList
				if err := k8sClient.List(ctx, &nsOneEqList, client.InNamespace(eqOne.Namespace)); err != nil {
					return false
				}
				if len(nsOneEqList.Items) > 0 {
					return false
				}
				var nsTwoEqList v1alpha1.ElasticQuotaList
				if err := k8sClient.List(ctx, &nsTwoEqList, client.InNamespace(eqTwo.Namespace)); err != nil {
					return false
				}
				if len(nsTwoEqList.Items) > 0 {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Checking that the non-overlapping ElasticQuota does not get deleted")
			var eqInstance v1alpha1.ElasticQuota
			lookupKey := types.NamespacedName{
				Name:      eqThree.Name,
				Namespace: eqThree.Namespace,
			}
			Expect(k8sClient.Get(ctx, lookupKey, &eqInstance)).To(Succeed())
		})
	})
})
