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

// OptimizationTarget foo
//+kubebuilder:validation:Enum=Latency;Cost;Emissions
type OptimizationTarget string

const (
	OptimizationTargetLatency   OptimizationTarget = "Latency"
	OptimizationTargetCost                         = "Cost"
	OptimizationTargetEmissions                    = "Emissions"
)

type StatusState string

const (
	StatusStateOptimizingModel StatusState = "OptimizingModel"
	StatusStateDeployingModel              = "DeployingModel"
	StatusStateAvailable                   = "Available"
	StatusStateFailed                      = "Failed"
)

type OptimizationSpec struct {
	// OptimizationTarget specifies the target for which the model that has to be deployed will be optimized for
	Target OptimizationTarget `json:"target"`
	// ModelOptimizerImageName is the name of the Docker image of the inference optimization service that will be
	// used for optimizing the model
	//+kubebuilder:default="nebuly.ai/model-optimizer"
	//+optional
	ModelOptimizerImageName string `json:"modelOptimizerImageName,omitempty"`
	// ModelOptimizerImageVersion is the version of the Docker image of the inference optimization service that
	// will be used for optimizing the model
	//+kubebuilder:default="0.0.1"
	//+optional
	ModelOptimizerImageVersion string `json:"modelOptimizerImageVersion,omitempty"`
	// OptimizationJobBackoffLimit is the number of retries before declaring an optimization job failed
	//+kubebuilder:default=1
	//+optional
	OptimizationJobBackoffLimit int8 `json:"optimizationJobBackoffLimit"`
}

type SourceModel struct {
	// Uri is a URI pointing to the model that has to be deployed
	Uri string `json:"uri"`
}

// ModelDeploymentSpec defines the desired state of ModelDeployment
type ModelDeploymentSpec struct {
	// Optimization defines the configuration of the model optimization
	Optimization OptimizationSpec `json:"optimization"`
	// SourceModel
	SourceModel SourceModel `json:"sourceModel"`
}

// ModelDeploymentStatus defines the observed state of ModelDeployment
type ModelDeploymentStatus struct {
	ModelOptimizationJob v1.ObjectReference `json:"modelOptimizationJob"`
	State                StatusState        `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ModelDeployment is the Schema for the modeldeployments API
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ModelDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelDeploymentSpec   `json:"spec,omitempty"`
	Status ModelDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ModelDeploymentList contains a list of ModelDeployment
type ModelDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelDeployment{}, &ModelDeploymentList{})
}
