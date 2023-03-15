package gateway

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/status"
)

// GetOpenShiftBaseDomain returns base domain of OCP cluster.
func GetOpenShiftBaseDomain(ctx context.Context, client k8sclient.Client) (string, error) {
	ictrl := &operatorv1.IngressController{}
	nsn := types.NamespacedName{Name: "default", Namespace: "openshift-ingress-operator"}
	err := client.Get(ctx, nsn, ictrl)
	if err != nil {
		// The preferred way to get the base domain is via OCP ingress controller
		// this approach works well with CRC and normal OCP cluster.
		// Fallback on cluster DNS might not work with CRC because CRC uses .apps-crc. and not .apps.
		key := k8sclient.ObjectKey{Name: "cluster"}
		var clusterDNS configv1.DNS
		if err := client.Get(ctx, key, &clusterDNS); err != nil {
			if apierrors.IsNotFound(err) {
				return "", &status.DegradedError{
					Message: "Missing OpenShift ingresscontroller and cluster DNS configuration to read base domain",
					Reason:  v1alpha1.ReasonCouldNotGetOpenShiftBaseDomain,
					Requeue: true,
				}
			}
			return "", kverrors.Wrap(err, "failed to lookup gateway base domain",
				"name", key)
		}

		return "", err
	}
	return ictrl.Status.Domain, nil
}
