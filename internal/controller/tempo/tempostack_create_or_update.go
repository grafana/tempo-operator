package controllers

import (
	"context"
	"fmt"

	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/cloudcredentials"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

func (r *TempoStackReconciler) createOrUpdate(ctx context.Context, tempo v1alpha1.TempoStack) error {
	params := manifestutils.Params{
		Tempo:      tempo,
		CtrlConfig: r.CtrlConfig,
	}

	tokenCCOAuthEnv := cloudcredentials.DiscoverTokenCCOAuthConfig()

	// We can use this before inferred, as COO mode cannot be inferred and need to be set explicit
	// if is not set at this point, we need to clean up resources.
	if tokenCCOAuthEnv != nil && tempo.Spec.Storage.Secret.CredentialMode == v1alpha1.CredentialModeTokenCCO {

		ccoObjects, err := cloudcredentials.BuildCredentialsRequest(&tempo, tempo.Spec.ServiceAccount, tokenCCOAuthEnv)
		if err != nil {
			return err
		}

		ownedCCOObjects, err := r.findCCOOwnedByTempoOperator(ctx, tempo)
		if err != nil {
			return err
		}

		err = reconcileManagedObjects(ctx, r.Client, &tempo, r.Scheme, ccoObjects, ownedCCOObjects)

		if err != nil {
			return err
		}
	} else if tokenCCOAuthEnv == nil && tempo.Spec.Storage.Secret.CredentialMode == v1alpha1.CredentialModeTokenCCO {
		return &status.ConfigurationError{
			Reason: v1alpha1.ReasonInvalidStorageConfig,
			Message: listFieldErrors(
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "storage").Child("credentialMode"),
						v1alpha1.CredentialModeTokenCCO,
						"cannot configure tempo in CCO mode without CCO environment",
					),
				}),
		}
	}

	var errs field.ErrorList
	params.StorageParams, errs = storage.GetStorageParamsForTempoStack(ctx, r.Client, tempo)
	params.StorageParams.CloudCredentials.Environment = tokenCCOAuthEnv

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
	params.TLSProfile, err = tlsprofile.Get(ctx, r.CtrlConfig.Gates, r.Client)
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

func (r *TempoStackReconciler) findCCOOwnedByTempoOperator(ctx context.Context, tempo v1alpha1.TempoStack) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(manifestutils.CommonLabels(tempo.Name)),
	}

	credentialRequestsList := &cloudcredentialv1.CredentialsRequestList{}
	err := r.List(ctx, credentialRequestsList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing ingress: %w", err)
	}
	for i := range credentialRequestsList.Items {
		ownedObjects[credentialRequestsList.Items[i].GetUID()] = &credentialRequestsList.Items[i]
	}

	return ownedObjects, nil
}

func (r *TempoStackReconciler) findObjectsOwnedByTempoOperator(ctx context.Context, tempo v1alpha1.TempoStack) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOps := &client.ListOptions{
		Namespace:     tempo.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(manifestutils.CommonLabels(tempo.Name)),
	}
	clusterWideListOps := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(manifestutils.ClusterScopedCommonLabels(tempo.ObjectMeta)),
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
