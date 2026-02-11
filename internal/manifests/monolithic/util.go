package monolithic

import (
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func tlsSecretAndBundleEmptyGRPC(tempo v1alpha1.TempoMonolithic) bool {
	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil &&
		tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.TLS != nil {
		return tempo.Spec.Ingestion.OTLP.GRPC.TLS.Cert == "" && tempo.Spec.Ingestion.OTLP.GRPC.TLS.CA == ""
	}
	return false
}

func tlsSecretAndBundleEmptyHTTP(tempo v1alpha1.TempoMonolithic) bool {
	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil &&
		tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.TLS != nil {
		return tempo.Spec.Ingestion.OTLP.HTTP.TLS.Cert == "" && tempo.Spec.Ingestion.OTLP.HTTP.TLS.CA == ""
	}
	return false
}

func ingestionGRPCTLSEnabled(tempo v1alpha1.TempoMonolithic) bool {
	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil &&
		tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.TLS != nil {
		return tempo.Spec.Ingestion.OTLP.GRPC.TLS.Enabled
	}
	return false
}

func ingestionHTTPTLSEnabled(tempo v1alpha1.TempoMonolithic) bool {
	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil &&
		tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.TLS != nil {
		return tempo.Spec.Ingestion.OTLP.HTTP.TLS.Enabled
	}
	return false
}

// if the gateway is enabled, the serving cert is generated for the gateway service, otherwise for the main Tempo service
func ingestionServingCertName(tempo v1alpha1.TempoMonolithic) string {
	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		return naming.ServingCertName(manifestutils.GatewayComponentName, tempo.Name)
	} else {
		return naming.ServingCertName(manifestutils.TempoMonolithComponentName, tempo.Name)
	}
}
