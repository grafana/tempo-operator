package manifestutils

import (
	"k8s.io/apimachinery/pkg/labels"
)

// ComponentLabels is a list of all commonLabels including the app.kubernetes.io/component:<component> label.
func ComponentLabels(component, instanceName string) labels.Set {
	return labels.Merge(CommonLabels(instanceName), map[string]string{
		"app.kubernetes.io/component": component,
	})
}

// CommonLabels returns common labels for each object created by the operator.
func CommonLabels(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "tempo",
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": "tempo-operator",
	}
}
