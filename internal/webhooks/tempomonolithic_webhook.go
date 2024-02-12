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

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	tempov1alpha1 "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
)

// TempoMonolithicWebhook provides webhooks for TempoMonolithic CR.
type TempoMonolithicWebhook struct {
}

// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (w *TempoMonolithicWebhook) SetupWebhookWithManager(mgr ctrl.Manager, ctrlConfig configv1alpha1.ProjectConfig) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&tempov1alpha1.TempoMonolithic{}).
		WithValidator(&monolithicValidator{client: mgr.GetClient(), ctrlConfig: ctrlConfig}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-tempomonolithic,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempomonolithics,verbs=create;update,versions=v1alpha1,name=vtempomonolithic.kb.io,admissionReviewVersions=v1

type monolithicValidator struct {
	client     client.Client
	ctrlConfig configv1alpha1.ProjectConfig
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

	warnings, errors := v.validateTempoMonolithic(ctx, *tempo)

	if len(errors) == 0 {
		return warnings, nil
	}
	return warnings, apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, errors)
}

func (v *monolithicValidator) validateTempoMonolithic(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) (admission.Warnings, field.ErrorList) {
	// We do not modify the Kubernetes object in the defaulter webhook,
	// but still apply some default values in-memory.
	tempo.Default()

	warnings := admission.Warnings{}
	errors := field.ErrorList{}
	addValidationResults := func(w admission.Warnings, e field.ErrorList) {
		warnings = append(warnings, w...)
		errors = append(errors, e...)
	}

	errors = append(errors, validateName(tempo.Name)...)
	addValidationResults(v.validateStorage(ctx, tempo))
	errors = append(errors, v.validateJaegerUI(tempo)...)
	errors = append(errors, v.validateObservability(tempo)...)
	warnings = append(warnings, v.validateExtraConfig(tempo)...)

	return warnings, errors
}

func (v *monolithicValidator) validateStorage(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) (admission.Warnings, field.ErrorList) {
	_, errs := storage.GetStorageParamsForTempoMonolithic(ctx, v.client, tempo)
	if len(errs) == 1 && (strings.HasPrefix(errs[0].Detail, storage.ErrFetchingSecret) || strings.HasPrefix(errs[0].Detail, storage.ErrFetchingConfigMap)) {
		// Do not fail the validation if the storage secret or TLS CA ConfigMap is not found, the user can create these objects later.
		// The operator will remain in a ConfigurationError status condition until the storage secret is created.
		return admission.Warnings{errs[0].Detail}, field.ErrorList{}
	}
	return nil, errs
}

func (v *monolithicValidator) validateJaegerUI(tempo tempov1alpha1.TempoMonolithic) field.ErrorList {
	if tempo.Spec.JaegerUI == nil {
		return nil
	}

	jaegerUIBase := field.NewPath("spec", "jaegerui")
	if tempo.Spec.JaegerUI.Ingress != nil && tempo.Spec.JaegerUI.Ingress.Enabled && !tempo.Spec.JaegerUI.Enabled {
		return field.ErrorList{field.Invalid(
			jaegerUIBase.Child("ingress", "enabled"),
			tempo.Spec.JaegerUI.Ingress.Enabled,
			"Jaeger UI must be enabled to create an ingress for Jaeger UI",
		)}
	}

	if tempo.Spec.JaegerUI.Route != nil && tempo.Spec.JaegerUI.Route.Enabled && !tempo.Spec.JaegerUI.Enabled {
		return field.ErrorList{field.Invalid(
			jaegerUIBase.Child("route", "enabled"),
			tempo.Spec.JaegerUI.Route.Enabled,
			"Jaeger UI must be enabled to create a route for Jaeger UI",
		)}
	}

	if tempo.Spec.JaegerUI.Route != nil && tempo.Spec.JaegerUI.Route.Enabled && !v.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
		return field.ErrorList{field.Invalid(
			jaegerUIBase.Child("route", "enabled"),
			tempo.Spec.JaegerUI.Route.Enabled,
			"the openshiftRoute feature gate must be enabled to create a route for Jaeger UI",
		)}
	}

	return nil
}

func (v *monolithicValidator) validateObservability(tempo tempov1alpha1.TempoMonolithic) field.ErrorList {
	if tempo.Spec.Observability == nil {
		return nil
	}

	observabilityBase := field.NewPath("spec", "observability")
	if tempo.Spec.Observability.Metrics != nil {
		metricsBase := observabilityBase.Child("metrics")

		if tempo.Spec.Observability.Metrics.ServiceMonitors != nil && tempo.Spec.Observability.Metrics.ServiceMonitors.Enabled &&
			!v.ctrlConfig.Gates.PrometheusOperator {
			return field.ErrorList{field.Invalid(
				metricsBase.Child("serviceMonitors", "enabled"),
				tempo.Spec.Observability.Metrics.ServiceMonitors.Enabled,
				"the prometheusOperator feature gate must be enabled to create ServiceMonitors for Tempo components",
			)}
		}

		if tempo.Spec.Observability.Metrics.PrometheusRules != nil && tempo.Spec.Observability.Metrics.PrometheusRules.Enabled &&
			!v.ctrlConfig.Gates.PrometheusOperator {
			return field.ErrorList{field.Invalid(
				metricsBase.Child("prometheusRules", "enabled"),
				tempo.Spec.Observability.Metrics.PrometheusRules.Enabled,
				"the prometheusOperator feature gate must be enabled to create PrometheusRules for Tempo components",
			)}
		}

		if tempo.Spec.Observability.Metrics.PrometheusRules != nil && tempo.Spec.Observability.Metrics.PrometheusRules.Enabled &&
			!(tempo.Spec.Observability.Metrics.ServiceMonitors != nil && tempo.Spec.Observability.Metrics.ServiceMonitors.Enabled) {
			return field.ErrorList{field.Invalid(
				metricsBase.Child("prometheusRules", "enabled"),
				tempo.Spec.Observability.Metrics.PrometheusRules.Enabled,
				"serviceMonitors must be enabled to create PrometheusRules (the rules alert based on collected metrics)",
			)}
		}
	}

	if tempo.Spec.Observability.Grafana != nil {
		grafanaBase := observabilityBase.Child("grafana")

		if tempo.Spec.Observability.Grafana.DataSource != nil && tempo.Spec.Observability.Grafana.DataSource.Enabled &&
			!v.ctrlConfig.Gates.GrafanaOperator {
			return field.ErrorList{field.Invalid(
				grafanaBase.Child("dataSource", "enabled"),
				tempo.Spec.Observability.Grafana.DataSource.Enabled,
				"the grafanaOperator feature gate must be enabled to create a data source for Tempo",
			)}
		}
	}

	return nil
}

func (v *monolithicValidator) validateExtraConfig(tempo tempov1alpha1.TempoMonolithic) admission.Warnings {
	if tempo.Spec.ExtraConfig != nil && len(tempo.Spec.ExtraConfig.Tempo.Raw) > 0 {
		return admission.Warnings{"overriding Tempo configuration could potentially break the deployment, use it carefully"}
	}
	return nil
}
