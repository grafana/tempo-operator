package naming

import (
	"fmt"
)

// Name returns component name.
func Name(component string, instanceName string) string {
	if component == "" {
		return fmt.Sprintf("tempo-%s", instanceName)
	}
	return fmt.Sprintf("tempo-%s-%s", instanceName, component)
}

// DefaultServiceAccountName returns the name of the default tempo service account to use.
func DefaultServiceAccountName(name string) string {
	return Name("serviceaccount", name)
}
