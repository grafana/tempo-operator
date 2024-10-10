package v1alpha1

type (
	// IngressType represents how a service should be exposed (ingress vs route).
	// +kubebuilder:validation:Enum=ingress;route;""
	// +kubebuilder:default=""
	IngressType string
)

const (
	// IngressTypeNone specifies that no ingress or route entry should be created.
	IngressTypeNone IngressType = ""
	// IngressTypeIngress specifies that an ingress entry should be created.
	IngressTypeIngress IngressType = "ingress"
	// IngressTypeRoute specifies that a route entry should be created.
	IngressTypeRoute IngressType = "route"
)

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
