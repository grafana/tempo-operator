package manifestutils

// CommonAnnotations returns common annotations for each pod created by the operator.
func CommonAnnotations(configChecksum string) map[string]string {
	return map[string]string{
		"tempo.grafana.com/config.hash": configChecksum,
	}
}
