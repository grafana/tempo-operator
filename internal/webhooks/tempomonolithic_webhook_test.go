package webhooks

import (
	"context"
	"testing"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestMonolithicValidate(t *testing.T) {
	ctx := admission.NewContextWithRequest(context.Background(), admission.Request{})

	tests := []struct {
		name       string
		ctrlConfig configv1alpha1.ProjectConfig
		tempo      v1alpha1.TempoMonolithic
		warnings   admission.Warnings
		errors     field.ErrorList
	}{
		{
			name: "valid instance",
			tempo: v1alpha1.TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample",
					Namespace: "default",
				},
				Spec: v1alpha1.TempoMonolithicSpec{},
			},
			warnings: admission.Warnings{},
			errors:   field.ErrorList{},
		},

		// Jaeger UI
		{
			name: "JaegerUI ingress enabled but Jaeger UI disabled",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
						Ingress: &v1alpha1.MonolithicJaegerUIIngressSpec{
							Enabled: true,
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "jaegerui", "ingress", "enabled"),
				true,
				"Jaeger UI must be enabled to create an ingress for Jaeger UI",
			)},
		},
		{
			name: "JaegerUI route enabled but Jaeger UI disabled",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
						Route: &v1alpha1.MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "jaegerui", "route", "enabled"),
				true,
				"Jaeger UI must be enabled to create a route for Jaeger UI",
			)},
		},
		{
			name: "JaegerUI route enabled but openShiftRoute feature gate not set",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: false,
					},
				},
			},
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
						Enabled: true,
						Route: &v1alpha1.MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "jaegerui", "route", "enabled"),
				true,
				"the openshiftRoute feature gate must be enabled to create a route for Jaeger UI",
			)},
		},
		{
			name: "JaegerUI route enabled and openshiftRoute feature gate set",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
						Enabled: true,
						Route: &v1alpha1.MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors:   field.ErrorList{},
		},

		// multitenancy
		{
			name: "multi-tenancy enabled, OpenShift mode, authorization set",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
						Enabled: true,
						TenantsSpec: v1alpha1.TenantsSpec{
							Mode: v1alpha1.ModeOpenShift,
							Authentication: []v1alpha1.AuthenticationSpec{{
								TenantName: "abc",
							}},
							Authorization: &v1alpha1.AuthorizationSpec{},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "multitenancy", "enabled"),
				true,
				"spec.tenants.authorization should not be defined in openshift mode",
			)},
		},
		{
			name: "RBAC and jaeger UI enabled",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Query: &v1alpha1.MonolithicQuerySpec{
						RBAC: v1alpha1.RBACSpec{
							Enabled: true,
						},
					},
					JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
						Enabled: true,
					},
					Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
						Enabled: true,
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "rbac", "enabled"),
				true,
				"cannot enable RBAC and jaeger query at the same time. The Jaeger UI does not support query RBAC",
			)},
		},

		{
			name: "RBAC and multitenancy disabled",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Query: &v1alpha1.MonolithicQuerySpec{
						RBAC: v1alpha1.RBACSpec{
							Enabled: true,
						},
					},
					Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
						Enabled: false,
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "rbac", "enabled"),
				true,
				"RBAC can only be enabled if multi-tenancy is enabled",
			)},
		},
		{
			name: "RBAC and multitenancy nil",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Query: &v1alpha1.MonolithicQuerySpec{
						RBAC: v1alpha1.RBACSpec{
							Enabled: true,
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "rbac", "enabled"),
				true,
				"RBAC can only be enabled if multi-tenancy is enabled",
			)},
		},

		// observability
		{
			name: "serviceMonitors enabled but prometheusOperator feature gate not set",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Metrics: &v1alpha1.MonolithicObservabilityMetricsSpec{
							ServiceMonitors: &v1alpha1.MonolithicObservabilityMetricsServiceMonitorsSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "observability", "metrics", "serviceMonitors", "enabled"),
				true,
				"the prometheusOperator feature gate must be enabled to create ServiceMonitors for Tempo components",
			)},
		},
		{
			name: "prometheusRules enabled but prometheusOperator feature gate not set",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Metrics: &v1alpha1.MonolithicObservabilityMetricsSpec{
							PrometheusRules: &v1alpha1.MonolithicObservabilityMetricsPrometheusRulesSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "observability", "metrics", "prometheusRules", "enabled"),
				true,
				"the prometheusOperator feature gate must be enabled to create PrometheusRules for Tempo components",
			)},
		},
		{
			name: "prometheusRules enabled but serviceMonitors disabled",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Metrics: &v1alpha1.MonolithicObservabilityMetricsSpec{
							PrometheusRules: &v1alpha1.MonolithicObservabilityMetricsPrometheusRulesSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "observability", "metrics", "prometheusRules", "enabled"),
				true,
				"serviceMonitors must be enabled to create PrometheusRules (the rules alert based on collected metrics)",
			)},
		},
		{
			name: "dataSource enabled but grafanaOperator feature gate not set",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Grafana: &v1alpha1.MonolithicObservabilityGrafanaSpec{
							DataSource: &v1alpha1.MonolithicObservabilityGrafanaDataSourceSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "observability", "grafana", "dataSource", "enabled"),
				true,
				"the grafanaOperator feature gate must be enabled to create a data source for Tempo",
			)},
		},
		{
			name: "dataSource enabled, grafanaOperator feature gate set, and gateway enabled",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					GrafanaOperator: true,
				},
			},
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Grafana: &v1alpha1.MonolithicObservabilityGrafanaSpec{
							DataSource: &v1alpha1.MonolithicObservabilityGrafanaDataSourceSpec{
								Enabled: true,
							},
						},
					},
					Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
						Enabled: true,
						TenantsSpec: v1alpha1.TenantsSpec{
							Authentication: []v1alpha1.AuthenticationSpec{{
								TenantName: "",
							}},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "observability", "grafana", "dataSource", "enabled"),
				true,
				"creating a data source for Tempo is not support if the gateway is enabled",
			)},
		},
		{
			name: "valid observability config",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
					GrafanaOperator:    true,
				},
			},
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Observability: &v1alpha1.MonolithicObservabilitySpec{
						Metrics: &v1alpha1.MonolithicObservabilityMetricsSpec{
							ServiceMonitors: &v1alpha1.MonolithicObservabilityMetricsServiceMonitorsSpec{
								Enabled: true,
							},
							PrometheusRules: &v1alpha1.MonolithicObservabilityMetricsPrometheusRulesSpec{
								Enabled: true,
							},
						},
						Grafana: &v1alpha1.MonolithicObservabilityGrafanaSpec{
							DataSource: &v1alpha1.MonolithicObservabilityGrafanaDataSourceSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors:   field.ErrorList{},
		},

		// service account
		{
			name: "custom service account set, however multi-tenancy is enabled with OpenShift mode",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					ServiceAccount: "abc",
					Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
						Enabled: true,
						TenantsSpec: v1alpha1.TenantsSpec{
							Mode: v1alpha1.ModeOpenShift,
							Authentication: []v1alpha1.AuthenticationSpec{{
								TenantName: "abc",
							}},
						},
					},
				},
			},
			warnings: admission.Warnings{},
			errors: field.ErrorList{field.Invalid(
				field.NewPath("spec", "serviceAccount"),
				"abc",
				"custom ServiceAccount is not supported if multi-tenancy with OpenShift mode is enabled",
			)},
		},

		// extra config
		{
			name: "extra config warning",
			tempo: v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					ExtraConfig: &v1alpha1.ExtraConfigSpec{
						Tempo: apiextensionsv1.JSON{Raw: []byte(`{}`)},
					},
				},
			},
			warnings: admission.Warnings{"overriding Tempo configuration could potentially break the deployment, use it carefully"},
			errors:   field.ErrorList{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &k8sFake{
				subjectAccessReview: &authorizationv1.SubjectAccessReview{
					Status: authorizationv1.SubjectAccessReviewStatus{
						Allowed: true,
					},
				},
			}
			v := &monolithicValidator{
				client:     client,
				ctrlConfig: test.ctrlConfig,
			}

			warnings, errors := v.validateTempoMonolithic(ctx, test.tempo)
			require.Equal(t, test.warnings, warnings)
			require.Equal(t, test.errors, errors)
		})
	}
}

func TestConflictTempoStackValidation(t *testing.T) {
	tempoMonolithic := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-obj",
			Namespace: "abc",
		},
	}

	tests := []struct {
		name     string
		input    runtime.Object
		expected field.ErrorList
		client   client.Client
	}{
		{
			name:  "should fail when monolithic exits",
			input: tempoMonolithic,
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("metadata").Child("name"),
					"test-obj",
					"Cannot create a TempoMonolithic with the same name as a TempoStack instance in the same namespace",
				)},
			client: &k8sFake{
				tempoStack: &v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-obj",
						Namespace: "abc",
					},
				},
			},
		},
		{
			name:   "should not fail",
			input:  tempoMonolithic,
			client: &k8sFake{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := &monolithicValidator{ctrlConfig: configv1alpha1.ProjectConfig{}, client: test.client}
			err := v.validateConflictWithTempoStack(ctx, *tempoMonolithic)
			assert.Equal(t, test.expected, err)
		})
	}
}
