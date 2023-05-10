package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

const requestsPercentage = 0.3

type componentResource struct {
	memory float32
	cpu    float32
}

var (
	resourcesMapNoGateway = map[string]componentResource{
		"distributor":    {cpu: 0.27, memory: 0.12},
		"ingester":       {cpu: 0.38, memory: 0.5},
		"compactor":      {cpu: 0.16, memory: 0.18},
		"querier":        {cpu: 0.1, memory: 0.15},
		"query-frontend": {cpu: 0.09, memory: 0.05},
	}
	resourcesMapWithGateway = map[string]componentResource{
		"distributor":    {cpu: 0.26, memory: 0.11},
		"ingester":       {cpu: 0.36, memory: 0.49},
		"compactor":      {cpu: 0.15, memory: 0.17},
		"querier":        {cpu: 0.09, memory: 0.14},
		"query-frontend": {cpu: 0.08, memory: 0.04},
		"gateway":        {cpu: 0.06, memory: 0.05},
	}
)

// Resources calculates the resource requirements of a specific component.
func Resources(tempo v1alpha1.TempoStack, component string) corev1.ResourceRequirements {

	resourcesMap := resourcesMapNoGateway
	if tempo.Spec.Template.Gateway.Enabled {
		resourcesMap = resourcesMapWithGateway
	}

	componentResources, ok := resourcesMap[component]
	if tempo.Spec.Resources.Total == nil || !ok {
		return corev1.ResourceRequirements{}
	}
	resources := corev1.ResourceRequirements{}
	totalCpu, ok := tempo.Spec.Resources.Total.Limits[corev1.ResourceCPU]
	if ok {
		totalCpuInt := totalCpu.MilliValue()
		cpu := float32(totalCpuInt) * componentResources.cpu

		resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewMilliQuantity(int64(cpu), resource.BinarySI),
		}
		resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewMilliQuantity(int64(cpu*requestsPercentage), resource.BinarySI),
		}
	}

	totalMemory, ok := tempo.Spec.Resources.Total.Limits[corev1.ResourceMemory]
	if ok {
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
		if resources.Requests == nil {
			resources.Requests = corev1.ResourceList{}
		}
		totalMemoryInt := totalMemory.Value()
		mem := float32(totalMemoryInt) * componentResources.memory
		resources.Limits[corev1.ResourceMemory] = *resource.NewQuantity(int64(mem), resource.BinarySI)
		resources.Requests[corev1.ResourceMemory] = *resource.NewQuantity(int64(mem*0.3), resource.BinarySI)
	}
	return resources
}
