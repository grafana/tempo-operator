package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// TempoContainerSecurityContext returns the default container security context.
func TempoContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		ReadOnlyRootFilesystem: ptr.To(true),
	}
}
