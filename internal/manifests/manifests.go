package manifests

import (
	tempov1alpha1 "github.com/os-observability/tempo-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/internal/manifests/config"
	"github.com/os-observability/tempo-operator/internal/manifests/distributor"
	"github.com/os-observability/tempo-operator/internal/manifests/ingester"
)

type Params struct {
	Tempo tempov1alpha1.Microservices
}

func BuildAll(params Params) []client.Object {
	var manifests []client.Object
	manifests = append(manifests, distributor.BuildDistributor(params.Tempo)...)
	manifests = append(manifests, ingester.BuildIngester(params.Tempo)...)
	manifests = append(manifests, config.BuildConfigMaps(params.Tempo))
	return manifests
}
