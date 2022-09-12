package factory

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceBuilder struct {
	v1.Namespace
}

func (b *namespaceBuilder) Get() v1.Namespace {
	return b.Namespace
}

func BuildNamespace(name string) *namespaceBuilder {
	namespace := v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
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

func (b *podBuilder) WithLabel(label, value string) *podBuilder {
	if b.Labels == nil {
		b.Labels = make(map[string]string)
	}
	b.Labels[label] = value
	return b
}

func (b *podBuilder) WithCreationTimestamp(timestamp metav1.Time) *podBuilder {
	b.Pod.CreationTimestamp = timestamp
	return b
}

func (b *podBuilder) WithPriority(priority int32) *podBuilder {
	b.Pod.Spec.Priority = &priority
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
	b.Container.Resources.Limits = limits
	return b
}

func (b *containerBuilder) WithRequests(requests v1.ResourceList) *containerBuilder {
	b.Container.Resources.Requests = requests
	return b
}

func (b *containerBuilder) WithCPUMilliLimit(cpuMilli int64) *containerBuilder {
	if b.Container.Resources.Limits == nil {
		b.Container.Resources.Limits = make(v1.ResourceList)
	}
	b.Container.Resources.Limits[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithCPUMilliRequest(cpuMilli int64) *containerBuilder {
	if b.Container.Resources.Requests == nil {
		b.Container.Resources.Requests = make(v1.ResourceList)
	}
	b.Container.Resources.Requests[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithGPUMemoryLimit(gpuMemory int64) *containerBuilder {
	if b.Container.Resources.Limits == nil {
		b.Container.Resources.Limits = make(v1.ResourceList)
	}
	b.Container.Resources.Limits[constant.ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)
	return b
}

func (b *containerBuilder) WithGPUMemoryRequest(gpuMemory int64) *containerBuilder {
	if b.Container.Resources.Requests == nil {
		b.Container.Resources.Requests = make(v1.ResourceList)
	}
	b.Container.Resources.Requests[constant.ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)
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
