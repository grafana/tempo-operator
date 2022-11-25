package memberlist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildGossip(t *testing.T) {
	service := BuildGossip(v1alpha1.Microservices{
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
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       componentName,
					Protocol:   corev1.ProtocolTCP,
					Port:       PortMemberlist,
					TargetPort: intstr.FromString("http-memberlist"),
				},
			},
		},
	}, service)
}
