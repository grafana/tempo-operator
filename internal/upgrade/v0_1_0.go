package upgrade

import "github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"

// This is a template for future versions.
func upgrade0_1_0(u Upgrade, tempo *v1alpha1.TempoStack) (*v1alpha1.TempoStack, error) {
	// no-op because 0.1.0 is the first released tempo-operator version
	return tempo, nil
}
