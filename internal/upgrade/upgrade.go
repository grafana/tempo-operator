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
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

const (
	metricUpgradesStateUpgraded = "upgraded"
	metricUpgradesStateUpToDate = "up-to-date"
	metricUpgradesStateFailed   = "failed"
)

var (
	metricUpgrades = promauto.With(metrics.Registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempooperator",
		Name:      "upgrades_total",
		Help:      "The number of up-to-date, upgraded and failed upgrades of TempoStack instances.",
	}, []string{"state"})
)

// Upgrade contains required objects to perform version upgrades.
type Upgrade struct {
	Client     client.Client
	Recorder   record.EventRecorder
	CtrlConfig configv1alpha1.ProjectConfig
	Version    version.Version
	Log        logr.Logger
}

// TempoStacks upgrades all TempoStacks in the cluster.
func (u Upgrade) TempoStacks(ctx context.Context) error {
	u.Log.Info("looking for instances to upgrade")

	listOps := []client.ListOption{}
	tempostackList := &v1alpha1.TempoStackList{}
	if err := u.Client.List(ctx, tempostackList, listOps...); err != nil {
		return fmt.Errorf("failed to list TempoStacks: %w", err)
	}

	for i := range tempostackList.Items {
		// u.TempoStack() logs the errors to the operator log
		// We continue upgrading the other operands even if an operand fails to upgrade.
		_ = u.TempoStack(ctx, tempostackList.Items[i])
	}

	if len(tempostackList.Items) == 0 {
		u.Log.Info("no instances found")
	}

	return nil
}

// TempoStack upgrades a TempoStack instance in the cluster.
func (u Upgrade) TempoStack(ctx context.Context, original v1alpha1.TempoStack) error {
	itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace, "from_version", original.Status.OperatorVersion)

	upgraded, err := u.updateTempoStackCR(ctx, original)
	itemLogger = itemLogger.WithValues("to_version", upgraded.Status.OperatorVersion)
	if err != nil {
		msg := "automated upgrade is not possible, the CR instance must be corrected and re-created manually"
		itemLogger.Info(msg)
		u.Recorder.Event(&original, "Error", "Upgrade", msg)
		metricUpgrades.WithLabelValues(metricUpgradesStateFailed).Inc()
		return err
	}

	// only save if there were changes to the CR
	if !reflect.DeepEqual(upgraded, original) {
		// the resource update overrides the status, so, keep it so that we can reset it later
		status := upgraded.Status
		patch := client.MergeFrom(&original)
		if err := u.Client.Patch(ctx, &upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance")
			metricUpgrades.WithLabelValues(metricUpgradesStateFailed).Inc()
			return err
		}

		// the status object requires its own update
		upgraded.Status = status
		if err := u.Client.Status().Patch(ctx, &upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance's status object")
			metricUpgrades.WithLabelValues(metricUpgradesStateFailed).Inc()
			return err
		}

		itemLogger.Info("upgraded instance")
		metricUpgrades.WithLabelValues(metricUpgradesStateUpgraded).Inc()
	} else {
		metricUpgrades.WithLabelValues(metricUpgradesStateUpToDate).Inc()
	}

	return nil
}

// updateTempoStackCR upgrades a single updateTempoStackCR CR to the latest known version.
// It runs all upgrade procedures between the current and latest operator version.
// Note: This method does not patch the TempoStack CR in the cluster.
func (u Upgrade) updateTempoStackCR(ctx context.Context, tempo v1alpha1.TempoStack) (v1alpha1.TempoStack, error) {
	log := u.Log.WithValues("namespace", tempo.Namespace, "tempo", tempo.Name)

	if tempo.Spec.ManagementState == v1alpha1.ManagementStateUnmanaged {
		log.Info("skipping unmanaged instance")
		return tempo, nil
	}

	if tempo.Status.OperatorVersion == u.Version.OperatorVersion {
		log.Info("instance is already up-to-date", "version", tempo.Status.OperatorVersion)
		return tempo, nil
	}

	// This field was empty in operator version 0.1.0 and 0.2.0.
	// Unfortunately this case can't be handled in the 0.3.0 upgrade procedure, because
	// the upgrade procedure requires a valid operator version.
	if tempo.Status.OperatorVersion == "" {
		tempo.Status.OperatorVersion = "0.1.0"
	}

	instanceVersion, err := semver.NewVersion(tempo.Status.OperatorVersion)
	if err != nil {
		log.Error(err, "failed to parse TempoStack operator version", "version", tempo.Status.OperatorVersion)
		return tempo, err
	}

	operatorVersion, err := semver.NewVersion(u.Version.OperatorVersion)
	if err != nil {
		u.Log.Error(err, "failed to parse current operator version", "version", u.Version.OperatorVersion)
		return tempo, err
	}

	if instanceVersion.GreaterThan(operatorVersion) {
		log.Info("skipping upgrading this instance because it's newer than the current running operator version", "version", tempo.Status.OperatorVersion, "operator_version", operatorVersion.String())
		return tempo, nil
	}

	for _, availableUpgrade := range upgrades {
		if availableUpgrade.version.GreaterThan(instanceVersion) {
			upgraded, err := availableUpgrade.upgrade(ctx, u, &tempo)
			if err != nil {
				log.Error(err, "failed to upgrade TempoStack instance", "from_version", tempo.Status.OperatorVersion, "to_version", availableUpgrade.version.String())
				return tempo, err
			}

			log.V(1).Info("performed upgrade step", "from_version", tempo.Status.OperatorVersion, "to_version", availableUpgrade.version.String())
			upgraded.Status.OperatorVersion = availableUpgrade.version.String()
			tempo = *upgraded
		}
	}

	// at the end of the upgrade process, the CR is up to date with the current running component versions (Operator, Tempo, TempoQuery)
	// update all component versions in the Status field of the CR with the current running versions
	updateTempoStackVersions(u, &tempo)

	return tempo, nil
}

// updateTempoStackVersions updates all component versions in the CR with the current running component versions.
func updateTempoStackVersions(u Upgrade, tempo *v1alpha1.TempoStack) {
	tempo.Status.OperatorVersion = u.Version.OperatorVersion
	tempo.Status.TempoVersion = u.Version.TempoVersion
	tempo.Status.TempoQueryVersion = "" // this field should be removed in the next version of the CRD
}
