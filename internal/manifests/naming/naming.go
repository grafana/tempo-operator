package naming

import (
	"fmt"
)

// Name returns the manifest name of a component.
// Example: tempo-simplest-compactor.
func Name(component string, tempoStackName string) string {
	if component == "" {
		return fmt.Sprintf("tempo-%s", tempoStackName)
	}
	return fmt.Sprintf("tempo-%s-%s", tempoStackName, component)
}

// ServiceName returns the name of a service of a component.
// This is the name of the TLS secret of a service and
// is part of the ServerName of TLS secured components.
//
// Example: tempo-simplest-compactor-http
// Note: This is not the name of the Kubernetes service object. Please use
// naming.Name(component, tempoStackName) to get the name of the Kubernetes object.
func ServiceName(tempoStackName string, component string, service string) string {
	return fmt.Sprintf("%s-%s", Name(component, tempoStackName), service)
}

// ServiceFqdn returns the fully qualified domain name of a service.
func ServiceFqdn(namespace string, tempoStackName string, component string, service string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", ServiceName(tempoStackName, component, service), namespace)
}

// DefaultServiceAccountName returns the name of the default tempo service account to use.
func DefaultServiceAccountName(name string) string {
	return Name("", name)
}

// SigningCABundleName return CA bundle configmap name.
func SigningCABundleName(name string) string {
	return fmt.Sprintf("tempo-%s-ca-bundle", name)
}
