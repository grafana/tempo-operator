package monolithic

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildAll generates all manifests.
func BuildAll(opts Options) ([]client.Object, error) {
	var manifests []client.Object

	configMap, configChecksum, err := BuildConfigMap(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, configMap)
	opts.ConfigChecksum = configChecksum

	statefulSet, err := BuildTempoStatefulset(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, statefulSet)

	service := BuildTempoService(opts)
	manifests = append(manifests, service)

	return manifests, nil
}