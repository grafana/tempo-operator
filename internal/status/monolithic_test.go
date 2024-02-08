package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

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
