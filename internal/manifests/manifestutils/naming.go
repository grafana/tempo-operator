package manifestutils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

// Name returns component name.
func Name(component string, instanceName string) string {
	if component == "" {
		return fmt.Sprintf("tempo-%s", instanceName)
	}
	return fmt.Sprintf("tempo-%s-%s", instanceName, component)
}

// ComponentLabels is a list of all commonLabels including the app.kubernetes.io/component:<component> label.
func ComponentLabels(component, instanceName string) labels.Set {
	return labels.Merge(commonLabels(instanceName), map[string]string{
		"app.kubernetes.io/component": component,
	})
}

func commonLabels(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "tempo",
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": "tempo-controller",
		"app.kubernetes.io/created-by": "tempo-controller",
	}
}
