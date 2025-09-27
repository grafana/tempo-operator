package controllers

import (
	"context"
	"errors"
	"fmt"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
	"github.com/grafana/tempo-operator/internal/handlers/storage"
	"github.com/grafana/tempo-operator/internal/manifests/cloudcredentials"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/monolithic"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
	"github.com/grafana/tempo-operator/internal/upgrade"
	"github.com/grafana/tempo-operator/internal/version"
)

// TempoMonolithicReconciler reconciles a TempoMonolithic object.
type TempoMonolithicReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	CtrlConfig configv1alpha1.ProjectConfig
	Version    version.Version
}

//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tempo.grafana.com,resources=tempomonolithics/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TempoMonolithicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithName("tempomonolithic-reconcile")
	ctx = ctrl.LoggerInto(ctx, log)

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	tempo := v1alpha1.TempoMonolithic{}
	if err := r.Get(ctx, req.NamespacedName, &tempo); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TempoMonolithic")
			return ctrl.Result{}, fmt.Errorf("could not fetch tempo: %w", err)
		}
		// instance is not found, metrics can be cleared
		status.ClearMonolithicMetrics(req.Namespace, req.Name)

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
			if err := finalize(ctx, r.Client, log, monolithic.ClusterScopedCommonLabels(tempo.ObjectMeta)); err != nil {
				return ctrl.Result{}, err
			}

			// Once all finalizers have been
			// removed, the object will be deleted.
			if controllerutil.RemoveFinalizer(&tempo, v1alpha1.TempoFinalizer) {
				err := r.Update(ctx, &tempo)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		return ctrl.Result{}, nil
	}

	if tempo.Spec.Management == v1alpha1.ManagementStateUnmanaged {
		log.Info("Skipping reconciliation for unmanaged TempoMonolithic resource", "name", req.String())
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
			return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
		}
		tempo = *upgraded.(*v1alpha1.TempoMonolithic)
	}

	if r.CtrlConfig.Gates.BuiltInCertManagement.Enabled {
		err := monolithic.CreateOrRotateCertificates(ctx, log, req, r.Client, r.Scheme, r.CtrlConfig.Gates, certrotation.MonolithicComponentCertSecretNames(req.Name))
		if err != nil {
			return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, fmt.Errorf("built in cert manager error: %w", err))
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

	// Apply ephemeral defaults after upgrade.
	// The ephemeral defaults should not be written back to the cluster.
	tempo.Default(r.CtrlConfig)

	// Discover CCO Environment configured (if any)
	tokenCCOAuthEnv := cloudcredentials.DiscoverTokenCCOAuthConfig()

	// We can use this before inferred, as COO mode cannot be inferred and need to be set explicit
	// if is not set at this point, we need to clean up resources.
	if tokenCCOAuthEnv != nil && r.getCredentialMode(tempo) == v1alpha1.CredentialModeTokenCCO {
		ccoObjects, err := cloudcredentials.BuildCredentialsRequest(&tempo, tempo.Spec.ServiceAccount, tokenCCOAuthEnv)
		if err != nil {
			return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
		}

		ownedCCOObjects, err := r.getCCOOwnedObjects(ctx, tempo)
		if err != nil {
			return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
		}

		err = reconcileManagedObjects(ctx, r.Client, &tempo, r.Scheme, ccoObjects, ownedCCOObjects)

		if err != nil {
			return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
		}
	} else if tokenCCOAuthEnv == nil && r.getCredentialMode(tempo) == v1alpha1.CredentialModeTokenCCO {
		return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo,
			errors.New("cannot configure tempo in CCO mode without CCO environment"))
	}

	err := r.createOrUpdate(ctx, tempo)
	if err != nil {
		return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
	}

	// Note: controller-runtime will always requeue a reconcile if Reconcile() returns any error except TerminalError.
	// Result.Requeue and Result.RequeueAfter are only respected if err == nil
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.15.0/pkg/internal/controller/controller.go#L315-L341
	return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, nil)
}

func (r *TempoMonolithicReconciler) getCredentialMode(tempo v1alpha1.TempoMonolithic) v1alpha1.CredentialMode {
	if tempo.Spec.Storage.Traces.Backend == v1alpha1.MonolithicTracesStorageBackendS3 {
		return tempo.Spec.Storage.Traces.S3.CredentialMode
	}

	// We only support inference mode for others, so return empty string
	return ""
}

func (r *TempoMonolithicReconciler) createOrUpdate(ctx context.Context, tempo v1alpha1.TempoMonolithic) error {
	opts := monolithic.Options{
		CtrlConfig: r.CtrlConfig,
		Tempo:      tempo,
	}

	var errs field.ErrorList
	opts.StorageParams, errs = storage.GetStorageParamsForTempoMonolithic(ctx, r.Client, tempo)
	if len(errs) > 0 {
		return &status.ConfigurationError{
			Reason:  v1alpha1.ReasonInvalidStorageConfig,
			Message: listFieldErrors(errs),
		}
	}

	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		var err error
		opts.GatewayTenantSecret, opts.GatewayTenantsData, err = getTenantParams(ctx, r.Client, &r.CtrlConfig, tempo.Namespace, tempo.Name, tempo.Spec.Multitenancy.TenantsSpec, true)
		if err != nil {
			return err
		}
	}

	var err error
	opts.TLSProfile, err = tlsprofile.Get(ctx, r.CtrlConfig.Gates, r.Client)
	if err != nil {
		switch err {
		case tlsprofile.ErrGetProfileFromCluster, tlsprofile.ErrGetInvalidProfile:
			return &status.ConfigurationError{
				Message: err.Error(),
				Reason:  v1alpha1.ReasonCouldNotGetOpenShiftTLSPolicy,
			}
		default:
			return err
		}
	}

	managedObjects, err := monolithic.BuildAll(opts)
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	ownedObjects, err := r.getOwnedObjects(ctx, tempo)
	if err != nil {
		return err
	}

	return reconcileManagedObjects(ctx, r.Client, &tempo, r.Scheme, managedObjects, ownedObjects)
}
func (r *TempoMonolithicReconciler) getCCOOwnedObjects(ctx context.Context, tempo v1alpha1.TempoMonolithic) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(monolithic.CommonLabels(tempo.Name)),
	}
	credentialRequestsList := &cloudcredentialv1.CredentialsRequestList{}
	err := r.List(ctx, credentialRequestsList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing cloud credential requests: %w", err)
	}
	for i := range credentialRequestsList.Items {
		ownedObjects[credentialRequestsList.Items[i].GetUID()] = &credentialRequestsList.Items[i]
	}
	return ownedObjects, nil
}

func (r *TempoMonolithicReconciler) getOwnedObjects(ctx context.Context, tempo v1alpha1.TempoMonolithic) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(monolithic.CommonLabels(tempo.Name)),
	}
	clusterWideListOps := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(monolithic.ClusterScopedCommonLabels(tempo.ObjectMeta)),
	}

	// Add all resources where the operator can conditionally create an object.
	// For example, Ingress and Route can be enabled or disabled in the CR.

	servicesList := &corev1.ServiceList{}
	err := r.List(ctx, servicesList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing services: %w", err)
	}
	for i := range servicesList.Items {
		ownedObjects[servicesList.Items[i].GetUID()] = &servicesList.Items[i]
	}

	ingressList := &networkingv1.IngressList{}
	err = r.List(ctx, ingressList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing ingress: %w", err)
	}
	for i := range ingressList.Items {
		ownedObjects[ingressList.Items[i].GetUID()] = &ingressList.Items[i]
	}

	// metrics reader for Jaeger UI Monitor Tab
	rolesList := &rbacv1.RoleList{}
	err = r.List(ctx, rolesList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing roles: %w", err)
	}
	for i := range rolesList.Items {
		ownedObjects[rolesList.Items[i].GetUID()] = &rolesList.Items[i]
	}

	// metrics reader for Jaeger UI Monitor Tab
	roleBindingList := &rbacv1.RoleBindingList{}
	err = r.List(ctx, roleBindingList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing role bindings: %w", err)
	}
	for i := range roleBindingList.Items {
		ownedObjects[roleBindingList.Items[i].GetUID()] = &roleBindingList.Items[i]
	}

	// TokenReview and SubjectAccessReview when gateway is configured with multi-tenancy in OpenShift mode
	clusterRoleList := &rbacv1.ClusterRoleList{}
	err = r.List(ctx, clusterRoleList, clusterWideListOps)
	if err != nil {
		return nil, fmt.Errorf("error listing cluster roles: %w", err)
	}
	for i := range clusterRoleList.Items {
		ownedObjects[clusterRoleList.Items[i].GetUID()] = &clusterRoleList.Items[i]
	}

	// TokenReview and SubjectAccessReview when gateway is configured with multi-tenancy in OpenShift mode
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	err = r.List(ctx, clusterRoleBindingList, clusterWideListOps)
	if err != nil {
		return nil, fmt.Errorf("error listing cluster role bindings: %w", err)
	}
	for i := range clusterRoleBindingList.Items {
		ownedObjects[clusterRoleBindingList.Items[i].GetUID()] = &clusterRoleBindingList.Items[i]
	}

	if r.CtrlConfig.Gates.PrometheusOperator {
		servicemonitorList := &monitoringv1.ServiceMonitorList{}
		err := r.List(ctx, servicemonitorList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing service monitors: %w", err)
		}
		for i := range servicemonitorList.Items {
			ownedObjects[servicemonitorList.Items[i].GetUID()] = servicemonitorList.Items[i]
		}

		prometheusRulesList := &monitoringv1.PrometheusRuleList{}
		err = r.List(ctx, prometheusRulesList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing prometheus rules: %w", err)
		}
		for i := range prometheusRulesList.Items {
			ownedObjects[prometheusRulesList.Items[i].GetUID()] = prometheusRulesList.Items[i]
		}
	}

	if r.CtrlConfig.Gates.OpenShift.OpenShiftRoute {
		routesList := &routev1.RouteList{}
		err := r.List(ctx, routesList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing routes: %w", err)
		}
		for i := range routesList.Items {
			ownedObjects[routesList.Items[i].GetUID()] = &routesList.Items[i]
		}
	}

	if r.CtrlConfig.Gates.GrafanaOperator {
		datasourceList := &grafanav1.GrafanaDatasourceList{}
		err := r.List(ctx, datasourceList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing datasources: %w", err)
		}
		for i := range datasourceList.Items {
			ownedObjects[datasourceList.Items[i].GetUID()] = &datasourceList.Items[i]
		}
	}

	return ownedObjects, nil
}

func (r *TempoMonolithicReconciler) findTempoMonolithicForStorageSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	monolithics := &v1alpha1.TempoMonolithicList{}

	err := r.List(ctx, monolithics, &client.ListOptions{
		Namespace: secret.GetNamespace(),
	})

	if err != nil {
		return []reconcile.Request{}
	}

	var requests []reconcile.Request

	for _, tempomonolithic := range monolithics.Items {
		if matchTempoMonolithicStorageSecret(tempomonolithic, secret.GetName()) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tempomonolithic.GetName(),
					Namespace: tempomonolithic.GetNamespace(),
				},
			})
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

// SetupWithManager sets up the controller with the Manager.
func (r *TempoMonolithicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		Named("tempomonolithic").
		For(&v1alpha1.TempoMonolithic{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findTempoMonolithicForStorageSecret),
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

func matchTempoMonolithicStorageSecret(item v1alpha1.TempoMonolithic, secretName string) bool {
	if item.Spec.Storage == nil {
		return false
	}
	storageConfig := item.Spec.Storage.Traces
	return (storageConfig.S3 != nil && storageConfig.S3.Secret == secretName) ||
		(storageConfig.Azure != nil && storageConfig.Azure.Secret == secretName) ||
		(storageConfig.GCS != nil && storageConfig.GCS.Secret == secretName)
}
