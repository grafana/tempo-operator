package v1alpha1

import (
	"context"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/autodetect"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

var (
	zeroQuantity  = resource.MustParse("0Gi")
	tenGBQuantity = resource.MustParse("10Gi")
)

const maxLabelLength = 63
const defaultRouteGatewayTLSTermination = TLSRouteTerminationTypePassthrough
const defaultUITLSTermination = TLSRouteTerminationTypeEdge

// SetupWebhookWithManager initializes the webhook.
func (r *TempoStack) SetupWebhookWithManager(mgr ctrl.Manager, ctrlConfig v1alpha1.ProjectConfig) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(NewDefaulter(ctrlConfig)).
		WithValidator(&validator{client: mgr.GetClient(), ctrlConfig: ctrlConfig}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-tempo-grafana-com-v1alpha1-tempostack,mutating=true,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempostacks,verbs=create;update,versions=v1alpha1,name=mtempostack.tempo.grafana.com,admissionReviewVersions=v1

// NewDefaulter creates a new instance of Defaulter, which implements functions for setting defaults on the Tempo CR.
func NewDefaulter(ctrlConfig v1alpha1.ProjectConfig) *Defaulter {
	return &Defaulter{
		ctrlConfig: ctrlConfig,
	}
}

// Defaulter implements the CustomDefaulter interface.
type Defaulter struct {
	ctrlConfig v1alpha1.ProjectConfig
}

// Default applies default values to a Kubernetes object.
func (d *Defaulter) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*TempoStack)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a TempoStack object but got %T", obj))
	}

	log := ctrl.LoggerFrom(ctx).WithName("tempostack-webhook")
	log.V(1).Info("running defaulter webhook", "name", r.Name)

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "tempo-operator"
	}
	r.Labels["tempo.grafana.com/distribution"] = d.ctrlConfig.Distribution

	if r.Spec.ServiceAccount == "" {
		r.Spec.ServiceAccount = naming.DefaultServiceAccountName(r.Name)
	}

	if r.Spec.Retention.Global.Traces.Duration == 0 {
		r.Spec.Retention.Global.Traces.Duration = 48 * time.Hour
	}

	if r.Spec.StorageSize.Cmp(zeroQuantity) <= 0 {
		r.Spec.StorageSize = tenGBQuantity
	}

	if r.Spec.SearchSpec.DefaultResultLimit == nil {
		defaultDefaultResultLimit := 20
		r.Spec.SearchSpec.DefaultResultLimit = &defaultDefaultResultLimit
	}

	defaultComponentReplicas := ptr.To(int32(1))
	defaultReplicationFactor := 1

	// Default replicas for all components if not specified.
	if r.Spec.Template.Ingester.Replicas == nil {
		r.Spec.Template.Ingester.Replicas = defaultComponentReplicas
	}
	if r.Spec.Template.Distributor.Replicas == nil {
		r.Spec.Template.Distributor.Replicas = defaultComponentReplicas
	}
	if r.Spec.Template.Compactor.Replicas == nil {
		r.Spec.Template.Compactor.Replicas = defaultComponentReplicas
	}
	if r.Spec.Template.Querier.Replicas == nil {
		r.Spec.Template.Querier.Replicas = defaultComponentReplicas
	}
	if r.Spec.Template.QueryFrontend.Replicas == nil {
		r.Spec.Template.QueryFrontend.Replicas = defaultComponentReplicas
	}

	// Default replication factor if not specified.
	if r.Spec.ReplicationFactor == 0 {
		r.Spec.ReplicationFactor = defaultReplicationFactor
	}

	// if tenant mode is Openshift, ingress type should be route by default.
	if r.Spec.Tenants != nil && r.Spec.Tenants.Mode == ModeOpenShift && r.Spec.Template.Gateway.Ingress.Type == "" {
		r.Spec.Template.Gateway.Ingress.Type = IngressTypeRoute
	}

	if r.Spec.Template.Gateway.Ingress.Type == IngressTypeRoute && r.Spec.Template.Gateway.Ingress.Route.Termination == "" {
		r.Spec.Template.Gateway.Ingress.Route.Termination = defaultRouteGatewayTLSTermination
	}

	// Terminate TLS of the JaegerQuery Route on the Edge by default
	if r.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type == IngressTypeRoute && r.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Route.Termination == "" {
		r.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Route.Termination = defaultUITLSTermination
	}

	// Enable IPv6 if the operator pod (and therefore most likely all other pods) only have IPv6 addresses assigned
	if r.Spec.HashRing.MemberList.EnableIPv6 == nil {
		if autodetect.DetectIPv6Only([]string{"eth0", "en0"}) {
			r.Spec.HashRing.MemberList.EnableIPv6 = ptr.To(true)
		}
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-tempostack,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=tempostacks,verbs=create;update,versions=v1alpha1,name=vtempostack.tempo.grafana.com,admissionReviewVersions=v1

type validator struct {
	client     client.Client
	ctrlConfig v1alpha1.ProjectConfig
}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// NOTE(agerstmayr): change verbs in +kubebuilder:webhook to "verbs=create;update;delete" if you want to enable deletion validation.
	return nil, nil
}

func (v *validator) validateServiceAccount(ctx context.Context, tempo TempoStack) field.ErrorList {
	var allErrs field.ErrorList

	// the default service account gets created later in the reconciliation loop
	if tempo.Spec.ServiceAccount != naming.DefaultServiceAccountName(tempo.Name) {
		// check if custom service account exists
		serviceAccount := &corev1.ServiceAccount{}
		err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.ServiceAccount}, serviceAccount)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("serviceAccount"),
				tempo.Spec.ServiceAccount,
				err.Error(),
			))
		}
	}
	return allErrs
}

func (v *validator) validateStorageSecret(ctx context.Context, tempo TempoStack) (admission.Warnings, field.ErrorList) {
	storageSecret := &corev1.Secret{}
	err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.Secret.Name}, storageSecret)
	if err != nil {
		// Do not fail the validation here, the user can create the storage secret later.
		// The operator will remain in a ConfigurationError status condition until the storage secret is set.
		return admission.Warnings{fmt.Sprintf("Secret '%s' does not exist", tempo.Spec.Storage.Secret.Name)}, field.ErrorList{}
	}

	return admission.Warnings{}, ValidateStorageSecret(tempo, *storageSecret)
}

func (v *validator) validateStorageCA(ctx context.Context, tempo TempoStack) (admission.Warnings, field.ErrorList) {
	caConfigMap := &corev1.ConfigMap{}
	err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.TLS.CA}, caConfigMap)
	if err != nil {
		// Do not fail the validation here, the user can create the ConfigMap later.
		// The operator will remain in a ConfigurationError status condition until the ConfigMap is created.
		return admission.Warnings{fmt.Sprintf("ConfigMap '%s' does not exist", tempo.Spec.Storage.TLS.CA)}, field.ErrorList{}
	}

	return admission.Warnings{}, ValidateStorageCAConfigMap(*caConfigMap)
}

func (v *validator) validateReplicationFactor(tempo TempoStack) field.ErrorList {
	// Validate minimum quorum on ingestors according to replicas and replication factor
	replicatonFactor := tempo.Spec.ReplicationFactor
	// Ingester replicas should not be nil at this point, due defauler.
	ingesterReplicas := int(*tempo.Spec.Template.Ingester.Replicas)
	quorum := int(math.Floor(float64(replicatonFactor)/2.0) + 1)
	// if ingester replicas less than quorum (which depends on replication factor), then doesn't allow to deploy as it is an
	// invalid configuration. Quorum equal to replicas doesn't allow you to lose ingesters but is a valid configuration.
	if ingesterReplicas < quorum {
		path := field.NewPath("spec").Child("ReplicationFactor")
		return field.ErrorList{
			field.Invalid(path, tempo.Spec.ReplicationFactor,
				fmt.Sprintf("replica factor of %d requires at least %d ingester replicas", replicatonFactor, quorum),
			)}
	}
	return nil
}

func (v *validator) validateQueryFrontend(tempo TempoStack) field.ErrorList {
	path := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("ingress").Child("type")

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type != IngressTypeNone && !tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		return field.ErrorList{field.Invalid(
			path,
			tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type,
			"Ingress cannot be enabled if jaegerQuery is disabled",
		)}
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type == IngressTypeRoute && !v.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
		return field.ErrorList{field.Invalid(
			path,
			tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type,
			"Please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
		)}
	}

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.Enabled {
		prometheusEndpointPath := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("monitorTab").Child("prometheusEndpoint")
		if tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint == "" {
			return field.ErrorList{field.Invalid(
				prometheusEndpointPath,
				tempo.Spec.Template.QueryFrontend.JaegerQuery.MonitorTab.PrometheusEndpoint,
				"Prometheus endpoint must be set when monitoring is enabled",
			)}
		}
	}

	return nil
}

func (v *validator) validateGateway(tempo TempoStack) field.ErrorList {
	path := field.NewPath("spec").Child("template").Child("gateway").Child("enabled")
	if tempo.Spec.Template.Gateway.Enabled {
		if tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type != IngressTypeNone {
			return field.ErrorList{
				field.Invalid(path, tempo.Spec.Template.Gateway.Enabled,
					"cannot enable gateway and jaeger query ingress at the same time, please use the Jaeger UI from the gateway",
				)}
		}

		if tempo.Spec.Tenants == nil {
			return field.ErrorList{
				field.Invalid(path, tempo.Spec.Template.Gateway.Enabled,
					"to enable the gateway, please configure tenants",
				)}
		}

		if tempo.Spec.Template.Gateway.Ingress.Type == IngressTypeRoute && !v.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
			return field.ErrorList{field.Invalid(
				field.NewPath("spec").Child("template").Child("gateway").Child("ingress").Child("type"),
				tempo.Spec.Template.Gateway.Ingress.Type,
				"please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
			)}
		}

		if tempo.Spec.Template.Gateway.Enabled && tempo.Spec.Template.Distributor.TLS.Enabled {
			return field.ErrorList{field.Invalid(
				field.NewPath("spec").Child("template").Child("gateway").Child("enabled"),
				tempo.Spec.Template.Gateway.Enabled,
				"Cannot enable gateway and distributor TLS at the same time",
			)}
		}
	}
	return nil
}

func (v *validator) validateObservability(tempo TempoStack) field.ErrorList {
	observabilityBase := field.NewPath("spec").Child("observability")
	metricsBase := observabilityBase.Child("metrics")
	grafanaBase := observabilityBase.Child("grafana")

	if tempo.Spec.Observability.Metrics.CreateServiceMonitors && !v.ctrlConfig.Gates.PrometheusOperator {
		return field.ErrorList{
			field.Invalid(metricsBase.Child("createServiceMonitors"), tempo.Spec.Observability.Metrics.CreateServiceMonitors,
				"the prometheusOperator feature gate must be enabled to create ServiceMonitors for Tempo components",
			)}
	}

	if tempo.Spec.Observability.Metrics.CreatePrometheusRules && !v.ctrlConfig.Gates.PrometheusOperator {
		return field.ErrorList{
			field.Invalid(metricsBase.Child("createPrometheusRules"), tempo.Spec.Observability.Metrics.CreatePrometheusRules,
				"the prometheusOperator feature gate must be enabled to create PrometheusRules for Tempo components",
			)}
	}

	if tempo.Spec.Observability.Metrics.CreatePrometheusRules && !tempo.Spec.Observability.Metrics.CreateServiceMonitors {
		return field.ErrorList{
			field.Invalid(metricsBase.Child("createPrometheusRules"), tempo.Spec.Observability.Metrics.CreatePrometheusRules,
				"the Prometheus rules alert based on collected metrics, therefore the createServiceMonitors feature must be enabled when enabling the createPrometheusRules feature",
			)}
	}

	tracingBase := observabilityBase.Child("tracing")
	if tempo.Spec.Observability.Tracing.SamplingFraction == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(tempo.Spec.Observability.Tracing.SamplingFraction, 64); err != nil {
		return field.ErrorList{
			field.Invalid(
				tracingBase.Child("sampling_fraction"),
				tempo.Spec.Observability.Tracing.SamplingFraction,
				err.Error(),
			)}
	}

	if tempo.Spec.Observability.Tracing.JaegerAgentEndpoint != "" {
		_, _, err := net.SplitHostPort(tempo.Spec.Observability.Tracing.JaegerAgentEndpoint)
		if err != nil {
			return field.ErrorList{
				field.Invalid(
					tracingBase.Child("jaeger_agent_endpoint"),
					tempo.Spec.Observability.Tracing.JaegerAgentEndpoint,
					err.Error(),
				)}
		}
	}

	if tempo.Spec.Observability.Grafana.CreateDatasource && !v.ctrlConfig.Gates.GrafanaOperator {
		return field.ErrorList{
			field.Invalid(grafanaBase.Child("createDatasource"), tempo.Spec.Observability.Grafana.CreateDatasource,
				"the grafanaOperator feature gate must be enabled to create a Datasource for Tempo",
			)}
	}

	return nil
}

func (v *validator) validateTenantConfigs(tempo TempoStack) field.ErrorList {
	if err := ValidateTenantConfigs(tempo); err != nil {
		return field.ErrorList{
			field.Invalid(
				field.NewPath("spec").Child("template").Child("tenants"),
				tempo.Spec.Template.Gateway.Enabled,
				err.Error(),
			)}
	}
	return nil
}

func (v *validator) validateStackName(tempo TempoStack) field.ErrorList {
	// We need to check this because the name is used as a label value for app.kubernetes.io/instance
	// Only validate the length, because the DNS rules are enforced by the functions in the `naming` package.
	if len(tempo.Name) > maxLabelLength {
		return field.ErrorList{
			field.Invalid(
				field.NewPath("metadata").Child("name"),
				tempo.Name,
				fmt.Sprintf("must be no more than %d characters", maxLabelLength),
			)}
	}
	return nil
}

func (v *validator) validateDeprecatedFields(tempo TempoStack) field.ErrorList {
	if tempo.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace != nil {
		return field.ErrorList{
			field.Invalid(
				field.NewPath("spec").Child("limits").Child("global").Child("query").Child("maxSearchBytesPerTrace"),
				tempo.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace,
				"this field is deprecated and must be unset",
			)}
	}
	for tenant, limits := range tempo.Spec.LimitSpec.PerTenant {
		if limits.Query.MaxSearchBytesPerTrace != nil {
			return field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("limits").Child("perTenant").Key(tenant).Child("query").Child("maxSearchBytesPerTrace"),
					limits.Query.MaxSearchBytesPerTrace,
					"this field is deprecated and must be unset",
				)}
		}
	}

	return nil
}

func (v *validator) validateReceiverTLS(tempo TempoStack) field.ErrorList {
	spec := tempo.Spec.Template.Distributor.TLS
	if spec.Enabled {
		if spec.Cert == "" {
			return field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("template").Child("distributor").Child("tls").Child("cert"),
					spec.Cert,
					"need to specify cert secret name",
				)}
		}
	}
	return nil
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tempo, ok := obj.(*TempoStack)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a TempoStack object but got %T", obj))
	}

	log := ctrl.LoggerFrom(ctx).WithName("tempostack-webhook")
	log.V(1).Info("running validating webhook", "name", tempo.Name)

	allWarnings := admission.Warnings{}
	allErrors := field.ErrorList{}
	var warnings admission.Warnings
	var errors field.ErrorList

	allErrors = append(allErrors, v.validateStackName(*tempo)...)
	allErrors = append(allErrors, v.validateServiceAccount(ctx, *tempo)...)

	warnings, errors = v.validateStorageSecret(ctx, *tempo)
	allWarnings = append(allWarnings, warnings...)
	allErrors = append(allErrors, errors...)

	if tempo.Spec.Storage.TLS.CA != "" {
		warnings, errors = v.validateStorageCA(ctx, *tempo)
		allWarnings = append(allWarnings, warnings...)
		allErrors = append(allErrors, errors...)
	}

	if tempo.Spec.ExtraConfig != nil && len(tempo.Spec.ExtraConfig.Tempo.Raw) > 0 {
		allWarnings = append(allWarnings, admission.Warnings{
			"override tempo configuration could potentially break the stack, use it carefully",
		}...)

	}

	allErrors = append(allErrors, v.validateReplicationFactor(*tempo)...)
	allErrors = append(allErrors, v.validateQueryFrontend(*tempo)...)
	allErrors = append(allErrors, v.validateGateway(*tempo)...)
	allErrors = append(allErrors, v.validateTenantConfigs(*tempo)...)
	allErrors = append(allErrors, v.validateObservability(*tempo)...)
	allErrors = append(allErrors, v.validateDeprecatedFields(*tempo)...)
	allErrors = append(allErrors, v.validateReceiverTLS(*tempo)...)

	if len(allErrors) == 0 {
		return allWarnings, nil
	}
	return allWarnings, apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, allErrors)
}

// ValidateTenantConfigs validates the tenants mode specification.
func ValidateTenantConfigs(tempo TempoStack) error {
	if tempo.Spec.Tenants == nil {
		return nil
	}

	tenants := tempo.Spec.Tenants
	if tenants.Mode == ModeStatic {
		// If the static mode is combined with the gateway, we will need the following fields
		// otherwise this will just enable tempo multitenancy without the gateway
		if tempo.Spec.Template.Gateway.Enabled {
			if tenants.Authentication == nil {
				return fmt.Errorf("spec.tenants.authentication is required in static mode")
			}

			if tenants.Authorization == nil {
				return fmt.Errorf("spec.tenants.authorization is required in static mode")
			}

			if tenants.Authorization.Roles == nil {
				return fmt.Errorf("spec.tenants.authorization.roles is required in static mode")
			}

			if tenants.Authorization.RoleBindings == nil {
				return fmt.Errorf("spec.tenants.authorization.roleBindings is required in static mode")
			}
		}
	} else if tenants.Mode == ModeOpenShift {
		if !tempo.Spec.Template.Gateway.Enabled {
			return fmt.Errorf("openshift mode requires gateway enabled")
		}
		if tenants.Authorization != nil {
			return fmt.Errorf("spec.tenants.authorization should not be defined in openshift mode")
		}
		for _, auth := range tenants.Authentication {
			if auth.OIDC != nil {
				return fmt.Errorf("spec.tenants.authentication.oidc should not be defined in openshift mode")
			}
		}
	}
	return nil
}
