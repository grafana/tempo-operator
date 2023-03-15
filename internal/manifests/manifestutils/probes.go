package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TempoReadinessProbe returns a readiness Probe spec for tempo components.
func TempoReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: TempoReadinessPath,
				Port: intstr.FromString(HttpPortName),
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      1,
	}
}
