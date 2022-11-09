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
	MemberList             []string
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
