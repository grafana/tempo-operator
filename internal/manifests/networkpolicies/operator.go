package networkpolicies

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// GenerateOperatorPolicies to limit network access.
func GenerateOperatorPolicies(namespace string, apiServerInfo manifestutils.KubeAPIServerInfo) []client.Object {
	const instanceName = "operator"
	labels := manifestutils.CommonOperatorLabels()
	objs := []client.Object{
		policyAPIServer(instanceName, namespace, apiServerInfo),
		policyDenyAll(instanceName, namespace, labels),
		policyIngressToMetrics(instanceName, namespace, labels),
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		objs = append(objs, policyWebhook(instanceName, namespace))
	}
	return objs
}
