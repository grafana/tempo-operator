package webhooks

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
