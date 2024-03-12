package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

func (r *TempoStackReconciler) createOrUpdate(ctx context.Context, log logr.Logger, tempo v1alpha1.TempoStack) error {
	params := manifestutils.Params{
		Tempo:      tempo,
		CtrlConfig: r.CtrlConfig,
	}

	var errs field.ErrorList
	params.StorageParams, errs = storage.GetStorageParamsForTempoStack(ctx, r.Client, tempo)
	if len(errs) > 0 {
		return &status.ConfigurationError{
			Reason:  v1alpha1.ReasonInvalidStorageConfig,
			Message: listFieldErrors(errs),
		}
	}

	if tempo.Spec.Tenants != nil {
		var err error
		params.GatewayTenantSecret, params.GatewayTenantsData, err = getTenantParams(ctx, r.Client, &r.CtrlConfig, tempo.Namespace, tempo.Name, *tempo.Spec.Tenants, tempo.Spec.Template.Gateway.Enabled)
		if err != nil {
			return err
		}
	}

	var err error
	params.TLSProfile, err = tlsprofile.Get(ctx, r.CtrlConfig.Gates, r.Client, log)
	if err != nil {
		switch err {
		case tlsprofile.ErrGetProfileFromCluster:
		case tlsprofile.ErrGetInvalidProfile:
			return &status.ConfigurationError{
				Message: err.Error(),
				Reason:  v1alpha1.ReasonCouldNotGetOpenShiftTLSPolicy,
			}
		default:
			return err
		}

	}

	managedObjects, err := manifests.BuildAll(params)
	// TODO (pavolloffay) check error type and change return appropriately
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	// Collect all objects owned by the operator, to be able to prune objects
	// which exist in the cluster but are not managed by the operator anymore.
	// For example, when the Jaeger Query Ingress is enabled and later disabled,
	// the Ingress object should be removed from the cluster.
	ownedObjects, err := r.findObjectsOwnedByTempoOperator(ctx, tempo)
	if err != nil {
		return err
	}

	err = reconcileManagedObjects(ctx, r.Client, &tempo, r.Scheme, managedObjects, ownedObjects)
	if err != nil {
		return err
	}

	return nil
}

func (r *TempoStackReconciler) findObjectsOwnedByTempoOperator(ctx context.Context, tempo v1alpha1.TempoStack) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(manifestutils.CommonLabels(tempo.Name)),
	}

	// Add all resources where the operator can conditionally create an object.
	// For example, Ingress and Route can be enabled or disabled in the CR.

	ingressList := &networkingv1.IngressList{}
	err := r.List(ctx, ingressList, listOps)
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
