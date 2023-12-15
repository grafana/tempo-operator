package monolithic

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildServices creates all services for a monolithic deployment.
func BuildServices(opts Options) []client.Object {
	services := []client.Object{
		buildTempoApiService(opts),
		buildTempoIngestService(opts),
	}

	if opts.Tempo.Spec.JaegerUI != nil && opts.Tempo.Spec.JaegerUI.Enabled {
		services = append(services, buildJaegerUIService(opts))
	}

	return services
}

func buildTempoApiService(opts Options) *corev1.Service {
	labels := ComponentLabels("tempo", opts.Tempo.Name)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("api", opts.Tempo.Name),
			Namespace: opts.Tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
			},
			Selector: labels,
		},
	}
}

func buildTempoIngestService(opts Options) *corev1.Service {
	tempo := opts.Tempo
	labels := ComponentLabels("tempo", tempo.Name)

	ports := []corev1.ServicePort{}
	// TODO: point to gateway
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
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("ingest", opts.Tempo.Name),
			Namespace: opts.Tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: labels,
		},
	}
}

func buildJaegerUIService(opts Options) *corev1.Service {
	labels := ComponentLabels("tempo", opts.Tempo.Name)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("jaegerui", opts.Tempo.Name),
			Namespace: opts.Tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.JaegerGRPCQuery,
					Port:       manifestutils.PortJaegerGRPCQuery,
					TargetPort: intstr.FromString(manifestutils.JaegerGRPCQuery),
				},
				{
					Name:       manifestutils.JaegerUIPortName,
					Port:       manifestutils.PortJaegerUI,
					TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
				},
				{
					Name:       manifestutils.JaegerMetricsPortName,
					Port:       manifestutils.PortJaegerMetrics,
					TargetPort: intstr.FromString(manifestutils.JaegerMetricsPortName),
				},
			},
			Selector: labels,
		},
	}
}
