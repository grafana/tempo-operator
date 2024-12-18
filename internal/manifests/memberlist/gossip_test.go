package memberlist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildGossip(t *testing.T) {
	service := BuildGossip(v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns1",
		},
	})
	labels := manifestutils.ComponentLabels("gossip-ring", "test")
	selector := k8slabels.Merge(manifestutils.CommonLabels("test"), GossipSelector)
	require.NotNil(t, service)
	assert.Equal(t, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-gossip-ring",
			Namespace: "ns1",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Selector:                 selector,
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpMemberlistPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortMemberlist,
					TargetPort: intstr.FromString(manifestutils.HttpMemberlistPortName),
				},
			},
		},
	}, service)
}

func TestConfigureHashRingEnv(t *testing.T) {

	tests := []struct {
		name                     string
		instanceAddrType         v1alpha1.InstanceAddrType
		expectedContainerEnvVars []corev1.EnvVar
	}{
		{
			name:             "with podIP",
			instanceAddrType: v1alpha1.InstanceAddrPodIP,
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name: "HASH_RING_INSTANCE_ADDR",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "status.podIP",
						},
					},
				},
			},
		},

		{
			name:             "with podIP",
			instanceAddrType: v1alpha1.InstanceAddrDefault,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			podSpec := &corev1.PodSpec{
				ServiceAccountName: "tempo-test-serviceaccount",
				Containers: []corev1.Container{
					{
						Name:  "tempo",
						Image: "docker.io/grafana/tempo:1.5.0",
						Args: []string{
							"-target=compactor",
							"-config.file=/conf/tempo.yaml",
							"-log.level=info",
							"-config.expand-env=true",
						},
					},
				},
			}

			tempoStackSpec := v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					HashRing: v1alpha1.HashRingSpec{
						MemberList: v1alpha1.MemberListSpec{
							InstanceAddrType: tc.instanceAddrType,
						},
					},
				},
			}

			err := ConfigureHashRingEnv(podSpec, tempoStackSpec)
			assert.NoError(t, err)

			assert.Equal(t, podSpec, &corev1.PodSpec{
				ServiceAccountName: "tempo-test-serviceaccount",
				Containers: []corev1.Container{
					{
						Name:  "tempo",
						Image: "docker.io/grafana/tempo:1.5.0",
						Args: []string{
							"-target=compactor",
							"-config.file=/conf/tempo.yaml",
							"-log.level=info",
							"-config.expand-env=true",
						},
						Env: tc.expectedContainerEnvVars,
					},
				},
			})
		})
	}
}
