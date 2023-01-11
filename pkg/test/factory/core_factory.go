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

package factory

import (
	"github.com/nebuly-ai/nos/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type nodeBuilder struct {
	v1.Node
}

func (b *nodeBuilder) Get() v1.Node {
	return b.Node
}

func (b *nodeBuilder) WithAnnotations(annotations map[string]string) *nodeBuilder {
	b.Node.Annotations = annotations
	return b
}

func (b *nodeBuilder) WithAllocatableResources(resourceList v1.ResourceList) *nodeBuilder {
	b.Node.Status.Allocatable = resourceList
	return b
}

func (b *nodeBuilder) WithLabels(labels map[string]string) *nodeBuilder {
	b.Node.Labels = labels
	return b
}

func BuildNode(name string) *nodeBuilder {
	node := v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &nodeBuilder{node}
}

type namespaceBuilder struct {
	v1.Namespace
}

func (b *namespaceBuilder) Get() v1.Namespace {
	return b.Namespace
}

func BuildNamespace(name string) *namespaceBuilder {
	namespace := v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespaceBuilder{namespace}
}

type podBuilder struct {
	v1.Pod
}

func (b *podBuilder) WithContainer(c v1.Container) *podBuilder {
	b.Spec.Containers = append(b.Spec.Containers, c)
	return b
}

func (b *podBuilder) WithPhase(f v1.PodPhase) *podBuilder {
	b.Status.Phase = f
	return b
}

func (b *podBuilder) WithUID(uid string) *podBuilder {
	b.UID = types.UID(uid)
	return b
}

func (b *podBuilder) WithInitContainer(c v1.Container) *podBuilder {
	b.Spec.InitContainers = append(b.Spec.InitContainers, c)
	return b
}

func (b *podBuilder) WithLabel(label, value string) *podBuilder {
	if b.Labels == nil {
		b.Labels = make(map[string]string)
	}
	b.Labels[label] = value
	return b
}

func (b *podBuilder) WithNodeName(nodeName string) *podBuilder {
	b.Spec.NodeName = nodeName
	return b
}

func (b *podBuilder) WithCreationTimestamp(timestamp metav1.Time) *podBuilder {
	b.CreationTimestamp = timestamp
	return b
}

func (b *podBuilder) WithPriority(priority int32) *podBuilder {
	b.Spec.Priority = &priority
	return b
}

func (b *podBuilder) Get() v1.Pod {
	return b.Pod
}

func BuildPod(namespace, name string) *podBuilder {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return &podBuilder{pod}
}

type containerBuilder struct {
	v1.Container
}

func (b *containerBuilder) WithLimits(limits v1.ResourceList) *containerBuilder {
	b.Resources.Limits = limits
	return b
}

func (b *containerBuilder) WithRequests(requests v1.ResourceList) *containerBuilder {
	b.Resources.Requests = requests
	return b
}

func (b *containerBuilder) WithCPUMilliLimit(cpuMilli int64) *containerBuilder {
	if b.Resources.Limits == nil {
		b.Resources.Limits = make(v1.ResourceList)
	}
	b.Resources.Limits[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithCPUMilliRequest(cpuMilli int64) *containerBuilder {
	if b.Resources.Requests == nil {
		b.Resources.Requests = make(v1.ResourceList)
	}
	b.Resources.Requests[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithNvidiaGPULimit(amount int64) *containerBuilder {
	if b.Resources.Limits == nil {
		b.Resources.Limits = make(v1.ResourceList)
	}
	b.Resources.Limits[constant.ResourceNvidiaGPU] = *resource.NewQuantity(amount, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithNvidiaGPURequest(amount int64) *containerBuilder {
	if b.Resources.Requests == nil {
		b.Resources.Requests = make(v1.ResourceList)
	}
	b.Resources.Requests[constant.ResourceNvidiaGPU] = *resource.NewQuantity(amount, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithScalarResourceLimit(resourceName v1.ResourceName, amount int) *containerBuilder {
	if b.Resources.Limits == nil {
		b.Resources.Limits = make(v1.ResourceList)
	}
	b.Resources.Limits[resourceName] = *resource.NewQuantity(int64(amount), resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithScalarResourceRequest(resourceName v1.ResourceName, amount int) *containerBuilder {
	if b.Resources.Requests == nil {
		b.Resources.Requests = make(v1.ResourceList)
	}
	b.Resources.Requests[resourceName] = *resource.NewQuantity(int64(amount), resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithResourceRequest(resourceName v1.ResourceName, quantity resource.Quantity) *containerBuilder {
	if b.Resources.Requests == nil {
		b.Resources.Requests = make(v1.ResourceList)
	}
	b.Resources.Requests[resourceName] = quantity
	return b
}

func (b *containerBuilder) Get() v1.Container {
	return b.Container
}

func BuildContainer(name, image string) *containerBuilder {
	c := v1.Container{
		Name:  name,
		Image: image,
	}
	return &containerBuilder{c}
}
