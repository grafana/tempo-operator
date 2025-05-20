package distributor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildDistributor(t *testing.T) {

	tests := []struct {
		name                       string
		enableGateway              bool
		expectedObjects            int
		receiverTLS                v1alpha1.TLSSpec
		expectedServiceAnnotations map[string]string
		expectCABundleConfigMap    bool
		enableServingCertsService  bool
		instanceAddrType           v1alpha1.InstanceAddrType
		expectedContainerPorts     []corev1.ContainerPort
		expectedServicePorts       []corev1.ServicePort
		expectedResources          corev1.ResourceRequirements
		expectedVolumes            []corev1.Volume
		expectedVolumeMounts       []corev1.VolumeMount
		expectedContainerEnvVars   []corev1.EnvVar
	}{
		{
			name:            "Gateway disabled",
			enableGateway:   false,
			expectedObjects: 2,
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftHTTPName,
					ContainerPort: manifestutils.PortJaegerThriftHTTP,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftCompactName,
					ContainerPort: manifestutils.PortJaegerThriftCompact,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerThriftBinaryName,
					ContainerPort: manifestutils.PortJaegerThriftBinary,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerGrpcName,
					ContainerPort: manifestutils.PortJaegerGrpc,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortZipkinName,
					ContainerPort: manifestutils.PortZipkin,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.PortJaegerThriftHTTPName,
					Port:       manifestutils.PortJaegerThriftHTTP,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortJaegerThriftCompactName,
					Port:       manifestutils.PortJaegerThriftCompact,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerThriftBinaryName,
					Port:       manifestutils.PortJaegerThriftBinary,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerGrpcName,
					Port:       manifestutils.PortJaegerGrpc,
					TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortZipkinName,
					Port:       manifestutils.PortZipkin,
					TargetPort: intstr.FromString(manifestutils.PortZipkinName),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "206158425",
				},
			},
		},
		{
			name:            "Receiver TLS enable",
			expectedObjects: 2,
			enableGateway:   false,
			receiverTLS: v1alpha1.TLSSpec{
				Enabled: true,
				CA:      "ca-custom",
				Cert:    "cert-custom",
			},
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftHTTPName,
					ContainerPort: manifestutils.PortJaegerThriftHTTP,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftCompactName,
					ContainerPort: manifestutils.PortJaegerThriftCompact,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerThriftBinaryName,
					ContainerPort: manifestutils.PortJaegerThriftBinary,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerGrpcName,
					ContainerPort: manifestutils.PortJaegerGrpc,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortZipkinName,
					ContainerPort: manifestutils.PortZipkin,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.PortJaegerThriftHTTPName,
					Port:       manifestutils.PortJaegerThriftHTTP,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortJaegerThriftCompactName,
					Port:       manifestutils.PortJaegerThriftCompact,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerThriftBinaryName,
					Port:       manifestutils.PortJaegerThriftBinary,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerGrpcName,
					Port:       manifestutils.PortJaegerGrpc,
					TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortZipkinName,
					Port:       manifestutils.PortZipkin,
					TargetPort: intstr.FromString(manifestutils.PortZipkinName),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "ca-custom",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "ca-custom",
							},
						},
					},
				},
				{
					Name: "cert-custom",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "cert-custom",
						},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
				{
					Name:      "ca-custom",
					MountPath: manifestutils.ReceiverTLSCADir,
					ReadOnly:  true,
				},
				{
					Name:      "cert-custom",
					MountPath: manifestutils.ReceiverTLSCertDir,
					ReadOnly:  true,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "206158425",
				},
			},
		},
		{
			name:            "Receiver TLS enable with ServingCertsService feature enabled",
			expectedObjects: 2,
			enableGateway:   false,
			receiverTLS: v1alpha1.TLSSpec{
				Enabled: true,
				CA:      "ca-custom",
				Cert:    "cert-custom",
			},
			enableServingCertsService: true,
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftHTTPName,
					ContainerPort: manifestutils.PortJaegerThriftHTTP,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftCompactName,
					ContainerPort: manifestutils.PortJaegerThriftCompact,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerThriftBinaryName,
					ContainerPort: manifestutils.PortJaegerThriftBinary,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerGrpcName,
					ContainerPort: manifestutils.PortJaegerGrpc,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortZipkinName,
					ContainerPort: manifestutils.PortZipkin,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.PortJaegerThriftHTTPName,
					Port:       manifestutils.PortJaegerThriftHTTP,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortJaegerThriftCompactName,
					Port:       manifestutils.PortJaegerThriftCompact,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerThriftBinaryName,
					Port:       manifestutils.PortJaegerThriftBinary,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerGrpcName,
					Port:       manifestutils.PortJaegerGrpc,
					TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortZipkinName,
					Port:       manifestutils.PortZipkin,
					TargetPort: intstr.FromString(manifestutils.PortZipkinName),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "ca-custom",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "ca-custom",
							},
						},
					},
				},
				{
					Name: "cert-custom",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "cert-custom",
						},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
				{
					Name:      "ca-custom",
					MountPath: manifestutils.ReceiverTLSCADir,
					ReadOnly:  true,
				},
				{
					Name:      "cert-custom",
					MountPath: manifestutils.ReceiverTLSCertDir,
					ReadOnly:  true,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "206158425",
				},
			},
		},
		{
			name:          "Receiver TLS enable with ServingCertsService feature enabled no custom certs",
			enableGateway: false,
			expectedServiceAnnotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "tempo-test-distributor-serving-cert",
			},
			expectedObjects: 3,
			receiverTLS: v1alpha1.TLSSpec{
				Enabled: true,
			},
			expectCABundleConfigMap:   true,
			enableServingCertsService: true,
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftHTTPName,
					ContainerPort: manifestutils.PortJaegerThriftHTTP,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftCompactName,
					ContainerPort: manifestutils.PortJaegerThriftCompact,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerThriftBinaryName,
					ContainerPort: manifestutils.PortJaegerThriftBinary,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerGrpcName,
					ContainerPort: manifestutils.PortJaegerGrpc,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortZipkinName,
					ContainerPort: manifestutils.PortZipkin,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.PortJaegerThriftHTTPName,
					Port:       manifestutils.PortJaegerThriftHTTP,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortJaegerThriftCompactName,
					Port:       manifestutils.PortJaegerThriftCompact,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerThriftBinaryName,
					Port:       manifestutils.PortJaegerThriftBinary,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerGrpcName,
					Port:       manifestutils.PortJaegerGrpc,
					TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortZipkinName,
					Port:       manifestutils.PortZipkin,
					TargetPort: intstr.FromString(manifestutils.PortZipkinName),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "tempo-test-serving-cabundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test-serving-cabundle",
							},
						},
					},
				},
				{
					Name: "tempo-test-distributor-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "tempo-test-distributor-serving-cert",
						},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
				{
					Name:      "tempo-test-serving-cabundle",
					MountPath: manifestutils.ReceiverTLSCADir,
					ReadOnly:  true,
				},
				{
					Name:      "tempo-test-distributor-serving-cert",
					MountPath: manifestutils.ReceiverTLSCertDir,
					ReadOnly:  true,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "206158425",
				},
			},
		},
		{
			name:            "Gateway enable",
			enableGateway:   true,
			expectedObjects: 2,
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(260, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(236223200, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(78, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(70866960, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "GOMEMLIMIT",
					Value: "188978560",
				},
			},
		},
		{
			name:            "set InstanceAddrType to PodIP",
			enableGateway:   false,
			expectedObjects: 2,
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          manifestutils.PortOtlpHttpName,
					ContainerPort: manifestutils.PortOtlpHttp,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.OtlpGrpcPortName,
					ContainerPort: manifestutils.PortOtlpGrpcServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpPortName,
					ContainerPort: manifestutils.PortHTTPServer,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.HttpMemberlistPortName,
					ContainerPort: manifestutils.PortMemberlist,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftHTTPName,
					ContainerPort: manifestutils.PortJaegerThriftHTTP,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortJaegerThriftCompactName,
					ContainerPort: manifestutils.PortJaegerThriftCompact,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerThriftBinaryName,
					ContainerPort: manifestutils.PortJaegerThriftBinary,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          manifestutils.PortJaegerGrpcName,
					ContainerPort: manifestutils.PortJaegerGrpc,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          manifestutils.PortZipkinName,
					ContainerPort: manifestutils.PortZipkin,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       manifestutils.PortOtlpHttpName,
					Port:       manifestutils.PortOtlpHttp,
					TargetPort: intstr.FromString(manifestutils.PortOtlpHttpName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.OtlpGrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortOtlpGrpcServer,
					TargetPort: intstr.FromString(manifestutils.OtlpGrpcPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.PortJaegerThriftHTTPName,
					Port:       manifestutils.PortJaegerThriftHTTP,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftHTTPName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortJaegerThriftCompactName,
					Port:       manifestutils.PortJaegerThriftCompact,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftCompactName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerThriftBinaryName,
					Port:       manifestutils.PortJaegerThriftBinary,
					TargetPort: intstr.FromString(manifestutils.PortJaegerThriftBinaryName),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       manifestutils.PortJaegerGrpcName,
					Port:       manifestutils.PortJaegerGrpc,
					TargetPort: intstr.FromString(manifestutils.PortJaegerGrpcName),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       manifestutils.PortZipkinName,
					Port:       manifestutils.PortZipkin,
					TargetPort: intstr.FromString(manifestutils.PortZipkinName),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedResources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(270, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(257698032, resource.BinarySI),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(81, resource.BinarySI),
					corev1.ResourceMemory: *resource.NewQuantity(77309416, resource.BinarySI),
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: manifestutils.ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test",
							},
						},
					},
				},
				{
					Name: manifestutils.TmpStorageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:      manifestutils.ConfigVolumeName,
					MountPath: "/conf",
					ReadOnly:  true,
				},
				{
					Name:      manifestutils.TmpStorageVolumeName,
					MountPath: manifestutils.TmpTempoStoragePath,
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name: "HASH_RING_INSTANCE_ADDR",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "status.podIP",
						},
					},
				},
				{
					Name:  "GOMEMLIMIT",
					Value: "206158425",
				},
			},
			instanceAddrType: v1alpha1.InstanceAddrPodIP,
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {

			instanceAddrType := v1alpha1.InstanceAddrDefault
			if ts.instanceAddrType != "" {
				instanceAddrType = ts.instanceAddrType
			}

			objects, err := BuildDistributor(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "project1",
					},
					Spec: v1alpha1.TempoStackSpec{
						Images: configv1alpha1.ImagesSpec{
							Tempo: "docker.io/grafana/tempo:1.5.0",
						},
						ServiceAccount: "tempo-test-serviceaccount",
						Template: v1alpha1.TempoTemplateSpec{
							Distributor: v1alpha1.TempoDistributorSpec{
								TLS: ts.receiverTLS,
								TempoComponentSpec: v1alpha1.TempoComponentSpec{
									Replicas:     ptr.To(int32(1)),
									NodeSelector: map[string]string{"a": "b"},
									Tolerations: []corev1.Toleration{
										{
											Key: "c",
										},
									},
								},
							},
							Gateway: v1alpha1.TempoGatewaySpec{
								Enabled: ts.enableGateway,
							},
						},
						Resources: v1alpha1.Resources{
							Total: &corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						},
						HashRing: v1alpha1.HashRingSpec{
							MemberList: v1alpha1.MemberListSpec{
								InstanceAddrType: instanceAddrType,
							},
						},
					},
				},
				CtrlConfig: configv1alpha1.ProjectConfig{
					Gates: configv1alpha1.FeatureGates{
						OpenShift: configv1alpha1.OpenShiftFeatureGates{
							ServingCertsService: ts.enableServingCertsService,
						},
					},
				},
			},
			)
			require.NoError(t, err)

			labels := manifestutils.ComponentLabels("distributor", "test")
			annotations := manifestutils.CommonAnnotations("")
			assert.Equal(t, ts.expectedObjects, len(objects))
			assert.Equal(t, &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-test-distributor",
					Namespace: "project1",
					Labels:    labels,
				},
				Spec: v1.DeploymentSpec{
					Replicas: ptr.To(int32(1)),
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      k8slabels.Merge(labels, map[string]string{"tempo-gossip-member": "true"}),
							Annotations: annotations,
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: "tempo-test-serviceaccount",
							NodeSelector:       map[string]string{"a": "b"},
							Tolerations: []corev1.Toleration{
								{
									Key: "c",
								},
							},
							Affinity: manifestutils.DefaultAffinity(labels),
							Containers: []corev1.Container{
								{
									Name:  "tempo",
									Image: "docker.io/grafana/tempo:1.5.0",
									Args: []string{
										"-target=distributor",
										"-config.file=/conf/tempo.yaml",
										"-log.level=info",
										"-config.expand-env=true",
									},
									Env:          ts.expectedContainerEnvVars,
									VolumeMounts: ts.expectedVolumeMounts,
									Ports:        ts.expectedContainerPorts,
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
												Path:   manifestutils.TempoReadinessPath,
												Port:   intstr.FromString(manifestutils.HttpPortName),
											},
										},
										InitialDelaySeconds: 15,
										TimeoutSeconds:      1,
									},
									Resources:       ts.expectedResources,
									SecurityContext: manifestutils.TempoContainerSecurityContext(),
								},
							},
							Volumes: ts.expectedVolumes,
						},
					},
				},
			}, objects[0])

			assert.NoError(t, err)
			assert.Equal(t, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "tempo-test-distributor",
					Namespace:   "project1",
					Labels:      labels,
					Annotations: ts.expectedServiceAnnotations,
				},
				Spec: corev1.ServiceSpec{
					Ports:    ts.expectedServicePorts,
					Selector: labels,
				},
			}, objects[1])

			if ts.expectCABundleConfigMap {
				assert.Equal(t, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-test-serving-cabundle",
						Namespace: "project1",
						Labels:    labels,
						Annotations: map[string]string{
							"service.beta.openshift.io/inject-cabundle": "true",
						},
					},
				}, objects[2])
			}
		})
	}
}

func TestOverrideResources(t *testing.T) {
	overrideResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}

	objects, err := BuildDistributor(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			ServiceAccount: "tempo-test-serviceaccount",
			Template: v1alpha1.TempoTemplateSpec{
				Distributor: v1alpha1.TempoDistributorSpec{
					TempoComponentSpec: v1alpha1.TempoComponentSpec{
						Replicas:     ptr.To(int32(1)),
						NodeSelector: map[string]string{"a": "b"},
						Tolerations: []corev1.Toleration{
							{
								Key: "c",
							},
						},
						Resources: &overrideResources,
					},
				},
			},
			Resources: v1alpha1.Resources{
				Total: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}})
	require.NoError(t, err)
	dep, ok := objects[0].(*v1.Deployment)
	require.True(t, ok)
	assert.Equal(t, dep.Spec.Template.Spec.Containers[0].Resources, overrideResources)
}
