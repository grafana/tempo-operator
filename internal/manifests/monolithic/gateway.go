package monolithic

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/gateway"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	opaPackage = "tempomonolithic"
)

// BuildGatewayObjects builds auxiliary objects required for multitenancy.
//
// Enabling multi-tenancy (multitenancy.enabled=true) and configuring at least one tenant has the following effect on the deployment:
// * a tempo-$name-gateway ConfigMap (rbac.yaml) and Secret (tenants.yaml) will be created
// * a tempo-gateway container is added to the StatefulSet
// * a tempo-$name-gateway service will be created and the tempo-$name and tempo-$name-jaegerui will be removed
//
// additionally, if mode=openshift
// * a tempo-gateway-opa container is added to the StatefulSet
// * the ServiceAccount will get additional annotations (serviceaccounts.openshift.io/oauth-redirectreference.$tenantName)
// * a ClusterRole (tokenreviews/create) and ClusterRoleBinding will be created for the ServiceAccount.
func BuildGatewayObjects(opts Options) ([]client.Object, map[string]string, error) {
	tempo := opts.Tempo
	manifests := []client.Object{}
	extraAnnotations := map[string]string{}
	labels := ComponentLabels(manifestutils.GatewayComponentName, tempo.Name)
	gatewayObjectName := naming.Name(manifestutils.GatewayComponentName, tempo.Name)

	cfgOpts := gateway.NewConfigOptions(
		tempo.Namespace,
		tempo.Name,
		naming.DefaultServiceAccountName(tempo.Name),
		naming.RouteFqdn(tempo.Namespace, tempo.Name, "jaegerui", opts.CtrlConfig.Gates.OpenShift.BaseDomain),
		opaPackage,
		tempo.Spec.Multitenancy.TenantsSpec,
		opts.GatewayTenantSecret,
		opts.GatewayTenantsData,
	)

	rbacConfigMap, rbacHash, err := gateway.NewRBACConfigMap(cfgOpts, tempo.Namespace, gatewayObjectName, labels)
	if err != nil {
		return nil, nil, err
	}
	extraAnnotations["tempo.grafana.com/rbacConfig.hash"] = rbacHash
	manifests = append(manifests, rbacConfigMap)

	tenantsSecret, tenantsHash, err := gateway.NewTenantsSecret(cfgOpts, tempo.Namespace, gatewayObjectName, labels)
	if err != nil {
		return nil, nil, err
	}
	extraAnnotations["tempo.grafana.com/tenantsConfig.hash"] = tenantsHash
	manifests = append(manifests, tenantsSecret)

	if tempo.Spec.Multitenancy.TenantsSpec.Mode == v1alpha1.ModeOpenShift {
		manifests = append(manifests, gateway.NewAccessReviewClusterRole(
			// ClusterRole is a cluster scoped resource, therefore we need to add the namespace to the name
			fmt.Sprintf("%s-%s", gatewayObjectName, tempo.Namespace),
			ClusterScopedComponentLabels(tempo.ObjectMeta, manifestutils.GatewayComponentName),
		))

		manifests = append(manifests, gateway.NewAccessReviewClusterRoleBinding(
			// ClusterRole is a cluster scoped resource, therefore we need to add the namespace to the name
			fmt.Sprintf("%s-%s", gatewayObjectName, tempo.Namespace),
			ClusterScopedComponentLabels(tempo.ObjectMeta, manifestutils.GatewayComponentName),
			tempo.Namespace,
			naming.DefaultServiceAccountName(tempo.Name),
		))
	}

	return manifests, extraAnnotations, nil
}
