package upgrade

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// ManagedInstances upgrades all managed instances in the cluster.
func (u Upgrade) ManagedInstances(ctx context.Context) error {
	u.Log.Info("looking for instances to upgrade")

	listOps := []client.ListOption{}
	tempostackList := &v1alpha1.TempoStackList{}
	if err := u.Client.List(ctx, tempostackList, listOps...); err != nil {
		return fmt.Errorf("failed to list TempoStacks: %w", err)
	}

	for i := range tempostackList.Items {
		original := tempostackList.Items[i]
		itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)

		if original.Spec.ManagementState == v1alpha1.ManagementStateUnmanaged {
			itemLogger.Info("skipping unmanaged instance")
			continue
		}

		// u.Upgrade() logs the errors to the operator log
		// We continue upgrading the other operands even if an operand fails to upgrade.
		_, _ = u.Upgrade(ctx, &original)
	}

	tempomonolithicList := &v1alpha1.TempoMonolithicList{}
	if err := u.Client.List(ctx, tempomonolithicList, listOps...); err != nil {
		return fmt.Errorf("failed to list TempoMonolithics: %w", err)
	}

	for i := range tempomonolithicList.Items {
		original := tempomonolithicList.Items[i]
		itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)

		if original.Spec.Management == v1alpha1.ManagementStateUnmanaged {
			itemLogger.Info("skipping unmanaged instance")
			continue
		}

		// u.Upgrade() logs the errors to the operator log
		// We continue upgrading the other operands even if an operand fails to upgrade.
		_, _ = u.Upgrade(ctx, &original)
	}

	if len(tempostackList.Items) == 0 && len(tempomonolithicList.Items) == 0 {
		u.Log.Info("no instances found")
	}
	return nil
}
