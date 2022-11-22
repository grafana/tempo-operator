package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	zeroQuantity                = resource.MustParse("0Gi")
	tenGBQuantity               = resource.MustParse("10Gi")
	ErrNoDefaultTempoImage      = errors.New("please specify a tempo image in the CR or in the operator configuration")
	ErrNoDefaultTempoQueryImage = errors.New("please specify a tempo-query image in the CR or in the operator configuration")
)

// log is for logging in this package.
var microserviceslog = logf.Log.WithName("microservices-resource")

func (r *Microservices) SetupWebhookWithManager(mgr ctrl.Manager, defaultImages ImagesSpec) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&defaulter{defaultImages: defaultImages}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

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
			return ErrNoDefaultTempoImage
		}
		r.Spec.Images.Tempo = d.defaultImages.Tempo
	}
	if r.Spec.Images.TempoQuery == "" {
		if d.defaultImages.TempoQuery == "" {
			return ErrNoDefaultTempoQueryImage
		}
		r.Spec.Images.TempoQuery = d.defaultImages.TempoQuery
	}

	if r.Spec.Retention.Global.Traces == 0 {
		r.Spec.Retention.Global.Traces = 48 * time.Hour
	}

	if r.Spec.StorageSize.Cmp(zeroQuantity) <= 0 {
		r.Spec.StorageSize = tenGBQuantity
	}

	if r.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace == nil {
		defaultMaxSearchBytesPerTrace := 0
		r.Spec.LimitSpec.Global.Query.MaxSearchBytesPerTrace = &defaultMaxSearchBytesPerTrace
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-tempo-grafana-com-v1alpha1-microservices,mutating=false,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=vmicroservices.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Microservices{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *Microservices) ValidateCreate() error {
	microserviceslog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *Microservices) ValidateUpdate(old runtime.Object) error {
	microserviceslog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *Microservices) ValidateDelete() error {
	microserviceslog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
