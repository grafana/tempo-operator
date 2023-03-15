package controllers

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/certrotation"
	"github.com/os-observability/tempo-operator/internal/certrotation/handlers"
)

// CertRotationReconciler reconciles the `tempo.grafana.com/certRotationRequiredAt` annotation on
// any TempoStack object associated with any of the owned signer/client/serving certificates secrets
// and CA bundle configmap.
type CertRotationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	FeatureGates configv1alpha1.FeatureGates
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Compare the state specified by the TempoStack object against the actual cluster state,
// and then perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *CertRotationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)
	log = log.WithValues("tempo", req.NamespacedName)

	rt, err := certrotation.ParseRotation(r.FeatureGates.BuiltInCertManagement)
	if err != nil {
		return ctrl.Result{}, err
	}

	checkExpiryAfter := expiryRetryAfter(rt.TargetCertRefresh)
	log.Info("Checking if TempoStack certificates expired", "name", req.String(), "interval", checkExpiryAfter.String())

	var expired *certrotation.CertExpiredError

	err = handlers.CheckCertExpiry(ctx, log, req, r.Client, r.FeatureGates)
	switch {
	case errors.As(err, &expired):
		log.Info("Certificate expired", "msg", expired.Error())
	case err != nil:
		return ctrl.Result{}, err
	default:
		log.Info("Skipping cert rotation, all TempoStack certificates still valid", "name", req.String())
		return ctrl.Result{
			RequeueAfter: checkExpiryAfter,
		}, nil
	}

	log.Error(err, "TempoStack certificates expired", "name", req.String())
	err = handlers.AnnotateForRequiredCertRotation(ctx, r.Client, req.Name, req.Namespace)
	if err != nil {
		log.Error(err, "failed to annotate required cert rotation", "name", req.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: checkExpiryAfter,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertRotationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TempoStack{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func expiryRetryAfter(certRefresh time.Duration) time.Duration {
	day := 24 * time.Hour
	if certRefresh > day {
		return 12 * time.Hour
	}

	return certRefresh / 4
}
