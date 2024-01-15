package v1alpha1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (r *TempoMonolithic) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
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

	if r.Spec.Storage.Traces.Backend == MonolithicTracesStorageBackendPersistentVolume && r.Spec.Storage.Traces.PV == nil {
		r.Spec.Storage.Traces.PV = &MonolithicTracesStoragePVSpec{
			Size: tenGBQuantity,
		}
	}

	if r.Spec.Ingestion == nil {
		r.Spec.Ingestion = &MonolithicIngestionSpec{
			OTLP: &MonolithicIngestionOTLPSpec{
				GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
					Enabled: true,
				},
			},
		}
	}
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-tempomonolithic,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempomonolithics,verbs=create;update,versions=v1alpha1,name=vtempomonolithic.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &TempoMonolithic{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *TempoMonolithic) ValidateCreate() (admission.Warnings, error) {
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *TempoMonolithic) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *TempoMonolithic) ValidateDelete() (admission.Warnings, error) {
	// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
	return r.validate()
}

func (tempo *TempoMonolithic) validate() (admission.Warnings, error) {
	log := ctrl.Log.WithName("tempomonolithic-webhook")
	log.V(1).Info("running validating webhook", "name", tempo.Name)

	allWarnings := admission.Warnings{}
	allErrors := field.ErrorList{}

	if tempo.Spec.ExtraConfig != nil && len(tempo.Spec.ExtraConfig.Tempo.Raw) > 0 {
		allWarnings = append(allWarnings, "overriding Tempo configuration could potentially break the deployment, use it carefully")
	}

	if len(allErrors) == 0 {
		return allWarnings, nil
	}
	return allWarnings, apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, allErrors)
}
