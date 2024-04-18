package gateway

import (
	"bytes"
	"embed"
	"fmt"
	"math/rand"
	"text/template"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	//go:embed gateway-rbac.yaml
	tempoGatewayRbacYAMLTmplFile embed.FS

	//go:embed gateway-tenants.yaml
	tempoGatewayTenantsYAMLTmplFile embed.FS

	rbacTemplate = template.Must(template.ParseFS(tempoGatewayRbacYAMLTmplFile, "gateway-rbac.yaml"))

	tenantsTemplate = template.Must(template.ParseFS(tempoGatewayTenantsYAMLTmplFile, "gateway-tenants.yaml"))
)

// generate gateway RBAC configuration file.
func buildRBACConfig(opts options) (rbacCfg string, err error) {
	// Build tempo gateway rbac yaml
	byteBuffer := &bytes.Buffer{}
	err = rbacTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create tempo gateway rbac configuration, err: %w", err)
	}
	rbacCfg = byteBuffer.String()
	return rbacCfg, nil
}

// generate gateway tenants configuration file.
func buildTenantsConfig(opts options) (tenantsCfg string, err error) {
	// Build tempo gateway tenants yaml
	byteBuffer := &bytes.Buffer{}
	err = tenantsTemplate.Execute(byteBuffer, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create tempo gateway tenants configuration, err: %w", err)
	}
	tenantsCfg = byteBuffer.String()
	return tenantsCfg, nil
}

// NewConfigOptions initializes options required to build the configuration files for multitenancy.
func NewConfigOptions(
	namespace string,
	name string,
	saName string,
	routeHost string,
	opaPackage string,
	tenantsSpec v1alpha1.TenantsSpec,
	oidcSecrets []*manifestutils.GatewayTenantOIDCSecret,
	tenantsData []*manifestutils.GatewayTenantsData,
) options {
	var auths []authentication
	for _, tenantAuth := range tenantsSpec.Authentication {
		cookieSecret := ""
		tenantData := getTenantData(tenantAuth.TenantName, tenantsData)
		if tenantData != nil && tenantData.OpenShiftCookieSecret != "" {
			cookieSecret = tenantData.OpenShiftCookieSecret
		} else {
			cookieSecret = newCookieSecret()
		}

		auth := authentication{
			TenantName:            tenantAuth.TenantName,
			TenantID:              tenantAuth.TenantID,
			OpenShiftCookieSecret: cookieSecret,
			OIDC:                  tenantAuth.OIDC,
			RedirectURL:           fmt.Sprintf("https://%s/openshift/%s/callback", routeHost, tenantAuth.TenantName),
		}

		oidcTenantSecret := getOIDCSecret(tenantAuth.TenantName, oidcSecrets)
		if oidcTenantSecret != nil {
			auth.OIDCSecret = oidcSecret{
				ClientID:     oidcTenantSecret.ClientID,
				ClientSecret: oidcTenantSecret.ClientSecret,
				IssuerCAPath: oidcTenantSecret.IssuerCAPath,
			}
		}

		auths = append(auths, auth)
	}

	return options{
		Namespace:      namespace,
		Name:           name,
		ServiceAccount: saName,
		OPAUrl:         fmt.Sprintf("http://localhost:%d/v1/data/%s/allow", gatewayOPAHTTPPort, opaPackage),
		Tenants: &tenants{
			Mode:           tenantsSpec.Mode,
			Authentication: auths,
			Authorization:  tenantsSpec.Authorization,
		},
	}
}

func getTenantData(tenantName string, tenantsData []*manifestutils.GatewayTenantsData) *manifestutils.GatewayTenantsData {
	for _, d := range tenantsData {
		if d.TenantName == tenantName {
			return d
		}
	}
	return nil
}

func getOIDCSecret(tenantName string, OIDCSecrets []*manifestutils.GatewayTenantOIDCSecret) *manifestutils.GatewayTenantOIDCSecret {
	for _, s := range OIDCSecrets {
		if s.TenantName == tenantName {
			return s
		}
	}
	return nil
}

// options is used to render the rbac.yaml and tenants.yaml file template.
type options struct {
	Name      string
	Namespace string
	Tenants   *tenants

	ServiceAccount string
	OPAUrl         string
}

type tenants struct {
	Mode v1alpha1.ModeType

	Authentication []authentication
	Authorization  *v1alpha1.AuthorizationSpec
}

type authentication struct {
	TenantName string
	TenantID   string

	RedirectURL string
	// OpenShiftCookieSecret is used for encrypting the auth token when put into the browser session.
	OpenShiftCookieSecret string
	OIDC                  *v1alpha1.OIDCSpec
	OIDCSecret            oidcSecret
}

// secret for clientID, clientSecret and issuerCAPath for tenant's authentication.
type oidcSecret struct {
	ClientID     string
	ClientSecret string
	IssuerCAPath string
}

var (
	cookieSecretLength       = 32
	cookieSecretAllowedRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func newCookieSecret() string {
	b := make([]rune, cookieSecretLength)
	for i := range b {
		b[i] = cookieSecretAllowedRunes[rand.Intn(len(cookieSecretAllowedRunes))] // nolint:gosec
	}
	return string(b)
}
