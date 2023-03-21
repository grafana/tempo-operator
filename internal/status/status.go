package status

import (
	"context"
	"strings"

	dockerparser "github.com/novln/docker-parser"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

var (
	tempoStackStatusCondition = promauto.With(metrics.Registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempostack",
		Name:      "status_condition",
		Help:      "The status condition of a TempoStack instance.",
	}, []string{"stack_namespace", "stack_name", "condition", "status"})
)

// Refresh updates the status field with the tempo container image versions and updates the tempostack_status_condition metric.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) (bool, error) {

	changed := tempo.DeepCopy()
	changed.Status = *status

	tempoImage, err := dockerparser.Parse(tempo.Spec.Images.Tempo)
	if err != nil {
		return false, err
	}
	changed.Status.TempoVersion = tempoImage.Tag()

	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		tempoQueryImage, err := dockerparser.Parse(tempo.Spec.Images.TempoQuery)
		if err != nil {
			return false, err
		}
		changed.Status.TempoQueryVersion = tempoQueryImage.Tag()
	}

	for _, cond := range status.Conditions {
		var isActive float64
		if cond.Status == metav1.ConditionTrue {
			isActive = 1
		}
		status := strings.ToLower(string(cond.Status))
		tempoStackStatusCondition.WithLabelValues(tempo.Namespace, tempo.Name, cond.Type, status).Set(isActive)
	}

	err = k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return true, err
	}

	return false, nil
}
