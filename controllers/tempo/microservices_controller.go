package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/status"
)

const (
	storageSecretField = ".spec.storage.secret" // nolint #nosec
)

// MicroservicesReconciler reconciles a Microservices object.
type MicroservicesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=services;configmaps;serviceaccounts;secrets;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=tempo.grafana.com,resources=microservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=microservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=microservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Microservices object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MicroservicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log = log.WithValues("tempo", req.NamespacedName)
	tempo := v1alpha1.Microservices{}
	if err := r.Get(ctx, req.NamespacedName, &tempo); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TempoMicroservices")
			return ctrl.Result{}, fmt.Errorf("could not fetch tempo: %w", err)
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, nil
	}

	err := r.reconcileManifests(ctx, log, req, tempo)
	return r.handleStatus(ctx, tempo, err)
}

// handleStatus set all components status, then verify if an error is type status.DegradedError, if that is the case, it will update the CR
// status conditions to Degraded. if not it will throw an error as usual.
func (r *MicroservicesReconciler) handleStatus(ctx context.Context, tempo v1alpha1.Microservices, err error) (ctrl.Result, error) {
	// First refresh components
	newStatus, rerr := status.GetComponetsStatus(ctx, r, tempo)
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

func (r *MicroservicesReconciler) reconcileManifests(ctx context.Context, log logr.Logger, req ctrl.Request, tempo v1alpha1.Microservices) error {
	storageConfig, err := r.getStorageConfig(ctx, tempo)
	if err != nil {
		return &status.DegradedError{
			Reason:  v1alpha1.ReasonInvalidStorageConfig,
			Message: err.Error(),
			Requeue: false,
		}
	}

	objects, err := manifests.BuildAll(manifestutils.Params{Tempo: tempo, StorageParams: *storageConfig})
	// TODO (pavolloffay) check error type and change return appropriately
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	errCount := 0
	for _, obj := range objects {
		l := log.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
		)

		if isNamespaceScoped(obj) {
			obj.SetNamespace(req.Namespace)
			if err := ctrl.SetControllerReference(&tempo, obj, r.Scheme); err != nil {
				l.Error(err, "failed to set controller owner reference to resource")
				errCount++
				continue
			}
		}

		desired := obj.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(obj, desired)

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, obj, mutateFn)
		if err != nil {
			l.Error(err, "failed to configure resource")
			errCount++
			continue
		}

		l.Info(fmt.Sprintf("Resource has been %s", op))
	}

	if errCount > 0 {
		return fmt.Errorf("failed to create objects for Tempo %s", req.NamespacedName)
	}
	return nil
}

func (r *MicroservicesReconciler) getStorageConfig(ctx context.Context, tempo v1alpha1.Microservices) (*manifestutils.StorageParams, error) {
	storageSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.Secret}, storageSecret)
	if err != nil {
		return nil, fmt.Errorf("could not fetch storage secret: %w", err)
	}

	fieldErrs := v1alpha1.ValidateStorageSecret(tempo, *storageSecret)
	if len(fieldErrs) > 0 {
		msgs := make([]string, len(fieldErrs))
		for i, fieldErr := range fieldErrs {
			msgs[i] = fieldErr.Detail
		}
		return nil, fmt.Errorf("invalid storage secret: %s", strings.Join(msgs, ", "))
	}

	return &manifestutils.StorageParams{S3: manifestutils.S3{
		Endpoint: string(storageSecret.Data["endpoint"]),
		Bucket:   string(storageSecret.Data["bucket"]),
	}}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroservicesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add an index to the storage secret field in the Microservices CRD.
	// If the content of any secret in the cluster changes, the watcher can identify related Microservices CRs
	// and reconcile them (i.e. update the tempo configuration file and restart the pods)
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Microservices{}, storageSecretField, func(rawObj client.Object) []string {
		microservices := rawObj.(*v1alpha1.Microservices)
		if microservices.Spec.Storage.Secret == "" {
			return nil
		}
		return []string{microservices.Spec.Storage.Secret}
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Microservices{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&v1.StatefulSet{}).
		Owns(&v1.Deployment{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findMicroservicesForStorageSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func isNamespaceScoped(obj client.Object) bool {
	switch obj.(type) {
	case *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
		return false
	default:
		return true
	}
}

func (r *MicroservicesReconciler) findMicroservicesForStorageSecret(secret client.Object) []reconcile.Request {
	microservices := &v1alpha1.MicroservicesList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(storageSecretField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	err := r.List(context.TODO(), microservices, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(microservices.Items))
	for i, item := range microservices.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// StatusClient methods, used for fetch component pod status and refresh the status of the CR.
func (r *MicroservicesReconciler) GetPodsComponent(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
	pods := &corev1.PodList{}

	opts := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(componentName, stack.Name)),
		client.InNamespace(stack.Namespace),
	}
	err := r.Client.List(ctx, pods, opts...)
	return pods, err
}

func (r *MicroservicesReconciler) PatchStatus(ctx context.Context, changed, original *v1alpha1.Microservices) error {
	statusPatch := client.MergeFrom(original)
	return r.Client.Status().Patch(ctx, changed, statusPatch)
}
