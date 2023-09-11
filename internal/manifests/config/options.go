package config

import "github.com/grafana/tempo-operator/internal/manifests/manifestutils"

// options holds the configuration template options.
type options struct {
	StorageType            string
	GlobalRetention        string
	QueryFrontendDiscovery string
	StorageParams          manifestutils.StorageParams
	GlobalRateLimits       rateLimitsOptions
	TenantRateLimitsPath   string
	TLS                    tlsOptions
	MemberList             []string
	Search                 searchOptions
	ReplicationFactor      int
	Multitenancy           bool
	Gateway                bool
	Gates                  featureGates
}

type tempoQueryOptions struct {
	Gates        featureGates
	TLS          tlsOptions
	HTTPPort     int
	TenantHeader string
	Gateway      bool
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
	MaxSearchDuration       string
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
	// Same as MinTLSVersion but instead of VersionTLS12 will be 1.2
	MinTLSVersionShort string
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
