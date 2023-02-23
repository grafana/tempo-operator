package handlers

import (
	"context"
	"fmt"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	v1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/certrotation"
	"github.com/os-observability/tempo-operator/internal/manifests"
)

// CreateOrRotateCertificates handles the Microservices client and serving certificate creation and rotation
// including the signing CA and a ca bundle or else returns an error. It returns only a degrade-condition-worthy
// error if building the manifests fails for any reason.
func CreateOrRotateCertificates(ctx context.Context, log logr.Logger,
	req ctrl.Request, k client.Client, s *runtime.Scheme, fg configv1alpha1.FeatureGates) error {
	ll := log.WithValues("microservices", req.String(), "event", "createOrRotateCerts")
	var stack v1alpha1.Microservices
	if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			// maybe the user deleted it before we could react? Either way this isn't an issue
			ll.Error(err, "could not find the requested tempo microservices", "name", req.String())
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup microservices", "name", req.String())
	}

	opts, err := GetOptions(ctx, k, req)
	if err != nil {
		return kverrors.Wrap(err, "failed to lookup certificates secrets", "name", req.String())
	}

	if optErr := certrotation.ApplyDefaultSettings(&opts, fg.BuiltInCertManagement); optErr != nil {
		ll.Error(optErr, "failed to conform options to build settings")
		return kverrors.Wrap(err, "failed to conform options to build settings", "name", req.String())
	}

	objects, err := certrotation.BuildAll(opts)

	if err != nil {
		ll.Error(err, "failed to build certificate manifests")
		return kverrors.Wrap(err, "failed to build certificate manifests", "name", req.String())
	}

	ll.Info("certificate manifests built", "count", len(objects))

	var errCount int32

	for _, obj := range objects {
		l := ll.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
		)

		obj.SetNamespace(req.Namespace)

		if err := ctrl.SetControllerReference(&stack, obj, s); err != nil {
			l.Error(err, "failed to set controller owner reference to resource")
			errCount++
			continue
		}

		desired := obj.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(obj, desired)

		op, err := ctrl.CreateOrUpdate(ctx, k, obj, mutateFn)
		if err != nil {
			l.Error(err, "failed to configure resource")
			errCount++
			continue
		}

		msg := fmt.Sprintf("Resource has been %s", op)
		switch op {
		case ctrlutil.OperationResultNone:
			l.V(1).Info(msg)
		case ctrlutil.OperationResultCreated:
		case ctrlutil.OperationResultUpdated:
		case ctrlutil.OperationResultUpdatedStatus:
		case ctrlutil.OperationResultUpdatedStatusOnly:
			l.Info(msg)
		}
	}

	if errCount > 0 {
		return kverrors.New("failed to create or rotate Microservices certificates", "name", req.String())
	}

	return nil
}
