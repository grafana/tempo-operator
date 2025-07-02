package config

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"path"
	"time"

	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/memberlist"
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

func fromRateLimitSpecToTenantOverrides(spec v1alpha1.RateLimitSpec, retention *time.Duration) tenantOverrides {
	return tenantOverrides{
		IngestionRateLimitBytes: spec.Ingestion.IngestionRateLimitBytes,
		IngestionBurstSizeBytes: spec.Ingestion.IngestionBurstSizeBytes,
		MaxBytesPerTrace:        spec.Ingestion.MaxBytesPerTrace,
		MaxTracesPerUser:        spec.Ingestion.MaxTracesPerUser,
		MaxBytesPerTagValues:    spec.Query.MaxBytesPerTagValues,
		MaxSearchDuration:       spec.Query.MaxSearchDuration.Duration.String(),
		BlockRetention:          retention,
	}
}

func fromRateLimitSpecToRateLimitOptionsMap(rateLimits map[string]v1alpha1.RateLimitSpec, retentions map[string]v1alpha1.RetentionConfig) map[string]tenantOverrides {
	result := make(map[string]tenantOverrides, len(rateLimits))
	for tenant, spec := range rateLimits {
		retentionsSpec, ok := retentions[tenant]
		delete(retentions, tenant)
		var retention *time.Duration
		if ok {
			retention = &retentionsSpec.Traces.Duration
		}

		result[tenant] = fromRateLimitSpecToTenantOverrides(spec, retention)
	}
	for tenant, spec := range retentions {
		result[tenant] = fromRateLimitSpecToTenantOverrides(v1alpha1.RateLimitSpec{}, &spec.Traces.Duration)
	}
	return result
}

func buildQueryFrontEndConfig(params manifestutils.Params) ([]byte, error) {
	if !params.Tempo.Spec.Template.Gateway.Enabled {
		params.CtrlConfig.Gates.HTTPEncryption = false
	}

	return buildConfiguration(params)
}

func buildConfiguration(params manifestutils.Params) ([]byte, error) {
	tempo := params.Tempo
	tlsopts := tlsOptions{}
	var err error

	if params.CtrlConfig.Gates.GRPCEncryption || params.CtrlConfig.Gates.HTTPEncryption {
		tlsopts, err = buildTLSConfig(params)
		if err != nil {
			return []byte{}, err
		}
	}

	opts := options{
		StorageType:     string(tempo.Spec.Storage.Secret.Type),
		StorageParams:   params.StorageParams,
		GlobalRetention: tempo.Spec.Retention.Global.Traces.Duration.String(),
		MemberList: memberlistOptions{
			JoinMembers:  []string{naming.Name("gossip-ring", tempo.Name)},
			EnableIPv6:   ptr.Deref(tempo.Spec.HashRing.MemberList.EnableIPv6, false),
			InstanceAddr: gossipRingInstanceAddr(tempo.Spec.HashRing),
		},
		QueryFrontendDiscovery: fmt.Sprintf("%s:%d", naming.Name("query-frontend-discovery", tempo.Name), manifestutils.PortGRPCServer),
		GlobalRateLimits:       fromRateLimitSpecToTenantOverrides(tempo.Spec.LimitSpec.Global, nil),
		Search:                 fromSearchSpecToOptions(tempo.Spec.SearchSpec),
		ReplicationFactor:      tempo.Spec.ReplicationFactor,
		Multitenancy:           tempo.Spec.Tenants != nil,
		Gateway:                tempo.Spec.Template.Gateway.Enabled,
		Gates: featureGates{
			GRPCEncryption: params.CtrlConfig.Gates.GRPCEncryption,
			HTTPEncryption: params.CtrlConfig.Gates.HTTPEncryption,
		},
		TLS:          tlsopts,
		ReceiverTLS:  buildReceiverTLSConfig(tempo),
		S3StorageTLS: buildS3StorageTLSConfig(params),
		Timeout:      params.Tempo.Spec.Timeout.Duration,
	}

	if isTenantOverridesConfigRequired(tempo.Spec.LimitSpec, tempo.Spec.Retention) {
		opts.TenantRateLimitsPath = tenantOverridesMountPath
	}

	return renderTemplate(opts)
}

func isTenantOverridesConfigRequired(limitSpec v1alpha1.LimitSpec, retentionSpec v1alpha1.RetentionSpec) bool {
	return len(limitSpec.PerTenant) > 0 || len(retentionSpec.PerTenant) > 0
}

func buildTenantOverrides(tempo v1alpha1.TempoStack) ([]byte, error) {
	return renderTenantOverridesTemplate(tenantOptions{
		TenantOverrides: fromRateLimitSpecToRateLimitOptionsMap(tempo.Spec.LimitSpec.PerTenant, tempo.Spec.Retention.PerTenant),
	})
}

func buildReceiverTLSConfig(tempo v1alpha1.TempoStack) receiverTLSOptions {
	return receiverTLSOptions{
		Enabled:         tempo.Spec.Template.Distributor.TLS.Enabled,
		ClientCAEnabled: tempo.Spec.Template.Distributor.TLS.CA != "",
		Paths: tlsFilePaths{
			CA:          path.Join(manifestutils.ReceiverTLSCADir, manifestutils.TLSCAFilename),
			Key:         path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSKeyFilename),
			Certificate: path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSCertFilename),
		},
		MinTLSVersion: tempo.Spec.Template.Distributor.TLS.MinVersion,
	}
}

func buildS3StorageTLSConfig(params manifestutils.Params) storageTLSOptions {
	tempo := params.Tempo
	opts := storageTLSOptions{
		Enabled:       params.Tempo.Spec.Storage.TLS.Enabled,
		MinTLSVersion: params.Tempo.Spec.Storage.TLS.MinVersion,
	}
	if tempo.Spec.Storage.TLS.CA != "" {
		opts.CA = path.Join(manifestutils.StorageTLSCADir, params.StorageParams.S3.TLS.CAFilename)
	}
	if tempo.Spec.Storage.TLS.Cert != "" {
		opts.Certificate = path.Join(manifestutils.StorageTLSCertDir, manifestutils.TLSCertFilename)
		opts.Key = path.Join(manifestutils.StorageTLSCertDir, manifestutils.TLSKeyFilename)
	}
	return opts
}

func buildTLSConfig(params manifestutils.Params) (tlsOptions, error) {
	tempo := params.Tempo
	minTLSShort, err := params.TLSProfile.MinVersionShort()
	if err != nil {
		return tlsOptions{}, err
	}
	return tlsOptions{
		Paths: tlsFilePaths{
			CA:          path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename),
			Key:         path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename),
			Certificate: path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename),
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

	findTracesConcurrentRequests := params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.FindTracesConcurrentRequests
	if findTracesConcurrentRequests == 0 {
		querierReplicas := int32(1)
		if params.Tempo.Spec.Template.Querier.Replicas != nil {
			querierReplicas = *params.Tempo.Spec.Template.Querier.Replicas
		}
		findTracesConcurrentRequests = int(querierReplicas) * 2
	}
	return renderTempoQueryTemplate(tempoQueryOptions{
		TLS:      tlsopts,
		HTTPPort: manifestutils.PortHTTPServer,
		Gates: featureGates{
			GRPCEncryption: params.CtrlConfig.Gates.GRPCEncryption,
			HTTPEncryption: params.CtrlConfig.Gates.HTTPEncryption,
		},
		TenantHeader:                 manifestutils.TenantHeader,
		Gateway:                      params.Tempo.Spec.Template.Gateway.Enabled,
		ServicesQueryDuration:        params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.ServicesQueryDuration.Duration.String(),
		FindTracesConcurrentRequests: findTracesConcurrentRequests,
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
		ConcurrentJobs:       2000,
		MaxConcurrentQueries: 20,
		MaxResultLimit:       spec.MaxResultLimit,
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

func gossipRingInstanceAddr(spec v1alpha1.HashRingSpec) string {
	var instanceAddr string
	switch spec.MemberList.InstanceAddrType {
	case v1alpha1.InstanceAddrPodIP:
		instanceAddr = fmt.Sprintf("${%s}", memberlist.GossipInstanceAddrEnvVarName)
	case v1alpha1.InstanceAddrDefault:
		// Do nothing use tempo defaults
	default:
		// Do nothing use tempo defaults
	}

	return instanceAddr
}
