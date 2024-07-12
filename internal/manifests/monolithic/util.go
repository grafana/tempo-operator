package monolithic

import "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"

func ingestionTLSEnabled(tempo v1alpha1.TempoMonolithic) bool {
	return ingestionGRPCTLSEnabled(tempo) || ingestionHTTPTLSEnabled(tempo)
}

func tlsSecretAndBundleEmpty(tempo v1alpha1.TempoMonolithic) bool {
	return tlsSecretAndBundleEmptyGRPC(tempo) || tlsSecretAndBundleEmptyHTTP(tempo)

}
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
