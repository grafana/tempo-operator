// The upgrade process in this package is based on opentelemetry-operator's upgrade process,
// licensed under the Apache License, Version 2.0.
// https://github.com/open-telemetry/opentelemetry-operator/tree/0a92a119f5acdcd775169e946638217ff5c78a1d/pkg/collector/upgrade
package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/version"
)

const (
	metricUpgradesStateUpgraded = "upgraded"
	metricUpgradesStateFailed   = "failed"
)

var (
	metricUpgrades = promauto.With(metrics.Registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempooperator",
		Name:      "upgrades_total",
		Help:      "The number of upgraded and failed upgrades of TempoStack instances.",
	}, []string{"kind", "state"})
)

// Upgrade contains required objects to perform version upgrades.
type Upgrade struct {
	Client     client.Client
	Recorder   record.EventRecorder
	CtrlConfig configv1alpha1.ProjectConfig
	Version    version.Version
	Log        logr.Logger
}

// UpgradeableCR defines functions required for upgrading a managed Custom Resource (TempoStack, TempoMonolithic).
type UpgradeableCR interface {
	client.Object
	GetOperatorVersion() string
	SetOperatorVersion(v string)
	SetTempoVersion(v string)
	GetStatus() any
	SetStatus(s any)
}

// Upgrade performs an upgrade of an UpgradeableCR in the cluster.
func (u Upgrade) Upgrade(ctx context.Context, original UpgradeableCR) (UpgradeableCR, error) {
	kind := original.GetObjectKind().GroupVersionKind().Kind
	itemLogger := u.Log.WithValues(
		"namespace", original.GetNamespace(),
		"name", original.GetName(),
		"kind", kind,
		"from_version", original.GetOperatorVersion(),
		"to_version", u.Version.OperatorVersion,
	)

	upgraded, err := u.upgradeSpec(ctx, original)
	if err != nil {
		msg := "automated upgrade is not possible, the CR instance must be corrected manually"
		itemLogger.Info(msg)
		u.Recorder.Event(original, corev1.EventTypeWarning, "FailedUpgrade", msg)
		metricUpgrades.WithLabelValues(kind, metricUpgradesStateFailed).Inc()
		return original, &status.ConfigurationError{
			Message: fmt.Sprintf("error during upgrade: %s", err),
			Reason:  v1alpha1.ReasonFailedUpgrade,
		}
	}

	// only save if there were changes to the CR
	if !reflect.DeepEqual(upgraded, original) {
		// the resource update overrides the status, so, keep it so that we can reset it later
		status := upgraded.GetStatus()
		patch := client.MergeFrom(original)
		if err := u.Client.Patch(ctx, upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance")
			metricUpgrades.WithLabelValues(kind, metricUpgradesStateFailed).Inc()
			return original, err
		}

		// the status object requires its own update
		upgraded.SetStatus(status)
		if err := u.Client.Status().Patch(ctx, upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance's status object")
			metricUpgrades.WithLabelValues(kind, metricUpgradesStateFailed).Inc()
			return original, err
		}

		itemLogger.Info("upgraded instance")
		metricUpgrades.WithLabelValues(kind, metricUpgradesStateUpgraded).Inc()
	}

	return upgraded, nil
}

// upgradeSpec upgrades a single CR to the latest known version.
// It runs all upgrade procedures between the current and latest operator version.
// Note: This method does not patch the CR in the cluster.
func (u Upgrade) upgradeSpec(ctx context.Context, original UpgradeableCR) (UpgradeableCR, error) {
	kind := original.GetObjectKind().GroupVersionKind().Kind
	log := u.Log.WithValues("namespace", original.GetNamespace(), "name", original.GetName(), "kind", kind)

	// do not mutate the CR in place, otherwise a broken upgrade step can result in an inconsistent state
	upgraded := original.DeepCopyObject().(UpgradeableCR)

	if original.GetOperatorVersion() == u.Version.OperatorVersion {
		log.Info("instance is already up-to-date", "version", original.GetOperatorVersion())
		return original, nil
	}

	instanceVersion, err := semver.NewVersion(original.GetOperatorVersion())
	if err != nil {
		log.Error(err, "failed to parse instance operator version", "version", original.GetOperatorVersion())
		return original, err
	}

	operatorVersion, err := semver.NewVersion(u.Version.OperatorVersion)
	if err != nil {
		u.Log.Error(err, "failed to parse current operator version", "version", u.Version.OperatorVersion)
		return original, err
	}

	if instanceVersion.GreaterThan(operatorVersion) {
		log.Info("skipping upgrading this instance because it's newer than the current running operator version", "version", original.GetOperatorVersion(), "operator_version", operatorVersion.String())
		return original, nil
	}

	for _, availableUpgrade := range upgrades {
		if availableUpgrade.version.GreaterThan(instanceVersion) {
			itemLogger := log.WithValues("from_version", upgraded.GetOperatorVersion(), "to_version", availableUpgrade.version.String())

			// The upgrade callback requires an Upgrade parameter. Unfortunately we can't add this function
			// to the UpgradeableCR interface, because otherwise the v1alpha1 package would import the upgrade package,
			// however the upgrade package already imports the v1alpha1 package, resulting in an import loop.
			var err error
			switch t := upgraded.(type) {
			case *v1alpha1.TempoStack:
				if availableUpgrade.upgradeTempoStack != nil {
					err = availableUpgrade.upgradeTempoStack(ctx, u, t)
				}
			case *v1alpha1.TempoMonolithic:
				if availableUpgrade.upgradeTempoMonolithic != nil {
					err = availableUpgrade.upgradeTempoMonolithic(ctx, u, t)
				}
			}

			if err != nil {
				itemLogger.Error(err, "failed to run upgrade step")
				return original, err
			}

			itemLogger.V(1).Info("performed upgrade step")
			upgraded.SetOperatorVersion(availableUpgrade.version.String())
		}
	}

	// at the end of the upgrade process, the CR is up to date with the current running component versions (Operator, Tempo, TempoQuery)
	// update all component versions in the Status field of the CR with the current running versions
	upgraded.SetOperatorVersion(u.Version.OperatorVersion)
	upgraded.SetTempoVersion(u.Version.TempoVersion)

	return upgraded, nil
}
