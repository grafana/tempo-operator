package monolithic

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildJaegerUIIngress creates a Ingress object for Jaeger UI.
func BuildJaegerUIIngress(opts Options) *networkingv1.Ingress {
	tempo := opts.Tempo
	labels := ComponentLabels(manifestutils.JaegerUIComponentName, tempo.Name)
	targetService, targetPort := jaegerUIServiceAndPort(tempo)

	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.JaegerUIComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: tempo.Spec.JaegerUI.Ingress.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: tempo.Spec.JaegerUI.Ingress.IngressClassName,
		},
	}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: targetService,
			Port: networkingv1.ServiceBackendPort{
				Name: targetPort,
			},
		},
	}

	if tempo.Spec.JaegerUI.Ingress.Host == "" {
		ingress.Spec.DefaultBackend = &backend
	} else {
		ingress.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: tempo.Spec.JaegerUI.Ingress.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: ptr.To(networkingv1.PathTypePrefix),
								Backend:  backend,
							},
						},
					},
				},
			},
		}
	}

	return ingress
}

const timeoutRouteAnnotation = "haproxy.router.openshift.io/timeout"

// BuildJaegerUIRoute creates a Route object for Jaeger UI.
func BuildJaegerUIRoute(opts Options) (*routev1.Route, error) {
	tempo := opts.Tempo
	labels := ComponentLabels(manifestutils.JaegerUIComponentName, tempo.Name)
	targetService, targetPort := jaegerUIServiceAndPort(tempo)

	var tlsCfg *routev1.TLSConfig
	switch tempo.Spec.JaegerUI.Route.Termination {
	case v1alpha1.TLSRouteTerminationTypeInsecure:
		// NOTE: insecure, no tls cfg.
	case v1alpha1.TLSRouteTerminationTypeEdge:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge}
	case v1alpha1.TLSRouteTerminationTypePassthrough:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
	case v1alpha1.TLSRouteTerminationTypeReencrypt:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	default: // NOTE: if unsupported, end here.
		return nil, fmt.Errorf("unsupported tls termination '%s' specified for route", tempo.Spec.JaegerUI.Route.Termination)
	}

	annotations := opts.Tempo.Spec.JaegerUI.Route.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[timeoutRouteAnnotation] == "" {
		annotations[timeoutRouteAnnotation] = fmt.Sprintf("%ds", int(tempo.Spec.Timeout.Duration.Seconds()))
	}

	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.JaegerUIComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: routev1.RouteSpec{
			Host: tempo.Spec.JaegerUI.Route.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: targetService,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(targetPort),
			},
			TLS: tlsCfg,
		},
	}, nil
}

func jaegerUIServiceAndPort(tempo v1alpha1.TempoMonolithic) (string, string) {
	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		return naming.Name(manifestutils.GatewayComponentName, tempo.Name), manifestutils.GatewayHttpPortName
	} else {
		return naming.Name(manifestutils.JaegerUIComponentName, tempo.Name), manifestutils.JaegerUIPortName
	}
}
