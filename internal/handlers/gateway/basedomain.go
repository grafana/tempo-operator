package gateway

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/status"
)

// GetOpenShiftBaseDomain returns base domain of OCP cluster.
func GetOpenShiftBaseDomain(ctx context.Context, k8sClient client.Client) (string, error) {
	ictrl := &operatorv1.IngressController{}
	nsn := types.NamespacedName{Name: "default", Namespace: "openshift-ingress-operator"}
	err := k8sClient.Get(ctx, nsn, ictrl)
	if err != nil {
		// The preferred way to get the base domain is via OCP ingress controller
		// this approach works well with CRC and normal OCP cluster.
		// Fallback on cluster DNS might not work with CRC because CRC uses .apps-crc. and not .apps.
		key := client.ObjectKey{Name: "cluster"}
		var clusterDNS configv1.DNS
		if err := k8sClient.Get(ctx, key, &clusterDNS); err != nil {
			if apierrors.IsNotFound(err) {
				return "", &status.ConfigurationError{
					Message: "Missing OpenShift IngressController and cluster DNS configuration to read base domain",
					Reason:  v1alpha1.ReasonCouldNotGetOpenShiftBaseDomain,
				}
			}
			return "", kverrors.Wrap(err, "failed to lookup gateway base domain",
				"name", key)
		}

		return "", err
	}
	return ictrl.Status.Domain, nil
}
