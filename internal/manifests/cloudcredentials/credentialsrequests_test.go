package cloudcredentials

import (
	"fmt"
	"os"
	"testing"

	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildCredentialsRequest_CreateForTempoStack(t *testing.T) {
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}
	err := os.Setenv("ROLEARN", "test-rolearn")
	require.NoError(t, err)

	credReqs, err := BuildCredentialsRequest(&stack, stack.Spec.ServiceAccount, &manifestutils.TokenCCOAuthConfig{
		AWS: &manifestutils.TokenCCOAWSEnvironment{
			RoleARN: "test-rolearn",
		},
	})

	credReq := credReqs[0].(*cloudcredentialv1.CredentialsRequest)

	require.NoError(t, err)
	require.NotNil(t, credReq)

	require.Equal(t, stack.Namespace, credReq.Spec.SecretRef.Namespace)
	require.Len(t, credReq.Spec.ServiceAccountNames, 1)
	require.Equal(t, stack.Spec.ServiceAccount, credReq.Spec.ServiceAccountNames[0])
	require.Equal(t, stack.Name, credReq.Name)
	require.Equal(t, fmt.Sprintf("%s-managed-credentials", stack.Name), credReq.Spec.SecretRef.Name)
}

func TestBuildCredentialsRequest_NoEnvConfigured(t *testing.T) {
	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	cco, err := BuildCredentialsRequest(&stack, stack.Spec.ServiceAccount, &manifestutils.TokenCCOAuthConfig{})

	require.Error(t, err)
	assert.Equal(t, 0, len(cco))
}
