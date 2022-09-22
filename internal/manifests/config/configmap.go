package config

import (
	tempoapp "github.com/grafana/tempo/cmd/tempo/app"
	tempodistributor "github.com/grafana/tempo/modules/distributor"
	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

const (
	otlpGRPCEndpoint = "0.0.0.0:4317"
	otlpHTTPEndpoint = "0.0.0.0:4318"
)

func config(tempo v1alpha1.Microservices) (string, error) {
	cfg := tempoapp.Config{
		Distributor: tempodistributor.Config{
			Receivers: map[string]interface{}{
				"otlp": otlpreceiver.Config{
					Protocols: otlpreceiver.Protocols{
						GRPC: &configgrpc.GRPCServerSettings{
							NetAddr: confignet.NetAddr{
								Endpoint: otlpGRPCEndpoint,
							},
						},
						HTTP: &confighttp.HTTPServerSettings{
							Endpoint: otlpHTTPEndpoint,
						},
					},
				},
			},
		},
	}
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func BuildConfigMaps(tempo v1alpha1.Microservices) client.Object {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: manifestutils.Name(tempo.Name, "configmap"),
		},
	}
}
