package webhooks

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tempov1alpha1 "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
)

// TempoMonolithicWebhook provides webhooks for TempoMonolithic CR.
type TempoMonolithicWebhook struct {
}

// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (w *TempoMonolithicWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&tempov1alpha1.TempoMonolithic{}).
		WithValidator(&monolithicValidator{client: mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-tempomonolithic,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempomonolithics,verbs=create;update,versions=v1alpha1,name=vtempomonolithic.kb.io,admissionReviewVersions=v1

type monolithicValidator struct {
	client client.Client
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (v *monolithicValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (v *monolithicValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (v *monolithicValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
	return nil, nil
}

func (v *monolithicValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tempo, ok := obj.(*tempov1alpha1.TempoMonolithic)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a TempoMonolithic object but got %T", obj))
	}

	log := ctrl.LoggerFrom(ctx).WithName("tempomonolithic-webhook")
	log.V(1).Info("running validating webhook", "name", tempo.Name)

	// We do not modify the Kubernetes object in the defaulter webhook,
	// but still apply some default values in-memory.
	tempo.Default()

	allWarnings := admission.Warnings{}
	allErrors := field.ErrorList{}
	addValidationResults := func(warnings admission.Warnings, errors field.ErrorList) {
		allWarnings = append(allWarnings, warnings...)
		allErrors = append(allErrors, errors...)
	}

	addValidationResults(v.validateStorage(ctx, *tempo))
	addValidationResults(v.validateExtraConfig(*tempo))

	if len(allErrors) == 0 {
		return allWarnings, nil
	}
	return allWarnings, apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, allErrors)
}

func (v *monolithicValidator) validateStorage(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) (admission.Warnings, field.ErrorList) { //nolint:unparam
	_, errs := storage.GetStorageParamsForTempoMonolithic(ctx, v.client, tempo)
	if len(errs) == 1 && (strings.HasPrefix(errs[0].Detail, storage.ErrFetchingSecret) || strings.HasPrefix(errs[0].Detail, storage.ErrFetchingConfigMap)) {
		// Do not fail the validation if the storage secret or TLS CA ConfigMap is not found, the user can create these objects later.
		// The operator will remain in a ConfigurationError status condition until the storage secret is created.
		return admission.Warnings{errs[0].Detail}, field.ErrorList{}
	}
	return nil, errs
}

func (v *monolithicValidator) validateExtraConfig(tempo tempov1alpha1.TempoMonolithic) (admission.Warnings, field.ErrorList) { //nolint:unparam
	if tempo.Spec.ExtraConfig != nil && len(tempo.Spec.ExtraConfig.Tempo.Raw) > 0 {
		return admission.Warnings{"overriding Tempo configuration could potentially break the deployment, use it carefully"}, nil
	}
	return nil, nil
}
