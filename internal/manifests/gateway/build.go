package gateway

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"path"
	"text/template"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tempov1alpha1 "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const (
	// tempoComponentName is the name of the build tempo component.
	tempoComponentName = "gateway"
	// tempoGatewayTenantFileName is the name of the tenant config file in the configmap.
	tempoGatewayTenantFileName = "tenants.yaml"
	// tempoGatewayRbacFileName is the name of the rbac config file in the configmap.
	tempoGatewayRbacFileName = "rbac.yaml"
	// tempoGatewayRegoFileName is the name of the tempo-gateway rego config file in the configmap.
	tempoGatewayRegoFileName = "tempo-gateway.rego"

	// tempoGatewayMountDir is the path that is mounted from the configmap.
	tempoGatewayMountDir = "/etc/tempo-gateway"
)

var (
	//go:embed gateway-rbac.yaml
	tempoGatewayRbacYAMLTmplFile embed.FS

	//go:embed gateway-tenants.yaml
	tempoGatewayTenantsYAMLTmplFile embed.FS

	//go:embed tempo-gateway.rego
	tempoStackGatewayRegoTmplFile embed.FS

	tempoGatewayRbacYAMLTmpl = template.Must(template.ParseFS(tempoGatewayRbacYAMLTmplFile, "gateway-rbac.yaml"))

	tempoGatewayTenantsYAMLTmpl = template.Must(template.ParseFS(tempoGatewayTenantsYAMLTmplFile, "gateway-tenants.yaml"))

	tempoGatewayRegoTmpl = template.Must(template.ParseFS(tempoStackGatewayRegoTmplFile, "tempo-gateway.rego"))
)

// BuildGateway creates gateway objects.
func BuildGateway(params manifestutils.Params) ([]client.Object, error) {
	if !params.Tempo.Spec.Components.Gateway.Enabled ||
		params.Tempo.Spec.Tenants == nil {
		return []client.Object{}, nil
	}
	rbacCfg, tenantsCfg, regoCfg, err := getCfgs(options{
		Namespace: params.Tempo.Namespace,
		Name:      params.Tempo.Name,
		Tenants:   params.Tempo.Spec.Tenants.DeepCopy(),
	})
	if err != nil {
		return nil, err
	}
	return []client.Object{
		configMap(params.Tempo, rbacCfg, regoCfg),
		secrert(params.Tempo, tenantsCfg),
		deployment(params),
		service(params.Tempo),
	}, nil
}

// generate gateway configuration files.
func getCfgs(opts options) (rbacCfg []byte, tenantsCfg []byte, regoCfg []byte, err error) {
	// Build tempo gateway rbac yaml
	w := bytes.NewBuffer(nil)
	err = tempoGatewayRbacYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create tempo gateway rbac configuration, err: %w", err)
	}
	rbacCfg, err = io.ReadAll(w)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read rbac configuration, err: %w", err)
	}
	// Build tempo gateway tenants yaml
	w = bytes.NewBuffer(nil)
	err = tempoGatewayTenantsYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create tempo gateway tenants configuration, err: %w", err)
	}
	tenantsCfg, err = io.ReadAll(w)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read tenant configuration, err: %w", err)
	}
	// Build tempo gateway observatorium rego for static mode.
	if opts.Tenants.Mode == tempov1alpha1.Static {
		w = bytes.NewBuffer(nil)
		err = tempoGatewayRegoTmpl.Execute(w, opts)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create tempo gateway rego configuration, err: %w", err)
		}
		regoCfg, err = io.ReadAll(w)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read tempo rego configuration, err: %w", err)
		}
		return rbacCfg, tenantsCfg, regoCfg, nil
	}
	return rbacCfg, tenantsCfg, nil, nil
}

func deployment(params manifestutils.Params) *v1.Deployment {
	tempo := params.Tempo
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	annotations := manifestutils.CommonAnnotations(params.ConfigChecksum)
	cfg := tempo.Spec.Components.Gateway

	const (
		portGRPC     = 8090
		portInternal = 8081
		portPublic   = 8080
	)

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tempo.Spec.ServiceAccount,
					NodeSelector:       cfg.NodeSelector,
					Tolerations:        cfg.Tolerations,
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: tempo.Spec.Images.TempoGateway,
							Args: []string{
								fmt.Sprintf("--web.listen=0.0.0.0:%d", portPublic),
								fmt.Sprintf("--web.internal.listen=0.0.0.0:%d", portInternal),
								fmt.Sprintf("--traces.write.endpoint=%s:4317", naming.Name("distributor", tempo.Name)),
								fmt.Sprintf("--traces.read.endpoint=%s:16686", naming.Name("query", tempo.Name)),
								fmt.Sprintf("--grpc.listen=0.0.0.0:%d", portGRPC),
								fmt.Sprintf("--rbac.config=%s", path.Join(tempoGatewayMountDir, "cm", tempoGatewayRbacFileName)),
								fmt.Sprintf("--tenants.config=%s", path.Join(tempoGatewayMountDir, "secert", tempoGatewayTenantFileName)),
								"--log.level=warn",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc-public",
									ContainerPort: portGRPC,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "internal",
									ContainerPort: portInternal,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "public",
									ContainerPort: portPublic,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/live",
										Port:   intstr.FromInt(portInternal),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								TimeoutSeconds:   2,
								PeriodSeconds:    30,
								FailureThreshold: 10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/ready",
										Port:   intstr.FromInt(portInternal),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								TimeoutSeconds:   1,
								PeriodSeconds:    5,
								FailureThreshold: 12,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rbac-rego",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "cm"),
								},
								{
									Name:      "tenant",
									ReadOnly:  true,
									MountPath: path.Join(tempoGatewayMountDir, "secert", tempoGatewayTenantFileName),
									SubPath:   tempoGatewayTenantFileName,
								},
							},
							// TODO(frzifus): add gateway to resource pool.
							// Resources:       manifestutils.Resources(tempo, tempoComponentName),
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "rbac-rego",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name(tempoComponentName, tempo.Name),
									},
									Items: []corev1.KeyToPath{
										{
											Key:  tempoGatewayRbacFileName,
											Path: tempoGatewayRbacFileName,
										},
										{
											Key:  tempoGatewayRegoFileName,
											Path: tempoGatewayRegoFileName,
										},
									},
								},
							},
						},
						{
							Name: "tenant",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: naming.Name(tempoComponentName, tempo.Name),
									Items: []corev1.KeyToPath{
										{
											Key:  tempoGatewayTenantFileName,
											Path: tempoGatewayTenantFileName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func service(tempo tempov1alpha1.Microservices) *corev1.Service {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       manifestutils.HttpMemberlistPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortMemberlist,
					TargetPort: intstr.FromString(manifestutils.HttpMemberlistPortName),
				},
				{
					Name:       manifestutils.HttpPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortHTTPServer,
					TargetPort: intstr.FromString(manifestutils.HttpPortName),
				},
				{
					Name:       manifestutils.GrpcPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       manifestutils.PortGRPCServer,
					TargetPort: intstr.FromString(manifestutils.GrpcPortName),
				},
			},
			Selector: labels,
		},
	}
}

func configMap(tempo tempov1alpha1.Microservices, rbacCfg, regoCfg []byte) *corev1.ConfigMap {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string]string{
			tempoGatewayRbacFileName: string(rbacCfg),
			tempoGatewayRegoFileName: string(regoCfg),
		},
	}
}

func secrert(tempo tempov1alpha1.Microservices, tenantsCfg []byte) *corev1.Secret {
	labels := manifestutils.ComponentLabels(tempoComponentName, tempo.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(tempoComponentName, tempo.Name),
			Labels:    labels,
			Namespace: tempo.Namespace,
		},
		Data: map[string][]byte{
			tempoGatewayTenantFileName: tenantsCfg,
		},
	}
	return secret
}

// options is used to render the rbac.yaml and tenants.yaml file template.
type options struct {
	Namespace     string
	Name          string
	Tenants       *tempov1alpha1.TenantsSpec
	TenantSecrets []*secret
}

// secret for clientID, clientSecret and issuerCAPath for tenant's authentication.
type secret struct {
	TenantName   string
	ClientID     string
	ClientSecret string
	IssuerCAPath string
}
