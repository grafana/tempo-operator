package v1alpha1

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	zeroQuantity = resource.MustParse("0Gi")
)

// log is for logging in this package.
var microserviceslog = logf.Log.WithName("microservices-resource")

func (r *Microservices) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-tempo-grafana-com-v1alpha1-microservices,mutating=true,failurePolicy=fail,sideEffects=None,groups=tempo.grafana.com,resources=microservices,verbs=create;update,versions=v1alpha1,name=mmicroservices.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Microservices{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *Microservices) Default() {
	microserviceslog.Info("default", "name", r.Name)
	if r.Spec.Retention.Global.Traces == 0 {
		r.Spec.Retention.Global.Traces = 48 * time.Hour
	}

	if r.Spec.StorageSize.Cmp(zeroQuantity) <= 0 {
		r.Spec.StorageSize = resource.MustParse("10Gi")
	}
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
