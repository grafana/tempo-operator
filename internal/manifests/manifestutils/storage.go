package manifestutils

import (
	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/api/v1alpha1"
)

// ConfigureStorage configures storage.
func ConfigureStorage(tempo v1alpha1.Microservices, pod *corev1.PodSpec) error {
	if tempo.Spec.Storage.Secret != "" {
		ingesterContainer := pod.Containers[0].DeepCopy()
		ingesterContainer.Env = append(ingesterContainer.Env,
			corev1.EnvVar{
				Name: "S3_SECRET_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "access_key_secret",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: tempo.Spec.Storage.Secret,
						},
					},
				},
			}, corev1.EnvVar{
				Name: "S3_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "access_key_id",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: tempo.Spec.Storage.Secret,
						},
					},
				},
			})

		ingesterContainer.Args = append(ingesterContainer.Args, "--storage.trace.s3.secret_key=$(S3_SECRET_KEY)")
		ingesterContainer.Args = append(ingesterContainer.Args, "--storage.trace.s3.access_key=$(S3_ACCESS_KEY)")

		if err := mergo.Merge(&pod.Containers[0], ingesterContainer, mergo.WithOverride); err != nil {
			return kverrors.Wrap(err, "failed to merge ingester container spec")
		}
	}
	return nil
}
