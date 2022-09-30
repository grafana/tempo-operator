package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/config"
	"github.com/os-observability/tempo-operator/internal/manifests/distributor"
	"github.com/os-observability/tempo-operator/internal/manifests/ingester"
	"github.com/os-observability/tempo-operator/internal/manifests/memberlist"
)

// Params holds parameters used to create Tempo objects.
type Params struct {
	Tempo v1alpha1.Microservices
}

// BuildAll creates objects for Tempo deployment.
func BuildAll(params Params) ([]client.Object, error) {
	var manifests []client.Object
	configMaps, err := config.BuildConfigs(params.Tempo)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, distributor.BuildDistributor(params.Tempo)...)
	manifests = append(manifests, ingester.BuildIngester(params.Tempo)...)
	manifests = append(manifests, configMaps)
	manifests = append(manifests, memberlist.BuildGossip(params.Tempo))
	return manifests, nil
}
