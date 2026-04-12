package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestPatchEnvVars(t *testing.T) {
	t.Run("appends user env vars after existing env vars", func(t *testing.T) {
		pod := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "tempo",
					Env: []corev1.EnvVar{
						{Name: "EXISTING", Value: "value"},
					},
				},
			},
		}

		userEnv := []corev1.EnvVar{
			{Name: "REDIS_PASSWORD", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "redis-secret"},
					Key:                  "password",
				},
			}},
		}

		PatchEnvVars(&pod, "tempo", userEnv, nil)

		require.Len(t, pod.Containers[0].Env, 2)
		require.Equal(t, "EXISTING", pod.Containers[0].Env[0].Name)
		require.Equal(t, "REDIS_PASSWORD", pod.Containers[0].Env[1].Name)
	})

	t.Run("sets envFrom on the container", func(t *testing.T) {
		pod := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "tempo",
				},
			},
		}

		envFrom := []corev1.EnvFromSource{
			{SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
			}},
		}

		PatchEnvVars(&pod, "tempo", nil, envFrom)

		require.Len(t, pod.Containers[0].EnvFrom, 1)
		require.Equal(t, "my-secret", pod.Containers[0].EnvFrom[0].SecretRef.Name)
	})

	t.Run("no-op when env and envFrom are empty", func(t *testing.T) {
		pod := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "tempo",
					Env: []corev1.EnvVar{
						{Name: "EXISTING", Value: "value"},
					},
				},
			},
		}

		PatchEnvVars(&pod, "tempo", nil, nil)

		require.Len(t, pod.Containers[0].Env, 1)
		require.Nil(t, pod.Containers[0].EnvFrom)
	})

	t.Run("only modifies the named container", func(t *testing.T) {
		pod := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "jaeger-query",
				},
				{
					Name: "tempo",
				},
			},
		}

		userEnv := []corev1.EnvVar{
			{Name: "REDIS_PASSWORD", Value: "secret"},
		}

		PatchEnvVars(&pod, "tempo", userEnv, nil)

		require.Empty(t, pod.Containers[0].Env)
		require.Len(t, pod.Containers[1].Env, 1)
		require.Equal(t, "REDIS_PASSWORD", pod.Containers[1].Env[0].Name)
	})

	t.Run("does nothing when container not found", func(t *testing.T) {
		pod := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "other",
				},
			},
		}

		userEnv := []corev1.EnvVar{
			{Name: "REDIS_PASSWORD", Value: "secret"},
		}

		PatchEnvVars(&pod, "tempo", userEnv, nil)

		require.Empty(t, pod.Containers[0].Env)
	})
}
