package networking

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GenerateOperandPolicies to limit network access.
func GenerateOperandPolicies(tempo v1alpha1.TempoStack) []client.Object {
	if !tempo.Spec.Networking.Enabled {
		return nil
	}
	labels := manifestutils.CommonLabels(tempo.Name)
	policies := []client.Object{
		policyDenyAll(tempo.Name, tempo.Namespace, labels),
		policyIngressToMetrics(tempo.Name, tempo.Namespace, labels),
		policyEgressAllowDNS(tempo.Name, tempo.Namespace, labels),
	}

	policies = append(policies, generatePolicyFor(tempo, manifestutils.DistributorComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.IngesterComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.CompactorComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.QuerierComponentName))
	policies = append(policies, generatePolicyFor(tempo, manifestutils.QueryFrontendComponentName))

	return policies
}
