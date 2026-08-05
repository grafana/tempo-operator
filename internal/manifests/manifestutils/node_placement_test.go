package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func podTemplate() *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app.kubernetes.io/component": "ingester"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "tempo", Image: "docker.io/grafana/tempo:x.y.z"},
			},
			Volumes: []corev1.Volume{
				{Name: "tempo-conf"},
			},
		},
	}
}

func TestConfigureReplicationWithoutZones(t *testing.T) {
	for _, tc := range []struct {
		name        string
		replication *v1alpha1.ReplicationSpec
	}{
		{"nil replication", nil},
		{"no zones", &v1alpha1.ReplicationSpec{Factor: 3}},
		{"empty zones", &v1alpha1.ReplicationSpec{Factor: 3, Zones: []v1alpha1.ZoneSpec{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			template := podTemplate()
			require.NoError(t, ConfigureReplication(template, tc.replication, IngesterComponentName, "test"))

			assert.Equal(t, podTemplate(), template)
		})
	}
}

func TestConfigureReplicationSpreadsAcrossZones(t *testing.T) {
	template := podTemplate()
	replication := &v1alpha1.ReplicationSpec{
		Factor: 3,
		Zones: []v1alpha1.ZoneSpec{
			{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone"},
			{MaxSkew: 2, TopologyKey: "kubernetes.io/hostname"},
		},
	}

	require.NoError(t, ConfigureReplication(template, replication, IngesterComponentName, "test"))

	assert.Equal(t, []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/zone",
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/component": "ingester",
				"app.kubernetes.io/instance":  "test",
			}},
		},
		{
			MaxSkew:           2,
			TopologyKey:       "kubernetes.io/hostname",
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/component": "ingester",
				"app.kubernetes.io/instance":  "test",
			}},
		},
	}, template.Spec.TopologySpreadConstraints)

	// the zone-awareness controller selects the pods to annotate by this label,
	// and reads the node labels to look at from this annotation
	assert.Equal(t, "enabled", template.Labels[v1alpha1.LabelZoneAwarePod])
	assert.Equal(t, "topology.kubernetes.io/zone,kubernetes.io/hostname",
		template.Annotations[v1alpha1.AnnotationAvailabilityZoneLabels])

	// the existing labels must be preserved
	assert.Equal(t, "ingester", template.Labels["app.kubernetes.io/component"])
}

func TestConfigureReplicationExposesAvailabilityZone(t *testing.T) {
	template := podTemplate()
	replication := &v1alpha1.ReplicationSpec{
		Zones: []v1alpha1.ZoneSpec{{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone"}},
	}

	require.NoError(t, ConfigureReplication(template, replication, IngesterComponentName, "test"))

	require.Len(t, template.Spec.InitContainers, 1)
	initContainer := template.Spec.InitContainers[0]
	assert.Equal(t, "az-annotation-check", initContainer.Name)
	// the init container reuses the image of the tempo container
	assert.Equal(t, "docker.io/grafana/tempo:x.y.z", initContainer.Image)
	assert.Equal(t, []corev1.VolumeMount{{Name: "az-annotation", MountPath: "/etc/az-annotation"}}, initContainer.VolumeMounts)

	require.Len(t, template.Spec.Containers, 1)
	assert.Equal(t, []corev1.EnvVar{{
		Name: "INSTANCE_AVAILABILITY_ZONE",
		ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "metadata.annotations['tempo.grafana.com/availability-zone']",
		}},
	}}, template.Spec.Containers[0].Env)

	// the existing volumes must be preserved
	require.Len(t, template.Spec.Volumes, 2)
	assert.Equal(t, "tempo-conf", template.Spec.Volumes[0].Name)
	assert.Equal(t, corev1.Volume{
		Name: "az-annotation",
		VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{
			Items: []corev1.DownwardAPIVolumeFile{{
				Path: "az",
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.annotations['tempo.grafana.com/availability-zone']",
				},
			}},
		}},
	}, template.Spec.Volumes[1])
}

func TestConfigureReplicationSkipsAvailabilityZoneOfGateway(t *testing.T) {
	template := podTemplate()
	replication := &v1alpha1.ReplicationSpec{
		Zones: []v1alpha1.ZoneSpec{{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone"}},
	}

	require.NoError(t, ConfigureReplication(template, replication, GatewayComponentName, "test"))

	// the gateway does not join the hash ring, therefore it does not need to know its availability zone
	assert.Empty(t, template.Spec.InitContainers)
	assert.Empty(t, template.Spec.Containers[0].Env)
	require.Len(t, template.Spec.Volumes, 1)

	// it is spread across the zones nonetheless
	assert.Len(t, template.Spec.TopologySpreadConstraints, 1)
}
