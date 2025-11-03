package monolithic

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildServices creates all services for a monolithic deployment.
func BuildServices(opts Options) []client.Object {
	tempo := opts.Tempo

	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		return []client.Object{buildGatewayService(opts)}
	} else {
		objs := []client.Object{buildTempoService(opts)}
		if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
			objs = append(objs, buildJaegerUIService(opts))
		}
		return objs
	}
}

// buildTempoService creates the service for a monolithic deployment.
func buildTempoService(opts Options) *corev1.Service {
	tempo := opts.Tempo
	annotations := map[string]string{}

	if opts.useServiceCertsOnReceiver {
		annotations["service.beta.openshift.io/serving-cert-secret-name"] = naming.ServingCertName(manifestutils.TempoMonolithComponentName, tempo.Name)
	}

	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)
	ports := []corev1.ServicePort{{
		Name:       manifestutils.HttpPortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       manifestutils.PortHTTPServer,
		TargetPort: intstr.FromString(manifestutils.HttpPortName),
	}}

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil {
		if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
			ports = append(ports, corev1.ServicePort{
				Name:       manifestutils.OtlpGrpcPortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       manifestutils.PortOtlpGrpcServer,
				TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
			})
		}
		if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
			ports = append(ports, corev1.ServicePort{
				Name:       manifestutils.PortOtlpHttpName,
				Protocol:   corev1.ProtocolTCP,
				Port:       manifestutils.PortOtlpHttp,
				TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
			})
		}
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.TempoMonolithComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: labels,
		},
	}
}

// buildJaegerUIService creates the service for Jaeger UI.
func buildJaegerUIService(opts Options) *corev1.Service {
	tempo := opts.Tempo
	ports := []corev1.ServicePort{
		{
			Name:       manifestutils.JaegerGRPCQuery,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerGRPCQuery,
			TargetPort: intstr.FromString(manifestutils.JaegerGRPCQuery),
		},
		{
			Name:       manifestutils.JaegerUIPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerUI,
			TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
		},
		{
			Name:       manifestutils.JaegerMetricsPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerMetrics,
			TargetPort: intstr.FromString(manifestutils.JaegerMetricsPortName),
		},
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.JaegerUIComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    ComponentLabels(manifestutils.JaegerUIComponentName, tempo.Name),
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name),
		},
	}
}

// buildGatewayService creates the service for a monolithic deployment if the gateway is enabled.
// Unfortunately the gateway is not a "100% drop-in service", because the paths for Jaeger UI and Tempo API are different
// when accessed via gateway, and the gateway exposes a different set of metrics.
func buildGatewayService(opts Options) *corev1.Service {
	tempo := opts.Tempo
	annotations := map[string]string{}

	if opts.CtrlConfig.Gates.OpenShift.ServingCertsService {
		annotations["service.beta.openshift.io/serving-cert-secret-name"] = naming.ServingCertName(manifestutils.GatewayComponentName, tempo.Name)
	}

	ports := []corev1.ServicePort{
		{
			Name:     manifestutils.GatewayHttpPortName,
			Protocol: corev1.ProtocolTCP,
			Port:     manifestutils.GatewayPortHTTPServer,
			// proxies Tempo API and optionally Jaeger UI
			TargetPort: intstr.FromString(manifestutils.GatewayHttpPortName),
		},
		{
			Name:       manifestutils.GatewayInternalHttpPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.GatewayPortInternalHTTPServer,
			TargetPort: intstr.FromString(manifestutils.GatewayInternalHttpPortName),
		},
		{
			Name:       manifestutils.HttpPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortHTTPServer,
			TargetPort: intstr.FromString(manifestutils.HttpPortName),
		},
	}

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil &&
		tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
		ports = append(ports, corev1.ServicePort{
			Name:       manifestutils.OtlpGrpcPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortOtlpGrpcServer,
			TargetPort: intstr.FromString(manifestutils.GatewayGrpcPortName),
		})
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Namespace:   tempo.Namespace,
			Labels:      ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name),
		},
	}
}
