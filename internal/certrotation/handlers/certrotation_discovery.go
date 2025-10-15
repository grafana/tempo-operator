package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

const certRotationRequiredAtKey = "tempo.grafana.com/certRotationRequiredAt"

// AnnotateTempoStackForRequiredCertRotation adds/updates the `tempo.grafana.com/certRotationRequiredAt` annotation
// to the named TempoStack if any of the managed client/serving/ca certificates expired. If no TempoStack
// is found, then skip reconciliation.
func AnnotateTempoStackForRequiredCertRotation(ctx context.Context, k client.Client, name, namespace string) error {
	var s v1alpha1.TempoStack
	key := client.ObjectKey{Name: name, Namespace: namespace}

	if err := k.Get(ctx, key, &s); err != nil {
		if apierrors.IsNotFound(err) {
			// Do nothing
			return nil
		}

		return kverrors.Wrap(err, "failed to get tempo TempoStack", "key", key)
	}

	ss := s.DeepCopy()
	if ss.Annotations == nil {
		ss.Annotations = make(map[string]string)
	}

	ss.Annotations[certRotationRequiredAtKey] = time.Now().UTC().Format(time.RFC3339)

	if err := k.Update(ctx, ss); err != nil {
		return kverrors.Wrap(err, fmt.Sprintf("failed to update tempo TempoStack `%s` annotation", certRotationRequiredAtKey), "key", key)
	}

	return nil
}

// AnnotateMonolithicForRequiredCertRotation adds/updates the `tempo.grafana.com/certRotationRequiredAt` annotation
// to the named TempoStack if any of the managed client/serving/ca certificates expired. If no TempoStack
// is found, then skip reconciliation.
func AnnotateMonolithicForRequiredCertRotation(ctx context.Context, k client.Client, name, namespace string) error {
	var s v1alpha1.TempoMonolithic
	key := client.ObjectKey{Name: name, Namespace: namespace}

	if err := k.Get(ctx, key, &s); err != nil {
		if apierrors.IsNotFound(err) {
			// Do nothing
			return nil
		}

		return kverrors.Wrap(err, "failed to get tempo TempoStack", "key", key)
	}

	ss := s.DeepCopy()
	if ss.Annotations == nil {
		ss.Annotations = make(map[string]string)
	}

	ss.Annotations[certRotationRequiredAtKey] = time.Now().UTC().Format(time.RFC3339)

	if err := k.Update(ctx, ss); err != nil {
		return kverrors.Wrap(err, fmt.Sprintf("failed to update tempo TempoStack `%s` annotation", certRotationRequiredAtKey), "key", key)
	}

	return nil
}
