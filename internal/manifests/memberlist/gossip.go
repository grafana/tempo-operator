package memberlist

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	componentName = "gossip-ring"
)

var (
	GossipSelector = map[string]string{"tempo-gossip-member": "true"}
)

// BuildGossip creates Kubernetes objects that are needed for memberlist.
func BuildGossip(tempo v1alpha1.Microservices) *corev1.Service {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	selector := k8slabels.Merge(manifestutils.CommonLabels(tempo.Name), GossipSelector)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpMemberlistPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortMemberlist,
					TargetPort: intstr.FromString(manifestutils.HttpMemberlistPortName),
				},
			},
		},
	}
}
