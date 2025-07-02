package config

import (
	"testing"
	"time"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Second * 30},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeStatic,
			S3: &manifestutils.S3{
				Insecure: true,
				Endpoint: "minio:9000",
				Bucket:   "tempo",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  ingestion_rate_limit_bytes: 100
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  ingestion_burst_size_bytes: 100
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  max_bytes_per_trace: 100
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  max_traces_per_user: 100
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  max_bytes_per_tag_values_query: 100
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
      `,
		},
		{
			name: "only MaxSearchDuration",
			spec: v1alpha1.LimitSpec{
				Global: v1alpha1.RateLimitSpec{
					Query: v1alpha1.QueryLimit{
						MaxSearchDuration: metav1.Duration{Duration: 24 * time.Hour},
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  max_search_duration: 24h0m0s
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
						MaxBytesPerTagValues: intToPointer(500),
						MaxSearchDuration:    metav1.Duration{Duration: 24 * time.Hour},
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  ingestion_burst_size_bytes: 100
  ingestion_rate_limit_bytes: 200
  max_traces_per_user: 300
  max_bytes_per_trace: 400
  max_bytes_per_tag_values_query: 500
  max_search_duration: 24h0m0s
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
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
							MaxBytesPerTagValues: intToPointer(500),
							MaxSearchDuration:    metav1.Duration{Duration: 24 * time.Hour},
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
overrides:
  per_tenant_override_config: /conf/overrides.yaml
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
      `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha1.TempoStackSpec{
						Timeout: metav1.Duration{Duration: time.Minute * 3},
						Storage: v1alpha1.ObjectStorageSpec{
							Secret: v1alpha1.ObjectStorageSecretSpec{
								Type: v1alpha1.ObjectStorageSecretS3,
							},
						},
						ReplicationFactor: 1,
						Retention: v1alpha1.RetentionSpec{
							Global: v1alpha1.RetentionConfig{
								Traces: metav1.Duration{Duration: 48 * time.Hour},
							},
						},
						LimitSpec: tc.spec,
					},
				},
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeStatic,
					S3: &manifestutils.S3{
						Insecure: true,
						Endpoint: "minio:9000",
						Bucket:   "tempo",
					},
				},
				TLSProfile: tlsprofile.TLSProfileOptions{
					MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
				},
			})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}
}

func TestBuildTenantsOverrides_ingestion(t *testing.T) {
	expectedCfg := `
---
overrides:
  "mytenant":
    ingestion:
      burst_size_bytes: 100
    read:
`
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1alpha1.TempoStackSpec{
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

func TestBuildTenantsOverrides_retention(t *testing.T) {
	expectedCfg := `
---
overrides:
  "mytenant":
    ingestion:
    read:
    compaction:
      block_retention: 24h0m0s
`
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1alpha1.TempoStackSpec{
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{
					"mytenant": {
						Traces: metav1.Duration{Duration: time.Hour * 24},
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
	defaultResultLimit := 20
	testCases := []struct {
		name   string
		expect string
		spec   v1alpha1.SearchSpec
	}{
		{
			name: "defaults",
			spec: v1alpha1.SearchSpec{
				DefaultResultLimit: &defaultResultLimit,
			},
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: gcs
    blocklist_poll: 5m
    local:
      path: /var/tempo/traces
    gcs:
      bucket_name: test-bucket
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    default_result_limit: 20
    concurrent_jobs: 2000
    max_duration: 0s
    max_spans_per_span_set: 0
      `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha1.TempoStackSpec{
						Timeout: metav1.Duration{Duration: time.Minute * 3},
						Storage: v1alpha1.ObjectStorageSpec{
							Secret: v1alpha1.ObjectStorageSecretSpec{
								Type: v1alpha1.ObjectStorageSecretGCS,
							},
						},
						ReplicationFactor: 1,
						SearchSpec:        tc.spec,
					},
				},
				StorageParams: manifestutils.StorageParams{
					GCS: &manifestutils.GCS{
						Bucket: "test-bucket",
					},
				},
				TLSProfile: tlsprofile.TLSProfileOptions{
					MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
				},
			})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}
}

func TestBuildConfiguration_ReplicationFactor(t *testing.T) {

	replcationFactor := 10
	expect := `
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
      replication_factor: 10
    tokens_file_path: /var/tempo/tokens.json
  max_block_duration: 10m
memberlist:
  abort_if_cluster_join_fails: false
  join_members:
    - tempo-test-gossip-ring
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: azure
    blocklist_poll: 5m
    local:
      path: /var/tempo/traces
    azure:
      container_name: "container-test"
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
    max_spans_per_span_set: 0
      `

	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Minute * 3},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretAzure,
					},
				},
				ReplicationFactor: replcationFactor,
			},
		},
		StorageParams: manifestutils.StorageParams{
			AzureStorage: &manifestutils.AzureStorage{
				Container: "container-test",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expect, string(cfg))
}

func TestBuildConfiguration_Multitenancy(t *testing.T) {
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
      endpoint: 0.0.0.0:9411
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
multitenancy_enabled: true
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Minute * 3},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
				Tenants: &v1alpha1.TenantsSpec{},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeStatic,
			S3: &manifestutils.S3{
				Insecure: true,
				Endpoint: "minio:9000",
				Bucket:   "tempo",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}

func TestBuildConfigurationTLS(t *testing.T) {

	testCases := []struct {
		name    string
		options tlsprofile.TLSProfileOptions
		expect  string
	}{
		{
			name: "tls with options",
			options: tlsprofile.TLSProfileOptions{
				Ciphers: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
					"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"},
				MinTLSVersion: "VersionTLS12",
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
    grpc_client_config:
      tls_enabled: true
      tls_cert_path: /var/run/tls/server/tls.crt
      tls_key_path: /var/run/tls/server/tls.key
      tls_ca_path: /var/run/ca/service-ca.crt
      tls_server_name: tempo-test-query-frontend.nstest.svc.cluster.local
      tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
      tls_min_version: VersionTLS12
internal_server:
  enable: true
  http_listen_address: ""
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    key_file: /var/run/tls/server/tls.key
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12
  grpc_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
ingester_client:
  grpc_client_config:
    tls_enabled: true
    tls_cert_path: /var/run/tls/server/tls.crt
    tls_key_path: /var/run/tls/server/tls.key
    tls_ca_path: /var/run/ca/service-ca.crt
    tls_server_name: tempo-test-ingester.nstest.svc.cluster.local
    tls_insecure_skip_verify: false
    tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
    tls_min_version: VersionTLS12
`,
		},
		{
			name: "tls options no ciphers",
			options: tlsprofile.TLSProfileOptions{
				MinTLSVersion: "VersionTLS13",
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
    grpc_client_config:
      tls_enabled: true
      tls_cert_path: /var/run/tls/server/tls.crt
      tls_key_path: /var/run/tls/server/tls.key
      tls_ca_path: /var/run/ca/service-ca.crt
      tls_server_name: tempo-test-query-frontend.nstest.svc.cluster.local
      tls_min_version: VersionTLS13
internal_server:
  enable: true
  http_listen_address: ""
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    key_file: /var/run/tls/server/tls.key
  tls_min_version: VersionTLS13
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
  tls_min_version: VersionTLS13
  grpc_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
ingester_client:
  grpc_client_config:
    tls_enabled: true
    tls_cert_path: /var/run/tls/server/tls.crt
    tls_key_path: /var/run/tls/server/tls.key
    tls_ca_path: /var/run/ca/service-ca.crt
    tls_server_name: tempo-test-ingester.nstest.svc.cluster.local
    tls_insecure_skip_verify: false
    tls_min_version: VersionTLS13
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cfg, err := buildConfiguration(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "nstest",
					},
					Spec: v1alpha1.TempoStackSpec{
						Timeout: metav1.Duration{Duration: time.Minute * 3},
						Storage: v1alpha1.ObjectStorageSpec{
							Secret: v1alpha1.ObjectStorageSecretSpec{
								Type: v1alpha1.ObjectStorageSecretS3,
							},
						},
						ReplicationFactor: 1,
						Retention: v1alpha1.RetentionSpec{
							Global: v1alpha1.RetentionConfig{
								Traces: metav1.Duration{Duration: 48 * time.Hour},
							},
						},
					},
				},
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeStatic,
					S3: &manifestutils.S3{
						Insecure: true,
						Endpoint: "minio:9000",
						Bucket:   "tempo",
					},
				},
				TLSProfile: tc.options,
				CtrlConfig: configv1alpha1.ProjectConfig{
					Gates: configv1alpha1.FeatureGates{
						HTTPEncryption: true,
						GRPCEncryption: true,
					},
				},
			})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}
}

func TestBuildConfigurationTLSWithGateway(t *testing.T) {
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
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
          tls:
            cert_file: /var/run/tls/server/tls.crt
            client_ca_file: /var/run/ca/service-ca.crt
            key_file: /var/run/tls/server/tls.key
            min_version: "1.2"
        http:
          endpoint: "0.0.0.0:4318"
          tls:
            cert_file: /var/run/tls/server/tls.crt
            client_ca_file: /var/run/ca/service-ca.crt
            key_file: /var/run/tls/server/tls.key
            min_version: "1.2"

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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
    grpc_client_config:
      tls_enabled: true
      tls_cert_path: /var/run/tls/server/tls.crt
      tls_key_path: /var/run/tls/server/tls.key
      tls_ca_path: /var/run/ca/service-ca.crt
      tls_server_name: tempo-test-query-frontend.nstest.svc.cluster.local
      tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
      tls_min_version: VersionTLS12
internal_server:
  enable: true
  http_listen_address: ""
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    key_file: /var/run/tls/server/tls.key
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
  tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  tls_min_version: VersionTLS12
  grpc_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
  http_tls_config:
    cert_file: /var/run/tls/server/tls.crt
    client_auth_type: RequireAndVerifyClientCert
    client_ca_file: /var/run/ca/service-ca.crt
    key_file: /var/run/tls/server/tls.key
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
ingester_client:
  grpc_client_config:
    tls_enabled: true
    tls_cert_path: /var/run/tls/server/tls.crt
    tls_key_path: /var/run/tls/server/tls.key
    tls_ca_path: /var/run/ca/service-ca.crt
    tls_server_name: tempo-test-ingester.nstest.svc.cluster.local
    tls_insecure_skip_verify: false
    tls_cipher_suites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
    tls_min_version: VersionTLS12
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "nstest",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: 3 * time.Minute},
				Template: v1alpha1.TempoTemplateSpec{
					Gateway: v1alpha1.TempoGatewaySpec{
						Enabled: true,
					},
				},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeStatic,
			S3: &manifestutils.S3{
				Insecure: true,
				Endpoint: "minio:9000",
				Bucket:   "tempo",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			Ciphers: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"},
			MinTLSVersion: "VersionTLS12",
		},
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				HTTPEncryption: true,
				GRPCEncryption: true,
			},
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}

func TestBuildConfigurationReceiversTLS(t *testing.T) {
	testCases := []struct {
		name   string
		spec   v1alpha1.TempoDistributorSpec
		expect string
	}{
		{
			name: "receiver tls not enabled",
			spec: v1alpha1.TempoDistributorSpec{
				TLS: v1alpha1.TLSSpec{
					Enabled: false,
					Cert:    "my-cert",
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`,
		},
		{
			name: "specify cert secret name",
			spec: v1alpha1.TempoDistributorSpec{
				TLS: v1alpha1.TLSSpec{
					Enabled:    true,
					Cert:       "my-cert",
					MinVersion: string(openshiftconfigv1.VersionTLS13),
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
          endpoint: 0.0.0.0:14268
          tls:
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
          tls:
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
    zipkin:
       endpoint: 0.0.0.0:9411
       tls:
         cert_file: /var/run/tls/receiver/tls.crt
         key_file: /var/run/tls/receiver/tls.key
         min_version: VersionTLS13
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
          tls:
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13

        http:
          endpoint: "0.0.0.0:4318"
          tls:
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`,
		},
		{
			name: "specify  secret name and CA configmap",
			spec: v1alpha1.TempoDistributorSpec{
				TLS: v1alpha1.TLSSpec{
					Enabled:    true,
					Cert:       "my-cert",
					CA:         "my-ca",
					MinVersion: string(openshiftconfigv1.VersionTLS13),
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
          endpoint: 0.0.0.0:14268
          tls:
            client_ca_file: /var/run/ca-receiver/service-ca.crt
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
          tls:
            client_ca_file: /var/run/ca-receiver/service-ca.crt
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
    zipkin:
       endpoint: 0.0.0.0:9411
       tls:
         client_ca_file: /var/run/ca-receiver/service-ca.crt
         cert_file: /var/run/tls/receiver/tls.crt
         key_file: /var/run/tls/receiver/tls.key
         min_version: VersionTLS13
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
          tls:
            client_ca_file: /var/run/ca-receiver/service-ca.crt
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13

        http:
          endpoint: "0.0.0.0:4318"
          tls:
            client_ca_file: /var/run/ca-receiver/service-ca.crt
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha1.TempoStackSpec{
						Timeout: metav1.Duration{Duration: time.Minute * 3},
						Storage: v1alpha1.ObjectStorageSpec{
							Secret: v1alpha1.ObjectStorageSecretSpec{
								Type: v1alpha1.ObjectStorageSecretS3,
							},
						},
						ReplicationFactor: 1,
						Retention: v1alpha1.RetentionSpec{
							Global: v1alpha1.RetentionConfig{
								Traces: metav1.Duration{Duration: 48 * time.Hour},
							},
						},
						Template: v1alpha1.TempoTemplateSpec{
							Distributor: tc.spec,
						},
					},
				},
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeStatic,
					S3: &manifestutils.S3{
						Insecure: true,
						Endpoint: "minio:9000",
						Bucket:   "tempo",
					},
				},
				TLSProfile: tlsprofile.TLSProfileOptions{
					MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
				},
			})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}

}

func TestBuildConfigurationIPv6(t *testing.T) {
	testCases := []struct {
		name   string
		ipv6   *bool
		expect string
	}{
		{
			name: "ipv6 disabled",
			ipv6: ptr.To(false),
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
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`,
		},
		{
			name: "ipv6 enabled",
			ipv6: ptr.To(true),
			expect: `
---
compactor:
  compaction:
    block_retention: 48h0m0s
  ring:
    kvstore:
      store: memberlist
    enable_inet6: true
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
      endpoint: 0.0.0.0:9411
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
    enable_inet6: true
  max_block_duration: 10m
memberlist:
  abort_if_cluster_join_fails: false
  join_members:
    - tempo-test-gossip-ring
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildConfiguration(manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha1.TempoStackSpec{
						Timeout: metav1.Duration{Duration: 3 * time.Minute},
						Storage: v1alpha1.ObjectStorageSpec{
							Secret: v1alpha1.ObjectStorageSecretSpec{
								Type: v1alpha1.ObjectStorageSecretS3,
							},
						},
						ReplicationFactor: 1,
						Retention: v1alpha1.RetentionSpec{
							Global: v1alpha1.RetentionConfig{
								Traces: metav1.Duration{Duration: 48 * time.Hour},
							},
						},
						HashRing: v1alpha1.HashRingSpec{
							MemberList: v1alpha1.MemberListSpec{
								EnableIPv6: tc.ipv6,
							},
						},
					},
				},
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeStatic,
					S3: &manifestutils.S3{
						Insecure: true,
						Endpoint: "minio:9000",
						Bucket:   "tempo",
					},
				},
			})
			require.NoError(t, err)
			require.YAMLEq(t, tc.expect, string(cfg))
		})
	}

}

func TestBuildConfiguration_S3_short_lived(t *testing.T) {
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
      endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    local:
      path: /var/tempo/traces
    s3:
      bucket: tempo
      endpoint: "s3.us-east-2.amazonaws.com"
      insecure: true
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
    max_spans_per_span_set: 0
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Minute * 3},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeToken,
			S3: &manifestutils.S3{
				Insecure: true,
				Bucket:   "tempo",
				Region:   "us-east-2",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}

func TestBuildConfiguration_S3_short_livedSecure(t *testing.T) {
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
        endpoint: 0.0.0.0:9411
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
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 3m0s
  http_server_write_timeout: 3m0s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
    local:
      path: /var/tempo/traces
    s3:
      bucket: tempo
      endpoint: "s3.us-east-2.amazonaws.com"
      insecure: false
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
query_frontend:
  search:
    concurrent_jobs: 2000
    max_duration: 0s
    max_spans_per_span_set: 0
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Minute * 3},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeToken,
			S3: &manifestutils.S3{
				Insecure: false,
				Bucket:   "tempo",
				Region:   "us-east-2",
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}

func TestBuildConfigurationWithPodIPAddressType(t *testing.T) {
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
      endpoint: 0.0.0.0:9411
    otlp:
      protocols:
        grpc:
          endpoint: "0.0.0.0:4317"
        http:
          endpoint: "0.0.0.0:4318"
  ring:
    kvstore:
      store: memberlist
    instance_addr: ${HASH_RING_INSTANCE_ADDR}
ingester:
  lifecycler:
    ring:
      kvstore:
        store: memberlist
      replication_factor: 1
    address: ${HASH_RING_INSTANCE_ADDR}
    tokens_file_path: /var/tempo/tokens.json
  max_block_duration: 10m
memberlist:
  abort_if_cluster_join_fails: false
  join_members:
    - tempo-test-gossip-ring
  advertise_addr: ${HASH_RING_INSTANCE_ADDR}
multitenancy_enabled: false
querier:
  max_concurrent_queries: 20
  frontend_worker:
    frontend_address: "tempo-test-query-frontend-discovery:9095"
server:
  grpc_server_max_recv_msg_size: 4194304
  grpc_server_max_send_msg_size: 4194304
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
  log_format: logfmt
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
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
    max_spans_per_span_set: 0
`
	cfg, err := buildConfiguration(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Timeout: metav1.Duration{Duration: time.Second * 30},
				Storage: v1alpha1.ObjectStorageSpec{
					Secret: v1alpha1.ObjectStorageSecretSpec{
						Type: v1alpha1.ObjectStorageSecretS3,
					},
				},
				ReplicationFactor: 1,
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
				HashRing: v1alpha1.HashRingSpec{
					MemberList: v1alpha1.MemberListSpec{
						InstanceAddrType: v1alpha1.InstanceAddrPodIP,
					},
				},
			},
		},
		StorageParams: manifestutils.StorageParams{
			CredentialMode: v1alpha1.CredentialModeStatic,
			S3: &manifestutils.S3{
				Endpoint: "minio:9000",
				Bucket:   "tempo",
				Insecure: true,
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
	})
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(cfg))
}
