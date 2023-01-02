package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

const certRotationRequiredAtKey = "tempo.grafana.com/certRotationRequiredAt"

// AnnotateForRequiredCertRotation adds/updates the `loki.grafana.com/certRotationRequiredAt` annotation
// to the named Lokistack if any of the managed client/serving/ca certificates expired. If no LokiStack
// is found, then skip reconciliation.
func AnnotateForRequiredCertRotation(ctx context.Context, k client.Client, name, namespace string) error {
	var s v1alpha1.Microservices
	key := client.ObjectKey{Name: name, Namespace: namespace}

	if err := k.Get(ctx, key, &s); err != nil {
		if apierrors.IsNotFound(err) {
			// Do nothing
			return nil
		}

		return kverrors.Wrap(err, "failed to get lokistack", "key", key)
	}

	ss := s.DeepCopy()
	if ss.Annotations == nil {
		ss.Annotations = make(map[string]string)
	}

	ss.Annotations[certRotationRequiredAtKey] = time.Now().UTC().Format(time.RFC3339)

	if err := k.Update(ctx, ss); err != nil {
		return kverrors.Wrap(err, fmt.Sprintf("failed to update lokistack `%s` annotation", certRotationRequiredAtKey), "key", key)
	}

	return nil
}
