package v1alpha1

import (
	"errors"
	"fmt"

	dockerparser "github.com/novln/docker-parser"
)

// Validate validates the controller configuration (ProjectConfig).
func (c *ProjectConfig) Validate() error {
	switch c.Gates.TLSProfile {
	case string(TLSProfileOldType),
		string(TLSProfileIntermediateType),
		string(TLSProfileModernType):
		// valid setting
	default:
		return fmt.Errorf("invalid value '%s' for setting featureGates.tlsProfile (valid values: %s, %s and %s)", c.Gates.TLSProfile, TLSProfileOldType, TLSProfileIntermediateType, TLSProfileModernType)
	}

	if c.DefaultImages.Tempo != "" {
		_, err := dockerparser.Parse(c.DefaultImages.Tempo)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempo", c.DefaultImages.Tempo)
		}
	}
	if c.DefaultImages.TempoQuery != "" {
		_, err := dockerparser.Parse(c.DefaultImages.TempoQuery)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempoQuery", c.DefaultImages.TempoQuery)
		}
	}
	if c.DefaultImages.TempoGateway != "" {
		_, err := dockerparser.Parse(c.DefaultImages.TempoGateway)
		if err != nil {
			return fmt.Errorf("invalid value '%s' for setting images.tempoGateway", c.DefaultImages.TempoGateway)
		}
	}

	if c.Gates.Observability.Metrics.CreateServiceMonitors && !c.Gates.PrometheusOperator {
		return errors.New("the prometheusOperator feature gate must be enabled to create a ServiceMonitor for the operator")
	}
	if c.Gates.Observability.Metrics.CreatePrometheusRules && !c.Gates.PrometheusOperator {
		return errors.New("the prometheusOperator feature gate must be enabled to create PrometheusRules for the operator")
	}
	if c.Gates.Observability.Metrics.CreatePrometheusRules && !c.Gates.Observability.Metrics.CreateServiceMonitors {
		return errors.New("the Prometheus rules alert based on collected metrics, therefore the createServiceMonitors feature must be enabled when enabling the createPrometheusRules feature")
	}

	return nil
}
