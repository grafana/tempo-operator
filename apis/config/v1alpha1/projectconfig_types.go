package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"

	apiv1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API.
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`
	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	DefaultImages apiv1alpha1.ImagesSpec `json:"images"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
