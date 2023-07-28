package controllers

import (
	"context"
	"errors"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/operator"
)

// OperatorReconciler reconciles the operator configuration.
type OperatorReconciler struct {
	client.Client
}

func (r *OperatorReconciler) getOperatorDeployment(ctx context.Context) (v1.Deployment, error) {
	listOps := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/name": "tempo-operator",
		}),
	}
	operatordeploymentsList := &v1.DeploymentList{}
	err := r.Client.List(ctx, operatordeploymentsList, listOps...)
	if err != nil {
		return v1.Deployment{}, fmt.Errorf("failed to list operator deployments: %w", err)
	}
	if len(operatordeploymentsList.Items) != 1 {
		return v1.Deployment{}, fmt.Errorf("failed to find current operator deployment, found deployments: %v", operatordeploymentsList.Items)
	}

	return operatordeploymentsList.Items[0], nil
}

// Reconcile reconciles the operator configuration.
func (r *OperatorReconciler) Reconcile(ctx context.Context, ctrlConfig configv1alpha1.ProjectConfig) error {
	log := ctrl.LoggerFrom(ctx).WithName("operator-reconcile")

	log.V(1).Info("starting reconcile loop")
	defer log.V(1).Info("finished reconcile loop")

	operatorDeployment, err := r.getOperatorDeployment(ctx)
	if err != nil {
		return fmt.Errorf("failed to get operator deployment: %w", err)
	}

	managedObjects, err := operator.BuildAll(ctrlConfig.Gates, operatorDeployment.Namespace)
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	errs := []error{}
	for _, obj := range managedObjects {
		l := log.WithValues(
			"object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
		)

		desired := obj.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(obj, desired)

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, obj, mutateFn)
		if err != nil {
			l.Error(err, "failed to configure resource")
			errs = append(errs, err)
			continue
		}

		l.V(1).Info(fmt.Sprintf("resource has been %s", op))
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create objects for operator: %w", errors.Join(errs...))
	}

	err = r.pruneObjects(ctx, ctrlConfig.Gates, operatorDeployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to prune objects: %w", err)
	}

	return nil
}

func (r *OperatorReconciler) pruneObjects(ctx context.Context, featureGates configv1alpha1.FeatureGates, namespace string) error {
	listOps := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(manifestutils.CommonOperatorLabels()),
	}

	if featureGates.PrometheusOperator {
		if !featureGates.Observability.Metrics.CreateServiceMonitors {
			servicemonitorList := &monitoringv1.ServiceMonitorList{}
			err := r.List(ctx, servicemonitorList, listOps)
			if err != nil {
				return fmt.Errorf("error listing service monitors: %w", err)
			}
			for _, obj := range servicemonitorList.Items {
				err = r.Delete(ctx, obj, &client.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("failed to prune service monitor: %w", err)
				}
			}
		}
		if !featureGates.Observability.Metrics.CreatePrometheusRules {
			prometheusruleList := &monitoringv1.PrometheusRuleList{}
			err := r.List(ctx, prometheusruleList, listOps)
			if err != nil {
				return fmt.Errorf("error listing prometheus rules: %w", err)
			}
			for _, obj := range prometheusruleList.Items {
				err = r.Delete(ctx, obj, &client.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("failed to prune prometheus rule: %w", err)
				}
			}
		}
	}

	return nil
}
