package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TempoContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.Bool(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		ReadOnlyRootFilesystem: pointer.Bool(true),
	}
}
