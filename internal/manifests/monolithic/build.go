package monolithic

import (
	"maps"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildAll generates all manifests.
func BuildAll(opts Options) ([]client.Object, error) {
	tempo := opts.Tempo
	manifests := []client.Object{}
	extraStsAnnotations := map[string]string{}

	configMap, annotations, err := BuildConfigMap(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, configMap)
	maps.Copy(extraStsAnnotations, annotations)

	if tempo.Spec.ServiceAccount == "" {
		manifests = append(manifests, BuildServiceAccount(opts))
	}

	statefulSet, err := BuildTempoStatefulset(opts, extraStsAnnotations)
	if err != nil {
		return nil, err
	}

	manifests = append(manifests, statefulSet)

	manifests = append(manifests, BuildTempoService(opts))

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
