package status

import (
	"context"

	dockerparser "github.com/novln/docker-parser"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

// Refresh updates the status field with the tempo container image versions.
func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.TempoStack, status *v1alpha1.TempoStackStatus) (bool, error) {

	changed := tempo.DeepCopy()
	changed.Status = *status

	tempoImage, err := dockerparser.Parse(tempo.Spec.Images.Tempo)
	if err != nil {
		return false, err
	}
	changed.Status.TempoVersion = tempoImage.Tag()

	if tempo.Spec.Components.QueryFrontend.JaegerQuery.Enabled {
		tempoQueryImage, err := dockerparser.Parse(tempo.Spec.Images.TempoQuery)
		if err != nil {
			return false, err
		}
		changed.Status.TempoQueryVersion = tempoQueryImage.Tag()
	}

	err = k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return true, err
	}

	return false, nil
}
