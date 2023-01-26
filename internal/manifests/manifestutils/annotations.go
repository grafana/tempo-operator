package manifestutils

func CommonAnnotations(configChecksum string) map[string]string {
	return map[string]string{
		"tempo.grafana.com/config.hash": configChecksum,
	}
}
