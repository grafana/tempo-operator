package manifestutils

const (
	// PrometheusCAFile declares the path for prometheus CA file for service monitors.
	PrometheusCAFile string = "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"
	// nolint #nosec
	// BearerTokenFile declares the path for bearer token file for service monitors.
	BearerTokenFile string = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	ConfigVolumeName = "tempo-conf"

	HttpPortName   = "http"
	PortHTTPServer = 3100

	GrpcPortName   = "grpc"
	PortGRPCServer = 9095

	OtlpGrpcPortName   = "otlp-grpc"
	PortOtlpGrpcServer = 4317

	HttpMemberlistPortName = "http-memberlist"
	PortMemberlist         = 7946
)
