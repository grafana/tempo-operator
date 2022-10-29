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
	MemberList             []string
	S3                     s3
}
