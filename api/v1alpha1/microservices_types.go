package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MicroservicesSpec defines the desired state of Microservices
type MicroservicesSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Microservices. Edit microservices_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MicroservicesStatus defines the observed state of Microservices
type MicroservicesStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Microservices is the Schema for the microservices API
type Microservices struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroservicesSpec   `json:"spec,omitempty"`
	Status MicroservicesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MicroservicesList contains a list of Microservices
type MicroservicesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservices `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Microservices{}, &MicroservicesList{})
}
