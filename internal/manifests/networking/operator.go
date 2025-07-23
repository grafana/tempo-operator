package networking

import (
	"errors"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateOperatorPolicies() ([]client.Object, error) {
	if true {
		return nil, nil
	}
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, errors.New("Unable to generate Operator Network Policy. Can not determine namespace.")
	}
	namespace := string(ns)
	const instanceName = "tempo-operator"

	return []client.Object{
		policyDenyAll(instanceName, namespace),
		policyIngressToMetrics(instanceName, namespace),
		policyEgressAllowDNS(instanceName, namespace),
		policyAPIServer(instanceName, namespace),
	}, nil
}
