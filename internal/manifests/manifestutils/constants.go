package manifestutils

const (
	// PrometheusCAFile declares the path for prometheus CA file for service monitors.
	PrometheusCAFile string = "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"
	// nolint #nosec
	// BearerTokenFile declares the path for bearer token file for service monitors.
	BearerTokenFile string = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// ConfigVolumeName declares the name of the volume containing the tempo configuration.
	ConfigVolumeName = "tempo-conf"

	// HttpPortName declares the name of the tempo http port.
	HttpPortName = "http"
	// PortHTTPServer declares the port number of the tempo http port.
	PortHTTPServer = 3100

	// GrpcPortName declares the name of the tempo gRPC port.
	GrpcPortName = "grpc"
	// PortGRPCServer declares the port number of the tempo gRPC port.
	PortGRPCServer = 9095

	// OtlpGrpcPortName declares the name of the OpenTelemetry Collector gRPC receiver port.
	OtlpGrpcPortName = "otlp-grpc"
	// PortOtlpGrpcServer declares the port number of the OpenTelemetry Collector gRPC receiver port.
	PortOtlpGrpcServer = 4317

	// HttpMemberlistPortName declares the name of the tempo memberlist port.
	HttpMemberlistPortName = "http-memberlist"
	// PortMemberlist declares the port number of the tempo memberlist port.
	PortMemberlist = 7946
)
