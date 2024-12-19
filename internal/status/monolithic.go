package status

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/monolithic"
	"github.com/grafana/tempo-operator/internal/version"
)

func isPodReady(pod corev1.Pod) bool {
	for _, c := range pod.Status.ContainerStatuses {
		if !c.Ready {
			return false
		}
	}
	return true
}

func getStatefulSetStatus(ctx context.Context, c client.Client, namespace string, name string, component string) (v1alpha1.PodStatusMap, error) {
	psm := v1alpha1.PodStatusMap{}

	opts := []client.ListOption{
		client.MatchingLabels(monolithic.ComponentLabels(component, name)),
		client.InNamespace(namespace),
	}

	// After creation of a StatefulSet, but before the Pods are created, the list of Pods is empty
	// and therefore no Pod is in pending phase. However, this does not reflect the actual state,
	// therefore we additionally check if the StatefulSet has the required number of readyReplicas.
	//
	// This additional check also helps with Pods in terminating state, which otherwise would show up
	// as Pods with PodPhase = Running.
	stss := &appsv1.StatefulSetList{}
	err := c.List(ctx, stss, opts...)
	if err != nil {
		return nil, err
	}
	for _, sts := range stss.Items {
		if sts.Status.ReadyReplicas < ptr.Deref(sts.Spec.Replicas, 1) {
			psm[corev1.PodPending] = append(psm[corev1.PodPending], sts.Name)
			return psm, nil
		}
	}

	pods := &corev1.PodList{}
	err = c.List(ctx, pods, opts...)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		phase := pod.Status.Phase
		if phase == corev1.PodRunning {
			// for the component status consider running, but not ready, pods as pending
			if !isPodReady(pod) {
				phase = corev1.PodPending
			}
		}
		psm[phase] = append(psm[phase], pod.Name)
	}

	return psm, nil
}

func getComponentsStatus(ctx context.Context, client client.Client, tempo v1alpha1.TempoMonolithic) (v1alpha1.MonolithicComponentStatus, error) {
	var err error
	components := v1alpha1.MonolithicComponentStatus{}

	components.Tempo, err = getStatefulSetStatus(ctx, client, tempo.Namespace, tempo.Name, manifestutils.TempoMonolithComponentName)
	if err != nil {
		return v1alpha1.MonolithicComponentStatus{}, fmt.Errorf("cannot get pod status: %w", err)
	}

	return components, nil
}

func conditionStatus(active bool) metav1.ConditionStatus {
	if active {
		return metav1.ConditionTrue
	} else {
		return metav1.ConditionFalse
	}
}

// resetCondition disables the condition if it exists already (without changing any other field of the condition),
// otherwise creates a new disabled condition with a specified reason.
func resetCondition(conditions []metav1.Condition, conditionType v1alpha1.ConditionStatus, defaultReason v1alpha1.ConditionReason) metav1.Condition {
	existingCondition := meta.FindStatusCondition(conditions, string(conditionType))
	if existingCondition != nil {
		// do not modify the condition struct of the slice, otherwise
		// meta.SetStatusCondition() won't update the last transition time
		condition := existingCondition.DeepCopy()
		condition.Status = metav1.ConditionFalse
		return *condition
	} else {
		return metav1.Condition{
			Type:   string(conditionType),
			Reason: string(defaultReason),
			Status: metav1.ConditionFalse,
		}
	}
}

func updateConditions(conditions *[]metav1.Condition, componentsStatus v1alpha1.MonolithicComponentStatus, reconcileError error) bool {
	isTerminalError := false

	// set PendingComponents condition if any pod of any component is in pending phase (or running but not ready)
	pending := metav1.Condition{
		Type:    string(v1alpha1.ConditionPending),
		Reason:  string(v1alpha1.ReasonPendingComponents),
		Message: messagePending,
		Status: conditionStatus(
			len(componentsStatus.Tempo[corev1.PodPending]) > 0,
		),
	}

	// set ConfigurationError condition if the reconcile function returned a ConfigurationError
	var configurationError metav1.Condition
	var cerr *ConfigurationError
	if errors.As(reconcileError, &cerr) {
		configurationError = metav1.Condition{
			Type:    string(v1alpha1.ConditionConfigurationError),
			Reason:  string(cerr.Reason),
			Message: cerr.Message,
			Status:  metav1.ConditionTrue,
		}
		isTerminalError = true
	} else {
		configurationError = resetCondition(*conditions, v1alpha1.ConditionConfigurationError, v1alpha1.ReasonInvalidStorageConfig)
	}

	// set Failed condition if the reconcile function returned any error other than ConfigurationError,
	// or if any pod of any component is in failed phase
	var failed metav1.Condition
	if reconcileError != nil && cerr == nil {
		failed = metav1.Condition{
			Type:    string(v1alpha1.ConditionFailed),
			Reason:  string(v1alpha1.ReasonFailedReconciliation),
			Message: reconcileError.Error(),
			Status:  metav1.ConditionTrue,
		}
	} else if len(componentsStatus.Tempo[corev1.PodFailed]) > 0 {
		failed = metav1.Condition{
			Type:    string(v1alpha1.ConditionFailed),
			Reason:  string(v1alpha1.ReasonFailedComponents),
			Message: messageFailed,
			Status:  metav1.ConditionTrue,
		}
	} else {
		failed = resetCondition(*conditions, v1alpha1.ConditionFailed, v1alpha1.ReasonFailedComponents)
	}

	// set Ready condition if all above conditions are false
	ready := metav1.Condition{
		Type:    string(v1alpha1.ConditionReady),
		Reason:  string(v1alpha1.ReasonReady),
		Message: messageReady,
		Status: conditionStatus(
			pending.Status == metav1.ConditionFalse &&
				failed.Status == metav1.ConditionFalse &&
				configurationError.Status == metav1.ConditionFalse,
		),
	}

	meta.SetStatusCondition(conditions, pending)
	meta.SetStatusCondition(conditions, configurationError)
	meta.SetStatusCondition(conditions, failed)
	meta.SetStatusCondition(conditions, ready)
	return isTerminalError
}

func patchStatus(ctx context.Context, c client.Client, original v1alpha1.TempoMonolithic, status v1alpha1.TempoMonolithicStatus) error {
	patch := client.MergeFrom(&original)
	updated := original.DeepCopy()
	updated.Status = status
	return c.Status().Patch(ctx, updated, patch)
}

// HandleTempoMonolithicStatus updates the .status field of a TempoMonolithic CR
// Status Conditions API conventions: https://github.com/kubernetes/community/blob/c04227d209633696ad49d7f4546fc8cfd9c660ab/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
func HandleTempoMonolithicStatus(ctx context.Context, client client.Client, tempo v1alpha1.TempoMonolithic, reconcileError error) error {
	var err error
	log := ctrl.LoggerFrom(ctx)
	status := *tempo.Status.DeepCopy()

	// The version fields in the status are empty for new CRs
	if status.OperatorVersion == "" {
		status.OperatorVersion = version.Get().OperatorVersion
	}
	if status.TempoVersion == "" {
		status.TempoVersion = version.Get().TempoVersion
	}

	status.Components, err = getComponentsStatus(ctx, client, tempo)
	if err != nil {
		log.Error(err, "could not get status of each component")
	}

	isTerminalError := updateConditions(&status.Conditions, status.Components, reconcileError)
	if isTerminalError {
		// wrap error in reconcile.TerminalError to indicate human intervention is required
		// and the request should not be requeued.
		reconcileError = reconcile.TerminalError(reconcileError)
	}

	updateMetrics(metricTempoMonolithicStatusCondition, status.Conditions, tempo.Namespace, tempo.Name)

	err = patchStatus(ctx, client, tempo, status)
	if err != nil {
		return err
	}

	return reconcileError
}
