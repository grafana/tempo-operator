package webhooks

import (
	"context"
	"fmt"

	"github.com/grafana/tempo-operator/internal/manifests/gateway"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const maxLabelLength = 63

func validateName(name string) field.ErrorList {
	// We need to check this because the name is used as a label value for app.kubernetes.io/instance
	// Only validate the length, because the DNS rules are enforced by the functions in the `naming` package.
	if len(name) > maxLabelLength {
		return field.ErrorList{field.Invalid(
			field.NewPath("metadata").Child("name"),
			name,
			fmt.Sprintf("must be no more than %d characters", maxLabelLength),
		)}
	}
	return nil
}

func validateTempoNameConflict(getFn func() error, instanceName string, to string, from string) field.ErrorList {
	var allErrs field.ErrorList
	err := getFn()
	if err != nil {
		if !apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("name"),
				instanceName,
				err.Error(),
			))
		}
	} else {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("name"),
			instanceName,
			fmt.Sprintf("Cannot create a %s with the same name as a %s instance in the same namespace", to, from),
		))
	}
	return allErrs
}

func subjectAccessReviewsForClusterRole(user authenticationv1.UserInfo, clusterRole rbacv1.ClusterRole) []authorizationv1.SubjectAccessReview {
	reviews := []authorizationv1.SubjectAccessReview{}
	for _, rule := range clusterRole.Rules {
		for _, apiGroup := range rule.APIGroups {
			for _, resource := range rule.Resources {
				for _, verb := range rule.Verbs {
					reviews = append(reviews, authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							UID:    user.UID,
							User:   user.Username,
							Groups: user.Groups,
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Group:    apiGroup,
								Resource: resource,
								Verb:     verb,
							},
						},
					})
				}
			}
		}
	}

	return reviews
}

// validateGatewayOpenShiftModeRBAC checks if the user requesting the change on the CR
// has already the permissions which the operator would grant to the ServiceAccount of the Tempo instance
// when enabling the OpenShift tenancy mode.
//
// In other words, the operator should not grant e.g. TokenReview permissions to the ServiceAccount of the Tempo instance
// if the user creating or modifying the TempoStack or TempoMonolithic doesn't have these permissions.
func validateGatewayOpenShiftModeRBAC(ctx context.Context, client client.Client) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return err
	}

	user := req.UserInfo
	clusterRole := gateway.NewAccessReviewClusterRole("", map[string]string{})
	reviews := subjectAccessReviewsForClusterRole(user, *clusterRole)

	for _, sar := range reviews {
		err := client.Create(ctx, &sar)
		if err != nil {
			return fmt.Errorf("failed to create subject access review: %w", err)
		}

		if !sar.Status.Allowed {
			return fmt.Errorf("user %s does not have permission to %s %s.%s", user.Username, sar.Spec.ResourceAttributes.Verb, sar.Spec.ResourceAttributes.Resource, sar.Spec.ResourceAttributes.Group)
		}
	}

	return nil
}
