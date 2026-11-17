package manifestutils

// TokenCCOAWSEnvironment expose AWS settings when using CCO.
type TokenCCOAWSEnvironment struct {
	RoleARN string
	// Partition is the ID of the AWS partition the role belongs to, e.g. aws-iso.
	// An empty value is treated as the commercial partition.
	Partition string
}

// TokenCCOAuthConfig CCO token config.
type TokenCCOAuthConfig struct {
	AWS *TokenCCOAWSEnvironment
}
