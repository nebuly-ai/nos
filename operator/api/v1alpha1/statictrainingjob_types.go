/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StaticTrainingJobSpec defines the desired state of StaticTrainingJob
type StaticTrainingJobSpec struct {
	Spec             v1.PodSpec `json:"spec,omitempty"`
	RequiredHardware []string   `json:"requiredHardware"`
}

// StaticTrainingJobStatus defines the observed state of StaticTrainingJob
type StaticTrainingJobStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// StaticTrainingJob is the Schema for the statictrainingjobs API
type StaticTrainingJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StaticTrainingJobSpec   `json:"spec,omitempty"`
	Status StaticTrainingJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StaticTrainingJobList contains a list of StaticTrainingJob
type StaticTrainingJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StaticTrainingJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StaticTrainingJob{}, &StaticTrainingJobList{})
}
