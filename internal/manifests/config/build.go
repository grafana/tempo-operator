package config

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

var (
	//go:embed tempo-config.yaml
	tempoConfigYAMLTmplFile embed.FS
	tempoConfigYAMLTmpl     = template.Must(template.ParseFS(tempoConfigYAMLTmplFile, "tempo-config.yaml"))

	//go:embed tempo-overrides.yaml
	tempoTenantsOverridesYAMLTmplFile embed.FS
	tempoTenantsOverridesYAMLTmpl     = template.Must(template.ParseFS(tempoTenantsOverridesYAMLTmplFile, "tempo-overrides.yaml"))
)

func s3FromParams(params Params) s3 {
	s3Endpoint := params.S3.Endpoint
	s3Insecure := !strings.HasPrefix(s3Endpoint, "https://")
	s3Endpoint = strings.TrimPrefix(s3Endpoint, "https://")
	s3Endpoint = strings.TrimPrefix(s3Endpoint, "http://")

	return s3{
		Endpoint: s3Endpoint,
		Bucket:   params.S3.Bucket,
		Insecure: s3Insecure,
	}
}

func fromRateLimitSpecToRateLimitOptions(spec v1alpha1.RateLimitSpec) rateLimitsOptions {
	return rateLimitsOptions{
		IngestionRateLimitBytes: spec.Ingestion.IngestionRateLimitBytes,
		IngestionBurstSizeBytes: spec.Ingestion.IngestionBurstSizeBytes,
		MaxBytesPerTrace:        spec.Ingestion.MaxBytesPerTrace,
		MaxTracesPerUser:        spec.Ingestion.MaxTracesPerUser,
		MaxBytesPerTagValues:    spec.Query.MaxBytesPerTagValues,
		MaxSearchBytesPerTrace:  spec.Query.MaxSearchBytesPerTrace,
	}
}

func fromRateLimitSpecToRateLimitOptionsMap(ratemaps map[string]v1alpha1.RateLimitSpec) map[string]rateLimitsOptions {
	result := make(map[string]rateLimitsOptions, len(ratemaps))
	for tenant, spec := range ratemaps {
		result[tenant] = fromRateLimitSpecToRateLimitOptions(spec)
	}
	return result
}

func buildConfiguration(tempo v1alpha1.Microservices, params Params) ([]byte, error) {
	opts := options{
		S3:              s3FromParams(params),
		GlobalRetention: tempo.Spec.Retention.Global.Traces.Duration.String(),
		MemberList: []string{
			naming.Name("gossip-ring", tempo.Name),
		},
		QueryFrontendDiscovery: fmt.Sprintf("%s:9095", naming.Name("query-frontend-discovery", tempo.Name)),
		GlobalRateLimits:       fromRateLimitSpecToRateLimitOptions(tempo.Spec.LimitSpec.Global),
		Search:                 fromSearchSpecToOptions(tempo.Spec.SearchSpec),
	}

	if isTenantOverridesConfigRequired(tempo.Spec.LimitSpec) {
		opts.TenantRateLimitsPath = tenantOverridesMountPath
	}

	return renderTemplate(opts)
}

func isTenantOverridesConfigRequired(limitSpec v1alpha1.LimitSpec) bool {
	return len(limitSpec.PerTenant) > 0
}

func buildTenantOverrides(tempo v1alpha1.Microservices) ([]byte, error) {
	return renderTenantOverridesTemplate(tenantOptions{
		RateLimits: fromRateLimitSpecToRateLimitOptionsMap(tempo.Spec.LimitSpec.PerTenant),
	})
}

func renderTemplate(opts options) ([]byte, error) {
	// Build Tempo config yaml
	w := bytes.NewBuffer(nil)
	err := tempoConfigYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, err
	}
	cfg, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func renderTenantOverridesTemplate(opts tenantOptions) ([]byte, error) {
	// Build tempo tenant overrides config yaml
	w := bytes.NewBuffer(nil)
	err := tempoTenantsOverridesYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, err
	}
	cfg, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func fromSearchSpecToOptions(spec v1alpha1.SearchSpec) searchOptions {
	options := searchOptions{
		// Those are recommended defaults taken from: https://grafana.com/docs/tempo/latest/operations/backend_search/
		// some of them could depend on the volumen and retention of the data, need to figure out how to set it.
		ExternalHedgeRequestsUpTo: 2,
		ConcurrentJobs:            2000,
		MaxConcurrentQueries:      20,
		ExternalHedgeRequestsAt:   "8s",
		// If not specified, will be zero,  means disable limit by default
		MaxSearchDuration: spec.MaxSearchDuration,
	}
	return options
}
