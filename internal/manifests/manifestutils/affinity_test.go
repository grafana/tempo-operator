package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigureAffinityDefault(t *testing.T) {
	labels := ComponentLabels(IngesterComponentName, "test")

	assert.Equal(t, DefaultAffinity(labels), ConfigureAffinity(labels, nil))
}

func TestConfigureAffinityOverride(t *testing.T) {
	labels := ComponentLabels(IngesterComponentName, "test")

	// a stack that requires, rather than prefers, one pod per node
	podAntiAffinity := &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
			LabelSelector: &metav1.LabelSelector{MatchLabels: labels},
			TopologyKey:   corev1.LabelHostname,
		}},
	}

	affinity := ConfigureAffinity(labels, podAntiAffinity)

	// the anti affinity of the component replaces the default, it is not merged into it
	assert.Equal(t, podAntiAffinity, affinity.PodAntiAffinity)
	assert.Empty(t, affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)

	// the default must not be modified for the other components
	assert.Len(t, DefaultAffinity(labels).PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, 2)
}
