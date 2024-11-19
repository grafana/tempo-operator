package controllers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
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

	// Apply ephemeral defaults after upgrade.
	// The ephemeral defaults should not be written back to the cluster.
	tempo.Default(r.CtrlConfig)

	err := r.createOrUpdate(ctx, tempo)
	if err != nil {
		return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, err)
	}

	// Note: controller-runtime will always requeue a reconcile if Reconcile() returns any error except TerminalError.
	// Result.Requeue and Result.RequeueAfter are only respected if err == nil
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.15.0/pkg/internal/controller/controller.go#L315-L341
	return ctrl.Result{}, status.HandleTempoMonolithicStatus(ctx, r.Client, tempo, nil)
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

func (r *TempoMonolithicReconciler) getOwnedObjects(ctx context.Context, tempo v1alpha1.TempoMonolithic) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(monolithic.CommonLabels(tempo.Name)),
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

// SetupWithManager sets up the controller with the Manager.
func (r *TempoMonolithicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		Named("tempomonolithic" + uuid.New().String()).
		For(&v1alpha1.TempoMonolithic{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&networkingv1.Ingress{})

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

	return builder.Complete(r)
}
