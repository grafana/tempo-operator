package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/internal/manifests/compactor"
	"github.com/os-observability/tempo-operator/internal/manifests/config"
	"github.com/os-observability/tempo-operator/internal/manifests/distributor"
	"github.com/os-observability/tempo-operator/internal/manifests/gateway"
	"github.com/os-observability/tempo-operator/internal/manifests/ingester"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/memberlist"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
	"github.com/os-observability/tempo-operator/internal/manifests/querier"
	"github.com/os-observability/tempo-operator/internal/manifests/queryfrontend"
	"github.com/os-observability/tempo-operator/internal/manifests/serviceaccount"
	"github.com/os-observability/tempo-operator/internal/manifests/servicemonitor"
)

// StorageParams holds storage configuration.
type StorageParams struct {
	AzureStorage *AzureStorage
	GCS          *GCS
	S3           *S3
}

// AzureStorage holds Azure Storage configuration.
type AzureStorage struct {
	Container string
	Env       string
}

// GCS holds Google Cloud Storage configuration.
type GCS struct {
	Bucket string
}

// S3 holds S3 configuration.
type S3 struct {
	Endpoint string
	Bucket   string
}

// BuildAll creates objects for Tempo deployment.
func BuildAll(params manifestutils.Params) ([]client.Object, error) {
	configMaps, configChecksum, err := config.BuildConfigMap(
		params.Tempo,
		config.Params{
			AzureStorage: config.AzureStorage{
				Container: params.StorageParams.AzureStorage.Container,
			},
			GCS: config.GCS{
				Bucket: params.StorageParams.GCS.Bucket,
			},
			S3: config.S3{
				Endpoint: params.StorageParams.S3.Endpoint,
				Bucket:   params.StorageParams.S3.Bucket,
			},
			HTTPEncryption: params.Gates.HTTPEncryption,
			GRPCEncryption: params.Gates.GRPCEncryption,
			TLSProfile:     params.TLSProfile,
		})
	if err != nil {
		return nil, err
	}
	params.ConfigChecksum = configChecksum

	ingesterObjs, err := ingester.BuildIngester(params)
	if err != nil {
		return nil, err
	}

	querierObjs, err := querier.BuildQuerier(params)
	if err != nil {
		return nil, err
	}
	frontendObjs, err := queryfrontend.BuildQueryFrontend(params)
	if err != nil {
		return nil, err
	}

	compactorObjs, err := compactor.BuildCompactor(params)
	if err != nil {
		return nil, err
	}

	gw, err := gateway.BuildGateway(params)
	if err != nil {
		return nil, err
	}

	distributorObjs, err := distributor.BuildDistributor(params)
	if err != nil {
		return nil, err
	}

	var manifests []client.Object
	manifests = append(manifests, configMaps)
	if params.Tempo.Spec.ServiceAccount == naming.DefaultServiceAccountName(params.Tempo.Name) {
		manifests = append(manifests, serviceaccount.BuildDefaultServiceAccount(params.Tempo))
	}
	manifests = append(manifests, distributorObjs...)
	manifests = append(manifests, ingesterObjs...)
	manifests = append(manifests, memberlist.BuildGossip(params.Tempo))
	manifests = append(manifests, frontendObjs...)
	manifests = append(manifests, querierObjs...)
	manifests = append(manifests, compactorObjs...)
	manifests = append(manifests, gw...)

	if params.Tempo.Spec.Observability.Metrics.CreateServiceMonitors {
		manifests = append(manifests, servicemonitor.BuildServiceMonitors(params)...)
	}

	return manifests, nil
}
