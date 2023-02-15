package config

// S3 holds S3 object storage configuration options.
type s3 struct {
	Endpoint string
	Bucket   string
	Insecure bool
}

// options holds the configuration template options.
type options struct {
	GlobalRetention        string
	QueryFrontendDiscovery string
	S3                     s3
	GlobalRateLimits       rateLimitsOptions
	TenantRateLimitsPath   string
	TLS                    tlsOptions
	MemberList             []string
	Search                 searchOptions
	ReplicationFactor      int
	Gates                  featureGates
}

type tempoQueryOptions struct {
	Gates    featureGates
	TLS      tlsOptions
	HTTPPort int
}

type featureGates struct {
	HTTPEncryption bool
	GRPCEncryption bool
}

type tenantOptions struct {
	RateLimits map[string]rateLimitsOptions
}

type rateLimitsOptions struct {
	IngestionBurstSizeBytes *int
	IngestionRateLimitBytes *int
	MaxBytesPerTrace        *int
	MaxTracesPerUser        *int
	MaxBytesPerTagValues    *int
	MaxSearchBytesPerTrace  *int
}

type searchOptions struct {
	MaxDuration               string
	QueryTimeout              string
	ExternalHedgeRequestsAt   string
	ExternalHedgeRequestsUpTo int
	ConcurrentJobs            int
	MaxConcurrentQueries      int
	DefaultResultLimit        int
	MaxResultLimit            int
	Enabled                   bool
}

type tlsOptions struct {
	Enabled     bool
	Paths       tlsFilePaths
	ServerNames tlsServerNames
}

type tlsFilePaths struct {
	CA   string
	GRPC tlsCertPath
	HTTP tlsCertPath
}

type tlsCertPath struct {
	Certificate string
	Key         string
}

type tlsServerNames struct {
	GRPC grpcServerNames
	HTTP httpServerNames
}

type grpcServerNames struct {
	Compactor     string
	Ingester      string
	QueryFrontend string
}

type httpServerNames struct {
	Querier       string
	QueryFrontend string
}
