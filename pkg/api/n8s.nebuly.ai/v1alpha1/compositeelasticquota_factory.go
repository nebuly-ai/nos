package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type compositeEqBuilder struct {
	CompositeElasticQuota
}

func (e *compositeEqBuilder) WithNamespaces(namespaces ...string) *compositeEqBuilder {
	e.CompositeElasticQuota.Spec.Namespaces = namespaces
	return e
}

func (e *compositeEqBuilder) WithMin(min v1.ResourceList) *compositeEqBuilder {
	e.CompositeElasticQuota.Spec.Min = min
	return e
}

func (e *compositeEqBuilder) WithMax(max v1.ResourceList) *compositeEqBuilder {
	e.CompositeElasticQuota.Spec.Max = max
	return e
}

func (e *compositeEqBuilder) WithMinGPUMemory(gpuMemory int64) *compositeEqBuilder {
	if e.CompositeElasticQuota.Spec.Min == nil {
		e.CompositeElasticQuota.Spec.Min = make(v1.ResourceList)
	}
	e.CompositeElasticQuota.Spec.Min[ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)
	return e
}

func (e *compositeEqBuilder) WithMaxGPUMemory(gpuMemory int64) *compositeEqBuilder {
	if e.CompositeElasticQuota.Spec.Max == nil {
		e.CompositeElasticQuota.Spec.Max = make(v1.ResourceList)
	}
	e.CompositeElasticQuota.Spec.Max[ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)
	return e
}

func (e *compositeEqBuilder) WithMinCPUMilli(cpuMilli int64) *compositeEqBuilder {
	if e.CompositeElasticQuota.Spec.Min == nil {
		e.CompositeElasticQuota.Spec.Min = make(v1.ResourceList)
	}
	e.CompositeElasticQuota.Spec.Min[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return e
}

func (e *compositeEqBuilder) WithMaxCPUMilli(cpuMilli int64) *compositeEqBuilder {
	if e.CompositeElasticQuota.Spec.Max == nil {
		e.CompositeElasticQuota.Spec.Max = make(v1.ResourceList)
	}
	e.CompositeElasticQuota.Spec.Max[v1.ResourceCPU] = *resource.NewMilliQuantity(cpuMilli, resource.DecimalSI)
	return e
}

func (e *compositeEqBuilder) Get() CompositeElasticQuota {
	return e.CompositeElasticQuota
}

func BuildCompositeEq(namespace, name string) *compositeEqBuilder {
	eq := CompositeElasticQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CompositeElasticQuota",
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return &compositeEqBuilder{eq}
}
