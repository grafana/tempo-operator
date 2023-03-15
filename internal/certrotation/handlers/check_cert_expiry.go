package handlers

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	v1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/certrotation"
)

// CheckCertExpiry handles the case if the TempoStack managed signing CA, client and/or serving
// certificates expired. Returns true if any of those expired and an error representing the reason
// of expiry.
func CheckCertExpiry(ctx context.Context, log logr.Logger, req ctrl.Request, k client.Client, fg configv1alpha1.FeatureGates) error {
	ll := log.WithValues("tempostacks", req.String(), "event", "checkCertExpiry")

	var stack v1alpha1.TempoStack
	if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			// maybe the user deleted it before we could react? Either way this isn't an issue
			ll.Error(err, "could not find the requested tempo tempostacks", "name", req.String())
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup tempostacks", "name", req.String())
	}

	opts, err := GetOptions(ctx, k, req)
	if err != nil {
		return kverrors.Wrap(err, "failed to lookup certificates secrets", "name", req.String())
	}

	if optErr := certrotation.ApplyDefaultSettings(&opts, fg.BuiltInCertManagement); optErr != nil {
		ll.Error(optErr, "failed to conform options to build settings")
		return optErr
	}

	if err := certrotation.SigningCAExpired(opts); err != nil {
		return err
	}

	if err := certrotation.CertificatesExpired(opts); err != nil {
		return err
	}

	return nil
}
