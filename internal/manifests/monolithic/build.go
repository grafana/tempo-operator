package monolithic

import (
	"maps"

	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
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

	var route *v1.Route

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		if tempo.Spec.JaegerUI.Ingress != nil && tempo.Spec.JaegerUI.Ingress.Enabled {
			manifests = append(manifests, BuildJaegerUIIngress(opts))
		}

		if tempo.Spec.JaegerUI.Route != nil && tempo.Spec.JaegerUI.Route.Enabled {
			route, err = BuildJaegerUIRoute(opts)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, route)

		}
	}

	if !tempo.Spec.Multitenancy.IsGatewayEnabled() {

		if isOauthProxyEnabled(tempo) {
			// Create common objects
			secret, err := oauthproxy.OAuthCookieSessionSecret(tempo.ObjectMeta)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, secret)

			if isOauthProxyEnabledForJaegerUI(tempo) {
				oauthproxy.PatchPodSpecForOauthProxy(
					oauthproxy.Params{
						TempoMeta:     tempo.ObjectMeta,
						ProjectConfig: opts.CtrlConfig,
						ProxyImage:    opts.CtrlConfig.DefaultImages.OauthProxy,
						ContainerName: "tempo-query",
						Port: corev1.ContainerPort{
							Name:          manifestutils.JaegerUIPortName,
							ContainerPort: manifestutils.PortJaegerUI,
							Protocol:      corev1.ProtocolTCP,
						},
						HTTPPort:               manifestutils.OAuthJaegerUIProxyPortHTTP,
						HTTPSPort:              manifestutils.OAuthJaegerUIProxyPortHTTPS,
						OverrideServiceAccount: false,
					}, &statefulSet.Spec.Template.Spec,
				)
				if route != nil {
					if serviceAccount != nil {
						oauthproxy.AddServiceAccountAnnotations(serviceAccount, route.Name)
					}
					oauthproxy.PatchRouteForOauthProxy(route)
				}
				oauthproxy.PatchQueryFrontEndService(getJaegerUIService(services, tempo), tempo.Name)
			}

			if isOauthProxyEnabledForTempo(tempo) {
				oauthproxy.PatchPodSpecForOauthProxy(
					oauthproxy.Params{
						TempoMeta:     tempo.ObjectMeta,
						ProjectConfig: opts.CtrlConfig,
						ProxyImage:    opts.CtrlConfig.DefaultImages.OauthProxy,
						ContainerName: "tempo",
						Port: corev1.ContainerPort{
							Name:          manifestutils.HttpPortName,
							ContainerPort: manifestutils.PortHTTPServer,
							Protocol:      corev1.ProtocolTCP,
						},
						HTTPPort:               manifestutils.OAuthQueryFrontendProxyPortHTTP,
						HTTPSPort:              manifestutils.OAuthQueryFrontendProxyPortHTTPS,
						OverrideServiceAccount: false,
						TLSSecretName:          naming.ServingCertName(manifestutils.TempoMonolithComponentName, tempo.Name),
					}, &statefulSet.Spec.Template.Spec,
				)
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
