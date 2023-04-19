package config

// AzureStorage holds Azure Storage configuration options.
type azureStorage struct {
	Container string
}

// GCS holds Google Cloud Storage configuration options.
type gcs struct {
	Bucket string
}

// S3 holds S3 object storage configuration options.
type s3 struct {
	Endpoint string
	Bucket   string
	Insecure bool
}

// options holds the configuration template options.
type options struct {
	StorageType            string
	GlobalRetention        string
	QueryFrontendDiscovery string
	AzureStorage           azureStorage
	GCS                    gcs
	S3                     s3
	GlobalRateLimits       rateLimitsOptions
	TenantRateLimitsPath   string
	TLS                    tlsOptions
	MemberList             []string
	Search                 searchOptions
	ReplicationFactor      int
	Multitenancy           bool
	Gates                  featureGates
}

type tempoQueryOptions struct {
	Gates        featureGates
	TLS          tlsOptions
	HTTPPort     int
	TenantHeader string
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
	ServerNames serverNames
	Profile     tlsProfileOptions
}

// TLSProfileSpec is the desired behavior of a TLSProfileType.
type tlsProfileOptions struct {
	// Ciphers is used to specify the cipher algorithms that are negotiated
	// during the TLS handshake.
	Ciphers string
	// MinTLSVersion is used to specify the minimal version of the TLS protocol
	// that is negotiated during the TLS handshake.
	MinTLSVersion string
}

type tlsFilePaths struct {
	CA          string
	Certificate string
	Key         string
}

type serverNames struct {
	Compactor     string
	Ingester      string
	QueryFrontend string
	Querier       string
}
