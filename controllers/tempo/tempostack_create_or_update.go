package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/gateway"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

func (r *TempoStackReconciler) getStorageConfig(ctx context.Context, tempo v1alpha1.TempoStack) (manifestutils.StorageParams, error) {
	storageSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: tempo.Namespace, Name: tempo.Spec.Storage.Secret.Name}, storageSecret)
	if err != nil {
		return manifestutils.StorageParams{}, fmt.Errorf("could not fetch storage secret: %w", err)
	}

	fieldErrs := v1alpha1.ValidateStorageSecret(tempo, *storageSecret)
	if len(fieldErrs) > 0 {
		msgs := make([]string, len(fieldErrs))
		for i, fieldErr := range fieldErrs {
			msgs[i] = fieldErr.Detail
		}
		return manifestutils.StorageParams{}, fmt.Errorf("invalid storage secret: %s", strings.Join(msgs, ", "))
	}

	params := manifestutils.StorageParams{}

	switch tempo.Spec.Storage.Secret.Type {
	case v1alpha1.ObjectStorageSecretAzure:
		params.AzureStorage = GetAzureParams(tempo, storageSecret)
	case v1alpha1.ObjectStorageSecretGCS:
		params.GCS = GetGCSParams(tempo, storageSecret)
	case v1alpha1.ObjectStorageSecretS3:
		params.S3 = GetS3Params(tempo, storageSecret)
	default:
		return manifestutils.StorageParams{}, fmt.Errorf("storage secret type is not recognized")
	}

	return params, nil
}

func isNamespaceScoped(obj client.Object) bool {
	switch obj.(type) {
	case *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
		return false
	default:
		return true
	}
}

func (r *TempoStackReconciler) createOrUpdate(ctx context.Context, log logr.Logger, req ctrl.Request, tempo v1alpha1.TempoStack) error {
	storageConfig, err := r.getStorageConfig(ctx, tempo)
	if err != nil {
		return &status.ConfigurationError{
			Reason:  v1alpha1.ReasonInvalidStorageConfig,
			Message: err.Error(),
		}
	}

	if err = v1alpha1.ValidateTenantConfigs(tempo); err != nil {
		return &status.ConfigurationError{
			Message: fmt.Sprintf("Invalid tenants configuration: %s", err),
			Reason:  v1alpha1.ReasonInvalidTenantsConfiguration,
		}
	}

	if tempo.Spec.Tenants != nil && tempo.Spec.Tenants.Mode == v1alpha1.ModeOpenShift && r.FeatureGates.OpenShift.BaseDomain == "" {
		domain, err := gateway.GetOpenShiftBaseDomain(ctx, r.Client)
		if err != nil {
			return err
		}
		log.Info("OpenShift base domain set", "openshift-base-domain", domain)
		r.FeatureGates.OpenShift.BaseDomain = domain
	}

	tlsProfile, err := tlsprofile.Get(ctx, r.FeatureGates, r.Client, log)
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

	// Collect all objects owned by the operator, to be able to prune objects
	// which exist in the cluster but are not managed by the operator anymore.
	// For example, when the Jaeger Query Ingress is enabled and later disabled,
	// the Ingress object should be removed from the cluster.
	pruneObjects, err := r.findObjectsOwnedByTempoOperator(ctx, tempo)
	if err != nil {
		return err
	}

	var tenantSecrets []*manifestutils.GatewayTenantOIDCSecret
	if tempo.Spec.Tenants != nil && tempo.Spec.Tenants.Mode == v1alpha1.ModeStatic {
		tenantSecrets, err = gateway.GetOIDCTenantSecrets(ctx, r.Client, tempo)
		if err != nil {
			return err
		}
	}

	var gatewayTenantsData []*manifestutils.GatewayTenantsData
	if tempo.Spec.Tenants != nil && tempo.Spec.Tenants.Mode == v1alpha1.ModeOpenShift {
		gatewayTenantsData, err = gateway.GetGatewayTenantsData(ctx, r.Client, tempo)
		if err != nil {
			// just log the error the secret is not created if the loop for an instance runs for the first time.
			log.Info("Failed to get gateway secret and/or tenants.yaml", "error", err)
		}
	}

	managedObjects, err := manifests.BuildAll(manifestutils.Params{
		Tempo:               tempo,
		StorageParams:       storageConfig,
		Gates:               r.FeatureGates,
		TLSProfile:          tlsProfile,
		GatewayTenantSecret: tenantSecrets,
		GatewayTenantsData:  gatewayTenantsData,
	})
	// TODO (pavolloffay) check error type and change return appropriately
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	errs := []error{}
	for _, obj := range managedObjects {
		l := log.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
		)

		if isNamespaceScoped(obj) {
			obj.SetNamespace(req.Namespace)
			if err := ctrl.SetControllerReference(&tempo, obj, r.Scheme); err != nil {
				l.Error(err, "failed to set controller owner reference to resource")
				errs = append(errs, err)
				continue
			}
		}

		desired := obj.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(obj, desired)

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, obj, mutateFn)
		if err != nil {
			l.Error(err, "failed to configure resource")
			errs = append(errs, err)
			continue
		}

		l.V(1).Info(fmt.Sprintf("resource has been %s", op))

		// This object is still managed by the operator, remove it from the list of objects to prune
		delete(pruneObjects, obj.GetUID())
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create objects for TempoStack %s: %w", req.NamespacedName, errors.Join(errs...))
	}

	// Prune owned objects in the cluster which are not managed anymore.
	pruneErrs := []error{}
	for _, obj := range pruneObjects {
		l := log.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
		)
		l.Info("pruning unmanaged resource")

		err = r.Delete(ctx, obj)
		if err != nil {
			l.Error(err, "failed to delete resource")
			pruneErrs = append(pruneErrs, err)
		}
	}
	if len(pruneErrs) > 0 {
		return fmt.Errorf("failed to prune objects of TempoStack %s: %w", req.NamespacedName, errors.Join(pruneErrs...))
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

	if r.FeatureGates.PrometheusOperator {
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

	if r.FeatureGates.OpenShift.OpenShiftRoute {
		routesList := &routev1.RouteList{}
		err := r.List(ctx, routesList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing routes: %w", err)
		}
		for i := range routesList.Items {
			ownedObjects[routesList.Items[i].GetUID()] = &routesList.Items[i]
		}
	}

	return ownedObjects, nil
}
