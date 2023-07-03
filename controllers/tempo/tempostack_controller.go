package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	tempov1alpha1 "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation/handlers"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/status"
)

const (
	storageSecretField = ".spec.storage.secret.name" // nolint #nosec
)

// TempoStackReconciler reconciles a TempoStack object.
type TempoStackReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	FeatureGates configv1alpha1.FeatureGates
}

// +kubebuilder:rbac:groups="",resources=services;configmaps;serviceaccounts;secrets;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=operator.openshift.io,resources=ingresscontrollers,verbs=get;list;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=dnses,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TempoStack object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *TempoStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithName("tempostack-reconcile").WithValues("tempo", req.NamespacedName)

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	tempo := v1alpha1.TempoStack{}
	if err := r.Get(ctx, req.NamespacedName, &tempo); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TempoTempoStack")
			return ctrl.Result{}, fmt.Errorf("could not fetch tempo: %w", err)
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, nil
	}

	if tempo.Spec.ManagementState != tempov1alpha1.ManagementStateManaged {
		log.Info("Skipping reconciliation for unmanaged TempoStack resource", "name", req.String())
		// Stop requeueing for unmanaged TempoStack custom resources
		return ctrl.Result{}, nil
	}

	if r.FeatureGates.BuiltInCertManagement.Enabled {
		err := handlers.CreateOrRotateCertificates(ctx, log, req, r.Client, r.Scheme, r.FeatureGates)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("built in cert manager error: %w", err)
		}
	}

	err := r.createOrUpdate(ctx, log, req, tempo)
	return r.handleStatus(ctx, tempo, err)
}

// handleStatus set all components status, then verify if an error is type status.DegradedError, if that is the case, it will update the CR
// status conditions to Degraded. if not it will throw an error as usual.
func (r *TempoStackReconciler) handleStatus(ctx context.Context, tempo v1alpha1.TempoStack, err error) (ctrl.Result, error) {
	// First refresh components
	newStatus, rerr := status.GetComponentsStatus(ctx, r, tempo)
	requeue := false

	if rerr != nil {
		return ctrl.Result{
			Requeue:      requeue,
			RequeueAfter: time.Second,
		}, err
	}

	// If is not degraded error, refresh and return error
	var degraded *status.DegradedError
	if !errors.As(err, &degraded) {
		requeue, rerr := status.Refresh(ctx, r, tempo, &newStatus)
		// Error refreshing components status
		if rerr != nil {
			return ctrl.Result{
				Requeue:      requeue,
				RequeueAfter: time.Second,
			}, err
		}
		// Return original error
		return ctrl.Result{
			Requeue: false,
		}, err
	}

	// Degraded error
	newStatus.Conditions = status.DegradedCondition(tempo, degraded.Message, degraded.Reason)

	// Refresh status
	requeue, err = status.Refresh(ctx, r, tempo, &newStatus)

	// Error refreshing status
	if err != nil {
		return ctrl.Result{
			Requeue:      requeue || degraded.Requeue,
			RequeueAfter: time.Second,
		}, err
	}
	// No errors at all.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TempoStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add an index to the storage secret field in the TempoStack CRD.
	// If the content of any secret in the cluster changes, the watcher can identify related TempoStack CRs
	// and reconcile them (i.e. update the tempo configuration file and restart the pods)
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.TempoStack{}, storageSecretField, func(rawObj client.Object) []string {
		tempostacks := rawObj.(*v1alpha1.TempoStack)
		if tempostacks.Spec.Storage.Secret.Name == "" {
			return nil
		}
		return []string{tempostacks.Spec.Storage.Secret.Name}
	})
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TempoStack{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&networkingv1.Ingress{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findTempoStackForStorageSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)

	if r.FeatureGates.PrometheusOperator {
		builder = builder.Owns(&monitoringv1.ServiceMonitor{})
		builder = builder.Owns(&monitoringv1.PrometheusRule{})
	}

	if r.FeatureGates.OpenShift.OpenShiftRoute {
		builder = builder.Owns(&routev1.Route{})
	}

	return builder.Complete(r)
}

func (r *TempoStackReconciler) findTempoStackForStorageSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	tempostacks := &v1alpha1.TempoStackList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(storageSecretField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	err := r.List(ctx, tempostacks, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(tempostacks.Items))
	for i, item := range tempostacks.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// GetPodsComponent is used for fetching component pod status and refreshing the status of the CR.
func (r *TempoStackReconciler) GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.TempoStack) (*corev1.PodList, error) {
	pods := &corev1.PodList{}

	opts := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(componentName, stack.Name)),
		client.InNamespace(stack.Namespace),
	}
	err := r.Client.List(ctx, pods, opts...)
	return pods, err
}

// PatchStatus patches the status field of the CR.
func (r *TempoStackReconciler) PatchStatus(ctx context.Context, changed, original *v1alpha1.TempoStack) error {
	statusPatch := client.MergeFrom(original)
	return r.Client.Status().Patch(ctx, changed, statusPatch)
}
