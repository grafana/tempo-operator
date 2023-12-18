package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func TestApplyTempoConfigLayer(t *testing.T) {
	input := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
  ring:
    kvstore:
      store: memberlist
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
        http:
          endpoint: "0.0.0.0:4318"
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
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m
  http_server_write_timeout: 3m
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    cache: none
    local:
      path: /var/tempo/traces
    s3:
      bucket: tempo
      endpoint: "minio:9000"
      insecure: true
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
`

	expCfg := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
  ring:
    kvstore:
      store: memberlist
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
        http:
          endpoint: "0.0.0.0:4318"
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
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 128
  grpc_server_max_send_msg_size: 128
  http_listen_port: 3200
  http_server_read_timeout: 3m
  http_server_write_timeout: 3m
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    cache: none
    local:
      path: /var/tempo/traces
    s3:
      bucket: tempo
      endpoint: "minio:9000"
      insecure: true
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
`

	override :=
		`
server:
  grpc_server_max_recv_msg_size: 128
  grpc_server_max_send_msg_size: 128
`

	rawInput := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(override), &rawInput)

	require.NoError(t, err)

	layers := make(v1alpha1.ConfigLayers)
	layers[tempoConfigKey] = v1alpha1.Config{Raw: rawInput}

	result, err := applyTempoConfigLayer(layers, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(result))

}

func TestApplyTempoConfigLayerNonExisting(t *testing.T) {
	input := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
  ring:
    kvstore:
      store: memberlist
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
        http:
          endpoint: "0.0.0.0:4318"
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
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m
  http_server_write_timeout: 3m
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    cache: none
    local:
      path: /var/tempo/traces
    s3:
      bucket: tempo
      endpoint: "minio:9000"
      insecure: true
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
`

	layers := make(v1alpha1.ConfigLayers)

	result, err := applyTempoConfigLayer(layers, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, input, string(result))

}

func TestApplyTempoConfigInvalidYAML(t *testing.T) {
	input := `21312312`
	override :=
		`
server:
  grpc_server_max_recv_msg_size: 128
  grpc_server_max_send_msg_size: 128
`

	rawInput := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(override), &rawInput)

	require.NoError(t, err)
	layers := make(v1alpha1.ConfigLayers)
	layers[tempoConfigKey] = v1alpha1.Config{Raw: rawInput}

	result, err := applyTempoConfigLayer(layers, []byte(input))

	require.Error(t, err)
	require.Nil(t, result)

}
