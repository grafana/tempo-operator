package upgrade

import (
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// This upgrade modifies the immutable field PodManagementPolicy of the ingester StatefulSet,
// therefore we delete the ingester StatefulSet, which will be recreated in the reconcile loop.
func upgrade0_5_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	listOps := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(manifestutils.IngesterComponentName, tempo.Name)),
	}
	ingesterList := &v1.StatefulSetList{}
	err := u.Client.List(ctx, ingesterList, listOps...)
	if err != nil {
		return fmt.Errorf("failed to list ingester stateful sets: %w", err)
	}

	for _, ingester := range ingesterList.Items {
		ingester := ingester
		u.Log.Info("deleting ingester (will be re-created)", "ingester", ingester.Name)
		err := u.Client.Delete(ctx, &ingester)
		if err != nil {
			return fmt.Errorf("failed to delete ingester %s: %w", ingester.Name, err)
		}
	}

	return nil
}
