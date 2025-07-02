package webhooks

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
	"github.com/grafana/tempo-operator/internal/status"
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

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-tempomonolithic,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempomonolithics,verbs=create;update;delete,versions=v1alpha1,name=vtempomonolithic.kb.io,admissionReviewVersions=v1

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
	oldTempo, ok := oldObj.(*tempov1alpha1.TempoMonolithic)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a TempoMonolithic object but got %T", oldObj))
	}
	newTempo, ok := newObj.(*tempov1alpha1.TempoMonolithic)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a TempoMonolithic object but got %T", newObj))
	}
	if newTempo.GetDeletionTimestamp() != nil &&
		controllerutil.ContainsFinalizer(oldTempo, tempov1alpha1.TempoFinalizer) && !controllerutil.ContainsFinalizer(newTempo, tempov1alpha1.TempoFinalizer) {
		// Do not validate if the specs are the same and only finalizer was removed
		// This is to avoid a situation when kubectl delete -f file.yaml is run and the file contains
		// Tempo CR and custom SA or storage secret. The reconcile loop will remove the finalizer and trigger the webhook which would fail.
		return nil, nil
	}
	return v.validate(ctx, newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (v *monolithicValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	tempo, ok := obj.(*tempov1alpha1.TempoMonolithic)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a TempoMonolithic object but got %T", obj))
	}
	status.ClearMonolithicMetrics(tempo.Namespace, tempo.Name)
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
	tempo.Default(v.ctrlConfig)

	warnings := admission.Warnings{}
	errors := field.ErrorList{}
	addValidationResults := func(w admission.Warnings, e field.ErrorList) {
		warnings = append(warnings, w...)
		errors = append(errors, e...)
	}

	errors = append(errors, validateName(tempo.Name)...)
	addValidationResults(v.validateStorage(ctx, tempo))
	errors = append(errors, v.validateJaegerUI(tempo)...)
	errors = append(errors, v.validateMultitenancy(ctx, tempo)...)
	errors = append(errors, v.validateObservability(tempo)...)
	errors = append(errors, v.validateServiceAccount(ctx, tempo)...)
	errors = append(errors, v.validateConflictWithTempoStack(ctx, tempo)...)

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
	if tempo.Spec.Query != nil && tempo.Spec.Query.RBAC.Enabled && tempo.Spec.JaegerUI.Enabled {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "rbac", "enabled"), tempo.Spec.Query.RBAC.Enabled,
				"cannot enable RBAC and jaeger query at the same time. The Jaeger UI does not support query RBAC",
			)}
	}

	return nil
}

func (v *monolithicValidator) validateMultitenancy(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) field.ErrorList {
	if tempo.Spec.Query != nil && tempo.Spec.Query.RBAC.Enabled && (tempo.Spec.Multitenancy == nil || !tempo.Spec.Multitenancy.Enabled) {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "rbac", "enabled"), tempo.Spec.Query.RBAC.Enabled,
				"RBAC can only be enabled if multi-tenancy is enabled",
			)}
	}

	if !tempo.Spec.Multitenancy.IsGatewayEnabled() {
		return nil
	}

	multitenancyBase := field.NewPath("spec", "multitenancy")

	if tempo.Spec.Multitenancy != nil && tempo.Spec.Multitenancy.Mode == tempov1alpha1.ModeOpenShift {
		err := validateGatewayOpenShiftModeRBAC(ctx, v.client)
		if err != nil {
			return field.ErrorList{field.Invalid(
				multitenancyBase.Child("mode"),
				tempo.Spec.Multitenancy.Mode,
				fmt.Sprintf("Cannot enable OpenShift tenancy mode: %v", err),
			)}
		}
	}

	err := ValidateTenantConfigs(&tempo.Spec.Multitenancy.TenantsSpec, tempo.Spec.Multitenancy.IsGatewayEnabled())
	if err != nil {
		return field.ErrorList{field.Invalid(multitenancyBase.Child("enabled"), tempo.Spec.Multitenancy.Enabled, err.Error())}
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

		if tempo.Spec.Observability.Grafana.DataSource != nil && tempo.Spec.Observability.Grafana.DataSource.Enabled &&
			tempo.Spec.Multitenancy.IsGatewayEnabled() {
			return field.ErrorList{field.Invalid(
				grafanaBase.Child("dataSource", "enabled"),
				tempo.Spec.Observability.Grafana.DataSource.Enabled,
				"creating a data source for Tempo is not support if the gateway is enabled",
			)}
		}
	}

	return nil
}

func (v *monolithicValidator) validateServiceAccount(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) field.ErrorList {
	if tempo.Spec.ServiceAccount == "" {
		return nil
	}

	if tempo.Spec.Multitenancy.IsGatewayEnabled() && tempo.Spec.Multitenancy.Mode == tempov1alpha1.ModeOpenShift {
		return field.ErrorList{field.Invalid(
			field.NewPath("spec").Child("serviceAccount"),
			tempo.Spec.ServiceAccount,
			"custom ServiceAccount is not supported if multi-tenancy with OpenShift mode is enabled",
		)}
	}

	serviceAccount := &corev1.ServiceAccount{}
	err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.ServiceAccount}, serviceAccount)
	if err != nil {
		return field.ErrorList{field.Invalid(
			field.NewPath("spec").Child("serviceAccount"),
			tempo.Spec.ServiceAccount,
			err.Error(),
		)}
	}

	return nil
}

func (v *monolithicValidator) validateExtraConfig(tempo tempov1alpha1.TempoMonolithic) admission.Warnings {
	if tempo.Spec.ExtraConfig != nil && len(tempo.Spec.ExtraConfig.Tempo.Raw) > 0 {
		return admission.Warnings{"overriding Tempo configuration could potentially break the deployment, use it carefully"}
	}
	return nil
}

func (v *monolithicValidator) validateConflictWithTempoStack(ctx context.Context, tempo tempov1alpha1.TempoMonolithic) field.ErrorList {
	return validateTempoNameConflict(func() error {
		stack := &tempov1alpha1.TempoStack{}
		return v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Name}, stack)
	},
		tempo.Name, "TempoMonolithic", "TempoStack",
	)
}
