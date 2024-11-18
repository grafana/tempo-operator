package memberlist

import (
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	componentName = "gossip-ring"
	// GossipInstanceAddrEnvVarName is the name of the hash ring instance address env var.
	GossipInstanceAddrEnvVarName = "HASH_RING_INSTANCE_ADDR"
)

var (
	// GossipSelector declares the labels for each gossip member.
	GossipSelector = map[string]string{"tempo-gossip-member": "true"}
)

// BuildGossip creates Kubernetes objects that are needed for memberlist.
func BuildGossip(tempo v1alpha1.TempoStack) *corev1.Service {
	labels := manifestutils.ComponentLabels(componentName, tempo.Name)
	selector := k8slabels.Merge(manifestutils.CommonLabels(tempo.Name), GossipSelector)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(componentName, tempo.Name),
			Namespace: tempo.Namespace,
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
	}
}

// ConfigureHashRingEnv adds an environment variable with the podIP if instanceAddressType = podIP.
func ConfigureHashRingEnv(p *corev1.PodSpec, tempo v1alpha1.TempoStack) error {

	memberList := tempo.Spec.HashRing.MemberList
	enableIPV6 := memberList.EnableIPv6 != nil && *memberList.EnableIPv6

	if !enableIPV6 && memberList.InstanceAddrType != v1alpha1.InstanceAddrPodIP {
		return nil
	}

	src := corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name: GossipInstanceAddrEnvVarName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		},
	}

	for i := range p.Containers {
		if err := mergo.Merge(&p.Containers[i], src, mergo.WithAppendSlice); err != nil {
			return err
		}
	}

	return nil
}
