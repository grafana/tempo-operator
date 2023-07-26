package prometheus

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/ViaQ/logerr/v2/kverrors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var (
	//go:embed prometheus-operator-alerts.yaml
	operatorAlertsFS       embed.FS
	operatorAlertsFilename = "prometheus-operator-alerts.yaml"
	operatorAlertsTmpl     = template.Must(template.New("").Delims("[[", "]]").ParseFS(operatorAlertsFS, operatorAlertsFilename))
)

const (
	// RunbookDefaultURL is the default url for the documentation of the Prometheus Operator alerts.
	RunbookDefaultURL = "https://github.com/grafana/tempo-operator/blob/main/operations/runbook.md"
)

// Options is used to configure Prometheus Alerts.
type Options struct {
	RunbookURL string
}

// PrometheusRule creates a PrometheusRule containing alerts of the operator.
func PrometheusRule(namespace string) (*monitoringv1.PrometheusRule, error) {
	opts := Options{
		RunbookURL: RunbookDefaultURL,
	}

	buf := &bytes.Buffer{}
	err := operatorAlertsTmpl.ExecuteTemplate(buf, operatorAlertsFilename, opts)
	if err != nil {
		return nil, kverrors.Wrap(err, "failed to execute template", "template", operatorAlertsFilename)
	}

	spec := monitoringv1.PrometheusRuleSpec{}
	err = yaml.NewYAMLOrJSONDecoder(buf, 8192).Decode(&spec)
	if err != nil {
		return nil, kverrors.Wrap(err, "failed to decode spec from reader")
	}

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "tempo-operator-controller-manager-prometheus-rule",
			Labels: labels.Merge(manifestutils.CommonOperatorLabels(), map[string]string{
				"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
			}),
		},
		Spec: spec,
	}, nil
}
