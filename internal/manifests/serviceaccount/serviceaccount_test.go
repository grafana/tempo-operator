package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildDefaultServiceAccount(t *testing.T) {
	serviceAccount := BuildDefaultServiceAccount(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns1",
			},
		}})

	labels := manifestutils.ComponentLabels("serviceaccount", "test")
	require.NotNil(t, serviceAccount)
	assert.Equal(t, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test",
			Namespace: "ns1",
			Labels:    labels,
		},
	}, serviceAccount)
}

func TestBuildDefaultServiceAccount_aws_sts(t *testing.T) {
	serviceAccount := BuildDefaultServiceAccount(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns1",
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeToken,
			S3: &manifestutils.S3{
				RoleARN: "arn:aws:iam::123456777012:role/aws-service-role",
			},
		}})

	labels := manifestutils.ComponentLabels("serviceaccount", "test")
	require.NotNil(t, serviceAccount)
	assert.Equal(t, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test",
			Namespace: "ns1",
			Labels:    labels,
			Annotations: map[string]string{
				"eks.amazonaws.com/audience": "sts.amazonaws.com",
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123456777012:role/aws-service-role",
			},
		},
	}, serviceAccount)
}

func TestBuildDefaultServiceAccount_azure_sts(t *testing.T) {
	serviceAccount := BuildDefaultServiceAccount(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns1",
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeToken,
			AzureStorage: &manifestutils.AzureStorage{
				ClientID: "client1",
				TenantID: "tenant1",
			},
		}})

	labels := manifestutils.ComponentLabels("serviceaccount", "test")
	require.NotNil(t, serviceAccount)
	assert.Equal(t, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test",
			Namespace: "ns1",
			Labels:    labels,
			Annotations: map[string]string{
				"azure.workload.identity/client-id": "client1",
				"azure.workload.identity/tenant-id": "tenant1",
			},
		},
	}, serviceAccount)
}
