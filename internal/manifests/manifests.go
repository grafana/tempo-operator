package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/config"
	"github.com/os-observability/tempo-operator/internal/manifests/distributor"
	"github.com/os-observability/tempo-operator/internal/manifests/ingester"
)

// Params holds parameters used to create Tempo objects.
type Params struct {
	StorageParams StorageParams
	Tempo         v1alpha1.Microservices
}

// StorageParams holds storage configuration.
type StorageParams struct {
	S3 S3
}

// S3 holds S3 configuration.
type S3 struct {
	Endpoint string
	Bucket   string
}

// BuildAll creates objects for Tempo deployment.
func BuildAll(params Params) ([]client.Object, error) {
	var manifests []client.Object
	configMaps, err := config.BuildConfigs(params.Tempo, config.Params{S3: config.S3{
		Endpoint: params.StorageParams.S3.Endpoint,
		Bucket:   params.StorageParams.S3.Bucket,
	}})
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, distributor.BuildDistributor(params.Tempo)...)
	ingesterObjs, err := ingester.BuildIngester(params.Tempo)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, ingesterObjs...)
	manifests = append(manifests, configMaps)
	return manifests, nil
}
