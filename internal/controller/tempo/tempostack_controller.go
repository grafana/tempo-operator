package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation"
	"github.com/grafana/tempo-operator/internal/certrotation/handlers"
	"github.com/grafana/tempo-operator/internal/manifests/cloudcredentials"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/upgrade"
	"github.com/grafana/tempo-operator/internal/version"
)

const (
	storageSecretField = ".spec.storage.secret.name" // nolint #nosec
)

// TempoStackReconciler reconciles a TempoStack object.
type TempoStackReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	CtrlConfig configv1alpha1.ProjectConfig
	Version    version.Version
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services;configmaps;serviceaccounts;secrets;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;clusterroles;rolebindings;roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=create;get
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=operator.openshift.io,resources=ingresscontrollers,verbs=get;list;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=dnses,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=grafana.integreatly.org,resources=grafanadatasources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Upgrate for 0.11.0 to Tempo 2.5
// +kubebuilder:rbac:groups="core",resources=persistentvolumeclaims,verbs=list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create

//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/finalizers,verbs=update
// +kubebuilder:rbac:groups=cloudcredential.openshift.io,resources=credentialsrequests,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TempoStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithName("tempostack-reconcile")
	ctx = ctrl.LoggerInto(ctx, log)

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	tempo := v1alpha1.TempoStack{}
	if err := r.Get(ctx, req.NamespacedName, &tempo); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TempoStack")
			return ctrl.Result{}, fmt.Errorf("could not fetch tempo: %w", err)
		}
		// instance is not found, metrics can be cleared
		status.ClearTempoStackMetrics(req.Namespace, req.Name)

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, nil
	}

	// We have a deletion, short circuit and let the deletion happen
	if deletionTimestamp := tempo.GetDeletionTimestamp(); deletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&tempo, v1alpha1.TempoFinalizer) {
			// If the finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := finalize(ctx, r.Client, log, manifestutils.ClusterScopedCommonLabels(tempo.ObjectMeta)); err != nil {
				log.Error(err, "failed to finalize, re-reconciling")
				return ctrl.Result{}, err
			}

			// Once all finalizers have been
			// removed, the object will be deleted.
			if controllerutil.RemoveFinalizer(&tempo, v1alpha1.TempoFinalizer) {
				err := r.Update(ctx, &tempo)
				if err != nil {
					log.Error(err, "failed to remove finalizer, re-reconciling")
					return ctrl.Result{}, err
				}
			}
		}

		return ctrl.Result{}, nil
	}

	if tempo.Spec.ManagementState != v1alpha1.ManagementStateManaged {
		log.Info("Skipping reconciliation for unmanaged TempoStack resource", "name", req.String())
		// Stop requeueing for unmanaged TempoStack custom resources
		return ctrl.Result{}, nil
	}

	// New CRs with empty OperatorVersion are ignored, as they're already up-to-date.
	// The versions will be set when the status field is refreshed.
	if tempo.Status.OperatorVersion != "" && tempo.Status.OperatorVersion != r.Version.OperatorVersion {
		upgraded, err := upgrade.Upgrade{
			Client:     r.Client,
			Recorder:   r.Recorder,
			CtrlConfig: r.CtrlConfig,
			Version:    r.Version,
			Log:        log.WithName("upgrade"),
		}.Upgrade(ctx, &tempo)
		if err != nil {
			return r.handleReconcileStatus(ctx, log, tempo, err)
		}
		tempo = *upgraded.(*v1alpha1.TempoStack)
	}

	if r.CtrlConfig.Gates.BuiltInCertManagement.Enabled {
		err := handlers.CreateOrRotateCertificates(ctx, log, req, r.Client, r.Scheme, r.CtrlConfig.Gates, certrotation.TempoStackComponentCertSecretNames(req.Name))
		if err != nil {
			return r.handleReconcileStatus(ctx, log, tempo, fmt.Errorf("built in cert manager error: %w", err))
		}
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(&tempo, v1alpha1.TempoFinalizer) {
		if controllerutil.AddFinalizer(&tempo, v1alpha1.TempoFinalizer) {
			err := r.Update(ctx, &tempo)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	err := r.createOrUpdate(ctx, tempo)
	if err != nil {
		return r.handleReconcileStatus(ctx, log, tempo, err)
	}

	// Update the components status also in case of no reconciliation errors.
	return r.handleReconcileStatus(ctx, log, tempo, nil)
}

// handleReconcileStatus updates the status of each component and sets an appropriate status condition:
//
//   - No error: Only update components status
//
//   - For ConfigurationError: Set the status condition to ConfigurationError.
//     Return a reconcile.TerminalError to indicate that human intervention is required
//     to resolve this error, and that the reconciliation request should not be requeued.
//
//   - For any other error: Set the status condition to Failed,
//     the Reason to "FailedReconciliation" and the message to the error message.
func (r *TempoStackReconciler) handleReconcileStatus(ctx context.Context, log logr.Logger, tempo v1alpha1.TempoStack, reconcileError error) (ctrl.Result, error) { //nolint:unparam
	// First refresh components
	newStatus, rerr := status.GetComponentsStatus(ctx, r, tempo)
	if rerr != nil {
		log.Error(rerr, "could not get components status")
	}

	var configurationError *status.ConfigurationError
	if reconcileError == nil {
		// No error.
	} else if errors.As(reconcileError, &configurationError) {
		// Handle configuration error
		newStatus.Conditions = status.UpdateCondition(tempo, metav1.Condition{
			Type:    string(v1alpha1.ConditionConfigurationError),
			Reason:  string(configurationError.Reason),
			Message: configurationError.Message,
		})

		// wrap error in reconcile.TerminalError to indicate human intervention is required
		// and the request should not be requeued.
		reconcileError = reconcile.TerminalError(configurationError)
	} else {
		// Handle all other errors (e.g. permission errors, etc.)
		newStatus.Conditions = status.UpdateCondition(tempo, metav1.Condition{
			Type:    string(v1alpha1.ConditionFailed),
			Reason:  string(v1alpha1.ReasonFailedReconciliation),
			Message: reconcileError.Error(),
		})
	}

	// Refresh status
	rerr = status.Refresh(ctx, r, tempo, &newStatus)
	if rerr != nil {
		return ctrl.Result{}, rerr
	}

	// Note: controller-runtime will always reconcile if this function returns any error except TerminalError.
	// Result.Requeue and Result.RequeueAfter are only respected if err == nil
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.15.0/pkg/internal/controller/controller.go#L315-L341
	return ctrl.Result{}, reconcileError
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
		Named("tempostack").
		For(&v1alpha1.TempoStack{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findTempoStackForStorageSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)

	if r.CtrlConfig.Gates.OpenShift.OpenShiftRoute {
		builder = builder.Owns(&routev1.Route{})
	}

	if r.CtrlConfig.Gates.PrometheusOperator {
		builder = builder.Owns(&monitoringv1.ServiceMonitor{})
		builder = builder.Owns(&monitoringv1.PrometheusRule{})
	}

	if r.CtrlConfig.Gates.GrafanaOperator {
		builder = builder.Owns(&grafanav1.GrafanaDatasource{})
	}

	tokenCCOAuthEnv := cloudcredentials.DiscoverTokenCCOAuthConfig()
	if tokenCCOAuthEnv != nil {
		builder = builder.Owns(&cloudcredentialv1.CredentialsRequest{})
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

	ccoStack := v1alpha1.TempoStack{}
	err = r.Get(ctx, client.ObjectKey{Namespace: secret.GetNamespace(),
		Name: manifestutils.TempoFromManagerCredentialSecretName(secret.GetName())}, &ccoStack)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return requests
		}
	} else {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ccoStack.GetName(),
				Namespace: ccoStack.GetNamespace(),
			},
		})
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
