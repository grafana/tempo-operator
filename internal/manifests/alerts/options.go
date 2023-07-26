package alerts

// Options is used to configure Prometheus Alerts.
type Options struct {
	RunbookURL string
	Cluster    string
	Namespace  string
}
