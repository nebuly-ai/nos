package v1beta3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+k8s:defaulter-gen=true

type CapacitySchedulingArgs struct {
	metav1.TypeMeta `json:",inline"`

	NvidiaGPUResourceMemoryGB *int64 `json:"nvidiaGPUResourceMemoryGB,omitempty"`
}
