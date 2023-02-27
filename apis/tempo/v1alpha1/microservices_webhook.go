package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

var (
	zeroQuantity                  = resource.MustParse("0Gi")
	tenGBQuantity                 = resource.MustParse("10Gi")
	errNoDefaultTempoImage        = errors.New("please specify a tempo image in the CR or in the operator configuration")
	errNoDefaultTempoGatewayImage = errors.New("please specify a tempo-gateway image in the CR or in the operator configuration")
	errNoDefaultTempoQueryImage   = errors.New("please specify a tempo-query image in the CR or in the operator configuration")
)

// log is for logging in this package.
var microserviceslog = logf.Log.WithName("microservices-resource")

// SetupWebhookWithManager initializes the webhook.
func (r *Microservices) SetupWebhookWithManager(mgr ctrl.Manager, ctrlConfig v1alpha1.ProjectConfig) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(NewDefaulter(ctrlConfig)).
		WithValidator(&validator{client: mgr.GetClient(), ctrlConfig: ctrlConfig}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-tempo-grafana-com-v1alpha1-microservices,mutating=true,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=mmicroservices.kb.io,admissionReviewVersions=v1

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
	r, ok := obj.(*Microservices)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Microservices object but got %T", obj))
	}
	microserviceslog.V(1).Info("default", "name", r.Name)

	if r.Spec.Images.Tempo == "" {
		if d.ctrlConfig.DefaultImages.Tempo == "" {
			return errNoDefaultTempoImage
		}
		r.Spec.Images.Tempo = d.ctrlConfig.DefaultImages.Tempo
	}
	if r.Spec.Images.TempoQuery == "" {
		if d.ctrlConfig.DefaultImages.TempoQuery == "" {
			return errNoDefaultTempoQueryImage
		}
		r.Spec.Images.TempoQuery = d.ctrlConfig.DefaultImages.TempoQuery
	}
	if r.Spec.Images.TempoGateway == "" {
		if d.ctrlConfig.DefaultImages.TempoGateway == "" {
			return errNoDefaultTempoGatewayImage
		}
		r.Spec.Images.TempoGateway = d.ctrlConfig.DefaultImages.TempoGateway
	}

	if r.Spec.ServiceAccount == "" {
		r.Spec.ServiceAccount = naming.DefaultServiceAccountName(r.Name)
	}

	if r.Spec.Retention.Global.Traces.Duration == 0 {
		r.Spec.Retention.Global.Traces.Duration = 48 * time.Hour
	}

	if r.Spec.StorageSize.Cmp(zeroQuantity) <= 0 {
		r.Spec.StorageSize = tenGBQuantity
	}

	if r.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace == nil {
		defaultMaxSearchBytesPerTrace := 0
		r.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace = &defaultMaxSearchBytesPerTrace
	}

	if r.Spec.SearchSpec.DefaultResultLimit == nil {
		defaultDefaultResultLimit := 20
		r.Spec.SearchSpec.DefaultResultLimit = &defaultDefaultResultLimit
	}

	defaultComponentReplicas := pointer.Int32(1)
	defaultReplicationFactor := 1

	// Default replicas for ingester if not specified.
	if r.Spec.Components.Ingester.Replicas == nil {
		r.Spec.Components.Ingester.Replicas = defaultComponentReplicas
	}

	// Default replicas for distributor if not specified.
	if r.Spec.Components.Distributor.Replicas == nil {
		r.Spec.Components.Distributor.Replicas = defaultComponentReplicas
	}

	// Default replication factor if not specified.
	if r.Spec.ReplicationFactor == 0 {
		r.Spec.ReplicationFactor = defaultReplicationFactor
	}

	// Create an Ingress or Route to Jaeger UI by default if jaegerQuery is enabled
	if r.Spec.Components.QueryFrontend.JaegerQuery.Enabled && r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type == "" {
		if d.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
			r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type = IngressTypeRoute
		} else {
			r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type = IngressTypeIngress
		}
	}

	// Terminate TLS of the JaegerQuery Route on the Edge by default
	if r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type == IngressTypeRoute && r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Route.Termination == "" {
		r.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Route.Termination = TLSRouteTerminationTypeEdge
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-microservices,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=vmicroservices.kb.io,admissionReviewVersions=v1

type validator struct {
	client     client.Client
	ctrlConfig v1alpha1.ProjectConfig
}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return v.validate(ctx, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return v.validate(ctx, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	// NOTE(agerstmayr): change verbs in +kubebuilder:webhook to "verbs=create;update;delete" if you want to enable deletion validation.
	return nil
}

func (v *validator) validateServiceAccount(ctx context.Context, tempo Microservices) field.ErrorList {
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

// ValidateStorageSecret validates the object storage secret required for tempo.
func ValidateStorageSecret(tempo Microservices, storageSecret corev1.Secret) field.ErrorList {
	path := field.NewPath("spec").Child("storage").Child("secret")

	if storageSecret.Data == nil {
		return field.ErrorList{field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret is empty")}
	}

	var allErrs field.ErrorList
	for _, key := range []string{
		"endpoint",
		"bucket",
		"access_key_id",
		"access_key_secret",
	} {
		if storageSecret.Data[key] == nil || len(storageSecret.Data[key]) == 0 {
			allErrs = append(allErrs, field.Invalid(
				path,
				tempo.Spec.Storage.Secret,
				fmt.Sprintf("storage secret must contain \"%s\" field", key),
			))
		} else if key == "endpoint" {
			u, err := url.ParseRequestURI(string(storageSecret.Data["endpoint"]))

			// ParseRequestURI also accepts absolute paths, therefore we need to check if the URL scheme is set
			if err != nil || u.Scheme == "" {
				allErrs = append(allErrs, field.Invalid(
					path,
					tempo.Spec.Storage.Secret,
					"\"endpoint\" field of storage secret must be a valid URL",
				))
			}
		}
	}
	return allErrs
}

func (v *validator) validateStorage(ctx context.Context, tempo Microservices) field.ErrorList {
	storageSecret := &corev1.Secret{}
	err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.Secret}, storageSecret)
	if err != nil {
		// Do not fail the validation here, the user can create the storage secret later.
		// The operator will remain in a degraded condition until the storage secret is set.
		return field.ErrorList{}
	}

	return ValidateStorageSecret(tempo, *storageSecret)
}

func (v *validator) validateReplicationFactor(tempo Microservices) field.ErrorList {
	// Validate minimum quorum on ingestors according to replicas and replication factor
	replicatonFactor := tempo.Spec.ReplicationFactor
	// Ingester replicas should not be nil at this point, due defauler.
	ingesterReplicas := int(*tempo.Spec.Components.Ingester.Replicas)
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

func (v *validator) validateQueryFrontend(tempo Microservices) field.ErrorList {
	path := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("ingress").Child("type")
	ingressEnabled := tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type != "" && tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type != IngressTypeNone

	if ingressEnabled && !tempo.Spec.Components.QueryFrontend.JaegerQuery.Enabled {
		return field.ErrorList{field.Invalid(
			path,
			tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type,
			"Ingress cannot be enabled if jaegerQuery is disabled",
		)}
	}

	if tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type == IngressTypeRoute && !v.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
		return field.ErrorList{field.Invalid(
			path,
			tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type,
			"Please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
		)}
	}

	if tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type == IngressTypeIngress && v.ctrlConfig.Gates.OpenShift.OpenShiftRoute {
		return field.ErrorList{field.Invalid(
			path,
			tempo.Spec.Components.QueryFrontend.JaegerQuery.Ingress.Type,
			"Please disable the featureGates.openshift.openshiftRoute feature gate to use Ingress",
		)}
	}

	return nil
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) error {
	tempo, ok := obj.(*Microservices)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Microservices object but got %T", obj))
	}
	microserviceslog.V(1).Info("validate", "name", tempo.Name)

	var allErrs field.ErrorList
	allErrs = append(allErrs, v.validateServiceAccount(ctx, *tempo)...)
	allErrs = append(allErrs, v.validateStorage(ctx, *tempo)...)
	allErrs = append(allErrs, v.validateReplicationFactor(*tempo)...)
	allErrs = append(allErrs, v.validateQueryFrontend(*tempo)...)

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, allErrs)
}
