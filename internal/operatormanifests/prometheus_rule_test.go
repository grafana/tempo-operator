package operatormanifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestPrometheusRule(t *testing.T) {
	prometheusrule, err := prometheusRule("tempo-operator-system")
	assert.NoError(t, err)

	assert.Equal(t, metav1.ObjectMeta{
		Name:      "tempo-operator-controller-manager-prometheus-rule",
		Namespace: "tempo-operator-system",
		Labels: labels.Merge(CommonLabels(), map[string]string{
			"openshift.io/prometheus-rule-evaluation-scope": "leaf-prometheus",
		}),
	}, prometheusrule.ObjectMeta)
	assert.Len(t, prometheusrule.Spec.Groups[0].Rules, 5)
}
