package v1alpha1

import "k8s.io/apimachinery/pkg/api/resource"

var (
	tenGBQuantity = resource.MustParse("10Gi")
)

// Default sets all default values in a central place, instead of setting it at every place where the value is accessed.
// NOTE: This function is called inside the Reconcile loop, NOT in the webhook.
// We want to keep the CR as minimal as the user configures it, and not modify it in any way (except for upgrades).
func (r *TempoMonolithic) Default() {
	if r.Spec.Storage == nil {
		r.Spec.Storage = &MonolithicStorageSpec{}
	}

	if r.Spec.Storage.Traces.Backend == "" {
		r.Spec.Storage.Traces.Backend = MonolithicTracesStorageBackendMemory
	}

	if r.Spec.Storage.Traces.Backend != MonolithicTracesStorageBackendMemory && r.Spec.Storage.Traces.WAL == nil {
		r.Spec.Storage.Traces.WAL = &MonolithicTracesStorageWALSpec{
			Size: tenGBQuantity,
		}
	}

	if r.Spec.Storage.Traces.Backend == MonolithicTracesStorageBackendPV && r.Spec.Storage.Traces.PV == nil {
		r.Spec.Storage.Traces.PV = &MonolithicTracesStoragePVSpec{
			Size: tenGBQuantity,
		}
	}

	if r.Spec.Ingestion == nil {
		r.Spec.Ingestion = &MonolithicIngestionSpec{}
	}
	if r.Spec.Ingestion.OTLP == nil {
		r.Spec.Ingestion.OTLP = &MonolithicIngestionOTLPSpec{}
	}
	if r.Spec.Ingestion.OTLP.GRPC == nil {
		r.Spec.Ingestion.OTLP.GRPC = &MonolithicIngestionOTLPProtocolsGRPCSpec{
			Enabled: true,
		}
	}
	if r.Spec.Ingestion.OTLP.HTTP == nil {
		r.Spec.Ingestion.OTLP.HTTP = &MonolithicIngestionOTLPProtocolsHTTPSpec{
			Enabled: true,
		}
	}

	if r.Spec.JaegerUI != nil && r.Spec.JaegerUI.Enabled &&
		r.Spec.JaegerUI.Route != nil && r.Spec.JaegerUI.Route.Enabled &&
		r.Spec.JaegerUI.Route.Termination == "" {
		r.Spec.JaegerUI.Route.Termination = "edge"
	}
}
