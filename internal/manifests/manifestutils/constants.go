package manifestutils

const (
	// PrometheusCAFile declares the path for prometheus CA file for service monitors.
	PrometheusCAFile string = "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"
	// nolint #nosec
	// BearerTokenFile declares the path for bearer token file for service monitors.
	BearerTokenFile string = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// ConfigVolumeName declares the name of the volume containing the tempo configuration.
	ConfigVolumeName = "tempo-conf"

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
)
