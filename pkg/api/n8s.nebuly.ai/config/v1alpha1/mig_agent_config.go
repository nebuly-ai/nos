package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"time"
)

// +kubebuilder:object:root=true

type MigAgentConfig struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
	ReportConfigIntervalSeconds            time.Duration `json:"reportConfigIntervalSeconds"`
}
