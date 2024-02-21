package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// GetGatewayTenantsData return parts of gateways tenants.yaml configuration file.
func GetGatewayTenantsData(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
) ([]*manifestutils.GatewayTenantsData, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Name: naming.Name(manifestutils.GatewayComponentName, name), Namespace: namespace}
	err := k8sClient.Get(ctx, key, secret)
	if err != nil {
		return nil, err
	}
	tenantConfigYAML := secret.Data[manifestutils.GatewayTenantFileName]
	tenantConfig, err := decodeTenantConfig(tenantConfigYAML)
	if err != nil {
		return nil, err
	}

	var tenantData []*manifestutils.GatewayTenantsData
	for _, t := range tenantConfig.Tenants {
		if t.OpenShift != nil && t.OpenShift.CookieSecret != "" {
			tenantData = append(tenantData, &manifestutils.GatewayTenantsData{
				TenantName:            t.Name,
				OpenShiftCookieSecret: t.OpenShift.CookieSecret,
			})
		}
	}
	return tenantData, nil
}

func decodeTenantConfig(tenantConfigYAML []byte) (*tenantsConfigJSON, error) {
	tenantConfigJSON, err := yaml.YAMLToJSON(tenantConfigYAML)
	if err != nil {
		return nil, fmt.Errorf("error in converting tenants.yaml to JSON: %w", err)
	}

	var tenantConfig tenantsConfigJSON
	err = json.Unmarshal(tenantConfigJSON, &tenantConfig)
	if err != nil {
		return nil, fmt.Errorf("error in unmarshalling tenant config to struct: %w", err)
	}
	return &tenantConfig, nil
}

// tenantConfigJSON is a helper struct to decode tenant configuration.
type tenantsConfigJSON struct {
	Tenants []tenantsSpec `json:"tenants,omitempty"`
}

type tenantsSpec struct {
	Name      string         `json:"name"`
	ID        string         `json:"id"`
	OpenShift *openShiftSpec `json:"openshift"`
}

type openShiftSpec struct {
	CookieSecret string `json:"cookieSecret"`
}
