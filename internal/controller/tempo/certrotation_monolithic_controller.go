package controllers

import (
	"context"
	"errors"

	"github.com/ViaQ/logerr/v2/kverrors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation"
	"github.com/grafana/tempo-operator/internal/certrotation/handlers"
)

// CertRotationMonolithicReconciler reconciles the `tempo.grafana.com/certRotationRequiredAt` annotation on
// any TempoMonolithic object associated with any of the owned signer/client/serving certificates secrets
// and CA bundle configmap.
type CertRotationMonolithicReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	FeatureGates configv1alpha1.FeatureGates
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Compare the state specified by the TempoMonolithic object against the actual cluster state,
// and then perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *CertRotationMonolithicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithName("certrotation-reconcile").WithValues("tempo", req.NamespacedName)

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	var monolithic v1alpha1.TempoMonolithic
	if err := r.Get(ctx, req.NamespacedName, &monolithic); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, kverrors.Wrap(err, "failed to lookup tempostack", "name", req.NamespacedName)
	}
	managed := monolithic.Spec.Management == v1alpha1.ManagementStateManaged

	if !managed {
		log.Info("Skipping reconciliation for unmanaged TempoMonolithic resource", "name", req.String())
		// Stop requeueing for unmanaged TempoMonolithic custom resources
		return ctrl.Result{}, nil
	}

	rt, err := certrotation.ParseRotation(r.FeatureGates.BuiltInCertManagement)
	if err != nil {
		return ctrl.Result{}, err
	}

	checkExpiryAfter := expiryRetryAfter(rt.TargetCertRefresh)
	log.V(1).Info("Checking if TempoMonolithic certificates expired", "name", req.String(), "interval", checkExpiryAfter.String())

	var expired *certrotation.CertExpiredError

	err = handlers.CheckCertExpiry("tempomonolithic", ctx, log, req, r.Client, r.FeatureGates,
		certrotation.MonolithicComponentCertSecretNames(req.Name))
	switch {
	case errors.As(err, &expired):
		log.Info("Certificate expired", "msg", expired.Error())
	case err != nil:
		return ctrl.Result{}, err
	default:
		log.V(1).Info("Skipping cert rotation, all TempoMonolithic certificates still valid", "name", req.String())
		return ctrl.Result{
			RequeueAfter: checkExpiryAfter,
		}, nil
	}

	log.Error(err, "TempoMonolithic certificates expired", "name", req.String())
	err = handlers.AnnotateMonolithicForRequiredCertRotation(ctx, r.Client, req.Name, req.Namespace)
	if err != nil {
		log.Error(err, "failed to annotate required cert rotation", "name", req.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: checkExpiryAfter,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertRotationMonolithicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("certrotation_monolithic").
		For(&v1alpha1.TempoMonolithic{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
