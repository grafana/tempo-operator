package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func intToPointer(i int) *int {
	return &i
}

func TestBuildConfiguration(t *testing.T) {
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
	cfg, err := buildConfiguration(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Retention: v1alpha1.RetentionSpec{
				Global: v1alpha1.RetentionConfig{
					Traces: metav1.Duration{Duration: 48 * time.Hour},
				},
			},
		},
	}, Params{S3: S3{
		Endpoint: "http://minio:9000",
		Bucket:   "tempo",
	}})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}

func TestBuildConfiguration_RateLimits(t *testing.T) {

	testCases := []struct {
		name   string
		spec   v1alpha1.LimitSpec
		expect string
	}{
		{
			name: "defaults",
			spec: v1alpha1.LimitSpec{},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only IngestionRateLimitBytes",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Ingestion: v1alpha1.IngestionLimitSpec{
						IngestionRateLimitBytes: intToPointer(100),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  ingestion_rate_limit_bytes: 100
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only IngestionBurstSizeBytes",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Ingestion: v1alpha1.IngestionLimitSpec{
						IngestionBurstSizeBytes: intToPointer(100),
					}},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  ingestion_burst_size_bytes: 100
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only MaxBytesPerTrace",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Ingestion: v1alpha1.IngestionLimitSpec{
						MaxBytesPerTrace: intToPointer(100),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  max_bytes_per_trace: 100
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only MaxTracesPerUser",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Ingestion: v1alpha1.IngestionLimitSpec{
						MaxTracesPerUser: intToPointer(100),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  max_traces_per_user: 100
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only MaxBytesPerTagValues",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Query: v1alpha1.QueryLimit{
						MaxBytesPerTagValues: intToPointer(100),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  max_bytes_per_tag_values_query: 100
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "only MaxSearchBytesPerTrace",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Query: v1alpha1.QueryLimit{
						MaxSearchBytesPerTrace: intToPointer(1000),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  max_search_bytes_per_trace: 1000
querier:
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "all set",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Ingestion: v1alpha1.IngestionLimitSpec{
						IngestionBurstSizeBytes: intToPointer(100),
						IngestionRateLimitBytes: intToPointer(200),
						MaxTracesPerUser:        intToPointer(300),
						MaxBytesPerTrace:        intToPointer(400),
					},
					Query: v1alpha1.QueryLimit{
						MaxBytesPerTagValues:   intToPointer(500),
						MaxSearchBytesPerTrace: intToPointer(1000),
					},
				},
			},
			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  ingestion_burst_size_bytes: 100
  ingestion_rate_limit_bytes: 200
  max_traces_per_user: 300
  max_bytes_per_trace: 400
  max_bytes_per_tag_values_query: 500
  max_search_bytes_per_trace: 1000
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
		{
			name: "per tenant overrides",
			spec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{
					"mytenant": {
						Ingestion: v1alpha1.IngestionLimitSpec{
							IngestionBurstSizeBytes: intToPointer(100),
							IngestionRateLimitBytes: intToPointer(200),
							MaxTracesPerUser:        intToPointer(300),
							MaxBytesPerTrace:        intToPointer(400),
						},
						Query: v1alpha1.QueryLimit{
							MaxBytesPerTagValues:   intToPointer(500),
							MaxSearchBytesPerTrace: intToPointer(1000),
						},
					},
				},
			},

			expect: `
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
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
overrides:
  per_tenant_override_config: /conf/overrides.yaml
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					LimitSpec: tc.spec,
				},
			}, Params{S3: S3{
				Endpoint: "http://minio:9000",
				Bucket:   "tempo",
			}})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.MicroservicesSpec{
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					LimitSpec: tc.spec,
				},
			}, Params{S3: S3{
				Endpoint: "http://minio:9000",
				Bucket:   "tempo",
			}})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}
}

func TestBuildTenantsOverrides(t *testing.T) {
	expectedCfg := `
---
overrides:
  "mytenant":
    ingestion_burst_size_bytes: 100
`
	tempo := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1alpha1.MicroservicesSpec{
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{
					"mytenant": {
						Ingestion: v1alpha1.IngestionLimitSpec{
							IngestionBurstSizeBytes: intToPointer(100),
						},
					},
				},
			},
		},
	}
	cfg, err := buildTenantOverrides(tempo)
	require.NoError(t, err)
	require.YAMLEq(t, expectedCfg, string(cfg))
}

func TestBuildConfiguration_SearchConfig(t *testing.T) {

	testCases := []struct {
		name   string
		expect string
		spec   v1alpha1.SearchSpec
	}{
		{
			name: "defaults",
			spec: v1alpha1.SearchSpec{},
			expect: `
--- 
compactor: 
  compaction: 
    block_retention: 0s
  ring: 
    kvstore: 
      store: memberlist
distributor: 
  receivers: 
    jaeger: 
      protocols: 
        thrift_http: 
          endpoint: "0.0.0.0:14268"
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
metrics_generator_enabled: false
multitenancy_enabled: false
querier: 
  max_concurrent_queries: 20
  search:
    external_hedge_requests_at: 8s
    external_hedge_requests_up_to: 2
  frontend_worker: 
    frontend_address: "tempo-test-query-frontend-discovery:9095"
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
      `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(v1alpha1.Microservices{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.MicroservicesSpec{
					SearchSpec: tc.spec,
				},
			}, Params{S3: S3{
				Endpoint: "http://minio:9000",
				Bucket:   "tempo",
			}})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}
}
