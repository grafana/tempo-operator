package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// controllerNameZoneAware is the name of the zone-awareness controller.
const controllerNameZoneAware = "tempostack-zoneaware-pod"

// createOrUpdatePodWithLabelPred only lets events of pods through that opted into zone awareness.
// Deletions are irrelevant, as the annotation lives on the pod itself.
var createOrUpdatePodWithLabelPred = ctrlbuilder.WithPredicates(predicate.Funcs{
	UpdateFunc:  func(e event.UpdateEvent) bool { return eventPodHasLabel(e.ObjectNew) },
	CreateFunc:  func(e event.CreateEvent) bool { return eventPodHasLabel(e.Object) },
	DeleteFunc:  func(e event.DeleteEvent) bool { return false },
	GenericFunc: func(e event.GenericEvent) bool { return false },
})

// TempoStackZoneAwarePodReconciler watches the pods of zone-aware TempoStack components and annotates
// them with the availability zone of the node they were scheduled on.
//
// The pods cannot read the topology labels of their node themselves: the downward API only exposes the
// node name, not its labels. This controller closes that gap, and an init container in each pod waits
// for the annotation before Tempo starts.
type TempoStackZoneAwarePodReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Reconcile annotates a single pod with its availability zone.
func (r *TempoStackZoneAwarePodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get pod %s: %w", req.NamespacedName, err)
	}

	return ctrl.Result{}, r.annotatePodWithAvailabilityZone(ctx, pod)
}

// annotatePodWithAvailabilityZone looks up the node the pod runs on and stores the values of the
// configured topology labels as an annotation on the pod.
func (r *TempoStackZoneAwarePodReconciler) annotatePodWithAvailabilityZone(ctx context.Context, pod *corev1.Pod) error {
	// The pod is not scheduled yet, a later event will carry the node name.
	if pod.Spec.NodeName == "" {
		return nil
	}

	node := &corev1.Node{}
	if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
		return fmt.Errorf("failed to lookup node %s: %w", pod.Spec.NodeName, err)
	}

	labelsAnnotation, ok := pod.Annotations[v1alpha1.AnnotationAvailabilityZoneLabels]
	if !ok {
		return fmt.Errorf("zone-aware pod %s is missing the %s annotation", pod.Name, v1alpha1.AnnotationAvailabilityZoneLabels)
	}

	availabilityZone, err := getAvailabilityZone(strings.Split(labelsAnnotation, ","), node.Labels)
	if err != nil {
		return fmt.Errorf("failed to get availability zone of pod %s: %w", pod.Name, err)
	}

	// Stop early if there is no annotation to set.
	if availabilityZone == "" {
		return nil
	}

	if pod.Annotations[v1alpha1.AnnotationAvailabilityZone] == availabilityZone {
		return nil
	}

	mergePatch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				v1alpha1.AnnotationAvailabilityZone: availabilityZone,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not format the annotations patch of pod %s: %w", pod.Name, err)
	}

	if err := r.Patch(ctx, pod, client.RawPatch(types.StrategicMergePatchType, mergePatch)); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("could not patch the annotations of pod %s: %w", pod.Name, err)
	}

	return nil
}

// getAvailabilityZone joins the values of the given node labels, in the order the topology keys were
// configured in the CR.
func getAvailabilityZone(labelKeys []string, nodeLabels map[string]string) (string, error) {
	labelValues := []string{}
	for _, key := range labelKeys {
		value, ok := nodeLabels[key]
		if !ok {
			return "", fmt.Errorf("scheduled node is missing the %s label", key)
		}

		labelValues = append(labelValues, value)
	}

	return strings.Join(labelValues, "_"), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TempoStackZoneAwarePodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerNameZoneAware).
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}, createOrUpdatePodWithLabelPred).
		Complete(r)
}

func eventPodHasLabel(object client.Object) bool {
	pod, isPod := object.(*corev1.Pod)
	if !isPod {
		return false
	}

	_, hasLabel := pod.Labels[v1alpha1.LabelZoneAwarePod]
	return hasLabel
}
