package status

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestGetStatefulSetStatus(t *testing.T) {
	tests := []struct {
		name     string
		client   client.Client
		expected v1alpha1.PodStatusMap
	}{
		{
			name: "sts rolling out",
			client: &k8sFake{
				stss: &appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo",
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.To[int32](1),
						},
						Status: appsv1.StatefulSetStatus{
							ReadyReplicas: 0,
						},
					}},
				},
			},
			expected: map[corev1.PodPhase][]string{
				corev1.PodPending: {"tempo"},
			},
		},
		{
			name: "pod pending",
			client: &k8sFake{
				stss: &appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo",
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.To[int32](1),
						},
						Status: appsv1.StatefulSetStatus{
							ReadyReplicas: 1,
						},
					}},
				},
				pods: &corev1.PodList{
					Items: []corev1.Pod{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo-xyz",
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodPending,
						},
					}},
				},
			},
			expected: map[corev1.PodPhase][]string{
				corev1.PodPending: {"tempo-xyz"},
			},
		},
		{
			name: "pod running but not ready",
			client: &k8sFake{
				stss: &appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo",
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.To[int32](1),
						},
						Status: appsv1.StatefulSetStatus{
							ReadyReplicas: 1,
						},
					}},
				},
				pods: &corev1.PodList{
					Items: []corev1.Pod{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo-xyz",
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							ContainerStatuses: []corev1.ContainerStatus{{
								Ready: false,
							}},
						},
					}},
				},
			},
			expected: map[corev1.PodPhase][]string{
				corev1.PodPending: {"tempo-xyz"},
			},
		},
		{
			name: "pod running and ready",
			client: &k8sFake{
				stss: &appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo",
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.To[int32](1),
						},
						Status: appsv1.StatefulSetStatus{
							ReadyReplicas: 1,
						},
					}},
				},
				pods: &corev1.PodList{
					Items: []corev1.Pod{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tempo-xyz",
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							ContainerStatuses: []corev1.ContainerStatus{{
								Ready: true,
							}},
						},
					}},
				},
			},
			expected: map[corev1.PodPhase][]string{
				corev1.PodRunning: {"tempo-xyz"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			psm, err := getStatefulSetStatus(context.Background(), tc.client, "", "", "")
			require.NoError(t, err)
			require.Equal(t, tc.expected, psm)
		})
	}
}

func TestUpdateConditions(t *testing.T) {
	tests := []struct {
		name                  string
		conditions            []metav1.Condition
		componentsStatus      v1alpha1.MonolithicComponentStatus
		reconcileError        error
		expectedConditions    []metav1.Condition
		expectedIsTerminalErr bool
	}{
		{
			name: "pod pending",
			componentsStatus: v1alpha1.MonolithicComponentStatus{
				Tempo: v1alpha1.PodStatusMap{
					corev1.PodPending: []string{"tempo-1"},
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionTrue,
				},
				{
					Type:   string(v1alpha1.ConditionConfigurationError),
					Reason: string(v1alpha1.ReasonInvalidStorageConfig),
					Status: metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Message: "",
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionFalse,
				},
			},
		},
		{
			name: "pod failed",
			componentsStatus: v1alpha1.MonolithicComponentStatus{
				Tempo: v1alpha1.PodStatusMap{
					corev1.PodFailed: []string{"tempo-1"},
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionFalse,
				},
				{
					Type:   string(v1alpha1.ConditionConfigurationError),
					Reason: string(v1alpha1.ReasonInvalidStorageConfig),
					Status: metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Message: messageFailed,
					Status:  metav1.ConditionTrue,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionFalse,
				},
			},
		},
		{
			name: "configuration error",
			componentsStatus: v1alpha1.MonolithicComponentStatus{
				Tempo: v1alpha1.PodStatusMap{
					corev1.PodRunning: []string{"tempo-1"},
				},
			},
			reconcileError: &ConfigurationError{
				Reason:  v1alpha1.ReasonInvalidStorageConfig,
				Message: "cannot get secret: abc",
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Reason:  string(v1alpha1.ReasonInvalidStorageConfig),
					Message: "cannot get secret: abc",
					Status:  metav1.ConditionTrue,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Message: "",
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionFalse,
				},
			},
			expectedIsTerminalErr: true,
		},
		{
			name: "transition from configuration error to no error",
			conditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Reason:  string(v1alpha1.ReasonInvalidStorageConfig),
					Message: "cannot get secret: abc",
					Status:  metav1.ConditionTrue,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Message: "",
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionTrue,
				},
			},
			componentsStatus: v1alpha1.MonolithicComponentStatus{
				Tempo: v1alpha1.PodStatusMap{
					corev1.PodRunning: []string{"tempo-1"},
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Reason:  string(v1alpha1.ReasonInvalidStorageConfig),
					Message: "cannot get secret: abc",
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Message: "",
					Status:  metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name: "other reconcile error",
			componentsStatus: v1alpha1.MonolithicComponentStatus{
				Tempo: v1alpha1.PodStatusMap{
					corev1.PodRunning: []string{"tempo-1"},
				},
			},
			reconcileError: errors.New("permission denied"),
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Message: messagePending,
					Status:  metav1.ConditionFalse,
				},
				{
					Type:   string(v1alpha1.ConditionConfigurationError),
					Reason: string(v1alpha1.ReasonInvalidStorageConfig),
					Status: metav1.ConditionFalse,
				},
				{
					Type:    string(v1alpha1.ConditionFailed),
					Reason:  string(v1alpha1.ReasonFailedReconciliation),
					Message: "permission denied",
					Status:  metav1.ConditionTrue,
				},
				{
					Type:    string(v1alpha1.ConditionReady),
					Reason:  string(v1alpha1.ReasonReady),
					Message: messageReady,
					Status:  metav1.ConditionFalse,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updatedConditions := make([]metav1.Condition, len(tc.conditions))
			_ = copy(updatedConditions, tc.conditions)

			isTerminalErr := updateConditions(&updatedConditions, tc.componentsStatus, tc.reconcileError)
			require.Equal(t, tc.expectedIsTerminalErr, isTerminalErr)

			// ignore times
			for i := range updatedConditions {
				updatedConditions[i].LastTransitionTime = metav1.Time{}
			}

			require.Equal(t, tc.expectedConditions, updatedConditions)
		})
	}
}

type k8sFake struct {
	client.Client
	stss *appsv1.StatefulSetList
	pods *corev1.PodList
}

func (k *k8sFake) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	switch typed := list.(type) {
	case *appsv1.StatefulSetList:
		if k.stss != nil {
			k.stss.DeepCopyInto(typed)
			return nil
		}
	case *corev1.PodList:
		if k.pods != nil {
			k.pods.DeepCopyInto(typed)
			return nil
		}
	}
	return fmt.Errorf("mock: not implemented")
}
