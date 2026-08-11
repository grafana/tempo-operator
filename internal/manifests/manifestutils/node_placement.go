package manifestutils

import (
	"fmt"
	"strings"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

const (
	// AvailabilityZoneEnvVarName is the name of the env var holding the availability zone of the pod.
	// It is referenced by the Tempo configuration, which is expanded at startup via -config.expand-env.
	AvailabilityZoneEnvVarName = "INSTANCE_AVAILABILITY_ZONE"

	availabilityZoneFieldPath           = "metadata.annotations['" + v1alpha1.AnnotationAvailabilityZone + "']"
	availabilityZoneInitVolumeName      = "az-annotation"
	availabilityZoneInitVolumeMountPath = "/etc/az-annotation"
	availabilityZoneInitVolumeFileName  = "az"
)

var availabilityZoneEnvVar = corev1.EnvVar{
	Name: AvailabilityZoneEnvVarName,
	ValueFrom: &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: availabilityZoneFieldPath,
		},
	},
}

// ConfigureReplication configures the zone awareness of a component.
//
// It spreads the pods of the component across the configured zones using topology spread constraints,
// and marks the pods for the zone-awareness controller, which annotates them with the availability zone
// of the node they were scheduled on. Every component except the gateway additionally waits for that
// annotation to be set and exposes it as an environment variable, so that Tempo can register itself in
// the ring with its availability zone.
func ConfigureReplication(podTemplate *corev1.PodTemplateSpec, zones []v1alpha1.ZoneSpec, component string, stackName string) error {
	if len(zones) == 0 {
		return nil
	}

	template := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				v1alpha1.LabelZoneAwarePod: "enabled",
			},
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: make([]corev1.Container, len(podTemplate.Spec.Containers)),
		},
	}

	zoneKeys := []string{}
	for _, zone := range zones {
		zoneKeys = append(zoneKeys, zone.TopologyKey)
		template.Spec.TopologySpreadConstraints = append(template.Spec.TopologySpreadConstraints, corev1.TopologySpreadConstraint{
			MaxSkew:           int32(zone.MaxSkew), //nolint:gosec // maxSkew is validated to be within [1, math.MaxInt32] by the API
			TopologyKey:       zone.TopologyKey,
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": component,
					"app.kubernetes.io/instance":  stackName,
				},
			},
		})
	}

	template.Annotations[v1alpha1.AnnotationAvailabilityZoneLabels] = strings.Join(zoneKeys, ",")

	// The gateway does not join the hash ring, therefore it does not need to know its availability zone.
	if component != GatewayComponentName {
		template.Spec.InitContainers = []corev1.Container{
			initContainerAZAnnotationCheck(podTemplate.Spec.Containers[0]),
		}

		src := corev1.Container{
			Env: []corev1.EnvVar{availabilityZoneEnvVar},
		}

		for i, dst := range podTemplate.Spec.Containers {
			if err := mergo.Merge(&dst, src, mergo.WithAppendSlice); err != nil {
				return err
			}
			podTemplate.Spec.Containers[i] = dst
		}

		vols := []corev1.Volume{azAnnotationVolume()}
		if err := mergo.Merge(&podTemplate.Spec.Volumes, vols, mergo.WithAppendSlice); err != nil {
			return err
		}
	}

	return mergo.Merge(podTemplate, template)
}

// initContainerAZAnnotationCheck returns an init container blocking the start of the pod until the
// zone-awareness controller annotated it with its availability zone.
//
// The image and the resource requirements of the Tempo container are reused. Init containers run
// before the regular containers, therefore reusing the resource requirements does not increase the
// effective resource requests of the pod.
func initContainerAZAnnotationCheck(tempoContainer corev1.Container) corev1.Container {
	azPath := fmt.Sprintf("%s/%s", availabilityZoneInitVolumeMountPath, availabilityZoneInitVolumeFileName)
	return corev1.Container{
		Name:  "az-annotation-check",
		Image: tempoContainer.Image,
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("while ! [ -s %s ]; do echo Waiting for availability zone annotation to be set; sleep 2; done; echo availability zone annotation is set; cat %s; echo", azPath, azPath),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      availabilityZoneInitVolumeName,
				MountPath: availabilityZoneInitVolumeMountPath,
			},
		},
		Resources:       tempoContainer.Resources,
		SecurityContext: TempoContainerSecurityContext(),
	}
}

// azAnnotationVolume exposes the availability zone annotation of the pod as a file, so that the init
// container can wait for it to be populated.
func azAnnotationVolume() corev1.Volume {
	return corev1.Volume{
		Name: availabilityZoneInitVolumeName,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: availabilityZoneInitVolumeFileName,
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: availabilityZoneFieldPath,
						},
					},
				},
			},
		},
	}
}
