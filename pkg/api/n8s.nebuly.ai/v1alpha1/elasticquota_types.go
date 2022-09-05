package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName={eq,eqs}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticQuota sets elastic quota restrictions per namespace
type ElasticQuota struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// ElasticQuotaSpec defines the Min and Max for Quota.
	// +optional
	Spec ElasticQuotaSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// ElasticQuotaStatus defines the observed use.
	// +optional
	Status ElasticQuotaStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ElasticQuotaSpec defines the Min and Max for Quota.
type ElasticQuotaSpec struct {
	// Min is the set of desired guaranteed limits for each named resource.
	// +optional
	Min v1.ResourceList `json:"min,omitempty" protobuf:"bytes,1,rep,name=min, casttype=ResourceList,castkey=ResourceName"`

	// Max is the set of desired max limits for each named resource. The usage of max is based on the resource configurations of
	// successfully scheduled pods.
	// +optional
	Max v1.ResourceList `json:"max,omitempty" protobuf:"bytes,2,rep,name=max, casttype=ResourceList,castkey=ResourceName"`
}

// ElasticQuotaStatus defines the observed use.
type ElasticQuotaStatus struct {
	// Used is the current observed total usage of the resource in the namespace.
	// +optional
	Used v1.ResourceList `json:"used,omitempty" protobuf:"bytes,1,rep,name=used,casttype=ResourceList,castkey=ResourceName"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticQuotaList is a list of ElasticQuota items.
type ElasticQuotaList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of ElasticQuota objects.
	Items []ElasticQuota `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func init() {
	SchemeBuilder.Register(&ElasticQuota{}, &ElasticQuotaList{})
}
