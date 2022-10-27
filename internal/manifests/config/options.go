package config

// S3 holds S3 object storage configuration options.
type S3Options struct {
	Endpoint string
	Bucket   string
	Insecure bool
}

// Options holds the configuration template options.
type Options struct {
	GlobalRetention        string
	QueryFrontendDiscovery string
	MemberList             []string
	S3                     S3Options
}
