package manifestutils_test

import (
	"testing"
	"time"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests"
	. "github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

// containerComponentMap maps container names to their component names for resource lookup.
// This is the mapping between what's in the Deployment containers and what's in SizeProfile.
var containerComponentMap = map[string]string{
	// Core components - all use "tempo" container name
	"distributor": DistributorComponentName,
	"ingester":    IngesterComponentName,
	"querier":     QuerierComponentName,
	"compactor":   CompactorComponentName,
	"tempo":       QueryFrontendComponentName, // query-frontend main container

	// Jaeger UI sidecars
	"jaeger-query": JaegerFrontendComponentName,
	"tempo-query":  QueryFrontendComponentName, // shares resources with query-frontend

	// Auth sidecars
	"oauth-proxy": QueryFrontendOauthProxyComponentName,

	// Gateway containers
	"tempo-gateway":     GatewayComponentName,
	"tempo-gateway-opa": GatewayOpaComponentName,
}

// getExpectedResources returns the expected ComponentResources from the profile for a given component.
func getExpectedResources(profile *SizeProfile, component string) *ComponentResources {
	if profile == nil {
		return nil
	}
	switch component {
	case IngesterComponentName:
		return &profile.Ingester
	case CompactorComponentName:
		return &profile.Compactor
	case QuerierComponentName:
		return &profile.Querier
	case QueryFrontendComponentName:
		return &profile.QueryFrontend
	case DistributorComponentName:
		return &profile.Distributor
	case GatewayComponentName:
		return &profile.Gateway
	case JaegerFrontendComponentName:
		return &profile.JaegerFrontend
	case QueryFrontendOauthProxyComponentName:
		return &profile.OauthProxy
	case GatewayOpaComponentName:
		return &profile.GatewayOpa
	default:
		return nil
	}
}

// sizesWithResources returns all sizes that have resource profiles (excludes demo).
func sizesWithResources() []v1alpha1.TempoStackSize {
	return []v1alpha1.TempoStackSize{
		v1alpha1.SizePico,
		v1alpha1.SizeExtraSmall,
		v1alpha1.SizeSmall,
		v1alpha1.SizeMedium,
	}
}

// --------------------------------------------------------------------------
// Unit tests for SizeProfile completeness and ResourcesForComponent function
// --------------------------------------------------------------------------

// TestAllSizesHaveAllComponentsDefined ensures every t-shirt size (except demo)
// has resource definitions for all components including sidecars.
func TestAllSizesHaveAllComponentsDefined(t *testing.T) {
	allComponents := []string{
		IngesterComponentName,
		CompactorComponentName,
		QuerierComponentName,
		QueryFrontendComponentName,
		DistributorComponentName,
		GatewayComponentName,
		JaegerFrontendComponentName,
		QueryFrontendOauthProxyComponentName,
		GatewayOpaComponentName,
	}

	for _, size := range sizesWithResources() {
		t.Run(string(size), func(t *testing.T) {
			profile := GetSizeProfile(size)
			require.NotNil(t, profile, "size %s should have a profile", size)

			for _, comp := range allComponents {
				t.Run(comp, func(t *testing.T) {
					res := ResourcesForComponent(size, comp)
					require.NotNil(t, res.Requests, "size %s should have requests for %s", size, comp)

					// Verify CPU is defined and positive
					cpu := res.Requests.Cpu()
					require.NotNil(t, cpu, "size %s should have CPU for %s", size, comp)
					assert.True(t, cpu.MilliValue() > 0, "size %s should have positive CPU for %s", size, comp)

					// Verify Memory is defined and positive
					mem := res.Requests.Memory()
					require.NotNil(t, mem, "size %s should have Memory for %s", size, comp)
					assert.True(t, mem.Value() > 0, "size %s should have positive Memory for %s", size, comp)
				})
			}
		})
	}
}

// TestDemoSizeReturnsEmptyResourcesForAllComponents verifies that demo size
// returns empty resources for all components (design decision).
func TestDemoSizeReturnsEmptyResourcesForAllComponents(t *testing.T) {
	allComponents := []string{
		IngesterComponentName,
		CompactorComponentName,
		QuerierComponentName,
		QueryFrontendComponentName,
		DistributorComponentName,
		GatewayComponentName,
		JaegerFrontendComponentName,
		QueryFrontendOauthProxyComponentName,
		GatewayOpaComponentName,
	}

	for _, comp := range allComponents {
		t.Run(comp, func(t *testing.T) {
			res := ResourcesForComponent(v1alpha1.SizeDemo, comp)
			assert.Nil(t, res.Requests, "demo size should have nil Requests for %s", comp)
			assert.Nil(t, res.Limits, "demo size should have nil Limits for %s", comp)
		})
	}
}

// TestUnknownComponentReturnsEmptyResources verifies that unknown component
// names return empty resources rather than panicking.
func TestUnknownComponentReturnsEmptyResources(t *testing.T) {
	unknownComponents := []string{
		"unknown",
		"",
		"not-a-component",
		"tempo-something-else",
	}

	for _, size := range sizesWithResources() {
		t.Run(string(size), func(t *testing.T) {
			for _, comp := range unknownComponents {
				t.Run(comp, func(t *testing.T) {
					res := ResourcesForComponent(size, comp)
					assert.Nil(t, res.Requests, "unknown component should have nil Requests")
					assert.Nil(t, res.Limits, "unknown component should have nil Limits")
				})
			}
		})
	}
}

// TestEmptySizeReturnsEmptyResources verifies that when no size is specified,
// empty resources are returned (allowing fallback to percentage-based).
func TestEmptySizeReturnsEmptyResources(t *testing.T) {
	allComponents := []string{
		IngesterComponentName,
		CompactorComponentName,
		QuerierComponentName,
		QueryFrontendComponentName,
		DistributorComponentName,
		GatewayComponentName,
		JaegerFrontendComponentName,
		QueryFrontendOauthProxyComponentName,
		GatewayOpaComponentName,
	}

	for _, comp := range allComponents {
		t.Run(comp, func(t *testing.T) {
			res := ResourcesForComponent("", comp)
			assert.Nil(t, res.Requests, "empty size should have nil Requests for %s", comp)
			assert.Nil(t, res.Limits, "empty size should have nil Limits for %s", comp)
		})
	}
}

// --------------------------------------------------------------------------
// Integration tests: Build actual manifests and verify container resources
// --------------------------------------------------------------------------

// testCase defines a test configuration for integration tests.
type testCase struct {
	name               string
	setupTempoStack    func(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params)
	expectedContainers map[string][]string // deployment name suffix -> container names
}

// testCases defines all test configurations.
var testCases = []testCase{
	{
		name:            "basic",
		setupTempoStack: setupBasicTempoStack,
		expectedContainers: map[string][]string{
			"distributor":    {"tempo"},
			"ingester":       {"tempo"},
			"querier":        {"tempo"},
			"compactor":      {"tempo"},
			"query-frontend": {"tempo"},
		},
	},
	{
		name:            "with-jaeger-ui",
		setupTempoStack: setupJaegerUITempoStack,
		expectedContainers: map[string][]string{
			"distributor":    {"tempo"},
			"ingester":       {"tempo"},
			"querier":        {"tempo"},
			"compactor":      {"tempo"},
			"query-frontend": {"tempo", "jaeger-query", "tempo-query"},
		},
	},
	{
		name:            "single-tenant-auth",
		setupTempoStack: setupSingleTenantAuthTempoStack,
		expectedContainers: map[string][]string{
			"distributor":    {"tempo"},
			"ingester":       {"tempo"},
			"querier":        {"tempo"},
			"compactor":      {"tempo"},
			"query-frontend": {"tempo", "jaeger-query", "tempo-query", "oauth-proxy"},
		},
	},
	{
		name:            "multi-tenant-static",
		setupTempoStack: setupMultiTenantStaticTempoStack,
		expectedContainers: map[string][]string{
			"distributor":    {"tempo"},
			"ingester":       {"tempo"},
			"querier":        {"tempo"},
			"compactor":      {"tempo"},
			"query-frontend": {"tempo"},
			"gateway":        {"tempo-gateway"},
		},
	},
	{
		name:            "multi-tenant-openshift",
		setupTempoStack: setupMultiTenantOpenShiftTempoStack,
		expectedContainers: map[string][]string{
			"distributor":    {"tempo"},
			"ingester":       {"tempo"},
			"querier":        {"tempo"},
			"compactor":      {"tempo"},
			"query-frontend": {"tempo"},
			"gateway":        {"tempo-gateway", "tempo-gateway-opa"},
		},
	},
}

// TestTempoStackContainerResourcesWithSize runs integration tests that build
// actual TempoStack manifests and verify all containers have correct resources.
func TestTempoStackContainerResourcesWithSize(t *testing.T) {
	sizes := sizesWithResources()

	for _, tc := range testCases {
		for _, size := range sizes {
			t.Run(tc.name+"/"+string(size), func(t *testing.T) {
				tempo, params := tc.setupTempoStack(size)
				_ = tempo // tempo is embedded in params

				objects, err := manifests.BuildAll(params)
				require.NoError(t, err, "BuildAll should succeed")

				verifyContainerResources(t, size, objects, tc.expectedContainers, params.Tempo.Name)
			})
		}
	}
}

// verifyContainerResources checks that all expected containers have the correct resources.
// It handles both Deployments and StatefulSets (ingester is a StatefulSet).
func verifyContainerResources(t *testing.T, size v1alpha1.TempoStackSize,
	objects []client.Object, expectedContainers map[string][]string, tempoName string) {
	t.Helper()

	profile := GetSizeProfile(size)
	require.NotNil(t, profile, "size %s should have a profile", size)

	// Track which workloads we've verified
	verifiedWorkloads := make(map[string]bool)

	for _, obj := range objects {
		var name string
		var containers []corev1.Container

		// Handle both Deployments and StatefulSets
		switch w := obj.(type) {
		case *appsv1.Deployment:
			name = w.Name
			containers = w.Spec.Template.Spec.Containers
		case *appsv1.StatefulSet:
			name = w.Name
			containers = w.Spec.Template.Spec.Containers
		default:
			continue
		}

		// Extract workload suffix from name (e.g., "tempo-test-distributor" -> "distributor")
		suffix := extractDeploymentSuffix(name, tempoName)

		expectedNames, exists := expectedContainers[suffix]
		if !exists {
			continue
		}

		verifiedWorkloads[suffix] = true

		// Verify all expected containers exist and have correct resources
		for _, expectedContainerName := range expectedNames {
			container := findContainer(containers, expectedContainerName)
			require.NotNil(t, container,
				"workload %s should have container %s", name, expectedContainerName)

			componentName := getComponentNameForContainer(expectedContainerName, suffix)
			expected := getExpectedResources(profile, componentName)
			require.NotNil(t, expected,
				"profile should have resources for component %s (container %s)", componentName, expectedContainerName)

			// Verify resources match
			require.NotNil(t, container.Resources.Requests,
				"container %s in %s should have Requests", expectedContainerName, name)

			assert.True(t, container.Resources.Requests.Cpu().Equal(expected.CPU),
				"container %s in %s: CPU mismatch - got %s, want %s",
				expectedContainerName, name,
				container.Resources.Requests.Cpu().String(), expected.CPU.String())

			assert.True(t, container.Resources.Requests.Memory().Equal(expected.Memory),
				"container %s in %s: Memory mismatch - got %s, want %s",
				expectedContainerName, name,
				container.Resources.Requests.Memory().String(), expected.Memory.String())

			// Design decision: only Requests, no Limits
			assert.Nil(t, container.Resources.Limits,
				"container %s in %s should have nil Limits", expectedContainerName, name)
		}
	}

	// Verify all expected workloads were found
	for suffix := range expectedContainers {
		assert.True(t, verifiedWorkloads[suffix],
			"expected workload with suffix %s was not found", suffix)
	}
}

// extractDeploymentSuffix extracts the component suffix from a deployment name.
// For example, "tempo-test-distributor" with tempoName "test" returns "distributor".
func extractDeploymentSuffix(depName, tempoName string) string {
	prefix := "tempo-" + tempoName + "-"
	if len(depName) > len(prefix) && depName[:len(prefix)] == prefix {
		return depName[len(prefix):]
	}
	return ""
}

// findContainer finds a container by name in the list.
func findContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

// getComponentNameForContainer maps container name to component name for resource lookup.
func getComponentNameForContainer(containerName, deploymentSuffix string) string {
	// Special case: the main "tempo" container's component depends on the deployment
	if containerName == "tempo" {
		switch deploymentSuffix {
		case "distributor":
			return DistributorComponentName
		case "ingester":
			return IngesterComponentName
		case "querier":
			return QuerierComponentName
		case "compactor":
			return CompactorComponentName
		case "query-frontend":
			return QueryFrontendComponentName
		}
	}
	// For other containers, use the mapping
	if comp, ok := containerComponentMap[containerName]; ok {
		return comp
	}
	return ""
}

// --------------------------------------------------------------------------
// Setup functions for different TempoStack configurations
// --------------------------------------------------------------------------

// baseStorageParams returns minimal storage params needed for manifest building.
func baseStorageParams() StorageParams {
	return StorageParams{
		S3: &S3{
			Endpoint: "http://minio:9000",
			Bucket:   "tempo",
		},
	}
}

// baseCtrlConfig returns a minimal controller config.
func baseCtrlConfig() configv1alpha1.ProjectConfig {
	return configv1alpha1.ProjectConfig{
		DefaultImages: configv1alpha1.ImagesSpec{
			Tempo:           "grafana/tempo:latest",
			TempoQuery:      "grafana/tempo-query:latest",
			JaegerQuery:     "jaegertracing/jaeger-query:latest",
			TempoGateway:    "quay.io/observatorium/api:latest",
			TempoGatewayOpa: "quay.io/observatorium/opa-openshift:latest",
			OauthProxy:      "quay.io/openshift/origin-oauth-proxy:latest",
		},
	}
}

// baseTLSProfile returns a minimal TLS profile for testing.
func baseTLSProfile() tlsprofile.TLSProfileOptions {
	return tlsprofile.TLSProfileOptions{
		MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
	}
}

// setupBasicTempoStack creates a basic TempoStack configuration (core components only).
func setupBasicTempoStack(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.TempoStackSpec{
			Size:           size,
			ServiceAccount: "tempo-test",
			StorageSize:    resource.MustParse("10Gi"), // Required for ingester StatefulSet
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "storage-secret",
					Type: v1alpha1.ObjectStorageSecretS3,
				},
			},
		},
	}

	params := Params{
		Tempo:         tempo,
		StorageParams: baseStorageParams(),
		CtrlConfig:    baseCtrlConfig(),
		TLSProfile:    baseTLSProfile(),
	}

	return tempo, params
}

// setupJaegerUITempoStack creates a TempoStack with Jaeger UI enabled.
func setupJaegerUITempoStack(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params) {
	tempo, params := setupBasicTempoStack(size)

	// Note: ServicesQueryDuration is required when JaegerQuery is enabled
	// (the webhook handles this in production, but we must set it manually in tests)
	defaultServicesDuration := metav1.Duration{Duration: 72 * time.Hour}
	tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled = true
	tempo.Spec.Template.QueryFrontend.JaegerQuery.ServicesQueryDuration = &defaultServicesDuration

	params.Tempo = tempo
	return tempo, params
}

// setupSingleTenantAuthTempoStack creates a TempoStack with Jaeger UI and OAuth proxy.
func setupSingleTenantAuthTempoStack(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params) {
	tempo, params := setupJaegerUITempoStack(size)

	tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type = v1alpha1.IngressTypeRoute
	tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeEdge
	tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication = &v1alpha1.JaegerQueryAuthenticationSpec{
		Enabled: true,
	}

	// OAuth proxy requires OpenShift feature gates
	params.CtrlConfig.Gates.OpenShift.OpenShiftRoute = true
	params.Tempo = tempo

	return tempo, params
}

// setupMultiTenantStaticTempoStack creates a TempoStack with gateway (multi-tenant static mode).
func setupMultiTenantStaticTempoStack(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params) {
	tempo, params := setupBasicTempoStack(size)

	tempo.Spec.Template.Gateway.Enabled = true
	tempo.Spec.Tenants = &v1alpha1.TenantsSpec{
		Mode: v1alpha1.ModeStatic,
		Authentication: []v1alpha1.AuthenticationSpec{
			{
				TenantName: "test-tenant",
				TenantID:   "test-tenant",
				OIDC: &v1alpha1.OIDCSpec{
					Secret: &v1alpha1.TenantSecretSpec{
						Name: "test-oidc-secret",
					},
					IssuerURL: "https://issuer.example.com",
				},
			},
		},
		Authorization: &v1alpha1.AuthorizationSpec{
			Roles: []v1alpha1.RoleSpec{
				{
					Name:        "read-write",
					Resources:   []string{"traces"},
					Tenants:     []string{"test-tenant"},
					Permissions: []v1alpha1.PermissionType{v1alpha1.Read, v1alpha1.Write},
				},
			},
			RoleBindings: []v1alpha1.RoleBindingsSpec{
				{
					Name:  "test-tenant-binding",
					Roles: []string{"read-write"},
					Subjects: []v1alpha1.Subject{
						{
							Name: "user@example.com",
							Kind: v1alpha1.User,
						},
					},
				},
			},
		},
	}

	params.Tempo = tempo
	return tempo, params
}

// setupMultiTenantOpenShiftTempoStack creates a TempoStack with gateway + OPA (OpenShift mode).
func setupMultiTenantOpenShiftTempoStack(size v1alpha1.TempoStackSize) (v1alpha1.TempoStack, Params) {
	tempo, params := setupBasicTempoStack(size)

	tempo.Spec.Template.Gateway.Enabled = true
	tempo.Spec.Tenants = &v1alpha1.TenantsSpec{
		Mode: v1alpha1.ModeOpenShift,
		Authentication: []v1alpha1.AuthenticationSpec{
			{
				TenantName: "dev",
				TenantID:   "dev-tenant-id",
			},
		},
	}

	// OpenShift mode requires ServingCertsService for the OPA container
	params.CtrlConfig.Gates.OpenShift.ServingCertsService = true
	params.CtrlConfig.Gates.OpenShift.OpenShiftRoute = true
	params.GatewayTenantsData = []*GatewayTenantsData{
		{
			TenantName:            "dev",
			OpenShiftCookieSecret: "test-cookie-secret",
		},
	}

	params.Tempo = tempo
	return tempo, params
}
