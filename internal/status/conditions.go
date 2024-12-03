package status

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

const (
	messageReady   = "All components are operational"
	messageFailed  = "Some Tempo components failed"
	messagePending = "Some Tempo components are pending on dependencies"
)

// ConfigurationError contains information about why the managed TempoStack has an invalid configuration.
type ConfigurationError struct {
	Reason  v1alpha1.ConditionReason
	Message string
}

func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("invalid configuration: %s", e.Message)
}

// ReadyCondition updates or appends the condition Ready to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func ReadyCondition(tempo v1alpha1.TempoStack) []metav1.Condition {
	ready := metav1.Condition{
		Type:    string(v1alpha1.ConditionReady),
		Message: messageReady,
		Reason:  string(v1alpha1.ReasonReady),
	}

	return UpdateCondition(tempo, ready)
}

// FailedCondition updates or appends the condition Failed to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func FailedCondition(tempo v1alpha1.TempoStack) []metav1.Condition {
	failed := metav1.Condition{
		Type:    string(v1alpha1.ConditionFailed),
		Message: messageFailed,
		Reason:  string(v1alpha1.ReasonFailedComponents),
	}

	return UpdateCondition(tempo, failed)
}

// PendingCondition updates or appends the condition Pending to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func PendingCondition(tempo v1alpha1.TempoStack) []metav1.Condition {
	pending := metav1.Condition{
		Type:    string(v1alpha1.ConditionPending),
		Message: messagePending,
		Reason:  string(v1alpha1.ReasonPendingComponents),
	}

	return UpdateCondition(tempo, pending)
}

// UpdateCondition updates or appends the condition to the TempoStack status conditions.
// In addition it resets all other status conditions to false.
func UpdateCondition(tempo v1alpha1.TempoStack, condition metav1.Condition) []metav1.Condition {

	for _, c := range tempo.Status.Conditions {
		if c.Type == condition.Type &&
			c.Reason == condition.Reason &&
			c.Message == condition.Message &&
			c.Status == metav1.ConditionTrue {
			// resource already has desired condition
			return tempo.Status.Conditions
		}
	}

	status := tempo.DeepCopy().Status

	condition.Status = metav1.ConditionTrue
	now := metav1.Now()
	condition.LastTransitionTime = now

	index := -1
	for i := range status.Conditions {
		// Reset all other conditions first
		status.Conditions[i].Status = metav1.ConditionFalse
		status.Conditions[i].LastTransitionTime = now

		// Locate existing pending condition if any
		if status.Conditions[i].Type == condition.Type {
			index = i
		}
	}

	if index == -1 {
		status.Conditions = append(status.Conditions, condition)
	} else {
		status.Conditions[index] = condition
	}

	return status.Conditions
}
