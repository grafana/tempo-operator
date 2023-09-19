package manifestutils

const (
	// PrometheusCAFile declares the path for prometheus CA file for service monitors.
	PrometheusCAFile string = "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"
	// nolint #nosec
	// BearerTokenFile declares the path for bearer token file for service monitors.
	BearerTokenFile string = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// ConfigVolumeName declares the name of the volume containing the tempo configuration.
	ConfigVolumeName = "tempo-conf"

	// GatewayTenantFileName the name of the tenant config file in the secret.
	GatewayTenantFileName = "tenants.yaml"

	// TmpStorageVolumeName declares the name of the volume containing temporary storage for tempo.
	TmpStorageVolumeName = "tempo-tmp-storage"

	// TmpStoragePath declares the path of temporary storage for tempo.
	TmpStoragePath = "/var/tempo"

	// HttpPortName declares the name of the tempo http port.
	HttpPortName = "http"
	// PortHTTPServer declares the port number of the tempo http port.
	PortHTTPServer = 3200
	// PortInternalHTTPServer declares the port number of the tempo http port.
	PortInternalHTTPServer = 3101
	// TempoReadinessPath specifies the path for the readiness probe.
	TempoReadinessPath = "/ready"
	// TempoLivenessPath specifies the path for the liveness probe.
	TempoLivenessPath = "/status/version"

	// GrpcPortName declares the name of the tempo gRPC port.
	GrpcPortName = "grpc"
	// PortGRPCServer declares the port number of the tempo gRPC port.
	PortGRPCServer = 9095

	// OtlpGrpcPortName declares the name of the OpenTelemetry Collector gRPC receiver port.
	OtlpGrpcPortName = "otlp-grpc"
	// PortOtlpGrpcServer declares the port number of the OpenTelemetry Collector gRPC receiver port.
	PortOtlpGrpcServer = 4317

	// PortOtlpHttpName declares the port name of the OpenTelemetry protocol over HTTP.
	PortOtlpHttpName = "otlp-http"
	// PortOtlpHttp declares the port number of the OpenTelemetry protocol over HTTP.
	PortOtlpHttp = 4318

	// PortJaegerThriftHTTPName declares the port name of the Jaeger Thrift HTTP protocol.
	PortJaegerThriftHTTPName = "thrift-http"
	// PortJaegerThriftHTTP declares the port number of the Jaeger Thrift HTTP protocol.
	PortJaegerThriftHTTP = 14268

	// PortJaegerThriftCompactName declares the port name of the Jaeger Thrift compact protocol.
	PortJaegerThriftCompactName = "thrift-compact"
	// PortJaegerThriftCompact declares the port number of the Jaeger Thrift compact protocol.
	PortJaegerThriftCompact = 6831

	// PortJaegerThriftBinaryName declares the port name of the Jaeger Thrift binary protocol.
	PortJaegerThriftBinaryName = "thrift-binary"
	// PortJaegerThriftBinary declares the port number of the Jaeger Thrift binary protocol.
	PortJaegerThriftBinary = 6832

	// PortJaegerGrpcName declares the port number of the Jaeger gRPC port.
	PortJaegerGrpcName = "jaeger-grpc"
	// PortJaegerGrpc declares the port number of the Jaeger gRPC port.
	PortJaegerGrpc = 14250

	// PortZipkinName declares the port number of zipkin receiver port.
	PortZipkinName = "http-zipkin"
	// PortZipkin declares the port number of zipkin receiver port.
	PortZipkin = 9411

	// HttpMemberlistPortName declares the name of the tempo memberlist port.
	HttpMemberlistPortName = "http-memberlist"
	// PortMemberlist declares the port number of the tempo memberlist port.
	PortMemberlist = 7946

	// CompactorComponentName declares the internal name of the compactor component.
	CompactorComponentName = "compactor"
	// QuerierComponentName declares the internal name of the querier component.
	QuerierComponentName = "querier"
	// DistributorComponentName declares the internal name of the distributor component.
	DistributorComponentName = "distributor"
	// QueryFrontendComponentName declares the internal name of the query-frontend component.
	QueryFrontendComponentName = "query-frontend"
	// IngesterComponentName declares the internal name of the ingester component.
	IngesterComponentName = "ingester"
	// GatewayComponentName declares the internal name of the gateway component.
	GatewayComponentName = "gateway"
	// TenantHeader is the header name that contains tenant name.
	TenantHeader = "x-scope-orgid"
)
