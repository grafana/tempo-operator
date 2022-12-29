package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

var (
	zeroQuantity                = resource.MustParse("0Gi")
	tenGBQuantity               = resource.MustParse("10Gi")
	errNoDefaultTempoImage      = errors.New("please specify a tempo image in the CR or in the operator configuration")
	errNoDefaultTempoQueryImage = errors.New("please specify a tempo-query image in the CR or in the operator configuration")
)

// log is for logging in this package.
var microserviceslog = logf.Log.WithName("microservices-resource")

func (r *Microservices) SetupWebhookWithManager(mgr ctrl.Manager, defaultImages ImagesSpec) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&defaulter{defaultImages: defaultImages}).
		WithValidator(&validator{client: mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-tempo-grafana-com-v1alpha1-microservices,mutating=true,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=mmicroservices.kb.io,admissionReviewVersions=v1

type defaulter struct {
	defaultImages ImagesSpec
}

func (d *defaulter) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*Microservices)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Microservices object but got %T", obj))
	}
	microserviceslog.Info("default", "name", r.Name)

	if r.Spec.Images.Tempo == "" {
		if d.defaultImages.Tempo == "" {
			return errNoDefaultTempoImage
		}
		r.Spec.Images.Tempo = d.defaultImages.Tempo
	}
	if r.Spec.Images.TempoQuery == "" {
		if d.defaultImages.TempoQuery == "" {
			return errNoDefaultTempoQueryImage
		}
		r.Spec.Images.TempoQuery = d.defaultImages.TempoQuery
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

	if r.Spec.SearchSpec.Enabled == nil {
		defaultSearchEnabled := true
		r.Spec.SearchSpec.Enabled = &defaultSearchEnabled
	}
	return nil
}

//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-microservices,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=vmicroservices.kb.io,admissionReviewVersions=v1

type validator struct {
	client client.Client
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

func (v *validator) validateServiceAccount(ctx context.Context, tempo *Microservices) field.ErrorList {
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

func (v *validator) validateStorageSecret(tempo *Microservices, storageSecret *corev1.Secret) field.ErrorList {
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

func (v *validator) validateStorage(ctx context.Context, tempo *Microservices) field.ErrorList {
	path := field.NewPath("spec").Child("storage").Child("secret")

	if tempo.Spec.Storage.Secret == "" {
		return field.ErrorList{field.Invalid(path, tempo.Spec.Storage.Secret, "storage secret is required")}
	}

	storageSecret := &corev1.Secret{}
	err := v.client.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.Secret}, storageSecret)
	if err != nil {
		return field.ErrorList{field.Invalid(path, tempo.Spec.Storage.Secret, err.Error())}
	}

	return v.validateStorageSecret(tempo, storageSecret)
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) error {
	tempo, ok := obj.(*Microservices)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Microservices object but got %T", obj))
	}
	microserviceslog.Info("validate", "name", tempo.Name)

	var allErrs field.ErrorList
	allErrs = append(allErrs, v.validateServiceAccount(ctx, tempo)...)
	allErrs = append(allErrs, v.validateStorage(ctx, tempo)...)
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(tempo.GroupVersionKind().GroupKind(), tempo.Name, allErrs)
}
