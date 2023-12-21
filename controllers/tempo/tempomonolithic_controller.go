package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/monolithic"
)

// TempoMonolithicReconciler reconciles a TempoMonolithic object.
type TempoMonolithicReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	CtrlConfig configv1alpha1.ProjectConfig
}

//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TempoMonolithicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("tempomonolithic-reconcile")

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	tempo := v1alpha1.TempoMonolithic{}
	if err := r.Get(ctx, req.NamespacedName, &tempo); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TempoMonolithic")
			return ctrl.Result{}, fmt.Errorf("could not fetch tempo: %w", err)
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, nil
	}

	if tempo.Spec.Management == v1alpha1.ManagementStateUnmanaged {
		log.Info("Skipping reconciliation for unmanaged TempoMonolithic resource", "name", req.String())
		return ctrl.Result{}, nil
	}

	managedObjects, err := monolithic.BuildAll(monolithic.Options{
		CtrlConfig: r.CtrlConfig,
		Tempo:      tempo,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building manifests: %w", err)
	}

	err = reconcileManagedObjects(ctx, log, r.Client, &tempo, r.Scheme, managedObjects)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TempoMonolithicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TempoMonolithic{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
