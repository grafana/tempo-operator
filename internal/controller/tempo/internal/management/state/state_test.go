package state_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/controller/tempo/internal/management/state"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var k8sClient client.Client
var testEnv *envtest.Environment
var testScheme *runtime.Scheme = scheme.Scheme

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "..", "..", "..", "config", "crd", "bases")},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start testEnv: %v", err)
		os.Exit(1)
	}

	if err := tempov1alpha1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		fmt.Printf("failed to setup a Kubernetes client: %v", err)
		os.Exit(1)
	}

	code := m.Run()
	err = testEnv.Stop()
	if err != nil {
		fmt.Printf("failed to stop testEnv: %v", err)
		os.Exit(1)
	}

	os.Exit(code)
}

func TestIsManaged(t *testing.T) {
	type test struct {
		name   string
		stack  tempov1alpha1.TempoStack
		wantOk bool
	}

	table := []test{
		{
			name: "managed",
			stack: tempov1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-managed",
					Namespace: "default",
				},
				Spec: tempov1alpha1.TempoStackSpec{
					ManagementState: tempov1alpha1.ManagementStateManaged,
					Storage: tempov1alpha1.ObjectStorageSpec{
						Secret: tempov1alpha1.ObjectStorageSecretSpec{
							Name: "test-storage",
							Type: tempov1alpha1.ObjectStorageSecretS3,
						},
					},
				},
			},
			wantOk: true,
		},
		{
			name: "unmanaged",
			stack: tempov1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-unmanaged",
					Namespace: "default",
				},
				Spec: tempov1alpha1.TempoStackSpec{
					ManagementState: tempov1alpha1.ManagementStateUnmanaged,
					Storage: tempov1alpha1.ObjectStorageSpec{
						Secret: tempov1alpha1.ObjectStorageSecretSpec{
							Name: "test-storage",
							Type: tempov1alpha1.ObjectStorageSecretS3,
						},
					},
				},
			},
			wantOk: false,
		},
	}
	for _, tst := range table {
		t.Run(tst.name, func(t *testing.T) {
			err := k8sClient.Create(context.TODO(), &tst.stack)
			require.NoError(t, err)

			r := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tst.stack.Name,
					Namespace: tst.stack.Namespace,
				},
			}

			ok, err := state.IsManaged(context.TODO(), r, k8sClient)
			require.NoError(t, err)
			require.Equal(t, ok, tst.wantOk)
		})
	}
}
