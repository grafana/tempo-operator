package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/config"
	"github.com/os-observability/tempo-operator/internal/manifests/distributor"
	"github.com/os-observability/tempo-operator/internal/manifests/ingester"
	"github.com/os-observability/tempo-operator/internal/manifests/memberlist"
	"github.com/os-observability/tempo-operator/internal/manifests/querier"
	"github.com/os-observability/tempo-operator/internal/manifests/queryfrontend"
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
	configMaps, err := config.BuildConfigs(params.Tempo, config.Params{S3: config.S3{
		Endpoint: params.StorageParams.S3.Endpoint,
		Bucket:   params.StorageParams.S3.Bucket,
	}})
	if err != nil {
		return nil, err
	}

	ingesterObjs, err := ingester.BuildIngester(params.Tempo)
	if err != nil {
		return nil, err
	}

	querierObjs, err := querier.BuildQuerier(params.Tempo)
	if err != nil {
		return nil, err
	}
	frontendObjs, err := queryfrontend.BuildQueryFrontend(params.Tempo)
	if err != nil {
		return nil, err
	}

	var manifests []client.Object
	manifests = append(manifests, configMaps)
	manifests = append(manifests, distributor.BuildDistributor(params.Tempo)...)
	manifests = append(manifests, ingesterObjs...)
	manifests = append(manifests, memberlist.BuildGossip(params.Tempo))
	manifests = append(manifests, frontendObjs...)
	manifests = append(manifests, querierObjs...)
	return manifests, nil
}
