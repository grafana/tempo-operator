package monolithic

import (
	"maps"
)

// CommonAnnotations returns common annotations for each pod created by the operator.
func CommonAnnotations(opts Options) map[string]string {
	annotations := map[string]string{}
	if opts.StorageParams.SecretHash != "" {
		annotations["tempo.grafana.com/storageSecret.hash"] = opts.StorageParams.SecretHash
	}
	if opts.StorageParams.CloudCredentials.ContentHash != "" {
		annotations["tempo.grafana.com/token.cco.auth.hash"] = opts.StorageParams.CloudCredentials.ContentHash
	}
	maps.Copy(annotations, opts.CertHashAnnotations)
	return annotations
}
