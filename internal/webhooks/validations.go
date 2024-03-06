package webhooks

import (
	"fmt"

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
