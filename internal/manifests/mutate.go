package manifests

import (
	"fmt"
	"reflect"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/google/go-cmp/cmp"
	grafanav1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ImmutableErr occurs if an immutable field should be changed.
type ImmutableErr struct {
	field    string
	existing interface{}
	desired  interface{}
}

func (m *ImmutableErr) Error() string {
	return fmt.Sprintf("update to immutable field %s is forbidden, diff: %s", m.field, cmp.Diff(m.existing, m.desired))
}

// MutateFuncFor returns a mutate function based on the
// existing resource's concrete type. It supports currently
// only the following types or else panics:
// - ConfigMap
// - Service
// - Deployment
// - StatefulSet
// - ServiceMonitor
// - Secret.
func MutateFuncFor(existing, desired client.Object) controllerutil.MutateFn {
	return func() error {
		existingAnnotations := existing.GetAnnotations()
		if err := mergeWithOverride(&existingAnnotations, desired.GetAnnotations()); err != nil {
			return err
		}
		existing.SetAnnotations(existingAnnotations)

		existingLabels := existing.GetLabels()
		if err := mergeWithOverride(&existingLabels, desired.GetLabels()); err != nil {
			return err
		}
		existing.SetLabels(existingLabels)

		if ownerRefs := desired.GetOwnerReferences(); len(ownerRefs) > 0 {
			existing.SetOwnerReferences(ownerRefs)
		}

		switch existing.(type) {
		case *corev1.ConfigMap:
			cm := existing.(*corev1.ConfigMap)
			wantCm := desired.(*corev1.ConfigMap)
			mutateConfigMap(cm, wantCm)

		case *corev1.Service:
			svc := existing.(*corev1.Service)
			wantSvc := desired.(*corev1.Service)
			return mutateService(svc, wantSvc)

		case *corev1.ServiceAccount:
			sa := existing.(*corev1.ServiceAccount)
			wantSa := desired.(*corev1.ServiceAccount)
			mutateServiceAccount(sa, wantSa)

		case *rbacv1.ClusterRole:
			cr := existing.(*rbacv1.ClusterRole)
			wantCr := desired.(*rbacv1.ClusterRole)
			mutateClusterRole(cr, wantCr)

		case *rbacv1.ClusterRoleBinding:
			crb := existing.(*rbacv1.ClusterRoleBinding)
			wantCrb := desired.(*rbacv1.ClusterRoleBinding)
			mutateClusterRoleBinding(crb, wantCrb)

		case *rbacv1.Role:
			r := existing.(*rbacv1.Role)
			wantR := desired.(*rbacv1.Role)
			mutateRole(r, wantR)

		case *rbacv1.RoleBinding:
			rb := existing.(*rbacv1.RoleBinding)
			wantRb := desired.(*rbacv1.RoleBinding)
			mutateRoleBinding(rb, wantRb)

		case *appsv1.Deployment:
			dpl := existing.(*appsv1.Deployment)
			wantDpl := desired.(*appsv1.Deployment)
			return mutateDeployment(dpl, wantDpl)

		case *appsv1.StatefulSet:
			sts := existing.(*appsv1.StatefulSet)
			wantSts := desired.(*appsv1.StatefulSet)
			return mutateStatefulSet(sts, wantSts)

		case *monitoringv1.ServiceMonitor:
			svcMonitor := existing.(*monitoringv1.ServiceMonitor)
			wantSvcMonitor := desired.(*monitoringv1.ServiceMonitor)
			mutateServiceMonitor(svcMonitor, wantSvcMonitor)

		case *networkingv1.Ingress:
			ing := existing.(*networkingv1.Ingress)
			wantIng := desired.(*networkingv1.Ingress)
			mutateIngress(ing, wantIng)

		case *routev1.Route:
			rt := existing.(*routev1.Route)
			wantRt := desired.(*routev1.Route)
			mutateRoute(rt, wantRt)

		case *monitoringv1.PrometheusRule:
			pr := existing.(*monitoringv1.PrometheusRule)
			wantPr := desired.(*monitoringv1.PrometheusRule)
			mutatePrometheusRule(pr, wantPr)

		case *corev1.Secret:
			pr := existing.(*corev1.Secret)
			wantPr := desired.(*corev1.Secret)
			mutateSecret(pr, wantPr)

		case *grafanav1.GrafanaDatasource:
			ds := existing.(*grafanav1.GrafanaDatasource)
			wantDs := desired.(*grafanav1.GrafanaDatasource)
			mutateGrafanaDatasource(ds, wantDs)

		default:
			t := reflect.TypeOf(existing).String()
			return kverrors.New("missing mutate implementation for resource type", "type", t)
		}
		return nil
	}
}

// Override non-empty dst attributes with non-empty src attributes values.
func mergeWithOverride(dst, src interface{}) error {
	err := mergo.Merge(dst, src, mergo.WithOverride)
	if err != nil {
		return kverrors.Wrap(err, "unable to mergeWithOverride", "dst", dst, "src", src)
	}
	return nil
}

func mutateSecret(existing, desired *corev1.Secret) {
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Data = desired.Data
}

func mutateConfigMap(existing, desired *corev1.ConfigMap) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels

	if _, ok := desired.Annotations["service.beta.openshift.io/inject-cabundle"]; ok {
		// The OpenShift service-ca operator will inject a service-ca.key into the ConfigMap.
		// Skip mutating this ConfigMap, otherwise the service-ca.key will be deleted by this operator
		// and re-added by the service-ca-operator in a loop.
		return
	}

	existing.BinaryData = desired.BinaryData
	existing.Data = desired.Data
}

func mutateServiceAccount(existing, desired *corev1.ServiceAccount) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
}

func mutateClusterRole(existing, desired *rbacv1.ClusterRole) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Rules = desired.Rules
}

func mutateClusterRoleBinding(existing, desired *rbacv1.ClusterRoleBinding) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Subjects = desired.Subjects
}

func mutateRole(existing, desired *rbacv1.Role) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Rules = desired.Rules
}

func mutateRoleBinding(existing, desired *rbacv1.RoleBinding) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Subjects = desired.Subjects
}

func mutateServiceMonitor(existing, desired *monitoringv1.ServiceMonitor) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateIngress(existing, desired *networkingv1.Ingress) {
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Spec.DefaultBackend = desired.Spec.DefaultBackend
	existing.Spec.Rules = desired.Spec.Rules
	existing.Spec.TLS = desired.Spec.TLS
}

func mutateRoute(existing, desired *routev1.Route) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutatePrometheusRule(existing, desired *monitoringv1.PrometheusRule) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateGrafanaDatasource(existing, desired *grafanav1.GrafanaDatasource) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateService(existing, desired *corev1.Service) error {
	existing.Spec.Ports = desired.Spec.Ports
	if err := mergeWithOverride(&existing.Spec.Selector, desired.Spec.Selector); err != nil {
		return err
	}
	return nil
}

func mutateDeployment(existing, desired *appsv1.Deployment) error {
	// Deployment selector is immutable so we set this value only if
	// a new object is going to be created
	if existing.CreationTimestamp.IsZero() {
		existing.Spec.Selector = desired.Spec.Selector
	}
	existing.Spec.Replicas = desired.Spec.Replicas
	if err := mergeWithOverride(&existing.Spec.Template, desired.Spec.Template); err != nil {
		return err
	}
	if err := mergeWithOverride(&existing.Spec.Strategy, desired.Spec.Strategy); err != nil {
		return err
	}
	return nil
}

func statefulSetVolumeClaimTemplatesChanged(existing, desired *appsv1.StatefulSet) bool {
	if len(desired.Spec.VolumeClaimTemplates) != len(existing.Spec.VolumeClaimTemplates) {
		return true
	}
	for i := range desired.Spec.VolumeClaimTemplates {
		if desired.Spec.VolumeClaimTemplates[i].Name != existing.Spec.VolumeClaimTemplates[i].Name ||
			!apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Annotations, existing.Spec.VolumeClaimTemplates[i].Annotations) ||
			!apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Spec, existing.Spec.VolumeClaimTemplates[i].Spec) {
			return true
		}
	}
	return false
}

func mutateStatefulSet(existing, desired *appsv1.StatefulSet) error {
	// list of mutable fields: https://github.com/kubernetes/kubernetes/blob/b1cf91b300a82bd05fdd7b115559e5b83680d768/pkg/apis/apps/validation/validation.go#L184
	if !existing.CreationTimestamp.IsZero() {
		if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
			return &ImmutableErr{".spec.selector", existing.Spec.Selector, desired.Spec.Selector}
		}
		if statefulSetVolumeClaimTemplatesChanged(existing, desired) {
			return &ImmutableErr{".spec.volumeClaimTemplates", existing.Spec.VolumeClaimTemplates, desired.Spec.VolumeClaimTemplates}
		}
	}

	existing.Spec.Selector = desired.Spec.Selector
	existing.Spec.PodManagementPolicy = desired.Spec.PodManagementPolicy
	existing.Spec.Replicas = desired.Spec.Replicas
	for i := range existing.Spec.VolumeClaimTemplates {
		existing.Spec.VolumeClaimTemplates[i].TypeMeta = desired.Spec.VolumeClaimTemplates[i].TypeMeta
		existing.Spec.VolumeClaimTemplates[i].ObjectMeta = desired.Spec.VolumeClaimTemplates[i].ObjectMeta
		existing.Spec.VolumeClaimTemplates[i].Spec = desired.Spec.VolumeClaimTemplates[i].Spec
	}
	if err := mergeWithOverride(&existing.Spec.Template, desired.Spec.Template); err != nil {
		return err
	}
	return nil
}
