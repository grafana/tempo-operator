package v1alpha1

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

var (
	twoGBQuantity                       = resource.MustParse("2Gi")
	tenGBQuantity                       = resource.MustParse("10Gi")
	defaultServicesDuration             = metav1.Duration{Duration: time.Hour * 24 * 3}
	defaultTimeout                      = metav1.Duration{Duration: time.Second * 30}
	defaultFindTracesConcurrentRequests = 2
)

// Default sets all default values in a central place, instead of setting it at every place where the value is accessed.
// NOTE: This function is called inside the Reconcile loop, NOT in the webhook.
// We want to keep the CR as minimal as the user configures it, and not modify it in any way (except for upgrades).
func (r *TempoMonolithic) Default(ctrlConfig configv1alpha1.ProjectConfig) {
	if r.Spec.Management == "" {
		r.Spec.Management = ManagementStateManaged
	}

	if r.Spec.Storage == nil {
		r.Spec.Storage = &MonolithicStorageSpec{}
	}
	if r.Spec.Storage.Traces.Backend == "" {
		r.Spec.Storage.Traces.Backend = MonolithicTracesStorageBackendMemory
	}
	if r.Spec.Storage.Traces.Size == nil {
		//exhaustive:ignore
		switch r.Spec.Storage.Traces.Backend {
		case MonolithicTracesStorageBackendMemory:
			r.Spec.Storage.Traces.Size = ptr.To(twoGBQuantity)
		default:
			r.Spec.Storage.Traces.Size = ptr.To(tenGBQuantity)
		}
	}

	if r.Spec.Ingestion == nil {
		r.Spec.Ingestion = &MonolithicIngestionSpec{}
	}
	if r.Spec.Ingestion.OTLP == nil {
		r.Spec.Ingestion.OTLP = &MonolithicIngestionOTLPSpec{}
	}
	if r.Spec.Ingestion.OTLP.GRPC == nil {
		r.Spec.Ingestion.OTLP.GRPC = &MonolithicIngestionOTLPProtocolsGRPCSpec{
			Enabled: true,
		}
	}
	// the gateway only supports OTLP/gRPC
	if r.Spec.Ingestion.OTLP.HTTP == nil {
		r.Spec.Ingestion.OTLP.HTTP = &MonolithicIngestionOTLPProtocolsHTTPSpec{
			Enabled: true,
		}
	}
	if r.Spec.JaegerUI != nil && r.Spec.JaegerUI.Enabled {

		if r.Spec.JaegerUI.Route != nil && r.Spec.JaegerUI.Route.Enabled {

			if r.Spec.JaegerUI.Route.Termination == "" {
				if r.Spec.Multitenancy.IsGatewayEnabled() && ctrlConfig.Gates.OpenShift.ServingCertsService {
					// gateway uses TLS
					r.Spec.JaegerUI.Route.Termination = TLSRouteTerminationTypeReencrypt
				} else {
					r.Spec.JaegerUI.Route.Termination = TLSRouteTerminationTypeEdge
				}
			}

			if r.Spec.JaegerUI.Authentication == nil {
				r.Spec.JaegerUI.Authentication = &JaegerQueryAuthenticationSpec{
					Enabled: ctrlConfig.Gates.OpenShift.OauthProxy.DefaultEnabled,
				}
			}

			if len(strings.TrimSpace(r.Spec.JaegerUI.Authentication.SAR)) == 0 {
				defaultSAR := fmt.Sprintf("{\"namespace\": \"%s\", \"resource\": \"pods\", \"verb\": \"get\"}", r.Namespace)
				r.Spec.JaegerUI.Authentication.SAR = defaultSAR
			}

			if r.Spec.JaegerUI.Authentication.Resources == nil {
				r.Spec.JaegerUI.Authentication.Resources = &corev1.ResourceRequirements{}
			}
		}

		if r.Spec.JaegerUI.ServicesQueryDuration == nil {
			r.Spec.JaegerUI.ServicesQueryDuration = &defaultServicesDuration
		}
		if r.Spec.JaegerUI.FindTracesConcurrentRequests == 0 {
			r.Spec.JaegerUI.FindTracesConcurrentRequests = defaultFindTracesConcurrentRequests
		}
	}

	if r.Spec.Timeout.Duration == 0 {
		r.Spec.Timeout = defaultTimeout
	}
	if r.Spec.Query == nil {
		r.Spec.Query = &MonolithicQuerySpec{}
	}
}
