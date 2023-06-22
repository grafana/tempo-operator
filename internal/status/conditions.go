package status

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

const (
	messageReady   = "All components are operational"
	messageFailed  = "Some TempoStack components failed"
	messagePending = "Some TempoStack components are pending on dependencies"
)

// DegradedError contains information about why the managed TempoStack has an invalid configuration.
type DegradedError struct {
	Message string
	Reason  v1alpha1.ConditionReason
	Requeue bool
}

func (e *DegradedError) Error() string {
	return fmt.Sprintf("cluster degraded: %s", e.Message)
}

// ReadyCondition updates or appends the condition Ready to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func ReadyCondition(k StatusClient, tempo v1alpha1.TempoStack) []metav1.Condition {
	ready := metav1.Condition{
		Type:    string(v1alpha1.ConditionReady),
		Message: messageReady,
		Reason:  string(v1alpha1.ReasonReady),
	}

	return updateCondition(tempo, ready)
}

// FailedCondition updates or appends the condition Failed to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func FailedCondition(k StatusClient, tempo v1alpha1.TempoStack) []metav1.Condition {
	failed := metav1.Condition{
		Type:    string(v1alpha1.ConditionFailed),
		Message: messageFailed,
		Reason:  string(v1alpha1.ReasonFailedComponents),
	}

	return updateCondition(tempo, failed)
}

// PendingCondition updates or appends the condition Pending to the TempoStack status conditions.
// In addition it resets all other Status conditions to false.
func PendingCondition(k StatusClient, tempo v1alpha1.TempoStack) []metav1.Condition {
	pending := metav1.Condition{
		Type:    string(v1alpha1.ConditionPending),
		Message: messagePending,
		Reason:  string(v1alpha1.ReasonPendingComponents),
	}

	return updateCondition(tempo, pending)
}

// DegradedCondition appends the condition Degraded to the TempoStack status conditions.
func DegradedCondition(tempo v1alpha1.TempoStack, msg string, reason v1alpha1.ConditionReason) []metav1.Condition {
	degraded := metav1.Condition{
		Type:    string(v1alpha1.ConditionDegraded),
		Message: msg,
		Reason:  string(reason),
	}

	return updateCondition(tempo, degraded)
}

func updateCondition(tempo v1alpha1.TempoStack, condition metav1.Condition) []metav1.Condition {

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
