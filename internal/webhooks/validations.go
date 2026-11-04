package webhooks

import (
	"context"
	"fmt"

	"github.com/grafana/tempo-operator/internal/manifests/gateway"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// subjectAccessReviewsForRules returns a SubjectAccessReview for every apiGroup/resource/verb combination of the given rules.
// An empty namespace checks the permissions cluster-wide.
func subjectAccessReviewsForRules(user authenticationv1.UserInfo, namespace string, rules []rbacv1.PolicyRule) []authorizationv1.SubjectAccessReview {
	extra := make(map[string]authorizationv1.ExtraValue, len(user.Extra))
	for k, v := range user.Extra {
		extra[k] = authorizationv1.ExtraValue(v)
	}

	reviews := []authorizationv1.SubjectAccessReview{}
	for _, rule := range rules {
		for _, apiGroup := range rule.APIGroups {
			for _, resource := range rule.Resources {
				for _, verb := range rule.Verbs {
					reviews = append(reviews, authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							UID:    user.UID,
							User:   user.Username,
							Groups: user.Groups,
							Extra:  extra,
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: namespace,
								Group:     apiGroup,
								Resource:  resource,
								Verb:      verb,
							},
						},
					})
				}
			}
		}
	}

	return reviews
}

// checkSubjectAccessReviews creates the SubjectAccessReviews and returns an error if any of them is not allowed.
func checkSubjectAccessReviews(ctx context.Context, client client.Client, user authenticationv1.UserInfo, sars []authorizationv1.SubjectAccessReview) error {
	for _, sar := range sars {
		err := client.Create(ctx, &sar)
		if err != nil {
			return fmt.Errorf("failed to create subject access review: %w", err)
		}

		if !sar.Status.Allowed {
			attrs := sar.Spec.ResourceAttributes
			resource := schema.GroupResource{Group: attrs.Group, Resource: attrs.Resource}
			if attrs.Namespace != "" {
				return fmt.Errorf("user %s does not have permission to %s %s in namespace %s", user.Username, attrs.Verb, resource, attrs.Namespace)
			}
			return fmt.Errorf("user %s does not have permission to %s %s", user.Username, attrs.Verb, resource)
		}
	}

	return nil
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

	clusterRole := gateway.NewAccessReviewClusterRole("", map[string]string{})
	sars := subjectAccessReviewsForRules(req.UserInfo, "", clusterRole.Rules)
	return checkSubjectAccessReviews(ctx, client, req.UserInfo, sars)
}

// validatePodCreatePermission checks if the user requesting the change on the CR
// has permission to create pods in the given namespace.
//
// Some fields of the CR (e.g. custom container images, a custom ServiceAccount or environment variables) directly control
// the pods created by the operator. Setting these fields must not allow a user without permission to create pods
// to run arbitrary images, to run pods with arbitrary ServiceAccounts or to expose arbitrary Secrets to the pods.
func validatePodCreatePermission(ctx context.Context, client client.Client, namespace string) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return err
	}

	rules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"create"},
	}}
	sars := subjectAccessReviewsForRules(req.UserInfo, namespace, rules)
	return checkSubjectAccessReviews(ctx, client, req.UserInfo, sars)
}
