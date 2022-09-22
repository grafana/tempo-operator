package config

import (
	"github.com/os-observability/tempo-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigMap(t *testing.T) {
	t.Skip()
	cfg, err := config(v1alpha1.Microservices{})
	require.NoError(t, err)

	expected := `
metrics_generator_enabled: false
http_api_prefix: ""
server:
    http_listen_network: ""
    http_listen_address: ""
    http_listen_port: 0
    http_listen_conn_limit: 0
    grpc_listen_network: ""
    grpc_listen_address: ""
    grpc_listen_port: 0
    grpc_listen_conn_limit: 0
    tls_cipher_suites: ""
    tls_min_version: ""
    http_tls_config:
        cert_file: ""
        key_file: ""
        client_auth_type: ""
        client_ca_file: ""
    grpc_tls_config:
        cert_file: ""
        key_file: ""
        client_auth_type: ""
        client_ca_file: ""
    register_instrumentation: false
    graceful_shutdown_timeout: 0s
    http_server_read_timeout: 0s
    http_server_write_timeout: 0s
    http_server_idle_timeout: 0s
    grpc_server_max_recv_msg_size: 0
    grpc_server_max_send_msg_size: 0
    grpc_server_max_concurrent_streams: 0
    grpc_server_max_connection_idle: 0s
    grpc_server_max_connection_age: 0s
    grpc_server_max_connection_age_grace: 0s
    grpc_server_keepalive_time: 0s
    grpc_server_keepalive_timeout: 0s
    grpc_server_min_time_between_pings: 0s
    grpc_server_ping_without_stream_allowed: false
    log_format: ""
    log_level: ""
    log_source_ips_enabled: false
    log_source_ips_header: ""
    log_source_ips_regex: ""
    log_request_at_info_level_enabled: false
    http_path_prefix: ""
distributor:
    ring:
        kvstore:
            store: ""
            prefix: ""
            consul:
                host: ""
                acl_token: ""
                http_client_timeout: 0s
                consistent_reads: false
                watch_rate_limit: 0
                watch_burst_size: 0
                cas_retry_delay: 0s
            etcd:
                endpoints: []
                dial_timeout: 0s
                max_retries: 0
                tls_enabled: false
                tls_cert_path: ""
                tls_key_path: ""
                tls_ca_path: ""
                tls_server_name: ""
                tls_insecure_skip_verify: false
                username: ""
                password: ""
            multi:
                primary: ""
                secondary: ""
                mirror_enabled: false
                mirror_timeout: 0s
        heartbeat_period: 0s
        heartbeat_timeout: 0s
        instance_id: ""
        instance_interface_names: []
        instance_port: 0
        instance_addr: ""
    receivers:
        otlp:
            receiversettings: {}
            protocols:
                grpc:
                    netaddr:
                        endpoint: 0.0.0.0:4317
                        transport: ""
                    tlssetting: null
                    maxrecvmsgsizemib: 0
                    maxconcurrentstreams: 0
                    readbuffersize: 0
                    writebuffersize: 0
                    keepalive: null
                    auth: null
                    includemetadata: false
                http: null
    override_ring_key: ""
    log_received_traces: false
    extend_writes: false
    search_tags_deny_list: []
compactor:
    ring:
        kvstore:
            store: ""
            prefix: ""
            consul:
                host: ""
                acl_token: ""
                http_client_timeout: 0s
                consistent_reads: false
                watch_rate_limit: 0
                watch_burst_size: 0
                cas_retry_delay: 0s
            etcd:
                endpoints: []
                dial_timeout: 0s
                max_retries: 0
                tls_enabled: false
                tls_cert_path: ""
                tls_key_path: ""
                tls_ca_path: ""
                tls_server_name: ""
                tls_insecure_skip_verify: false
                username: ""
                password: ""
            multi:
                primary: ""
                secondary: ""
                mirror_enabled: false
                mirror_timeout: 0s
        heartbeat_period: 0s
        heartbeat_timeout: 0s
        wait_stability_min_duration: 0s
        wait_stability_max_duration: 0s
        instance_id: ""
        instance_interface_names: []
        instance_port: 0
        instance_addr: ""
        wait_active_instance_timeout: 0s
    compaction:
        chunk_size_bytes: 0
        flush_size_bytes: 0
        compaction_window: 0s
        max_compaction_objects: 0
        max_block_bytes: 0
        block_retention: 0s
        compacted_block_retention: 0s
        retention_concurrency: 0
        iterator_buffer_size: 0
        max_time_per_tenant: 0s
        compaction_cycle: 0s
    override_ring_key: ""
ingester:
    lifecycler:
        ring:
            kvstore:
                store: ""
                prefix: ""
                consul:
                    host: ""
                    acl_token: ""
                    http_client_timeout: 0s
                    consistent_reads: false
                    watch_rate_limit: 0
                    watch_burst_size: 0
                    cas_retry_delay: 0s
                etcd:
                    endpoints: []
                    dial_timeout: 0s
                    max_retries: 0
                    tls_enabled: false
                    tls_cert_path: ""
                    tls_key_path: ""
                    tls_ca_path: ""
                    tls_server_name: ""
                    tls_insecure_skip_verify: false
                    username: ""
                    password: ""
                multi:
                    primary: ""
                    secondary: ""
                    mirror_enabled: false
                    mirror_timeout: 0s
            heartbeat_timeout: 0s
            replication_factor: 0
            zone_awareness_enabled: false
            excluded_zones: ""
        num_tokens: 0
        heartbeat_period: 0s
        heartbeat_timeout: 0s
        observe_period: 0s
        join_after: 0s
        min_ready_duration: 0s
        interface_names: []
        final_sleep: 0s
        tokens_file_path: ""
        availability_zone: ""
        unregister_on_shutdown: false
        readiness_check_ring_health: false
        address: ""
        port: 0
        id: ""
    concurrent_flushes: 0
    flush_check_period: 0s
    flush_op_timeout: 0s
    trace_idle_period: 0s
    max_block_duration: 0s
    max_block_bytes: 0
    complete_block_timeout: 0s
    override_ring_key: ""
    use_flatbuffer_search: false
metrics_generator:
    ring:
        kvstore:
            store: ""
            prefix: ""
            consul:
                host: ""
                acl_token: ""
                http_client_timeout: 0s
                consistent_reads: false
                watch_rate_limit: 0
                watch_burst_size: 0
                cas_retry_delay: 0s
            etcd:
                endpoints: []
                dial_timeout: 0s
                max_retries: 0
                tls_enabled: false
                tls_cert_path: ""
                tls_key_path: ""
                tls_ca_path: ""
                tls_server_name: ""
                tls_insecure_skip_verify: false
                username: ""
                password: ""
            multi:
                primary: ""
                secondary: ""
                mirror_enabled: false
                mirror_timeout: 0s
        heartbeat_period: 0s
        heartbeat_timeout: 0s
        instance_id: ""
        instance_interface_names: []
        instance_addr: ""
    processor:
        service_graphs:
            wait: 0s
            max_items: 0
            workers: 0
            histogram_buckets: []
            dimensions: []
        span_metrics:
            histogram_buckets: []
            dimensions: []
    registry:
        collection_interval: 0s
        stale_duration: 0s
    storage:
        path: ""
        wal:
            wal_segment_size: 0
            wal_compression: false
            stripe_size: 0
            truncate_frequency: 0s
            min_wal_time: 0
            max_wal_time: 0
            no_lockfile: false
        remote_write_flush_deadline: 0s
    metrics_ingestion_time_range_slack: 0s
`
	assert.Equal(t, expected, cfg)
}
