package upgrade

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// Upstream Tempo 2.5.0 has a breaking change https://github.com/grafana/tempo/releases/tag/v2.5.0
// The /var/tempo is created in the dockerfile with 10001:10001
// The user is changed to 10001:10001
// The previous user in 2.4.2 was root (0)
// The Red Hat Tempo image does not use root user (it uses 1001) and on OpenShift the /var/tempo PV has a different fsGroup
// so the issue does not happen on OpenShift
func upgrade0_11_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	// do nothing on OpenShift
	if u.CtrlConfig.Gates.OpenShift.OpenShiftRoute {
		return nil
	}

	image := tempo.Spec.Images.Tempo
	if image == "" {
		image = u.CtrlConfig.DefaultImages.Tempo
	}
	rootUser := int64(0)

	listOps := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(manifestutils.IngesterComponentName, tempo.Name)),
	}
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := u.Client.List(ctx, pvcs, listOps...)
	if err != nil {
		return err
	}

	// keep the jobs around for 1 day
	ttl := int32(60 * 60 * 24)
	var errs []error
	for _, pvc := range pvcs.Items {
		upgradeJob := v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("chown-%s", pvc.Name),
				Namespace: tempo.Namespace,
			},
			Spec: v1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						// Make sure the job runs on the same node as ingester
						NodeSelector:       tempo.Spec.Template.Ingester.NodeSelector,
						ServiceAccountName: naming.DefaultServiceAccountName(tempo.Name),
						Volumes: []corev1.Volume{
							{
								Name: "data",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvc.Name,
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:    "chwon",
								Image:   image,
								Command: []string{"chown", "-R", "10001:10001", "/var/tempo"},
								VolumeMounts: []corev1.VolumeMount{
									{Name: "data", MountPath: "/var/tempo"},
								},
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
						SecurityContext: &corev1.PodSecurityContext{
							RunAsUser: &rootUser,
						},
					},
				},
				TTLSecondsAfterFinished: &ttl,
			},
		}

		if errOwnerRef := ctrl.SetControllerReference(tempo, &upgradeJob, u.Client.Scheme()); errOwnerRef != nil {
			errs = append(errs, errOwnerRef)
		}
		errCreate := u.Client.Create(ctx, &upgradeJob)
		if errCreate != nil {
			errs = append(errs, errCreate)
		}
	}

	return errors.Join(errs...)
}
