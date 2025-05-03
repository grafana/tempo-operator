package cloudcredentials

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

var testScheme *runtime.Scheme = scheme.Scheme

func TestBuildCredentialsRequest_CreateForTempoStack(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	var credReq *cloudcredentialv1.CredentialsRequest // or whatever the expected type is

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cl.On("CreateOrUpdate", mock.Anything, mock.Anything, mock.Anything).
		Return(controllerutil.OperationResultCreated, nil).Run(func(args mock.Arguments) {
		credReq, _ = args.Get(1).(*cloudcredentialv1.CredentialsRequest)
	})

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeTokenCCO,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Spec.ServiceAccount,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	require.NoError(t, err)
	require.NotNil(t, credReq)

	/* Assert call numbers */
	cl.AssertNumberOfCalls(t, "Get", 0)
	cl.AssertNumberOfCalls(t, "CreateOrUpdate", 1)

	/* Compare key values of the CredentialsRequest*/

	require.Equal(t, stack.Namespace, credReq.Spec.SecretRef.Namespace)
	require.Len(t, credReq.Spec.ServiceAccountNames, 1)
	require.Equal(t, stack.Spec.ServiceAccount, credReq.Spec.ServiceAccountNames[0])
	require.Equal(t, stack.Name, credReq.Name)
	require.Equal(t, fmt.Sprintf("%s-managed-credentials", stack.Name), credReq.Spec.SecretRef.Name)
}

func TestBuildCredentialsRequest_NotSupportedEnv(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeTokenCCO,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	require.EqualError(t, err, "failed encoding credentialsrequest provider spec: unsupported CCO environment")

}

func TestBuildCredentialsRequest_CreateForTempoStackError(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	retErr := errors.New("some error")

	cl.On("CreateOrUpdate", mock.Anything, mock.Anything, mock.Anything).
		Return(controllerutil.OperationResultNone, retErr)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeTokenCCO,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	require.EqualError(t, err, "failed to configure CredentialRequest: some error")

}

func TestBuildCredentialsRequest_DeleteNoCCOMode(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cl.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeToken,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	require.NoError(t, err)

	/* Assert call numbers */
	cl.AssertNumberOfCalls(t, "Get", 1)
	cl.AssertNumberOfCalls(t, "Delete", 1)

}

func TestBuildCredentialsRequest_DeleteNoCCOModeNotFound(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}
	returnErr := apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(returnErr)
	cl.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeToken,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	require.NoError(t, err)

	/* Assert call numbers */
	cl.AssertNumberOfCalls(t, "Get", 1)
	cl.AssertNumberOfCalls(t, "Delete", 0)

}

func TestBuildCredentialsRequest_DeleteError(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	delError := errors.New("something went wrong")

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	cl.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(delError)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeStatic,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	assert.EqualError(t, err, "failed to remove CredentialsRequest: something went wrong")

}

func TestBuildCredentialsRequest_GetError(t *testing.T) {
	cl := &clientStub{}
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		assert.FailNow(t, "failed to register scheme")
	}

	getError := errors.New("something went wrong")

	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getError)

	params := CredentialRequestOptions{
		Controlled:     &stack,
		CredentialMode: v1alpha1.CredentialModeStatic,
		TokenCCOAuth: manifestutils.TokenCCOAuthConfig{
			HasCCOEnvironment: true,
			AWS:               &manifestutils.TokenCCOAWSEnvironment{RoleARN: "role-arn"},
		},
		ServiceAccount: stack.Name,
	}

	err := CreateUpdateDeleteCredentialsRequest(context.Background(), testScheme, params, cl)

	assert.EqualError(t, err, "failed to lookup CredentialsRequest: something went wrong")

}
