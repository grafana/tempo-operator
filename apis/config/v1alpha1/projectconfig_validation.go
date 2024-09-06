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

	// Validate container images if set
	for envName, envValue := range map[string]string{
		EnvRelatedImageTempo:           c.DefaultImages.Tempo,
		EnvRelatedImageJaegerQuery:     c.DefaultImages.JaegerQuery,
		EnvRelatedImageTempoQuery:      c.DefaultImages.TempoQuery,
		EnvRelatedImageTempoGateway:    c.DefaultImages.TempoGateway,
		EnvRelatedImageTempoGatewayOpa: c.DefaultImages.TempoGatewayOpa,
	} {
		if envValue != "" {
			_, err := dockerparser.Parse(envValue)
			if err != nil {
				return fmt.Errorf("invalid value '%s': please set the %s environment variable to a valid container image", envValue, envName)
			}
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
