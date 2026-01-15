package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestResourceSum(t *testing.T) {
	cpu := float32(0)
	mem := float32(0.0)
	for _, r := range resourcesMapNoGateway {
		mem += r.memory
		cpu += r.cpu
	}
	assert.InDelta(t, float32(1.0), cpu, 0.01)
	assert.InDelta(t, float32(1.0), mem, 0.01)
}

func TestResourceWithGatewaySum(t *testing.T) {
	cpu := float32(0)
	mem := float32(0.0)
	for _, r := range resourcesMapWithGateway {
		mem += r.memory
		cpu += r.cpu
	}
	assert.InDelta(t, float32(1.0), cpu, 0.01)
	assert.InDelta(t, float32(1.0), mem, 0.01)
}

func TestResources(t *testing.T) {
	tests := []struct {
		resources corev1.ResourceRequirements
		name      string
		tempo     v1alpha1.TempoStack
		replicas  *int32
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
			name: "cpu, memory resources set with replicas",
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
					Template: v1alpha1.TempoTemplateSpec{
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(2)),
						},
					},
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(135, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(128849016, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(40, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(38654708, resource.BinarySI),
				},
			},
			replicas: ptr.To(int32(2)),
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
			name: "cpu, memory resources set with replicas and gateway enable",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(2)),
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
					corev1.ResourceCPU:    *resource.NewMilliQuantity(130, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(118111600, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(39, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(35433480, resource.BinarySI),
				},
			},
			replicas: ptr.To(int32(2)),
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
			resources := Resources(test.tempo, "distributor", test.replicas)
			assert.Equal(t, test.resources, resources)
		})
	}
}

func TestResourcesWithSize(t *testing.T) {
	tests := []struct {
		name      string
		tempo     v1alpha1.TempoStack
		component string
		want      corev1.ResourceRequirements
	}{
		{
			name: "size extra-small returns correct resources for distributor",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeExtraSmall,
				},
			},
			component: DistributorComponentName,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
		{
			name: "size extra-small returns correct resources for ingester",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeExtraSmall,
				},
			},
			component: IngesterComponentName,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1500m"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
		},
		{
			name: "size demo returns empty resources",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeDemo,
				},
			},
			component: DistributorComponentName,
			want:      corev1.ResourceRequirements{},
		},
		{
			name: "size takes precedence over resources.total",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeSmall,
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
			component: DistributorComponentName,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("600m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
		{
			name: "size demo takes precedence over resources.total (returns empty)",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeDemo,
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
			component: DistributorComponentName,
			want:      corev1.ResourceRequirements{},
		},
		{
			name: "size small returns correct resources for gateway",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeSmall,
				},
			},
			component: GatewayComponentName,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("800m"),
					corev1.ResourceMemory: resource.MustParse("192Mi"),
				},
			},
		},
		{
			name: "size small returns correct resources for jaeger-frontend",
			tempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Size: v1alpha1.SizeSmall,
				},
			},
			component: JaegerFrontendComponentName,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("800m"),
					corev1.ResourceMemory: resource.MustParse("192Mi"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resources(tt.tempo, tt.component, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
