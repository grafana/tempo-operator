package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestConfigmap(t *testing.T) {
	cm, err := BuildConfigs(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, Params{S3: S3{
		Endpoint: "http://minio:9000",
		Bucket:   "tempo",
	}})

	require.NoError(t, err)
	require.NotNil(t, cm.Data)
	require.NotNil(t, cm.Data["tempo.yaml"])
	assert.Equal(t, `
compactor:
  compaction:
    block_retention: 48h
  ring:
    kvstore:
      store: memberlist
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
  ring:
    kvstore:
      store: memberlist
ingester:
  lifecycler:
    ring:
      kvstore:
        store: memberlist
      replication_factor: 1
    tokens_file_path: /var/tempo/tokens.json
  max_block_duration: 10m
memberlist:
  abort_if_cluster_join_fails: false
  join_members:
  - tempo-test-gossip-ring
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  ingestion_burst_size_bytes: 15000000000
  ingestion_rate_limit_bytes: 15000000000
  max_bytes_per_tag_values_query: 15000000000
  max_search_bytes_per_trace: 0
  max_traces_per_user: 50000000
querier:
  frontend_worker:
    frontend_address: tempo-test-query-frontend-discovery:9095
search_enabled: true
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3100
  http_server_read_timeout: 3m
  http_server_write_timeout: 3m
  log_format: logfmt
  log_level: debug
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    cache: none
    s3:
      endpoint: minio:9000
      bucket: tempo
      insecure: true
    local:
      path: /var/tempo/traces
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
`, cm.Data["tempo.yaml"])
}
