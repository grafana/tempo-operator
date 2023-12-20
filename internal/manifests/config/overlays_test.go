package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestApplyTempoConfigLayer(t *testing.T) {
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

	layers := make(map[string]apiextensionsv1.JSON)
	layers[tempoConfigKey] = apiextensionsv1.JSON{Raw: raw}

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
server:
  grpc_server_max_recv_msg_size: 4194304
  http_listen_port: 3200
  http_server_write_timeout: 3m
storage:
  trace:
    backend: s3
`

	layers := make(map[string]apiextensionsv1.JSON)

	result, err := applyTempoConfigLayer(layers, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, input, string(result))

}

func TestApplyTempoConfigLayerNil(t *testing.T) {
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

	result, err := applyTempoConfigLayer(nil, []byte(input))
	require.NoError(t, err)
	require.YAMLEq(t, input, string(result))

}
