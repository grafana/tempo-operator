package config

import (
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func mergeExtraConfigWithConfig(overrides map[string]interface{}, templateResults []byte) ([]byte, error) {
	renderedTemplateMap := make(map[string]interface{})

	if err := yaml.Unmarshal(templateResults, &renderedTemplateMap); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&renderedTemplateMap, overrides, mergo.WithOverride); err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(renderedTemplateMap)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func applyExtraConfigOverlay(layers v1alpha1.ConfigLayers, layer string, templateResults []byte) ([]byte, error) {
	if layers == nil {
		return templateResults, nil
	}

	config, ok := layers[layer]
	if !ok {
		return templateResults, nil
	}

	return mergeExtraConfigWithConfig(config.Raw, templateResults)
}

func applyTempoConfigLayer(layers v1alpha1.ConfigLayers, templateResults []byte) ([]byte, error) {
	return applyExtraConfigOverlay(layers, tempoConfigKey, templateResults)
}
