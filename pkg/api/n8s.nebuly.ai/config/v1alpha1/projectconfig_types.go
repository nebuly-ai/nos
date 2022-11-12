package v1alpha1

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

//+kubebuilder:object:root=true

type OperatorConfig struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
	NvidiaGPUResourceMemoryGB              *int64 `json:"nvidiaGPUResourceMemoryGB,omitempty"`
}

func (c *OperatorConfig) FillDefaultValues() {
	if c.NvidiaGPUResourceMemoryGB == nil {
		var defaultValue int64 = constant.DefaultNvidiaGPUResourceMemory
		c.NvidiaGPUResourceMemoryGB = &defaultValue
	}
}

// +kubebuilder:object:root=true

type GpuPartitionerConfig struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
}

func init() {
	SchemeBuilder.Register(&OperatorConfig{})
	SchemeBuilder.Register(&GpuPartitionerConfig{})
}
