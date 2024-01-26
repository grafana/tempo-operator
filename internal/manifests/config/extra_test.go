package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestApplyTempoExtraConfig(t *testing.T) {
	input := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
server:
  grpc_server_max_recv_msg_size: 4194304
  http_listen_port: 3200
  http_server_write_timeout: 3m
storage:
  trace:
    backend: s3
`

	expCfg := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
server:
  grpc_server_max_recv_msg_size: 128
  grpc_server_max_send_msg_size: 128
  http_listen_port: 3200
  http_server_write_timeout: 3m
storage:
  trace:
    backend: s3
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
	raw, err := json.Marshal(rawInput)
	require.NoError(t, err)

	extraConfig := apiextensionsv1.JSON{Raw: raw}

	result, err := MergeExtraConfigWithConfig(extraConfig, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, expCfg, string(result))

}

func TestApplyTempoExtraConfigEmpty(t *testing.T) {
	input := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
server:
  grpc_server_max_recv_msg_size: 4194304
  http_listen_port: 3200
  http_server_write_timeout: 3m
storage:
  trace:
    backend: s3
`
	extraConfig := apiextensionsv1.JSON{}

	result, err := MergeExtraConfigWithConfig(extraConfig, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, input, string(result))
}

func TestApplyTempoExtraConfigInvalid(t *testing.T) {
	input := `
---
compactor:
  compaction:
    block_retention: 48h0m0s
server:
  grpc_server_max_recv_msg_size: 4194304
  http_listen_port: 3200
  http_server_write_timeout: 3m
storage:
  trace:
    backend: s3
`
	extraConfig := apiextensionsv1.JSON{
		Raw: []byte("{{{{}"),
	}

	_, err := MergeExtraConfigWithConfig(extraConfig, []byte(input))
	require.Error(t, err)
}
