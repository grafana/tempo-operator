package manifestutils

// TokenCCOAWSEnvironment expose AWS settings when using CCO.
type TokenCCOAWSEnvironment struct {
	RoleARN string
}

// TokenCCOAuthConfig CCO token config.
type TokenCCOAuthConfig struct {
	AWS *TokenCCOAWSEnvironment
}
