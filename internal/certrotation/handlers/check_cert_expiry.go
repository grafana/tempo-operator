package handlers

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation"
)

// CheckCertExpiry handles the case if the TempoStack managed signing CA, client and/or serving
// certificates expired. Returns true if any of those expired and an error representing the reason
// of expiry.
func CheckCertExpiry(controllerName string, ctx context.Context, log logr.Logger, req ctrl.Request, k client.Client,
	fg configv1alpha1.FeatureGates, components map[string]string) error {
	ll := log.WithValues(controllerName, req.String(), "event", "checkCertExpiry")

	opts, err := GetOptions(ctx, k, req, components)
	if err != nil {
		return kverrors.Wrap(err, "failed to lookup certificates secrets", "name", req.String())
	}

	if optErr := certrotation.ApplyDefaultSettings(&opts, fg.BuiltInCertManagement, components); optErr != nil {
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
