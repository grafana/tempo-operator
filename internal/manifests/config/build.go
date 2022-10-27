package config

import (
	"bytes"
	"embed"
	"html/template"
	"io"
)

const (

	// TempoConfigFileName is the name of the config file in the configmap.
	TempoConfigFileName = "config.yaml"
)

var (
	//go:embed tempo-config.yaml
	tempoConfigYAMLTmplFile embed.FS
	tempoConfigYAMLTmpl     = template.Must(template.ParseFS(tempoConfigYAMLTmplFile, "tempo-config.yaml"))
)

// Build builds a tempo configuration.
func buildConfiguration(opts Options) ([]byte, error) {
	// Build Tempo config yaml
	w := bytes.NewBuffer(nil)
	err := tempoConfigYAMLTmpl.Execute(w, opts)
	if err != nil {
		return nil, err
	}
	cfg, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
