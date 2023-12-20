package config

import (
	"encoding/json"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func mergeExtraConfigWithConfig(overridesJSON apiextensionsv1.JSON, templateResults []byte) ([]byte, error) {
	renderedTemplateMap := make(map[string]interface{})
	overrides := make(map[string]interface{})

	if err := json.Unmarshal(overridesJSON.Raw, &overrides); err != nil {
		return nil, err
	}

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

func applyExtraConfigOverlay(layers map[string]apiextensionsv1.JSON, layer string, templateResults []byte) ([]byte, error) {
	if layers == nil {
		return templateResults, nil
	}

	config, ok := layers[layer]
	if !ok {
		return templateResults, nil
	}

	return mergeExtraConfigWithConfig(config, templateResults)
}

func applyTempoConfigLayer(layers map[string]apiextensionsv1.JSON, templateResults []byte) ([]byte, error) {
	return applyExtraConfigOverlay(layers, tempoConfigKey, templateResults)
}
