package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
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
	jaegerUIResourcePercentage = 0.2 // This is the percentage of the resource assigned to the querier component.
)

func ResourcesForQuerierAndJaegerUI(resources corev1.ResourceRequirements) (*corev1.ResourceRequirements, *corev1.ResourceRequirements) {
	queryFrontendResources := corev1.ResourceRequirements{}
	jaegerUIResources := corev1.ResourceRequirements{}

	cpu := resources.Requests.Cpu().Value()
	memory := resources.Requests.Memory().Value()

	jaegerUICPU := jaegerUIResourcePercentage * float64(cpu)
	tempoQueryFrontEndCPU := cpu - int64(jaegerUICPU)

	jaegerUIMem := jaegerUIResourcePercentage * float64(memory)
	tempoQueryFrontEndMem := memory - int64(jaegerUIMem)

	jaegerUIResources.Limits = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(jaegerUICPU*0.3), resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(int64(jaegerUIMem*0.3), resource.BinarySI),
	}
	jaegerUIResources.Requests = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(jaegerUICPU), resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(int64(jaegerUIMem), resource.BinarySI),
	}

	queryFrontendResources.Limits = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(float64(tempoQueryFrontEndCPU)*0.3), resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(int64(float64(tempoQueryFrontEndMem)*0.3), resource.BinarySI),
	}
	queryFrontendResources.Requests = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(tempoQueryFrontEndCPU, resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(tempoQueryFrontEndMem, resource.BinarySI),
	}

	return &queryFrontendResources, &jaegerUIResources
}

// Resources calculates the resource requirements of a specific component.
func Resources(tempo v1alpha1.TempoStack, component string, replicas *int32) corev1.ResourceRequirements {

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
		if replicas != nil && *replicas > 1 {
			cpu /= float32(*replicas)
		}

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
		if replicas != nil && *replicas > 1 {
			mem /= float32(*replicas)
		}
		resources.Limits[corev1.ResourceMemory] = *resource.NewQuantity(int64(mem), resource.BinarySI)
		resources.Requests[corev1.ResourceMemory] = *resource.NewQuantity(int64(mem*0.3), resource.BinarySI)
	}
	return resources
}
