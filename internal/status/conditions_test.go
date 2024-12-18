package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestReadyCondition(t *testing.T) {
	tests := []struct {
		name               string
		expectedConditions []metav1.Condition
		inputConditions    []metav1.Condition
	}{
		{
			name: "When Existing ReadyCondition set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReady),
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReady),
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name:            "When None exists append  ReadyCondition",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReady),
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name: "When existing ReadyCondition and true do nothing",
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReady),
					Status:  metav1.ConditionTrue,
				},
			},
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionReady),
					Message: messageReady,
					Reason:  string(v1alpha1.ReasonReady),
					Status:  metav1.ConditionTrue,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			stack := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.TempoStackStatus{
					Conditions: tc.inputConditions,
				},
			}

			conditions := ReadyCondition(stack)

			// Don't care about time
			now := metav1.Now()
			tc.expectedConditions[0].LastTransitionTime = now
			conditions[0].LastTransitionTime = now

			assert.Equal(t, tc.expectedConditions, conditions)

		})
	}
}

func TestFailedCondition(t *testing.T) {
	tests := []struct {
		name               string
		expectedConditions []metav1.Condition
		inputConditions    []metav1.Condition
	}{
		{
			name: "When Existing FailedCondition set it to true",
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
		},
		{
			name:            "When None exists append  FailedCondition",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name: "When existing FailedCondition and true do nothing",
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionFailed),
					Message: messageFailed,
					Reason:  string(v1alpha1.ReasonFailedComponents),
					Status:  metav1.ConditionTrue,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			stack := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.TempoStackStatus{
					Conditions: tc.inputConditions,
				},
			}

			conditions := FailedCondition(stack)

			// Don't care about time
			now := metav1.Now()
			tc.expectedConditions[0].LastTransitionTime = now
			conditions[0].LastTransitionTime = now

			assert.Equal(t, tc.expectedConditions, conditions)

		})
	}
}

func TestPendingCondition(t *testing.T) {
	tests := []struct {
		name               string
		expectedConditions []metav1.Condition
		inputConditions    []metav1.Condition
	}{
		{
			name: "When Existing PendingCondition set it to true",
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
		},
		{
			name:            "When None exists append  PendingCondition",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name: "When existing PendingCondition and true do nothing",
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionPending),
					Message: messagePending,
					Reason:  string(v1alpha1.ReasonPendingComponents),
					Status:  metav1.ConditionTrue,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			stack := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.TempoStackStatus{
					Conditions: tc.inputConditions,
				},
			}

			conditions := PendingCondition(stack)

			// Don't care about time
			now := metav1.Now()
			tc.expectedConditions[0].LastTransitionTime = now
			conditions[0].LastTransitionTime = now

			assert.Equal(t, tc.expectedConditions, conditions)

		})
	}
}

func TestConfigurationErrorCondition(t *testing.T) {
	configErrorMessage := "some configuration error"
	reasonString := "because I want"
	reason := v1alpha1.ConditionReason(reasonString)

	tests := []struct {
		name               string
		expectedConditions []metav1.Condition
		inputConditions    []metav1.Condition
	}{
		{
			name: "When Existing PendingCondition set it to true",
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Message: configErrorMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionFalse,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Message: configErrorMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name:            "When None exists append  PendingCondition",
			inputConditions: []metav1.Condition{},
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Message: configErrorMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
		},
		{
			name: "When existing PendingCondition and true do nothing",
			expectedConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Message: configErrorMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
			inputConditions: []metav1.Condition{
				{
					Type:    string(v1alpha1.ConditionConfigurationError),
					Message: configErrorMessage,
					Reason:  reasonString,
					Status:  metav1.ConditionTrue,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			stack := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-stack",
					Namespace: "some-ns",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						Tempo: "local:2.0",
					},
				},
				Status: v1alpha1.TempoStackStatus{
					Conditions: tc.inputConditions,
				},
			}

			conditions := UpdateCondition(stack, metav1.Condition{
				Type:    string(v1alpha1.ConditionConfigurationError),
				Reason:  string(reason),
				Message: configErrorMessage,
			})

			// Don't care about time
			now := metav1.Now()
			tc.expectedConditions[0].LastTransitionTime = now
			conditions[0].LastTransitionTime = now

			assert.Equal(t, tc.expectedConditions, conditions)
		})
	}
}

func TestConfigurationError(t *testing.T) {
	err := ConfigurationError{
		Message: "my message",
	}
	assert.Equal(t, "invalid configuration: my message", err.Error())
}
