package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// ComponentResources defines CPU and Memory requests for a component.
type ComponentResources struct {
	CPU    resource.Quantity
	Memory resource.Quantity
}

// SizeProfile defines resources for all components at a given size.
type SizeProfile struct {
	Ingester          ComponentResources
	Compactor         ComponentResources
	Querier           ComponentResources
	QueryFrontend     ComponentResources
	Distributor       ComponentResources
	Gateway           ComponentResources
	JaegerFrontend    ComponentResources
	ReplicationFactor int
}

// sizeProfiles maps each size to its resource profile.
// Resources are based on performance testing results.
// 1x.demo has no resources (nil profile).
var sizeProfiles = map[v1alpha1.TempoStackSize]*SizeProfile{
	// 1x.demo: No resources assigned (similar to LokiStack pattern)
	// Pods run without resource constraints for local dev/demo environments
	v1alpha1.SizeDemo: nil,

	// 1x.pico: Small production workloads with HA support
	v1alpha1.SizePico: {
		Ingester:          ComponentResources{CPU: resource.MustParse("500m"), Memory: resource.MustParse("3Gi")},
		Compactor:         ComponentResources{CPU: resource.MustParse("500m"), Memory: resource.MustParse("500Mi")},
		Querier:           ComponentResources{CPU: resource.MustParse("750m"), Memory: resource.MustParse("1536Mi")},
		QueryFrontend:     ComponentResources{CPU: resource.MustParse("500m"), Memory: resource.MustParse("500Mi")},
		Distributor:       ComponentResources{CPU: resource.MustParse("500m"), Memory: resource.MustParse("500Mi")},
		Gateway:           ComponentResources{CPU: resource.MustParse("100m"), Memory: resource.MustParse("64Mi")},
		JaegerFrontend:          ComponentResources{CPU: resource.MustParse("100m"), Memory: resource.MustParse("64Mi")},
		ReplicationFactor: 2,
	},

	// 1x.extra-small: Medium production workloads (~100GB/day) with HA support
	// Based on performance test XS configuration
	v1alpha1.SizeExtraSmall: {
		Ingester:          ComponentResources{CPU: resource.MustParse("1500m"), Memory: resource.MustParse("8Gi")},
		Compactor:         ComponentResources{CPU: resource.MustParse("200m"), Memory: resource.MustParse("4Gi")},
		Querier:           ComponentResources{CPU: resource.MustParse("1000m"), Memory: resource.MustParse("1Gi")},
		QueryFrontend:     ComponentResources{CPU: resource.MustParse("200m"), Memory: resource.MustParse("1Gi")},
		Distributor:       ComponentResources{CPU: resource.MustParse("200m"), Memory: resource.MustParse("128Mi")},
		Gateway:           ComponentResources{CPU: resource.MustParse("400m"), Memory: resource.MustParse("128Mi")},
		JaegerFrontend:          ComponentResources{CPU: resource.MustParse("400m"), Memory: resource.MustParse("128Mi")},
		ReplicationFactor: 2,
	},

	// 1x.small: Larger production workloads (~500GB/day) with HA support
	// Based on performance test S configuration
	v1alpha1.SizeSmall: {
		Ingester:          ComponentResources{CPU: resource.MustParse("2000m"), Memory: resource.MustParse("10Gi")},
		Compactor:         ComponentResources{CPU: resource.MustParse("400m"), Memory: resource.MustParse("6Gi")},
		Querier:           ComponentResources{CPU: resource.MustParse("2500m"), Memory: resource.MustParse("3Gi")},
		QueryFrontend:     ComponentResources{CPU: resource.MustParse("400m"), Memory: resource.MustParse("1Gi")},
		Distributor:       ComponentResources{CPU: resource.MustParse("600m"), Memory: resource.MustParse("128Mi")},
		Gateway:           ComponentResources{CPU: resource.MustParse("800m"), Memory: resource.MustParse("192Mi")},
		JaegerFrontend:          ComponentResources{CPU: resource.MustParse("800m"), Memory: resource.MustParse("192Mi")},
		ReplicationFactor: 2,
	},

	// 1x.medium: High-scale production workloads (~2TB/day) with HA support
	// Based on performance test M configuration
	v1alpha1.SizeMedium: {
		Ingester:          ComponentResources{CPU: resource.MustParse("8000m"), Memory: resource.MustParse("16Gi")},
		Compactor:         ComponentResources{CPU: resource.MustParse("600m"), Memory: resource.MustParse("10Gi")},
		Querier:           ComponentResources{CPU: resource.MustParse("5000m"), Memory: resource.MustParse("4Gi")},
		QueryFrontend:     ComponentResources{CPU: resource.MustParse("800m"), Memory: resource.MustParse("1Gi")},
		Distributor:       ComponentResources{CPU: resource.MustParse("1500m"), Memory: resource.MustParse("128Mi")},
		Gateway:           ComponentResources{CPU: resource.MustParse("4000m"), Memory: resource.MustParse("192Mi")},
		JaegerFrontend:          ComponentResources{CPU: resource.MustParse("4000m"), Memory: resource.MustParse("192Mi")},
		ReplicationFactor: 2,
	},
}

// RateLimitProfile defines rate limit defaults for a given size.
// Pointer fields allow distinguishing between "not set" (nil) and "set to zero".
type RateLimitProfile struct {
	IngestionRateLimitBytes *int
	IngestionBurstSizeBytes *int
	MaxTracesPerUser        *int
}

// rateLimitProfiles maps each size to its rate limit profile.
// Rate limits are based on performance testing results.
// nil profile means no rate limits should be applied (use Tempo defaults).
var rateLimitProfiles = map[v1alpha1.TempoStackSize]*RateLimitProfile{
	// 1x.demo: No rate limits (development/demo environment)
	v1alpha1.SizeDemo: nil,

	// 1x.pico: Small production workloads
	// ~0.6 MB/s ingestion rate, burst 2x
	// MaxTracesPerUser uses Tempo default (10K is sufficient)
	v1alpha1.SizePico: {
		IngestionRateLimitBytes: ptr.To(600_000),
		IngestionBurstSizeBytes: ptr.To(1_200_000),
		MaxTracesPerUser:        nil,
	},

	// 1x.extra-small: Medium production workloads (~100GB/day)
	// ~1.2 MB/s ingestion rate, burst 2x
	// MaxTracesPerUser uses Tempo default (10K is sufficient)
	v1alpha1.SizeExtraSmall: {
		IngestionRateLimitBytes: ptr.To(1_200_000),
		IngestionBurstSizeBytes: ptr.To(2_400_000),
		MaxTracesPerUser:        nil,
	},

	// 1x.small: Larger production workloads (~500GB/day)
	// ~5.8 MB/s ingestion rate, burst 2x
	// MaxTracesPerUser: observed 13.8K + 20% headroom
	v1alpha1.SizeSmall: {
		IngestionRateLimitBytes: ptr.To(5_800_000),
		IngestionBurstSizeBytes: ptr.To(11_600_000),
		MaxTracesPerUser:        ptr.To(17_000),
	},

	// 1x.medium: High-scale production workloads (~2TB/day)
	// ~23 MB/s ingestion rate, burst 2x
	// MaxTracesPerUser: observed 44.3K + 20% headroom
	v1alpha1.SizeMedium: {
		IngestionRateLimitBytes: ptr.To(23_000_000),
		IngestionBurstSizeBytes: ptr.To(46_000_000),
		MaxTracesPerUser:        ptr.To(54_000),
	},
}

// replicationFactors maps each size to its default replication factor.
var replicationFactors = map[v1alpha1.TempoStackSize]int{
	v1alpha1.SizeDemo:       1,
	v1alpha1.SizePico:       2,
	v1alpha1.SizeExtraSmall: 2,
	v1alpha1.SizeSmall:      2,
	v1alpha1.SizeMedium:     2,
}

// GetSizeProfile returns the resource profile for the given size.
// Returns nil if size is empty or is SizeDemo (which has no resources).
func GetSizeProfile(size v1alpha1.TempoStackSize) *SizeProfile {
	if size == "" {
		return nil
	}
	return sizeProfiles[size]
}

// GetRateLimitProfile returns the rate limit profile for the given size.
// Returns nil if size is empty or is SizeDemo (which has no rate limits).
func GetRateLimitProfile(size v1alpha1.TempoStackSize) *RateLimitProfile {
	if size == "" {
		return nil
	}
	return rateLimitProfiles[size]
}

// ReplicationFactorForSize returns the default replication factor for the given size.
// Returns 0 if size is empty or unknown.
func ReplicationFactorForSize(size v1alpha1.TempoStackSize) int {
	if size == "" {
		return 0
	}
	if rf, ok := replicationFactors[size]; ok {
		return rf
	}
	return 0
}

// ResourcesForComponent returns the resource requirements for a specific component based on size.
// Returns only resource Requests (no Limits) as per the design.
// Returns empty ResourceRequirements if size is empty, SizeDemo, or component is unknown.
func ResourcesForComponent(size v1alpha1.TempoStackSize, component string) corev1.ResourceRequirements {
	profile := GetSizeProfile(size)
	if profile == nil {
		return corev1.ResourceRequirements{}
	}

	var compRes ComponentResources
	switch component {
	case IngesterComponentName:
		compRes = profile.Ingester
	case CompactorComponentName:
		compRes = profile.Compactor
	case QuerierComponentName:
		compRes = profile.Querier
	case QueryFrontendComponentName:
		compRes = profile.QueryFrontend
	case DistributorComponentName:
		compRes = profile.Distributor
	case GatewayComponentName:
		compRes = profile.Gateway
	case JaegerFrontendComponentName:
		compRes = profile.JaegerFrontend
	default:
		return corev1.ResourceRequirements{}
	}

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    compRes.CPU,
			corev1.ResourceMemory: compRes.Memory,
		},
	}
}
