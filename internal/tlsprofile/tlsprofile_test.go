package tlsprofile

import (
	"context"
	"testing"

	"github.com/ViaQ/logerr/v2/kverrors"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

func TestGetTLSSecurityProfile(t *testing.T) {
	type tt struct {
		desc        string
		profile     configv1alpha1.TLSProfileType
		expected    openshiftconfigv1.TLSSecurityProfile
		expectedErr error
	}

	tc := []tt{
		{
			desc:    "Old profile",
			profile: configv1alpha1.TLSProfileOldType,
			expected: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileOldType,
			},
		},
		{
			desc:    "Intermediate profile",
			profile: configv1alpha1.TLSProfileIntermediateType,
			expected: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileIntermediateType,
			},
		},
		{
			desc:    "Modern profile",
			profile: configv1alpha1.TLSProfileModernType,
			expected: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileModernType,
			},
		},
		{
			desc:        "Unknow profile",
			profile:     configv1alpha1.TLSProfileType(""),
			expected:    openshiftconfigv1.TLSSecurityProfile{},
			expectedErr: kverrors.New("unable to determine tls profile settings %s", ""),
		},
	}

	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			profile, err := getTLSSecurityProfile(tc.profile)
			assert.Equal(t, tc.expectedErr, err)
			assert.EqualValues(t, tc.expected, profile)
		})
	}
}

func TestGetTLSSettings(t *testing.T) {
	type tt struct {
		desc        string
		profile     openshiftconfigv1.TLSSecurityProfile
		expectedErr error
	}

	tc := []tt{
		{
			desc: "Old profile",
			profile: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileOldType,
			},
		},
		{
			desc: "Intermediate profile",
			profile: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileIntermediateType,
			},
		},
		{
			desc: "Modern profile",
			profile: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileModernType,
			},
		},
		{
			desc:        "Unknow profile",
			profile:     openshiftconfigv1.TLSSecurityProfile{},
			expectedErr: kverrors.New("unable to determine tls profile settings %s", ""),
		},
		{
			desc: "Custom without spec",
			profile: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileCustomType,
			},
			expectedErr: kverrors.New("missing TLS custom profile spec"),
		},
	}

	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			options, err := GetTLSSettings(tc.profile)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				expected := TLSProfileOptions{
					MinTLSVersion: string(openshiftconfigv1.TLSProfiles[tc.profile.Type].MinTLSVersion),
					Ciphers:       crypto.OpenSSLToIANACipherSuites(openshiftconfigv1.TLSProfiles[tc.profile.Type].Ciphers),
				}
				assert.EqualValues(t, expected, options)
			}
		})
	}
}

func TestGetTLSSettingsCustom(t *testing.T) {
	profile := openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileCustomType,
		Custom: &openshiftconfigv1.CustomTLSProfile{
			TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
				Ciphers:       []string{"DES-CBC3-SHA"},
				MinTLSVersion: "TLSv1.1",
			},
		},
	}

	expected := TLSProfileOptions{
		Ciphers:       crypto.OpenSSLToIANACipherSuites([]string{"DES-CBC3-SHA"}),
		MinTLSVersion: "TLSv1.1",
	}

	options, err := GetTLSSettings(profile)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, options)
}

func TestGetDefaultTLSSecurityProfile(t *testing.T) {
	profile := GetDefaultTLSSecurityProfile()
	assert.EqualValues(t, openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileIntermediateType,
	}, profile)

}

func TestGetTLSSecurityProfile_APIServerNotFound(t *testing.T) {

	type tt struct {
		desc            string
		mockFn          func(args mock.Arguments)
		returnErr       error
		expectedProfile openshiftconfigv1.TLSSecurityProfile
	}

	nopFn := func(args mock.Arguments) {}

	tc := []tt{
		{
			desc:            "Profile not found",
			returnErr:       apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found"),
			expectedProfile: openshiftconfigv1.TLSSecurityProfile{},
			mockFn:          nopFn,
		},
		{
			desc: "Profile found",
			mockFn: func(args mock.Arguments) {
				v := args.Get(2).(*openshiftconfigv1.APIServer)
				v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileModernType,
				}
			},
			expectedProfile: openshiftconfigv1.TLSSecurityProfile{
				Type: openshiftconfigv1.TLSProfileModernType,
			},
		},
	}

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			sw := &clientStub{}
			sw.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.returnErr).Run(tc.mockFn)
			profile, err := getTLSProfileFromCluster(context.TODO(), sw)
			if tc.returnErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			sw.AssertExpectations(t)
			sw.AssertNumberOfCalls(t, "Get", 1)
			assert.Equal(t, tc.expectedProfile, profile)
		})
	}
}
