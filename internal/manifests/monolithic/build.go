package monolithic

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildAll generates all manifests.
func BuildAll(opts Options) ([]client.Object, error) {
	tempo := opts.Tempo
	manifests := []client.Object{}

	configMap, configChecksum, err := BuildConfigMap(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, configMap)
	opts.ConfigChecksum = configChecksum

	manifests = append(manifests, BuildServiceAccount(opts))

	statefulSet, err := BuildTempoStatefulset(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, statefulSet)

	service := BuildTempoService(opts)
	manifests = append(manifests, service)

	ingresses, err := BuildTempoIngress(opts)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, ingresses...)

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
