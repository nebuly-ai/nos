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

package capacityscheduling

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type filterFunc func(obj interface{}) bool

var unstructuredFilter filterFunc = func(obj interface{}) bool {
	switch t := obj.(type) {
	case *unstructured.Unstructured:
		return true
	case cache.DeletedFinalStateUnknown:
		if _, ok := t.Obj.(*unstructured.Unstructured); ok {
			return true
		}
		utilruntime.HandleError(fmt.Errorf("cannot convert to *unstructured.Unstructured: %v", obj))
		return false
	default:
		utilruntime.HandleError(fmt.Errorf("unable to handle object in %T", obj))
		return false
	}
}

func NewElasticQuotaInfoInformer(kubeConfig *restclient.Config, resourceCalculator resource.Calculator) (*ElasticQuotaInfoInformer, error) {
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	sharedInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

	elasticQuotaGvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "elasticquotas",
	}
	compositeElasticQuotaGvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "compositeelasticquotas",
	}

	return &ElasticQuotaInfoInformer{
		compositeElasticQuotaInformer: sharedInformerFactory.ForResource(compositeElasticQuotaGvr),
		elasticQuotaInformer:          sharedInformerFactory.ForResource(elasticQuotaGvr),
		sharedInformerFactory:         sharedInformerFactory,
		resourceCalculator:            resourceCalculator,
	}, nil
}

// ElasticQuotaInfoInformer is a wrapper around ElasticQuota and CompositeElasticQuota informers that
// exposes their respective types as ElasticQuotaInfo
type ElasticQuotaInfoInformer struct {
	compositeElasticQuotaInformer informers.GenericInformer
	elasticQuotaInformer          informers.GenericInformer
	sharedInformerFactory         dynamicinformer.DynamicSharedInformerFactory
	resourceCalculator            resource.Calculator
}

func (i ElasticQuotaInfoInformer) Start(stopCh <-chan struct{}) {
	i.sharedInformerFactory.Start(stopCh)
}

func (i ElasticQuotaInfoInformer) HasSynced() bool {
	return i.elasticQuotaInformer.Informer().HasSynced() && i.compositeElasticQuotaInformer.Informer().HasSynced()
}

// GetAssociatedCompositeElasticQuota returns, if present, the CompositeElasticQuota to which the namespace provided
// as argument is subject to
func (i ElasticQuotaInfoInformer) GetAssociatedCompositeElasticQuota(namespace string) (*ElasticQuotaInfo, error) {
	objs, err := i.compositeElasticQuotaInformer.Lister().List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	for _, o := range objs {
		compositeEqInfo, err := i.fromUnstructuredCompositeEqToElasticQuotaInfo(o.(*unstructured.Unstructured))
		if err != nil {
			return nil, err
		}
		if util.InSlice(namespace, compositeEqInfo.Namespaces.List()) {
			return compositeEqInfo, nil
		}
	}

	return nil, nil
}

// GetAssociatedElasticQuota returns, if present, the ElasticQuotaInfo that sets the quota limits on the
// namespace provided as argument.
//
// If namespace is not associated with any ElasticQuota then nil is returned.
func (i ElasticQuotaInfoInformer) GetAssociatedElasticQuota(namespace string) (*ElasticQuotaInfo, error) {
	objs, err := i.elasticQuotaInformer.Lister().ByNamespace(namespace).List(labels.NewSelector())
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, nil
	}
	if len(objs) > 1 {
		return nil, fmt.Errorf("")
	}
	return i.fromUnstructuredEqToElasticQuotaInfo(objs[0].(*unstructured.Unstructured))
}

// AddEventHandler adds an event handler that receives events for both CompositeElasticQuota and ElasticQuota resources.
//
// The handler always receives ElasticQuotaInfo objects, no matters if the event was generated by
// an ElasticQuota or a CompositeElasticQuota resource.
//
// Events for ElasticQuota resources are sent only if the namespace to which the ElasticQuota belongs to is not
// subject to any CompositeElasticQuota. In such a case, the CompositeElasticQuota takes precedence and only its
// events are sent to the handler, while ElasticQuota events are ignored.
func (i ElasticQuotaInfoInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	i.elasticQuotaInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: i.excludeNamespacesSubjectToCompositeEqFilter(),
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					eqInfo, err := i.fromUnstructuredEqToElasticQuotaInfo(obj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert unstructured to ElasticQuota")
						return
					}
					handler.OnAdd(eqInfo)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					oldEqInfo, err := i.fromUnstructuredEqToElasticQuotaInfo(oldObj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert oldObj unstructured to ElasticQuota")
						return
					}
					newEqInfo, err := i.fromUnstructuredEqToElasticQuotaInfo(newObj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert newObj unstructured to ElasticQuota")
						return
					}
					handler.OnUpdate(oldEqInfo, newEqInfo)
				},
				DeleteFunc: func(obj interface{}) {
					eqInfo, err := i.fromUnstructuredEqToElasticQuotaInfo(obj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert unstructured to ElasticQuota")
						return
					}
					handler.OnDelete(eqInfo)
				},
			},
		},
	)

	i.compositeElasticQuotaInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: unstructuredFilter,
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					eqInfo, err := i.fromUnstructuredCompositeEqToElasticQuotaInfo(obj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert unstructured to CompositeElasticQuota")
						return
					}
					handler.OnAdd(eqInfo)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					oldEqInfo, err := i.fromUnstructuredCompositeEqToElasticQuotaInfo(oldObj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert oldObj unstructured to CompositeElasticQuota")
						return
					}
					newEqInfo, err := i.fromUnstructuredCompositeEqToElasticQuotaInfo(newObj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert newObj unstructured to CompositeElasticQuota")
						return
					}
					handler.OnUpdate(oldEqInfo, newEqInfo)
				},
				DeleteFunc: func(obj interface{}) {
					eqInfo, err := i.fromUnstructuredCompositeEqToElasticQuotaInfo(obj.(*unstructured.Unstructured))
					if err != nil {
						klog.ErrorS(err, "unable to convert unstructured to CompositeElasticQuota")
						return
					}
					handler.OnDelete(eqInfo)
				},
			},
		},
	)
}

// excludeNamespacesSubjectToCompositeEqFilter returns a filter function that excludes all the resources belonging to
// a namespace subject to any CompositeElasticQuota
func (i ElasticQuotaInfoInformer) excludeNamespacesSubjectToCompositeEqFilter() filterFunc {
	return func(obj interface{}) bool {
		if isUnstructured := unstructuredFilter(obj); !isUnstructured {
			return false
		}
		unstructuredObj := obj.(*unstructured.Unstructured)
		namespaces, err := i.getNamespacesSubjectToCompositeEq()
		if err != nil {
			klog.ErrorS(err, "unable to get all namespaces subject to any CompositeElasticQuota")
			return false
		}
		if util.InSlice(unstructuredObj.GetNamespace(), namespaces.List()) {
			return false
		}
		return true
	}
}

// getNamespacesSubjectToCompositeEq returns the namespaces which are subject to any CompositeElasticQuota resource
func (i ElasticQuotaInfoInformer) getNamespacesSubjectToCompositeEq() (sets.String, error) {
	var result = sets.NewString()
	objList, err := i.compositeElasticQuotaInformer.Lister().List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	var compositeEq v1alpha1.CompositeElasticQuota
	for _, obj := range objList {
		unstructObj := obj.(*unstructured.Unstructured)
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.UnstructuredContent(), &compositeEq); err != nil {
			return nil, err
		}
		result.Insert(compositeEq.Spec.Namespaces...)
	}
	return result, nil
}

// fromUnstructuredEqToElasticQuotaInfo converts an Unstructured object containing an ElasticQuota to
// an object of type ElasticQuotaInfo
func (i ElasticQuotaInfoInformer) fromUnstructuredEqToElasticQuotaInfo(u *unstructured.Unstructured) (*ElasticQuotaInfo, error) {
	var eq v1alpha1.ElasticQuota
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &eq); err != nil {
		return nil, err
	}
	return &ElasticQuotaInfo{
		ResourceName:       eq.Name,
		ResourceNamespace:  eq.Namespace,
		Namespaces:         sets.NewString(eq.Namespace),
		pods:               sets.NewString(),
		Min:                framework.NewResource(eq.Spec.Min),
		Max:                framework.NewResource(eq.Spec.Max),
		Used:               framework.NewResource(nil), // used is calculated by the scheduler plugin afterwards
		MaxEnforced:        eq.Spec.Max != nil,
		resourceCalculator: i.resourceCalculator,
	}, nil
}

// fromUnstructuredCompositeEqToElasticQuotaInfo converts an Unstructured object containing a CompositeElasticQuota to
// an object of type ElasticQuotaInfo
func (i ElasticQuotaInfoInformer) fromUnstructuredCompositeEqToElasticQuotaInfo(u *unstructured.Unstructured) (*ElasticQuotaInfo, error) {
	var compositeEq v1alpha1.CompositeElasticQuota
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &compositeEq); err != nil {
		return nil, err
	}
	return &ElasticQuotaInfo{
		ResourceName:       compositeEq.Name,
		ResourceNamespace:  compositeEq.Namespace,
		Namespaces:         sets.NewString(compositeEq.Spec.Namespaces...),
		pods:               sets.NewString(),
		Min:                framework.NewResource(compositeEq.Spec.Min),
		Max:                framework.NewResource(compositeEq.Spec.Max),
		Used:               framework.NewResource(nil), // used is calculated by the scheduler plugin afterwards
		MaxEnforced:        compositeEq.Spec.Max != nil,
		resourceCalculator: i.resourceCalculator,
	}, nil
}
