package gateway

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

var (
	//go:embed gateway-rbac.yaml
	tempoGatewayRbacYAMLTmplFile embed.FS

	//go:embed gateway-tenants.yaml
	tempoGatewayTenantsYAMLTmplFile embed.FS

	rbacTemplate = template.Must(template.ParseFS(tempoGatewayRbacYAMLTmplFile, "gateway-rbac.yaml"))

	tenantsTemplate = template.Must(template.ParseFS(tempoGatewayTenantsYAMLTmplFile, "gateway-tenants.yaml"))
)

// generate gateway configuration files.
func buildConfigFiles(opts options) (rbacCfg string, tenantsCfg string, err error) {
	// Build tempo gateway rbac yaml
	byteBuffer := &bytes.Buffer{}
	err = rbacTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to create tempo gateway rbac configuration, err: %w", err)
	}
	rbacCfg = byteBuffer.String()
	// Build tempo gateway tenants yaml
	byteBuffer.Reset()
	err = tenantsTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to create tempo gateway tenants configuration, err: %w", err)
	}
	tenantsCfg = byteBuffer.String()
	return rbacCfg, tenantsCfg, nil
}

// options is used to render the rbac.yaml and tenants.yaml file template.
type options struct {
	Namespace     string
	Name          string
	Tenants       *v1alpha1.TenantsSpec
	TenantSecrets []*tenantSecret
}

// secret for clientID, clientSecret and issuerCAPath for tenant's authentication.
type tenantSecret struct {
	TenantName   string
	ClientID     string
	ClientSecret string
	IssuerCAPath string
}
