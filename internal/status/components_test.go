package status

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestSetComponentsStatus_WhenListReturnError_ReturnError(t *testing.T) {

	s := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
	}

	tests := []struct {
		componentNotFound string
	}{
		{componentNotFound: "compactor"},
		{componentNotFound: "querier"},
		{componentNotFound: "ingester"},
		{componentNotFound: "distributor"},
		{componentNotFound: "query-frontend"},
	}
	for _, tc := range tests {
		t.Run(tc.componentNotFound, func(t *testing.T) {
			k := &StatusClientStub{}
			k.UpdateStatusStub = func(ctx context.Context, s v1alpha1.Microservices) error {
				return nil
			}
			k.GetPodsComponentStub = func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
				if tc.componentNotFound == componentName {
					return nil, apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
				}
				pods := v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-a",
							},
							Status: v1.PodStatus{
								Phase: v1.PodPending,
							},
						},
					},
				}
				return &pods, nil

			}
			err := SetComponentsStatus(context.TODO(), k, s)
			require.Error(t, err)
		})
	}
}

func TestSetComponentsStatus_WhenPodListExisting_SetPodStatusMap(t *testing.T) {
	k := &StatusClientStub{}

	k.GetPodsComponentStub = func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
		pods := v1.PodList{
			Items: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod-a",
					},
					Status: v1.PodStatus{
						Phase: v1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod-b",
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
					},
				},
			},
		}
		return &pods, nil

	}

	s := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
	}

	expected := v1alpha1.PodStatusMap{
		"Pending": []string{"pod-a"},
		"Running": []string{"pod-b"},
	}

	k.UpdateStatusStub = func(ctx context.Context, s v1alpha1.Microservices) error {
		assert.Equal(t, expected, s.Status.Components.Compactor)
		return nil
	}
	err := SetComponentsStatus(context.TODO(), k, s)

	require.NoError(t, err)
}
