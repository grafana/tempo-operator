package monolithic

// Labels returns common labels for each TempoMonolithic object created by the operator.
func Labels(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "tempo-monolithic",
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": "tempo-operator",
	}
}
