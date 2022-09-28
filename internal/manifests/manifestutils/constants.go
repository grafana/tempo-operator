package manifestutils

const (
	// PrometheusCAFile declares the path for prometheus CA file for service monitors.
	PrometheusCAFile string = "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"
	// nolint #nosec
	// BearerTokenFile declares the path for bearer token file for service monitors.
	BearerTokenFile string = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)
