package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API.
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`
	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
