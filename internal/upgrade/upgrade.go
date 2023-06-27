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
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
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
		original := tempostackList.Items[i]
		itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)

		upgraded, err := u.TempoStack(ctx, original)
		if err != nil {
			msg := "automated upgrade is not possible, the CR instance must be corrected and re-created manually"
			itemLogger.Info(msg)
			u.Recorder.Event(&original, "Error", "Upgrade", msg)
			continue
		}

		// only save if there were changes to the CR
		if !reflect.DeepEqual(upgraded, tempostackList.Items[i]) {
			// the resource update overrides the status, so, keep it so that we can reset it later
			status := upgraded.Status
			patch := client.MergeFrom(&original)
			if err := u.Client.Patch(ctx, &upgraded, patch); err != nil {
				itemLogger.Error(err, "failed to apply changes to instance")
				continue
			}

			// the status object requires its own update
			upgraded.Status = status
			if err := u.Client.Status().Patch(ctx, &upgraded, patch); err != nil {
				itemLogger.Error(err, "failed to apply changes to instance's status object")
				continue
			}

			itemLogger.Info("upgraded instance", "from_version", tempostackList.Items[i].Status.OperatorVersion, "to_version", upgraded.Status.OperatorVersion)
		}
	}

	if len(tempostackList.Items) == 0 {
		u.Log.Info("no instances found")
	}

	return nil
}

// TempoStack upgrades a single TempoStack CR to the latest known version.
// It runs all upgrade procedures between the current and latest operator version.
// Note: It does not save/apply the changes to the CR.
func (u Upgrade) TempoStack(ctx context.Context, tempo v1alpha1.TempoStack) (v1alpha1.TempoStack, error) {
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
			upgraded, err := availableUpgrade.upgrade(u, &tempo)
			if err != nil {
				log.Error(err, "failed to upgrade TempoStack instance", "from_version", tempo.Status.OperatorVersion, "to_version", availableUpgrade.version.String())
				return tempo, err
			}

			log.V(1).Info("performed upgrade step", "from_version", tempo.Status.OperatorVersion, "to_version", availableUpgrade.version.String())
			upgraded.Status.OperatorVersion = availableUpgrade.version.String()
			tempo = *upgraded
		}
	}

	// update all tempo images to the new default images on every upgrade
	updateTempoStackImages(u, &tempo)

	// at the end of the upgrade process, the CR is up to date with the current running versions
	// update all versions with the current running versions
	updateTempoStackVersions(u, &tempo)

	return tempo, nil
}

// updateTempoStackImages updates all images with the default images of the operator configuration.
func updateTempoStackImages(u Upgrade, tempo *v1alpha1.TempoStack) {
	if u.CtrlConfig.DefaultImages.Tempo != "" {
		tempo.Spec.Images.Tempo = u.CtrlConfig.DefaultImages.Tempo
	}

	if u.CtrlConfig.DefaultImages.TempoQuery != "" {
		tempo.Spec.Images.TempoQuery = u.CtrlConfig.DefaultImages.TempoQuery
	}

	if u.CtrlConfig.DefaultImages.TempoGateway != "" {
		tempo.Spec.Images.TempoGateway = u.CtrlConfig.DefaultImages.TempoGateway
	}

	if u.CtrlConfig.DefaultImages.TempoGatewayOpa != "" {
		tempo.Spec.Images.TempoGatewayOpa = u.CtrlConfig.DefaultImages.TempoGatewayOpa
	}
}

// updateTempoStackVersions updates all versions in the CR with the current running versions.
func updateTempoStackVersions(u Upgrade, tempo *v1alpha1.TempoStack) {
	tempo.Status.OperatorVersion = u.Version.OperatorVersion
	tempo.Status.TempoVersion = u.Version.TempoVersion
	tempo.Status.TempoQueryVersion = "" // this field should be removed in the next version of the CRD
}
