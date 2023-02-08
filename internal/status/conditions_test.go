package status

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestSetReadyCondition(t *testing.T) {
	tests := []struct {
		name                 string
		patchError           error
		expectedError        error
		expectedConditions   []metav1.Condition
		inputConditions      []metav1.Condition
		statusPatchCallCount int
	}{
		{
			name:                 "When Patch Tempo CRD returns error",
			patchError:           apierrors.NewBadRequest("something wasn't found"),
			expectedError:        apierrors.NewBadRequest("something wasn't found"),
			statusPatchCallCount: 1,
		},
		{
			name: "When Existing ReadyCondition set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReadyComponents),
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReadyComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name:            "When None exists append  ReadyCondition",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReadyComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ReadyCondition and true do nothing",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReadyComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statucPatchCallsCount := 0

			client := &StatusClientStub{}

			client.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
				statucPatchCallsCount++
				if tc.patchError != nil {
					return tc.patchError
				}

				if tc.statusPatchCallCount != 0 {
					// Don't care about time
					now := metav1.Now()
					tc.expectedConditions[0].LastTransitionTime = now
					changed.Status.Conditions[0].LastTransitionTime = now
					assert.Equal(t, tc.expectedConditions, changed.Status.Conditions)
				}
				return nil
			}

			stack := v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Images: v1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.MicroservicesStatus{
					Conditions: tc.inputConditions,
				},
			}

			err := SetReadyCondition(context.Background(), client, stack)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, statucPatchCallsCount, tc.statusPatchCallCount)

		})
	}
}

func TestSetFailedCondition(t *testing.T) {
	tests := []struct {
		name                 string
		patchError           error
		expectedError        error
		expectedConditions   []metav1.Condition
		inputConditions      []metav1.Condition
		statusPatchCallCount int
	}{
		{
			name:                 "when patch Tempo CRD returns error",
			patchError:           apierrors.NewBadRequest("something wasn't found"),
			expectedError:        apierrors.NewBadRequest("something wasn't found"),
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionFailed set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name:            "When none exists append ConditionFailed",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionFailed and true do nothing",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statucPatchCallsCount := 0

			client := &StatusClientStub{}

			client.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
				statucPatchCallsCount++
				if tc.patchError != nil {
					return tc.patchError
				}

				if tc.statusPatchCallCount != 0 {
					// Don't care about time
					now := metav1.Now()
					tc.expectedConditions[0].LastTransitionTime = now
					changed.Status.Conditions[0].LastTransitionTime = now
					assert.Equal(t, tc.expectedConditions, changed.Status.Conditions)
				}
				return nil
			}

			stack := v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Images: v1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.MicroservicesStatus{
					Conditions: tc.inputConditions,
				},
			}

			err := SetFailedCondition(context.Background(), client, stack)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, statucPatchCallsCount, tc.statusPatchCallCount)

		})
	}
}

func TestSetPendingCondition(t *testing.T) {
	tests := []struct {
		name                 string
		patchError           error
		expectedError        error
		expectedConditions   []metav1.Condition
		inputConditions      []metav1.Condition
		statusPatchCallCount int
	}{
		{
			name:                 "when patch Tempo CRD returns error",
			patchError:           apierrors.NewBadRequest("something wasn't found"),
			expectedError:        apierrors.NewBadRequest("something wasn't found"),
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionPending set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name:            "When none exists append ConditionPending",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionPending and true do nothing",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statucPatchCallsCount := 0

			client := &StatusClientStub{}

			client.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
				statucPatchCallsCount++
				if tc.patchError != nil {
					return tc.patchError
				}

				if tc.statusPatchCallCount != 0 {
					// Don't care about time
					now := metav1.Now()
					tc.expectedConditions[0].LastTransitionTime = now
					changed.Status.Conditions[0].LastTransitionTime = now
					assert.Equal(t, tc.expectedConditions, changed.Status.Conditions)
				}
				return nil
			}

			stack := v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Images: v1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.MicroservicesStatus{
					Conditions: tc.inputConditions,
				},
			}

			err := SetPendingCondition(context.Background(), client, stack)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, statucPatchCallsCount, tc.statusPatchCallCount)

		})
	}
}

func TestSetDegradedCondition(t *testing.T) {
	degradedMessage := "super degraded config"
	reasonString := "because I want"
	reason := v1alpha1.ConditionReason(reasonString)
	tests := []struct {
		name                 string
		patchError           error
		expectedError        error
		expectedConditions   []metav1.Condition
		inputConditions      []metav1.Condition
		statusPatchCallCount int
	}{
		{
			name:                 "when patch Tempo CRD returns error",
			patchError:           apierrors.NewBadRequest("something wasn't found"),
			expectedError:        apierrors.NewBadRequest("something wasn't found"),
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionDegraded set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionDegraded),
					Message: degradedMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionDegraded),
					Message: degradedMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name:            "When none exists append ConditionDegraded",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionDegraded),
					Message: degradedMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 1,
		},
		{
			name: "When existing ConditionDegraded and true do nothing",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionDegraded),
					Message: degradedMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
			statusPatchCallCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statucPatchCallsCount := 0

			client := &StatusClientStub{}

			client.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
				statucPatchCallsCount++
				if tc.patchError != nil {
					return tc.patchError
				}

				if tc.statusPatchCallCount != 0 {
					// Don't care about time
					now := metav1.Now()
					tc.expectedConditions[0].LastTransitionTime = now
					changed.Status.Conditions[0].LastTransitionTime = now
					assert.Equal(t, tc.expectedConditions, changed.Status.Conditions)
				}
				return nil
			}

			stack := v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Images: v1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.MicroservicesStatus{
					Conditions: tc.inputConditions,
				},
			}

			err := SetDegradedCondition(context.Background(), client, stack, degradedMessage, reason)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, statucPatchCallsCount, tc.statusPatchCallCount)

		})
	}
}
