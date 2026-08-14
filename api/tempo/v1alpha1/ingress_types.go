package v1alpha1

type (
	// IngressType represents how a service should be exposed (ingress vs route).
	// +kubebuilder:validation:Enum=ingress;route;none;""
	// +kubebuilder:default=""
	IngressType string
)

const (
	// IngressTypeNone specifies that no ingress or route entry should be created.
	IngressTypeNone IngressType = "none"
	// IngressTypeUnspecified specifies that ingress or route entry should only be created if necessary.
	IngressTypeUnspecified IngressType = ""
	// IngressTypeIngress specifies that an ingress entry should be created.
	IngressTypeIngress IngressType = "ingress"
	// IngressTypeRoute specifies that a route entry should be created.
	IngressTypeRoute IngressType = "route"
)

// IsEnabled returns true if the ingress type is explicitly set to a specific resource.
func (i IngressType) IsEnabled() bool {
	switch i {
	case IngressTypeNone, IngressTypeUnspecified:
		return false
	case IngressTypeIngress, IngressTypeRoute:
		return true
	}
	return false
}

type (
	// TLSRouteTerminationType is used to indicate which TLS settings should be used.
	// +kubebuilder:validation:Enum=insecure;edge;passthrough;reencrypt
	TLSRouteTerminationType string
)

const (
	// TLSRouteTerminationTypeInsecure indicates that insecure connections are allowed.
	TLSRouteTerminationTypeInsecure TLSRouteTerminationType = "insecure"
	// TLSRouteTerminationTypeEdge indicates that encryption should be terminated
	// at the edge router.
	TLSRouteTerminationTypeEdge TLSRouteTerminationType = "edge"
	// TLSRouteTerminationTypePassthrough indicates that the destination service is
	// responsible for decrypting traffic.
	TLSRouteTerminationTypePassthrough TLSRouteTerminationType = "passthrough"
	// TLSRouteTerminationTypeReencrypt indicates that traffic will be decrypted on the edge
	// and re-encrypt using a new certificate.
	TLSRouteTerminationTypeReencrypt TLSRouteTerminationType = "reencrypt"
)
