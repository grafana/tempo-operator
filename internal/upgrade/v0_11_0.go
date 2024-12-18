package upgrade

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	pollInterval = 2 * time.Second
	pollTimeout  = 5 * time.Minute
)

// Upstream Tempo 2.5.0 has a breaking change https://github.com/grafana/tempo/releases/tag/v2.5.0
// The /var/tempo is created in the dockerfile with 10001:10001
// The user is changed to 10001:10001
// The previous user in 2.4.2 was root (0)
// The Red Hat Tempo image does not use root user (it uses 1001) and on OpenShift the /var/tempo PV has a different fsGroup
// so the issue does not happen on OpenShift.
func upgrade0_11_0(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoStack) error {
	// do nothing on OpenShift
	if u.CtrlConfig.Gates.OpenShift.OpenShiftRoute {
		return nil
	}

	image := tempo.Spec.Images.Tempo
	if image == "" {
		image = u.CtrlConfig.DefaultImages.Tempo
	}

	listOps := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(manifestutils.IngesterComponentName, tempo.Name)),
	}
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := u.Client.List(ctx, pvcs, listOps...)
	if err != nil {
		return err
	}
	if len(pvcs.Items) == 0 {
		return nil
	}

	err = scale_down_ingester(ctx, u, client.ObjectKey{Namespace: tempo.GetNamespace(), Name: naming.Name(manifestutils.IngesterComponentName, tempo.GetName())})
	if err != nil {
		return err
	}

	return chown_pvcs(ctx, u, tempo, tempo.Spec.Template.Ingester.NodeSelector, image, pvcs)
}

func upgrade0_11_0_monolithic(ctx context.Context, u Upgrade, tempo *v1alpha1.TempoMonolithic) error {
	// do nothing on OpenShift
	if u.CtrlConfig.Gates.OpenShift.OpenShiftRoute {
		return nil
	}

	listOps := []client.ListOption{
		client.MatchingLabels(manifestutils.ComponentLabels(manifestutils.TempoMonolithComponentName, tempo.Name)),
	}
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := u.Client.List(ctx, pvcs, listOps...)
	if err != nil {
		return err
	}
	if len(pvcs.Items) == 0 {
		return nil
	}

	err = scale_down_ingester(ctx, u, client.ObjectKey{Namespace: tempo.GetNamespace(), Name: naming.Name(manifestutils.TempoMonolithComponentName, tempo.GetName())})
	if err != nil {
		return err
	}

	return chown_pvcs(ctx, u, tempo, tempo.Spec.NodeSelector, u.CtrlConfig.DefaultImages.Tempo, pvcs)
}

func scale_down_ingester(ctx context.Context, u Upgrade, ingesterQuery client.ObjectKey) error {
	ingester := &appsv1.StatefulSet{}
	err := u.Client.Get(ctx, ingesterQuery, ingester)
	if err != nil {
		// ingester does not exist, maybe scaled down?
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}

	patch := ingester.DeepCopy()
	zero := int32(0)
	patch.Spec.Replicas = &zero
	err = u.Client.Patch(ctx, patch, client.MergeFrom(ingester))
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (done bool, err error) {
		ingester := &appsv1.StatefulSet{}
		err = u.Client.Get(ctx, ingesterQuery, ingester)
		if err != nil {
			return false, err
		}
		if ingester.Status.Replicas == 0 {
			return true, nil
		}

		return false, nil
	})
}

func chown_pvcs(ctx context.Context, u Upgrade, tempo metav1.Object, nodeSelector map[string]string, image string, pvcs *corev1.PersistentVolumeClaimList) error {
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	for _, pvc := range pvcs.Items {
		volumes = append(volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      pvc.Name,
			MountPath: fmt.Sprintf("/var/tempo/%s", pvc.Name),
		})
	}

	// keep the jobs around for 1 day
	ttl := int32(60 * 60 * 24)
	rootUser := int64(0)
	upgradeJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("chown-%s", tempo.GetName()),
			Namespace: tempo.GetNamespace(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					// Make sure the job runs on the same node as ingester
					NodeSelector:       nodeSelector,
					ServiceAccountName: naming.DefaultServiceAccountName(tempo.GetName()),
					Volumes:            volumes,
					Containers: []corev1.Container{
						{
							Name:         "chown",
							Image:        image,
							Command:      []string{"chown", "-R", "10001:10001", "/var/tempo"},
							VolumeMounts: volumeMounts,
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

	if err := ctrl.SetControllerReference(tempo, &upgradeJob, u.Client.Scheme()); err != nil {
		return err
	}
	err := u.Client.Create(ctx, &upgradeJob)
	if err != nil {
		return err
	}
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (done bool, err error) {
		job := &batchv1.Job{}
		objectKey := client.ObjectKey{
			Namespace: upgradeJob.Namespace,
			Name:      upgradeJob.Name,
		}
		err = u.Client.Get(ctx, objectKey, job)
		if err != nil {
			return false, err
		}
		if job.Status.Succeeded == 1 {
			return true, nil
		}

		return false, nil
	})

}
