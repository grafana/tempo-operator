package monolithic

import (
	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// Options defines calculated options required to generate all manifests.
type Options struct {
	CtrlConfig          configv1alpha1.ProjectConfig
	Tempo               v1alpha1.TempoMonolithic
	StorageParams       manifestutils.StorageParams
	GatewayTenantSecret []*manifestutils.GatewayTenantOIDCSecret
	GatewayTenantsData  []*manifestutils.GatewayTenantsData
}
