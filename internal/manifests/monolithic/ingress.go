package monolithic

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildTempoIngress creates the ingress for a monolithic deployment.
func BuildTempoIngress(opts Options) ([]client.Object, error) {
	var manifests []client.Object
	tempo := opts.Tempo

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		if tempo.Spec.JaegerUI.Ingress != nil && tempo.Spec.JaegerUI.Ingress.Enabled {
			manifests = append(manifests, buildJaegerUIIngress(opts))
		}
		if tempo.Spec.JaegerUI.Route != nil && tempo.Spec.JaegerUI.Route.Enabled {
			route, err := buildJaegerUIRoute(opts)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, route)
		}
	}

	return manifests, nil
}

func buildJaegerUIIngress(opts Options) *networkingv1.Ingress {
	tempo := opts.Tempo
	labels := Labels(tempo.Name)
	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name("jaegerui", tempo.Name),
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
			Name: naming.Name("", tempo.Name),
			Port: networkingv1.ServiceBackendPort{
				Name: manifestutils.JaegerUIPortName,
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

func buildJaegerUIRoute(opts Options) (*routev1.Route, error) {
	tempo := opts.Tempo
	labels := Labels(tempo.Name)

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

	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name("jaegerui", tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: tempo.Spec.JaegerUI.Route.Annotations,
		},
		Spec: routev1.RouteSpec{
			Host: tempo.Spec.JaegerUI.Route.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name("", tempo.Name),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
			},
			TLS: tlsCfg,
		},
	}, nil
}
