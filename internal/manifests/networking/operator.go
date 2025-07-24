package networking

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GenerateOperatorPolicies to limit network access.
func GenerateOperatorPolicies(namespace string) []client.Object {
	const instanceName = "operator"
	labels := manifestutils.CommonOperatorLabels()
	objs := []client.Object{
		policyDenyAll(instanceName, namespace, labels),
		policyIngressToMetrics(instanceName, namespace, labels),
		policyEgressAllowDNS(instanceName, namespace, labels),
		policyAPIServer(instanceName, namespace),
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		objs = append(objs, policyWebhook(instanceName, namespace))
	}
	return objs
}
