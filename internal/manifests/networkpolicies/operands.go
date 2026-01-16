package networkpolicies

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GenerateOperandPolicies to limit network access.
func GenerateOperandPolicies(params manifestutils.Params) []client.Object {
	tempo := params.Tempo
	if !tempo.Spec.NetworkPolicy.Enabled {
		return nil
	}

	labels := manifestutils.CommonLabels(tempo.Name)
	delete(labels, "app.kubernetes.io/name")

	policies := []client.Object{
		policyTempoGossip(tempo.Name, tempo.Namespace, labels),
	}

	// Add platform-specific DNS policy
	if params.CtrlConfig.Distribution == "openshift" {
		policies = append(policies, policyEgressAllowDNSOpenShift(tempo.Name, tempo.Namespace, labels))
	} else {
		policies = append(policies, policyEgressAllowDNS(tempo.Name, tempo.Namespace, labels))
	}

	policies = append(policies, generatePolicyFor(params, manifestutils.DistributorComponentName))
	policies = append(policies, generatePolicyFor(params, manifestutils.IngesterComponentName))
	policies = append(policies, generatePolicyFor(params, manifestutils.CompactorComponentName))
	policies = append(policies, generatePolicyFor(params, manifestutils.QuerierComponentName))
	policies = append(policies, generatePolicyFor(params, manifestutils.QueryFrontendComponentName))

	if tempo.Spec.Template.Gateway.Enabled {
		policies = append(policies, generatePolicyFor(params, manifestutils.GatewayComponentName))
	}

	return policies
}
