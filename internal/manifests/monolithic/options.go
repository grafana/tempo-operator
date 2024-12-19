package monolithic

import (
	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

// Options defines calculated options required to generate all manifests.
type Options struct {
	CtrlConfig                configv1alpha1.ProjectConfig
	Tempo                     v1alpha1.TempoMonolithic
	StorageParams             manifestutils.StorageParams
	GatewayTenantSecret       []*manifestutils.GatewayTenantOIDCSecret
	GatewayTenantsData        []*manifestutils.GatewayTenantsData
	TLSProfile                tlsprofile.TLSProfileOptions
	useServiceCertsOnReceiver bool
}
