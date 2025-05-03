package monolithic

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildServiceAccount(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}
	sa := BuildServiceAccount(opts)

	labels := ComponentLabels("serviceaccount", "sample")
	require.Equal(t, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample",
			Namespace: "default",
			Labels:    labels,
		},
	}, sa)
}

func TestBuildServiceAccount_aws_sts(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeToken,
			S3: &manifestutils.S3{
				RoleARN: "arn:aws:iam::123456777012:role/aws-service-role",
			},
		},
	}
	sa := BuildServiceAccount(opts)

	labels := ComponentLabels("serviceaccount", "sample")
	require.Equal(t, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample",
			Namespace: "default",
			Labels:    labels,
			Annotations: map[string]string{
				"eks.amazonaws.com/audience": "sts.amazonaws.com",
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123456777012:role/aws-service-role",
			},
		},
	}, sa)
}
