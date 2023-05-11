package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestResourceSum(t *testing.T) {
	cpu := float32(0)
	mem := float32(0.0)
	for _, r := range resourcesMapNoGateway {
		mem += r.memory
		cpu += r.cpu
	}
	assert.Equal(t, float32(1.0), cpu)
	assert.Equal(t, float32(1.0), mem)
}

func TestResourceWithGatewaySum(t *testing.T) {
	cpu := float32(0)
	mem := float32(0.0)
	for _, r := range resourcesMapWithGateway {
		mem += r.memory
		cpu += r.cpu
	}
	assert.Equal(t, float32(1.0), cpu)
	assert.Equal(t, float32(1.0), mem)
}

func TestResources(t *testing.T) {
	tests := []struct {
		resources corev1.ResourceRequirements
		name      string
		tempo     v1alpha1.TempoStack
	}{
		{
			name: "resources not set",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{},
			},
			resources: corev1.ResourceRequirements{},
		},
		{
			name: "cpu, memory resources set",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Resources: v1alpha1.Resources{
						Total: &corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("2Gi"),
								corev1.ResourceCPU:    resource.MustParse("1000m"),
							},
						},
					},
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
		},
		{
			name: "cpu, memory resources set and gateway enable",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
					Resources: v1alpha1.Resources{
						Total: &corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("2Gi"),
								corev1.ResourceCPU:    resource.MustParse("1000m"),
							},
						},
					},
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(260, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(236223200, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(78, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(70866960, resource.BinarySI),
				},
			},
		},
		{
			name: "missing cpu resources",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Resources: v1alpha1.Resources{
						Total: &corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
		},
		{
			name: "missing memory resources",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Resources: v1alpha1.Resources{
						Total: &corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("2Gi"),
								corev1.ResourceCPU:    resource.MustParse("1000m"),
							},
						},
					},
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resources := Resources(test.tempo, "distributor")
			assert.Equal(t, test.resources, resources)
		})
	}
}
