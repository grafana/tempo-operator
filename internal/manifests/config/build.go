package config

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

var (
	//go:embed tempo-config.yaml
	tempoConfigYAMLTmplFile embed.FS
	tempoConfigYAMLTmpl     = template.Must(template.ParseFS(tempoConfigYAMLTmplFile, "tempo-config.yaml"))

	//go:embed tempo-overrides.yaml
	tempoTenantsOverridesYAMLTmplFile embed.FS
	tempoTenantsOverridesYAMLTmpl     = template.Must(template.ParseFS(tempoTenantsOverridesYAMLTmplFile, "tempo-overrides.yaml"))

	//go:embed tempo-query.yaml
	tempoQueryYAMLTmplFile embed.FS
	tempoQueryYAMLTmpl     = template.Must(template.ParseFS(tempoQueryYAMLTmplFile, "tempo-query.yaml"))
)

func fromRateLimitSpecToRateLimitOptions(spec v1alpha1.RateLimitSpec) rateLimitsOptions {
	return rateLimitsOptions{
		IngestionRateLimitBytes: spec.Ingestion.IngestionRateLimitBytes,
		IngestionBurstSizeBytes: spec.Ingestion.IngestionBurstSizeBytes,
		MaxBytesPerTrace:        spec.Ingestion.MaxBytesPerTrace,
		MaxTracesPerUser:        spec.Ingestion.MaxTracesPerUser,
		MaxBytesPerTagValues:    spec.Query.MaxBytesPerTagValues,
		MaxSearchDuration:       spec.Query.MaxSearchDuration.Duration.String(),
	}
}

func fromRateLimitSpecToRateLimitOptionsMap(ratemaps map[string]v1alpha1.RateLimitSpec) map[string]rateLimitsOptions {
	result := make(map[string]rateLimitsOptions, len(ratemaps))
	for tenant, spec := range ratemaps {
		result[tenant] = fromRateLimitSpecToRateLimitOptions(spec)
	}
	return result
}

func buildQueryFrontEndConfig(params manifestutils.Params) ([]byte, error) {
	if !params.Tempo.Spec.Template.Gateway.Enabled {
		params.Gates.HTTPEncryption = false
	}

	return buildConfiguration(params)
}

func buildConfiguration(params manifestutils.Params) ([]byte, error) {
	tempo := params.Tempo
	tlsopts := tlsOptions{}
	var err error

	if params.Gates.GRPCEncryption || params.Gates.HTTPEncryption {
		tlsopts, err = buildTLSConfig(params)
		if err != nil {
			return []byte{}, err
		}
	}

	opts := options{
		StorageType:     string(tempo.Spec.Storage.Secret.Type),
		StorageParams:   params.StorageParams,
		GlobalRetention: tempo.Spec.Retention.Global.Traces.Duration.String(),
		MemberList: []string{
			naming.Name("gossip-ring", tempo.Name),
		},
		QueryFrontendDiscovery: fmt.Sprintf("%s:%d", naming.Name("query-frontend-discovery", tempo.Name), manifestutils.PortGRPCServer),
		GlobalRateLimits:       fromRateLimitSpecToRateLimitOptions(tempo.Spec.LimitSpec.Global),
		Search:                 fromSearchSpecToOptions(tempo.Spec.SearchSpec),
		ReplicationFactor:      tempo.Spec.ReplicationFactor,
		Multitenancy:           tempo.Spec.Tenants != nil,
		Gateway:                tempo.Spec.Template.Gateway.Enabled,
		Gates: featureGates{
			GRPCEncryption: params.Gates.GRPCEncryption,
			HTTPEncryption: params.Gates.HTTPEncryption,
		},
		TLS: tlsopts,
	}

	if isTenantOverridesConfigRequired(tempo.Spec.LimitSpec) {
		opts.TenantRateLimitsPath = tenantOverridesMountPath
	}

	return renderTemplate(opts)
}

func isTenantOverridesConfigRequired(limitSpec v1alpha1.LimitSpec) bool {
	return len(limitSpec.PerTenant) > 0
}

func buildTenantOverrides(tempo v1alpha1.TempoStack) ([]byte, error) {
	return renderTenantOverridesTemplate(tenantOptions{
		RateLimits: fromRateLimitSpecToRateLimitOptionsMap(tempo.Spec.LimitSpec.PerTenant),
	})
}

func buildTLSConfig(params manifestutils.Params) (tlsOptions, error) {
	tempo := params.Tempo
	minTLSShort, err := params.TLSProfile.MinVersionShort()
	if err != nil {
		return tlsOptions{}, err
	}
	return tlsOptions{
		Paths: tlsFilePaths{
			CA:          fmt.Sprintf("%s/service-ca.crt", manifestutils.CABundleDir),
			Key:         fmt.Sprintf("%s/tls.key", manifestutils.TempoServerTLSDir()),
			Certificate: fmt.Sprintf("%s/tls.crt", manifestutils.TempoServerTLSDir()),
		},
		ServerNames: serverNames{
			QueryFrontend: naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName),
			Ingester:      naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.IngesterComponentName),
		},
		Profile: tlsProfileOptions{
			MinTLSVersion:      params.TLSProfile.MinTLSVersion,
			Ciphers:            params.TLSProfile.TLSCipherSuites(),
			MinTLSVersionShort: minTLSShort,
		},
	}, nil

}

func buildTempoQueryConfig(params manifestutils.Params) ([]byte, error) {
	tlsopts, err := buildTLSConfig(params)
	if err != nil {
		return []byte{}, err
	}

	return renderTempoQueryTemplate(tempoQueryOptions{
		TLS:      tlsopts,
		HTTPPort: manifestutils.PortHTTPServer,
		Gates: featureGates{
			GRPCEncryption: params.Gates.GRPCEncryption,
			HTTPEncryption: params.Gates.HTTPEncryption,
		},
		TenantHeader: manifestutils.TenantHeader,
		Gateway:      params.Tempo.Spec.Template.Gateway.Enabled,
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
		MaxResultLimit:            spec.MaxResultLimit,
		// If not specified, will be zero,  means disable limit by default
		MaxDuration: spec.MaxDuration.Duration.String(),
	}

	if spec.DefaultResultLimit != nil {
		options.DefaultResultLimit = *spec.DefaultResultLimit
	}

	return options
}

func renderTempoQueryTemplate(opts tempoQueryOptions) ([]byte, error) {
	// Build tempo query config yaml
	w := bytes.NewBuffer(nil)
	err := tempoQueryYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, err
	}
	cfg, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
