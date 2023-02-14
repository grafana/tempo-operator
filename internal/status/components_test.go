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
			k := &statusClientStub{}

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
			_, err := GetComponetsStatus(context.TODO(), k, s)
			require.Error(t, err)
		})
	}
}

func TestSetComponentsStatus_WhenSomePodPending(t *testing.T) {
	k := &statusClientStub{}

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

	expectedComponents := v1alpha1.PodStatusMap{
		"Pending": []string{"pod-a"},
		"Running": []string{"pod-b"},
	}

	expected := v1alpha1.MicroservicesStatus{
		Components: v1alpha1.ComponentStatus{
			Compactor:     expectedComponents,
			Ingester:      expectedComponents,
			Distributor:   expectedComponents,
			Querier:       expectedComponents,
			QueryFrontend: expectedComponents,
		},
	}

	components, err := GetComponetsStatus(context.TODO(), k, s)

	// Don't care about timing
	now := metav1.Now()
	expected.Conditions = append(expected.Conditions, metav1.Condition{
		Type:               string(v1alpha1.ConditionPending),
		Message:            messagePending,
		Reason:             string(v1alpha1.ReasonPendingComponents),
		LastTransitionTime: now,
		Status:             metav1.ConditionTrue,
	})
	components.Conditions[0].LastTransitionTime = now

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}

func TestSetComponentsStatus_WhenSomePodFailed(t *testing.T) {
	k := &statusClientStub{}

	k.GetPodsComponentStub = func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
		pods := v1.PodList{
			Items: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod-a",
					},
					Status: v1.PodStatus{
						Phase: v1.PodFailed,
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

	expectedComponents := v1alpha1.PodStatusMap{
		"Failed":  []string{"pod-a"},
		"Running": []string{"pod-b"},
	}

	expected := v1alpha1.MicroservicesStatus{
		Components: v1alpha1.ComponentStatus{
			Compactor:     expectedComponents,
			Ingester:      expectedComponents,
			Distributor:   expectedComponents,
			Querier:       expectedComponents,
			QueryFrontend: expectedComponents,
		},
	}

	components, err := GetComponetsStatus(context.TODO(), k, s)

	// Don't care about timing
	now := metav1.Now()
	expected.Conditions = append(expected.Conditions, metav1.Condition{
		Type:               string(v1alpha1.ConditionFailed),
		Message:            messageFailed,
		Reason:             string(v1alpha1.ReasonFailedComponents),
		LastTransitionTime: now,
		Status:             metav1.ConditionTrue,
	})
	components.Conditions[0].LastTransitionTime = now

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}

func TestSetComponentsStatus_WhenSomePodUnknow(t *testing.T) {
	k := &statusClientStub{}

	k.GetPodsComponentStub = func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
		pods := v1.PodList{
			Items: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod-a",
					},
					Status: v1.PodStatus{
						Phase: v1.PodUnknown,
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

	expectedComponents := v1alpha1.PodStatusMap{
		"Unknown": []string{"pod-a"},
		"Running": []string{"pod-b"},
	}

	expected := v1alpha1.MicroservicesStatus{
		Components: v1alpha1.ComponentStatus{
			Compactor:     expectedComponents,
			Ingester:      expectedComponents,
			Distributor:   expectedComponents,
			Querier:       expectedComponents,
			QueryFrontend: expectedComponents,
		},
	}

	components, err := GetComponetsStatus(context.TODO(), k, s)

	// Don't care about timing
	now := metav1.Now()
	expected.Conditions = append(expected.Conditions, metav1.Condition{
		Type:               string(v1alpha1.ConditionFailed),
		Message:            messageFailed,
		Reason:             string(v1alpha1.ReasonFailedComponents),
		LastTransitionTime: now,
		Status:             metav1.ConditionTrue,
	})
	components.Conditions[0].LastTransitionTime = now

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}

func TestSetComponentsStatus_WhenAllReady(t *testing.T) {
	k := &statusClientStub{}

	k.GetPodsComponentStub = func(ctx context.Context, componentName string, stack v1alpha1.Microservices) (*corev1.PodList, error) {
		pods := v1.PodList{
			Items: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod-a",
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
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

	expectedComponents := v1alpha1.PodStatusMap{
		"Running": []string{"pod-a", "pod-b"},
	}

	expected := v1alpha1.MicroservicesStatus{
		Components: v1alpha1.ComponentStatus{
			Compactor:     expectedComponents,
			Ingester:      expectedComponents,
			Distributor:   expectedComponents,
			Querier:       expectedComponents,
			QueryFrontend: expectedComponents,
		},
	}

	components, err := GetComponetsStatus(context.TODO(), k, s)

	// Don't care about timing
	now := metav1.Now()
	expected.Conditions = append(expected.Conditions, metav1.Condition{
		Type:               string(v1alpha1.ConditionReady),
		Message:            messageReady,
		Reason:             string(v1alpha1.ReasonReady),
		LastTransitionTime: now,
		Status:             metav1.ConditionTrue,
	})
	components.Conditions[0].LastTransitionTime = now

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}
