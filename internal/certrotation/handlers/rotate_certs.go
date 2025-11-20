package handlers

import (
	"context"
	"fmt"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	v1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// CreateOrRotateCertificates handles the TempoStack client and serving certificate creation and rotation
// including the signing CA and a ca bundle or else returns an error. It returns only a degrade-condition-worthy
// error if building the manifests fails for any reason.
func CreateOrRotateCertificates(ctx context.Context, log logr.Logger,
	req ctrl.Request, k client.Client, s *runtime.Scheme, fg configv1alpha1.FeatureGates, cs map[string]string) error {
	ll := log.WithValues("tempostacks", req.String(), "event", "createOrRotateCerts")
	var stack v1alpha1.TempoStack
	if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			// maybe the user deleted it before we could react? Either way this isn't an issue
			ll.Error(err, "could not find the requested tempo tempostacks", "name", req.String())
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup tempostacks", "name", req.String())
	}

	opts, err := GetOptions(ctx, k, req, cs)
	if err != nil {
		return kverrors.Wrap(err, "failed to lookup certificates secrets", "name", req.String())
	}

	if optErr := certrotation.ApplyDefaultSettings(&opts, fg.BuiltInCertManagement, certrotation.TempoStackComponentCertSecretNames(opts.StackName)); optErr != nil {
		ll.Error(optErr, "failed to conform options to build settings")
		return kverrors.Wrap(err, "failed to conform options to build settings", "name", req.String())
	}

	objects, err := certrotation.BuildAll(opts)

	if err != nil {
		ll.Error(err, "failed to build certificate manifests")
		return kverrors.Wrap(err, "failed to build certificate manifests", "name", req.String())
	}

	ll.V(1).Info("certificate manifests built", "count", len(objects))

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
		return kverrors.New("failed to create or rotate TempoStack certificates", "name", req.String())
	}

	// Add certificate hash annotations to trigger pod restart after successful certificate rotation
	if err := addCertificateHashAnnotations(ctx, k, &stack, opts); err != nil {
		ll.Error(err, "failed to add certificate hash annotations after certificate rotation")
		return kverrors.Wrap(err, "failed to add certificate hash annotations", "name", req.String())
	}

	ll.Info("certificate rotation completed successfully, hash annotations added")
	return nil
}

// addCertificateHashAnnotations adds certificate hash annotations to trigger pod restart after certificate rotation.
func addCertificateHashAnnotations(ctx context.Context, k client.Client, stack *v1alpha1.TempoStack, opts certrotation.Options) error {
	certSecrets := make(map[string]*corev1.Secret, len(opts.Certificates))
	for name, cert := range opts.Certificates {
		certSecrets[name] = cert.Secret
	}

	stack.Annotations = labels.Merge(stack.Annotations, manifestutils.CertificateHashAnnotations(certSecrets))
	return k.Update(ctx, stack)
}
