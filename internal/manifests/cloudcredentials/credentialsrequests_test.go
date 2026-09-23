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

func TestBuildCredentialsRequest_ARNPartition(t *testing.T) {
	tests := []struct {
		name             string
		partition        string
		expectedResource string
	}{
		{
			name:             "commercial partition",
			partition:        "aws",
			expectedResource: "arn:aws:s3:*:*:*",
		},
		{
			// An empty partition must keep the behavior of the commercial partition.
			name:             "unset partition",
			partition:        "",
			expectedResource: "arn:aws:s3:*:*:*",
		},
		{
			name:             "iso partition",
			partition:        "aws-iso",
			expectedResource: "arn:aws-iso:s3:*:*:*",
		},
		{
			name:             "isob partition",
			partition:        "aws-iso-b",
			expectedResource: "arn:aws-iso-b:s3:*:*:*",
		},
	}

	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			ServiceAccount: "test-service-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credReqs, err := BuildCredentialsRequest(&stack, stack.Spec.ServiceAccount, &manifestutils.TokenCCOAuthConfig{
				AWS: &manifestutils.TokenCCOAWSEnvironment{
					RoleARN:   "test-rolearn",
					Partition: tt.partition,
				},
			})
			require.NoError(t, err)

			credReq := credReqs[0].(*cloudcredentialv1.CredentialsRequest)
			decoded := &cloudcredentialv1.AWSProviderSpec{}
			require.NoError(t, cloudcredentialv1.Codec.DecodeProviderSpec(credReq.Spec.ProviderSpec, decoded))

			require.Len(t, decoded.StatementEntries, 1)
			require.Equal(t, tt.expectedResource, decoded.StatementEntries[0].Resource)
			require.Equal(t, "test-rolearn", decoded.STSIAMRoleARN)
		})
	}
}

func TestDiscoverTokenCCOAuthConfig_Partition(t *testing.T) {
	tests := []struct {
		name              string
		roleARN           string
		expectedPartition string
	}{
		{
			name:              "commercial role arn",
			roleARN:           "arn:aws:iam::123456789012:role/tempo",
			expectedPartition: "aws",
		},
		{
			name:              "iso role arn",
			roleARN:           "arn:aws-iso:iam::123456789012:role/tempo",
			expectedPartition: "aws-iso",
		},
		{
			name:              "unparseable role arn falls back to the commercial partition",
			roleARN:           "test-rolearn",
			expectedPartition: "aws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ROLEARN", tt.roleARN)

			config := DiscoverTokenCCOAuthConfig()

			require.NotNil(t, config)
			require.NotNil(t, config.AWS)
			require.Equal(t, tt.roleARN, config.AWS.RoleARN)
			require.Equal(t, tt.expectedPartition, config.AWS.Partition)
		})
	}
}
