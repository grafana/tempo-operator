package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	config "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// TempoOperatorConfigSpec defines the desired state of TempoOperatorConfig.
type TempoOperatorConfigSpec struct{}

// TempoOperatorConfigStatus defines the observed state of TempoOperatorConfig.
type TempoOperatorConfigStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TempoOperatorConfig is the Schema for the tempooperatorconfigs API.
type TempoOperatorConfig struct {
	Spec   TempoOperatorConfigSpec   `json:"spec,omitempty"`
	Status TempoOperatorConfigStatus `json:"status,omitempty"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	config.ControllerManagerConfigurationSpec `json:",inline"`
	metav1.TypeMeta                           `json:",inline"`
	metav1.ObjectMeta                         `json:"metadata,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectConfigList contains a list of ProjectConfig.
type TempoOperatorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TempoOperatorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TempoOperatorConfig{}, &TempoOperatorConfigList{})
}
