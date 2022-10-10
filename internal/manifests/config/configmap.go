package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/memberlist"
	"github.com/grafana/dskit/ring"
	tempoapp "github.com/grafana/tempo/cmd/tempo/app"
	tempodistributor "github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

const (
	kvStore = "memberlist"
)

// nolint
func config(tempo v1alpha1.Microservices) (string, error) {
	logLevel := logging.Level{}
	logLevel.Set("warn")
	logFormat := logging.Format{}
	logFormat.Set("logfmt")
	cfg := tempoapp.Config{
		Server: server.Config{
			HTTPListenPort: 3100,
			CipherSuites:   "TLS_AES_128_GCM_SHA256",
			LogFormat:      logFormat,
			LogLevel:       logLevel,
		},
		Distributor: tempodistributor.Config{
			DistributorRing: tempodistributor.RingConfig{
				KVStore: kv.Config{
					Store: kvStore,
				},
			},
			Receivers: map[string]interface{}{
				"otlp": struct {
					Protocols struct {
						Grpc struct {
							Endpoint string `yaml:"endpoint"`
						} `yaml:"grpc"`
					} `yaml:"protocols"`
				}{
					Protocols: struct {
						Grpc struct {
							Endpoint string `yaml:"endpoint"`
						} `yaml:"grpc"`
					}{
						Grpc: struct {
							Endpoint string `yaml:"endpoint"`
						}{
							Endpoint: "0.0.0.0:4317",
						},
					},
				},
			},
		},
		Ingester: ingester.Config{
			LifecyclerConfig: ring.LifecyclerConfig{
				RingConfig: ring.Config{
					KVStore: kv.Config{
						Store: kvStore,
					},
					ReplicationFactor: 1,
				},
			},
			MaxBlockDuration: time.Minute * 60,
		},
		StorageConfig: storage.Config{
			Trace: tempodb.Config{
				WAL: &wal.Config{
					Filepath: "/war/tempo/wal",
					Encoding: backend.EncSnappy,
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1000000,
					IndexPageSizeBytes:   1000000,
					BloomFP:              0.01,
					BloomShardSizeBytes:  100000,
					Version:              "v2",
					Encoding:             backend.EncZstd,
					SearchEncoding:       backend.EncSnappy,
					SearchPageSizeBytes:  1000000,
				},
				Backend: "local",
				Local: &local.Config{
					Path: "/tmp/tempo",
				},
			},
		},
		MemberlistKV: memberlist.KVConfig{
			JoinMembers: []string{fmt.Sprintf("tempo-%s-distributed-gossip-ring", tempo.Name)},
		},
	}

	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	cfgStr := string(bytes)

	// TODO(pavolloffay) find a nicer approach
	// The config struct does not define omitemty which reflects an empty fields to be rendered
	// This causes issues when Tempo is parsing the config
	// e.g. field use_flatbuffer_search not found in type ingester.Config
	re := regexp.MustCompile("(?m)[\r\n]+^.*tls_cipher_suites.*$")
	cfgStr = re.ReplaceAllString(cfgStr, "")
	re = regexp.MustCompile("(?m)[\r\n]+^.*tls_min_version.*$")
	cfgStr = re.ReplaceAllString(cfgStr, "")
	re = regexp.MustCompile("(?m)[\r\n]+^.*use_flatbuffer_search.*$")
	cfgStr = re.ReplaceAllString(cfgStr, "")
	re = regexp.MustCompile("(?m)[\r\n]+^.*metrics_ingestion_time_range_slack.*$")
	cfgStr = re.ReplaceAllString(cfgStr, "")
	// ingester.lifecycler.address
	re = regexp.MustCompile("(?m)[\r\n]+^.*address.*$")
	cfgStr = re.ReplaceAllString(cfgStr, "")
	return cfgStr, nil
}

// Params holds configuration parameters.
type Params struct {
	S3 S3
}

// S3 holds S3 object storage configuration options.
type S3 struct {
	Endpoint string
	Bucket   string
}

// BuildConfigs creates configuration objects.
func BuildConfigs(tempo v1alpha1.Microservices, params Params) (*corev1.ConfigMap, error) {
	s3Insecure := false
	s3Endpoint := params.S3.Endpoint
	if strings.HasPrefix(s3Endpoint, "http://") {
		s3Insecure = true
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "http://")
	} else if !strings.HasPrefix(s3Endpoint, "https://") {
		s3Insecure = true
	} else {
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "https://")
	}

	config := strings.Replace(hardcodedConfig, "${S3_INSECURE}", strconv.FormatBool(s3Insecure), 1)
	config = strings.Replace(config, "${S3_ENDPOINT}", s3Endpoint, 1)
	config = strings.Replace(config, "${S3_BUCKET}", params.S3.Bucket, 1)
	config = strings.Replace(config, "${MEMBERLIST}", manifestutils.Name("gossip-ring", tempo.Name), 1)
	config = strings.Replace(config, "${QUERY_FRONTEND_DISCOVERY}", fmt.Sprintf("%s:9095", manifestutils.Name("query-frontend-discovery", tempo.Name)), 1)

	labels := manifestutils.ComponentLabels("config", tempo.Name)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   manifestutils.Name("", tempo.Name),
			Labels: labels,
		},
		Data: map[string]string{
			"tempo.yaml":       config,
			"tempo-query.yaml": "backend: 127.0.0.1:3100\n",
		},
	}, nil
}

// TODO(pavolloffay) This is a temporary solution.
// The config should be created from code.
const hardcodedConfig = `
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
  - ${MEMBERLIST}
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
    frontend_address: ${QUERY_FRONTEND_DISCOVERY}
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
      endpoint: ${S3_ENDPOINT}
      bucket: ${S3_BUCKET}
      insecure: ${S3_INSECURE}
    local:
      path: /var/tempo/traces
    wal:
      path: /var/tempo/wal
usage_report:
  reporting_enabled: false
`
