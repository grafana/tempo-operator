package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TempoReadinessProbe returns a readiness Probe spec for tempo components.
func TempoReadinessProbe(tlsEnable bool) *corev1.Probe {

	scheme := corev1.URISchemeHTTP
	port := intstr.FromString(HttpPortName)

	if tlsEnable {
		scheme = corev1.URISchemeHTTPS
		port = intstr.FromInt(PortInternalHTTPServer)
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: scheme,
				Path:   TempoReadinessPath,
				Port:   port,
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      1,
	}
}
