package monolithic

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	"github.com/grafana/tempo-operator/internal/manifests/oauthproxy"
)

func getJaegerUIService(services []client.Object, tempo v1alpha1.TempoMonolithic) *corev1.Service {
	serviceName := naming.Name(manifestutils.JaegerUIComponentName, tempo.Name)
	for _, clientObject := range services {
		svc, ok := clientObject.(*corev1.Service)
		if ok {
			if svc.Name == serviceName {
				return svc
			}
		}
	}
	return nil
}

// BuildAll generates all manifests.
func BuildAll(opts Options) ([]client.Object, error) {
	tempo := opts.Tempo
	manifests := []client.Object{}
	extraStsAnnotations := map[string]string{}

	if opts.CtrlConfig.Gates.OpenShift.ServingCertsService {
		manifests = append(manifests, manifestutils.NewConfigMapCABundle(
			tempo.Namespace,
			naming.ServingCABundleName(tempo.Name),
			CommonLabels(tempo.Name),
		))
		if ingestionHTTPTLSEnabled(tempo) && tlsSecretAndBundleEmptyHTTP(tempo) {
			tempo.Spec.Ingestion.OTLP.HTTP.TLS.Cert = naming.ServingCertName(manifestutils.TempoMonolithComponentName, tempo.Name)
			opts.useServiceCertsOnReceiver = true
		}

		if ingestionGRPCTLSEnabled(tempo) && tlsSecretAndBundleEmptyGRPC(tempo) {
			tempo.Spec.Ingestion.OTLP.GRPC.TLS.Cert = naming.ServingCertName(manifestutils.TempoMonolithComponentName, tempo.Name)
			opts.useServiceCertsOnReceiver = true
		}
	}

	configMap, annotations, err := BuildConfigMap(opts)
	if err != nil {
		return nil, err
	}

	manifests = append(manifests, configMap)
	maps.Copy(extraStsAnnotations, annotations)

	var serviceAccount *corev1.ServiceAccount
	if tempo.Spec.ServiceAccount == "" {
		serviceAccount = BuildServiceAccount(opts)
		manifests = append(manifests, serviceAccount)
	}

	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		objs, annotations, err := BuildGatewayObjects(opts)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, objs...)
		maps.Copy(extraStsAnnotations, annotations)
	}

	statefulSet, err := BuildTempoStatefulset(opts, extraStsAnnotations)
	if err != nil {
		return nil, err
	}

	manifests = append(manifests, statefulSet)
	services := BuildServices(opts)
	manifests = append(manifests, services...)

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		if tempo.Spec.JaegerUI.Ingress != nil && tempo.Spec.JaegerUI.Ingress.Enabled {
			manifests = append(manifests, BuildJaegerUIIngress(opts))
		}

		if tempo.Spec.JaegerUI.Route != nil && tempo.Spec.JaegerUI.Route.Enabled {
			route, err := BuildJaegerUIRoute(opts)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, route)
			if tempo.Spec.JaegerUI.Authentication.Enabled && !tempo.Spec.Multitenancy.IsGatewayEnabled() {

				oauthproxy.PatchStatefulSetForOauthProxy(
					tempo.ObjectMeta,
					tempo.Spec.JaegerUI.Authentication,
					tempo.Spec.Timeout.Duration,
					opts.CtrlConfig,
					statefulSet,
					tempo.Spec.JaegerUI.Authentication.Resources,
				)
				oauthproxy.PatchQueryFrontEndService(getJaegerUIService(services, tempo), tempo.Name)
				if serviceAccount != nil {
					oauthproxy.AddServiceAccountAnnotations(serviceAccount, route.Name)
				}
				oauthproxy.PatchRouteForOauthProxy(route)
			}
		}
	}

	if tempo.Spec.Observability != nil {
		if tempo.Spec.Observability.Metrics != nil {
			if tempo.Spec.Observability.Metrics.ServiceMonitors != nil && tempo.Spec.Observability.Metrics.ServiceMonitors.Enabled {
				manifests = append(manifests, BuildServiceMonitor(opts))
			}

			if tempo.Spec.Observability.Metrics.PrometheusRules != nil && tempo.Spec.Observability.Metrics.PrometheusRules.Enabled {
				prometheusRules, err := BuildPrometheusRules(opts)
				if err != nil {
					return nil, err
				}
				manifests = append(manifests, prometheusRules...)
			}
		}

		if tempo.Spec.Observability.Grafana != nil &&
			tempo.Spec.Observability.Grafana.DataSource != nil && tempo.Spec.Observability.Grafana.DataSource.Enabled {
			manifests = append(manifests, BuildGrafanaDatasource(opts))
		}
	}

	return manifests, nil
}
