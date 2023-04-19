package certrotation

import (
	"fmt"
	"time"

	"k8s.io/apiserver/pkg/authentication/user"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
)

var defaultUserInfo = &user.DefaultInfo{Name: "system:tempostacks", Groups: []string{"system:logging"}}

// BuildAll builds all secrets and configmaps containing
// CA certificates, CA bundles and client certificates for
// a TempoStack.
func BuildAll(opts Options) ([]client.Object, error) {
	res := make([]client.Object, 0)

	obj, err := buildSigningCASecret(&opts)
	if err != nil {
		return nil, err
	}
	res = append(res, obj)

	obj, err = buildCABundle(&opts)
	if err != nil {
		return nil, err
	}
	res = append(res, obj)

	objs, err := buildTargetCertKeyPairSecrets(opts)
	if err != nil {
		return nil, err
	}
	res = append(res, objs...)

	return res, nil
}

// ApplyDefaultSettings merges the default options with the ones we give.
func ApplyDefaultSettings(opts *Options, cfg configv1alpha1.BuiltInCertManagement) error {
	rotation, err := ParseRotation(cfg)
	if err != nil {
		return err
	}
	opts.Rotation = rotation

	clock := time.Now
	opts.Signer.Rotation = signerRotation{
		Clock: clock,
	}

	if opts.Certificates == nil {
		opts.Certificates = make(map[string]SelfSignedCertKey)
	}
	for service, name := range ComponentCertSecretNames(opts.StackName) {
		r := certificateRotation{
			Clock:    clock,
			UserInfo: defaultUserInfo,
			Hostnames: []string{
				fmt.Sprintf("%s.%s.svc.cluster.local", service, opts.StackNamespace),
				fmt.Sprintf("%s.%s.svc", service, opts.StackNamespace),
			},
		}

		cert, ok := opts.Certificates[name]
		if !ok {
			cert = SelfSignedCertKey{}
		}
		cert.Rotation = r
		opts.Certificates[name] = cert
	}

	return nil
}
