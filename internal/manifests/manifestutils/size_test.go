package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestGetSizeProfile(t *testing.T) {
	tests := []struct {
		name    string
		size    v1alpha1.TempoStackSize
		wantNil bool
	}{
		{
			name:    "empty size returns nil",
			size:    "",
			wantNil: true,
		},
		{
			name:    "demo size returns nil (no resources)",
			size:    v1alpha1.SizeDemo,
			wantNil: true,
		},
		{
			name:    "pico size returns profile",
			size:    v1alpha1.SizePico,
			wantNil: false,
		},
		{
			name:    "extra-small size returns profile",
			size:    v1alpha1.SizeExtraSmall,
			wantNil: false,
		},
		{
			name:    "small size returns profile",
			size:    v1alpha1.SizeSmall,
			wantNil: false,
		},
		{
			name:    "medium size returns profile",
			size:    v1alpha1.SizeMedium,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := GetSizeProfile(tt.size)
			if tt.wantNil {
				assert.Nil(t, profile)
			} else {
				require.NotNil(t, profile)
				// Verify profile has component resources defined
				assert.NotEmpty(t, profile.Ingester.CPU.String())
				assert.NotEmpty(t, profile.Ingester.Memory.String())
			}
		})
	}
}

func TestReplicationFactorForSize(t *testing.T) {
	tests := []struct {
		name   string
		size   v1alpha1.TempoStackSize
		wantRF int
	}{
		{
			name:   "empty size returns 0",
			size:   "",
			wantRF: 0,
		},
		{
			name:   "demo size returns 1",
			size:   v1alpha1.SizeDemo,
			wantRF: 1,
		},
		{
			name:   "pico size returns 2",
			size:   v1alpha1.SizePico,
			wantRF: 2,
		},
		{
			name:   "extra-small size returns 2",
			size:   v1alpha1.SizeExtraSmall,
			wantRF: 2,
		},
		{
			name:   "small size returns 2",
			size:   v1alpha1.SizeSmall,
			wantRF: 2,
		},
		{
			name:   "medium size returns 2",
			size:   v1alpha1.SizeMedium,
			wantRF: 2,
		},
		{
			name:   "unknown size returns 0",
			size:   v1alpha1.TempoStackSize("unknown"),
			wantRF: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := ReplicationFactorForSize(tt.size)
			assert.Equal(t, tt.wantRF, rf)
		})
	}
}

func TestResourcesForComponent(t *testing.T) {
	// Test demo size returns empty resources
	t.Run("demo size returns empty resources for all components", func(t *testing.T) {
		components := []string{
			IngesterComponentName,
			CompactorComponentName,
			QuerierComponentName,
			QueryFrontendComponentName,
			DistributorComponentName,
			GatewayComponentName,
			JaegerFrontendComponentName,
		}

		for _, comp := range components {
			res := ResourcesForComponent(v1alpha1.SizeDemo, comp)
			assert.Nil(t, res.Requests, "demo size should return nil requests for %s", comp)
			assert.Nil(t, res.Limits, "demo size should return nil limits for %s", comp)
		}
	})

	// Test empty size returns empty resources
	t.Run("empty size returns empty resources", func(t *testing.T) {
		res := ResourcesForComponent("", IngesterComponentName)
		assert.Nil(t, res.Requests)
		assert.Nil(t, res.Limits)
	})

	// Test unknown component returns empty resources
	t.Run("unknown component returns empty resources", func(t *testing.T) {
		res := ResourcesForComponent(v1alpha1.SizeSmall, "unknown-component")
		assert.Nil(t, res.Requests)
		assert.Nil(t, res.Limits)
	})

	// Test that all sizes have all components defined (except demo)
	t.Run("all sizes have all components defined", func(t *testing.T) {
		sizes := []v1alpha1.TempoStackSize{
			v1alpha1.SizePico,
			v1alpha1.SizeExtraSmall,
			v1alpha1.SizeSmall,
			v1alpha1.SizeMedium,
		}

		components := []string{
			IngesterComponentName,
			CompactorComponentName,
			QuerierComponentName,
			QueryFrontendComponentName,
			DistributorComponentName,
			GatewayComponentName,
			JaegerFrontendComponentName,
		}

		for _, size := range sizes {
			for _, comp := range components {
				res := ResourcesForComponent(size, comp)
				require.NotNil(t, res.Requests, "size %s should have requests for %s", size, comp)
				assert.Contains(t, res.Requests, corev1.ResourceCPU, "size %s should have CPU request for %s", size, comp)
				assert.Contains(t, res.Requests, corev1.ResourceMemory, "size %s should have memory request for %s", size, comp)
				// Limits should be nil (we only set requests per design)
				assert.Nil(t, res.Limits, "size %s should not have limits for %s", size, comp)
			}
		}
	})

	// Test specific resource values for extra-small size (based on perf tests)
	t.Run("extra-small size has correct values", func(t *testing.T) {
		ingesterRes := ResourcesForComponent(v1alpha1.SizeExtraSmall, IngesterComponentName)
		assert.True(t, ingesterRes.Requests[corev1.ResourceCPU].Equal(resource.MustParse("1500m")))
		assert.True(t, ingesterRes.Requests[corev1.ResourceMemory].Equal(resource.MustParse("8Gi")))

		distributorRes := ResourcesForComponent(v1alpha1.SizeExtraSmall, DistributorComponentName)
		assert.True(t, distributorRes.Requests[corev1.ResourceCPU].Equal(resource.MustParse("200m")))
		assert.True(t, distributorRes.Requests[corev1.ResourceMemory].Equal(resource.MustParse("128Mi")))
	})

	// Test specific resource values for small size (based on perf tests)
	t.Run("small size has correct values", func(t *testing.T) {
		ingesterRes := ResourcesForComponent(v1alpha1.SizeSmall, IngesterComponentName)
		assert.True(t, ingesterRes.Requests[corev1.ResourceCPU].Equal(resource.MustParse("2000m")))
		assert.True(t, ingesterRes.Requests[corev1.ResourceMemory].Equal(resource.MustParse("10Gi")))
	})

	// Test specific resource values for medium size (based on perf tests)
	t.Run("medium size has correct values", func(t *testing.T) {
		ingesterRes := ResourcesForComponent(v1alpha1.SizeMedium, IngesterComponentName)
		assert.True(t, ingesterRes.Requests[corev1.ResourceCPU].Equal(resource.MustParse("8000m")))
		assert.True(t, ingesterRes.Requests[corev1.ResourceMemory].Equal(resource.MustParse("16Gi")))

		gatewayRes := ResourcesForComponent(v1alpha1.SizeMedium, GatewayComponentName)
		assert.True(t, gatewayRes.Requests[corev1.ResourceCPU].Equal(resource.MustParse("4000m")))
		assert.True(t, gatewayRes.Requests[corev1.ResourceMemory].Equal(resource.MustParse("192Mi")))
	})

	// Test JaegerFrontend has same resources as Gateway
	t.Run("JaegerFrontend has same resources as Gateway", func(t *testing.T) {
		sizes := []v1alpha1.TempoStackSize{
			v1alpha1.SizePico,
			v1alpha1.SizeExtraSmall,
			v1alpha1.SizeSmall,
			v1alpha1.SizeMedium,
		}

		for _, size := range sizes {
			gatewayRes := ResourcesForComponent(size, GatewayComponentName)
			jaegerRes := ResourcesForComponent(size, JaegerFrontendComponentName)

			assert.Equal(t, gatewayRes.Requests[corev1.ResourceCPU], jaegerRes.Requests[corev1.ResourceCPU],
				"size %s: JaegerFrontend should have same CPU as Gateway", size)
			assert.Equal(t, gatewayRes.Requests[corev1.ResourceMemory], jaegerRes.Requests[corev1.ResourceMemory],
				"size %s: JaegerFrontend should have same Memory as Gateway", size)
		}
	})
}

func TestResourcesIncreaseFromSmallToMedium(t *testing.T) {
	// Verify that resources increase from small to medium (both from perf tests)
	// Note: pico and extra-small are from different sources so may not be monotonic
	components := []string{
		IngesterComponentName,
		QuerierComponentName,
		CompactorComponentName,
	}

	for _, comp := range components {
		t.Run("resources increase for "+comp, func(t *testing.T) {
			smallRes := ResourcesForComponent(v1alpha1.SizeSmall, comp)
			mediumRes := ResourcesForComponent(v1alpha1.SizeMedium, comp)

			smallCPU := smallRes.Requests.Cpu().MilliValue()
			mediumCPU := mediumRes.Requests.Cpu().MilliValue()
			smallMem := smallRes.Requests.Memory().Value()
			mediumMem := mediumRes.Requests.Memory().Value()

			assert.GreaterOrEqual(t, mediumCPU, smallCPU, "Medium CPU for %s should be >= Small", comp)
			assert.GreaterOrEqual(t, mediumMem, smallMem, "Medium Memory for %s should be >= Small", comp)
		})
	}
}

func TestGetRateLimitProfile(t *testing.T) {
	tests := []struct {
		name                        string
		size                        v1alpha1.TempoStackSize
		wantNil                     bool
		wantIngestionRateLimitBytes *int
		wantIngestionBurstSizeBytes *int
		wantMaxTracesPerUserNil     bool
		wantMaxTracesPerUser        *int
	}{
		{
			name:    "empty size returns nil",
			size:    "",
			wantNil: true,
		},
		{
			name:    "demo size returns nil (no rate limits)",
			size:    v1alpha1.SizeDemo,
			wantNil: true,
		},
		{
			name:                        "pico size returns profile with no MaxTracesPerUser",
			size:                        v1alpha1.SizePico,
			wantNil:                     false,
			wantIngestionRateLimitBytes: intToPointer(600_000),
			wantIngestionBurstSizeBytes: intToPointer(1_200_000),
			wantMaxTracesPerUserNil:     true,
		},
		{
			name:                        "extra-small size returns profile with no MaxTracesPerUser",
			size:                        v1alpha1.SizeExtraSmall,
			wantNil:                     false,
			wantIngestionRateLimitBytes: intToPointer(1_200_000),
			wantIngestionBurstSizeBytes: intToPointer(2_400_000),
			wantMaxTracesPerUserNil:     true,
		},
		{
			name:                        "small size returns profile with MaxTracesPerUser",
			size:                        v1alpha1.SizeSmall,
			wantNil:                     false,
			wantIngestionRateLimitBytes: intToPointer(5_800_000),
			wantIngestionBurstSizeBytes: intToPointer(11_600_000),
			wantMaxTracesPerUserNil:     false,
			wantMaxTracesPerUser:        intToPointer(17_000),
		},
		{
			name:                        "medium size returns profile with MaxTracesPerUser",
			size:                        v1alpha1.SizeMedium,
			wantNil:                     false,
			wantIngestionRateLimitBytes: intToPointer(23_000_000),
			wantIngestionBurstSizeBytes: intToPointer(46_000_000),
			wantMaxTracesPerUserNil:     false,
			wantMaxTracesPerUser:        intToPointer(54_000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := GetRateLimitProfile(tt.size)
			if tt.wantNil {
				assert.Nil(t, profile)
				return
			}
			require.NotNil(t, profile)
			assert.Equal(t, tt.wantIngestionRateLimitBytes, profile.IngestionRateLimitBytes)
			assert.Equal(t, tt.wantIngestionBurstSizeBytes, profile.IngestionBurstSizeBytes)
			if tt.wantMaxTracesPerUserNil {
				assert.Nil(t, profile.MaxTracesPerUser)
			} else {
				assert.Equal(t, tt.wantMaxTracesPerUser, profile.MaxTracesPerUser)
			}
		})
	}
}

func TestRateLimitsIncreaseWithSize(t *testing.T) {
	// Verify that rate limits increase from smaller to larger sizes
	sizes := []v1alpha1.TempoStackSize{
		v1alpha1.SizePico,
		v1alpha1.SizeExtraSmall,
		v1alpha1.SizeSmall,
		v1alpha1.SizeMedium,
	}

	for i := 1; i < len(sizes); i++ {
		t.Run("rate limits increase from "+string(sizes[i-1])+" to "+string(sizes[i]), func(t *testing.T) {
			smaller := GetRateLimitProfile(sizes[i-1])
			larger := GetRateLimitProfile(sizes[i])

			require.NotNil(t, smaller)
			require.NotNil(t, larger)

			assert.Greater(t, *larger.IngestionRateLimitBytes, *smaller.IngestionRateLimitBytes,
				"Ingestion rate limit should increase")
			assert.Greater(t, *larger.IngestionBurstSizeBytes, *smaller.IngestionBurstSizeBytes,
				"Ingestion burst size should increase")
		})
	}
}

func TestBurstSizeIsTwiceRateLimit(t *testing.T) {
	// Verify that burst size is 2x the rate limit for all sizes
	sizes := []v1alpha1.TempoStackSize{
		v1alpha1.SizePico,
		v1alpha1.SizeExtraSmall,
		v1alpha1.SizeSmall,
		v1alpha1.SizeMedium,
	}

	for _, size := range sizes {
		t.Run("burst size is 2x rate limit for "+string(size), func(t *testing.T) {
			profile := GetRateLimitProfile(size)
			require.NotNil(t, profile)

			expectedBurst := *profile.IngestionRateLimitBytes * 2
			assert.Equal(t, expectedBurst, *profile.IngestionBurstSizeBytes,
				"Burst size should be 2x the rate limit")
		})
	}
}

func intToPointer(i int) *int {
	return &i
}
