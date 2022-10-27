package config

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
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
	s3Insecure := false
	s3Endpoint := params.S3.Endpoint
	if strings.HasPrefix(s3Endpoint, "http://") {
		s3Insecure = true
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "http://")
	} else if !strings.HasPrefix(s3Endpoint, "https://") {
		s3Insecure = true
	} else {
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "https://")
	}
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
		GlobalRetention: tempo.Spec.Retention.Global.Traces.String(),
		MemberList: []string{
			manifestutils.Name("gossip-ring", tempo.Name),
		},
		QueryFrontendDiscovery: fmt.Sprintf("%s:9095", manifestutils.Name("query-frontend-discovery", tempo.Name)),
		GlobalRateLimits:       fromRateLimitSpecToRateLimitOptions(tempo.Spec.LimitSpec.Global),
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
