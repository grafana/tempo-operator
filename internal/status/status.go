package status

import (
	"context"

	dockerparser "github.com/novln/docker-parser"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func Refresh(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices, status *v1alpha1.MicroservicesStatus) (bool, error) {
	tempoImage, err := dockerparser.Parse(tempo.Spec.Images.Tempo)
	if err != nil {
		return false, err
	}

	changed := tempo.DeepCopy()
	changed.Status = *status
	changed.Status.TempoVersion = tempoImage.Tag()
	err = k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return true, err
	}

	return false, nil
}
