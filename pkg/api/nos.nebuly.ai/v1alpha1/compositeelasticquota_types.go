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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName={ceq,ceqs}
//+kubebuilder:subresource:status
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CompositeElasticQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// CompositeElasticQuotaSpec defines the Min and Max for Quota.
	Spec CompositeElasticQuotaSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// CompositeElasticQuotaStatus defines the observed use.
	Status CompositeElasticQuotaStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type CompositeElasticQuotaSpec struct {
	// Namespaces is the desired list of namespaces in which the specified limits will be enforced
	//+kubebuilder:validation:MinItems:=1
	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,1,rep,name=namespaces"`

	// Min is the set of desired guaranteed limits for each named resource.
	Min v1.ResourceList `json:"min,omitempty" protobuf:"bytes,1,rep,name=min, casttype=ResourceList,castkey=ResourceName"`

	// Max is the set of desired max limits for each named resource. The usage of max is based on the resource configurations of
	// successfully scheduled pods.
	Max v1.ResourceList `json:"max,omitempty" protobuf:"bytes,2,rep,name=max, casttype=ResourceList,castkey=ResourceName"`
}

type CompositeElasticQuotaStatus struct {
	// Used is the current observed total usage of the resource in the namespace.
	Used v1.ResourceList `json:"used,omitempty" protobuf:"bytes,1,rep,name=used,casttype=ResourceList,castkey=ResourceName"`
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CompositeElasticQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []CompositeElasticQuota `json:"items" protobuf:"bytes,2,rep,name=items"`
}
