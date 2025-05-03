package cloudcredentials

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

type k8client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	CreateOrUpdate(ctx context.Context, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error)
}

// CredentialRequestOptions options for creating the credential request CCO CR.
type CredentialRequestOptions struct {
	TokenCCOAuth   manifestutils.TokenCCOAuthConfig
	ServiceAccount string
	Controlled     metav1.Object
	CredentialMode v1alpha1.CredentialMode
}

// CreateUpdateDeleteCredentialsRequest creates a new CredentialsRequest resource for a TempoStack
// to request a cloud credentials Secret resource from the OpenShift cloud-credentials-operator.
func CreateUpdateDeleteCredentialsRequest(ctx context.Context, scheme *runtime.Scheme,
	params CredentialRequestOptions, k k8client) error {

	if params.CredentialMode != v1alpha1.CredentialModeTokenCCO {
		var credReq cloudcredentialv1.CredentialsRequest
		if err := k.Get(ctx, client.ObjectKey{
			Namespace: params.Controlled.GetNamespace(),
			Name:      params.Controlled.GetName(),
		}, &credReq); err != nil {
			if apierrors.IsNotFound(err) {
				// CredentialsRequest does not exist -> this is what we want
				return nil
			}
			return kverrors.Wrap(err, "failed to lookup CredentialsRequest")
		}

		if err := k.Delete(ctx, &credReq); err != nil {
			return kverrors.Wrap(err, "failed to remove CredentialsRequest")
		}

		return nil
	}
	credReq, err := buildCredentialsRequest(params)
	if err != nil {
		return err
	}

	err = ctrl.SetControllerReference(params.Controlled, credReq, scheme)
	if err != nil {
		return kverrors.Wrap(err, "failed to set controller owner reference to resource")
	}

	desired := credReq.DeepCopyObject().(client.Object)
	mutateFn := manifests.MutateFuncFor(credReq, desired)

	_, err = k.CreateOrUpdate(ctx, credReq, mutateFn)
	if err != nil {
		return kverrors.Wrap(err, "failed to configure CredentialRequest")
	}

	return nil
}

func buildCredentialsRequest(params CredentialRequestOptions) (*cloudcredentialv1.CredentialsRequest, error) {
	stack := client.ObjectKey{Name: params.Controlled.GetName(), Namespace: params.Controlled.GetNamespace()}

	providerSpec, err := encodeProviderSpec(params.TokenCCOAuth)
	if err != nil {
		return nil, kverrors.Wrap(err, "failed encoding credentialsrequest provider spec")
	}

	return &cloudcredentialv1.CredentialsRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stack.Name,
			Namespace: stack.Namespace,
		},
		Spec: cloudcredentialv1.CredentialsRequestSpec{
			SecretRef: corev1.ObjectReference{
				Name:      manifestutils.ManagedCredentialsSecretName(stack.Name),
				Namespace: stack.Namespace,
			},
			ProviderSpec: providerSpec,
			ServiceAccountNames: []string{
				params.ServiceAccount,
			},
		},
	}, nil
}

func encodeProviderSpec(env manifestutils.TokenCCOAuthConfig) (*runtime.RawExtension, error) {
	var spec runtime.Object

	if env.AWS != nil {
		spec = &cloudcredentialv1.AWSProviderSpec{
			StatementEntries: []cloudcredentialv1.StatementEntry{
				{
					Action: []string{
						"s3:ListBucket",
						"s3:PutObject",
						"s3:GetObject",
						"s3:DeleteObject",
					},
					Effect:   "Allow",
					Resource: "arn:aws:s3:*:*:*",
				},
			},
			STSIAMRoleARN: env.AWS.RoleARN,
		}
	}

	if spec != nil {
		encodedSpec, err := cloudcredentialv1.Codec.EncodeProviderSpec(spec.DeepCopyObject())
		return encodedSpec, err
	} else {
		return nil, kverrors.New("unsupported CCO environment")
	}
}
