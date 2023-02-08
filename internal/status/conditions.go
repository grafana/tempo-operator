package status

import (
	"context"
	"fmt"

	dockerparser "github.com/novln/docker-parser"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

const (
	messageReady   = "All components are operational"
	messageFailed  = "Some Microservice components failed"
	messagePending = "Some Microservice components pending on dependencies"
)

// DegradedError contains information about why the managed Microservice has an invalid configuration.
type DegradedError struct {
	Message string
	Reason  v1alpha1.ConditionReason
	Requeue bool
}

func (e *DegradedError) Error() string {
	return fmt.Sprintf("cluster degraded: %s", e.Message)
}

// SetReadyCondition updates or appends the condition Ready to the Microservice status conditions.
// In addition it resets all other Status conditions to false.
func SetReadyCondition(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices) (bool, error) {
	ready := metav1.Condition{
		Type:    string(v1alpha1.ConditionReady),
		Message: messageReady,
		Reason:  string(v1alpha1.ReasonReady),
	}

	return updateCondition(ctx, k, tempo, ready)
}

// SetFailedCondition updates or appends the condition Failed to the Microservice status conditions.
// In addition it resets all other Status conditions to false.
func SetFailedCondition(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices) (bool, error) {
	failed := metav1.Condition{
		Type:    string(v1alpha1.ConditionFailed),
		Message: messageFailed,
		Reason:  string(v1alpha1.ReasonFailedComponents),
	}

	return updateCondition(ctx, k, tempo, failed)
}

// SetPendingCondition updates or appends the condition Pending to the Microservice status conditions.
// In addition it resets all other Status conditions to false.
func SetPendingCondition(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices) (bool, error) {
	pending := metav1.Condition{
		Type:    string(v1alpha1.ConditionPending),
		Message: messagePending,
		Reason:  string(v1alpha1.ReasonPendingComponents),
	}

	return updateCondition(ctx, k, tempo, pending)
}

// SetDegradedCondition appends the condition Degraded to the Microservice status conditions.
func SetDegradedCondition(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices, msg string, reason v1alpha1.ConditionReason) (bool, error) {
	degraded := metav1.Condition{
		Type:    string(v1alpha1.ConditionDegraded),
		Message: msg,
		Reason:  string(reason),
	}

	return updateCondition(ctx, k, tempo, degraded)
}

func updateCondition(ctx context.Context, k StatusClient, tempo v1alpha1.Microservices, condition metav1.Condition) (bool, error) {

	tempoImage, err := dockerparser.Parse(tempo.Spec.Images.Tempo)
	if err != nil {
		return false, err
	}

	for _, c := range tempo.Status.Conditions {
		if c.Type == condition.Type &&
			c.Reason == condition.Reason &&
			c.Message == condition.Message &&
			c.Status == metav1.ConditionTrue {
			// resource already has desired condition
			return false, nil
		}
	}

	changed := tempo.DeepCopy()
	changed.Status.TempoVersion = tempoImage.Tag()

	condition.Status = metav1.ConditionTrue
	now := metav1.Now()
	condition.LastTransitionTime = now

	index := -1
	for i := range changed.Status.Conditions {
		// Reset all other conditions first
		changed.Status.Conditions[i].Status = metav1.ConditionFalse
		changed.Status.Conditions[i].LastTransitionTime = now

		// Locate existing pending condition if any
		if changed.Status.Conditions[i].Type == condition.Type {
			index = i
		}
	}

	if index == -1 {
		changed.Status.Conditions = append(changed.Status.Conditions, condition)
	} else {
		changed.Status.Conditions[index] = condition
	}

	err = k.PatchStatus(ctx, changed, &tempo)
	if err != nil {
		return true, err
	}

	return false, nil
}
