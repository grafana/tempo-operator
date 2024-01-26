package config

import (
	"encoding/json"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// MergeExtraConfigWithConfig overlays configuration from overridesJSON onto templateResults.
func MergeExtraConfigWithConfig(overridesJSON apiextensionsv1.JSON, templateResults []byte) ([]byte, error) {
	// mergo.Merge requires that both variables have the same type
	renderedTemplateMap := make(map[string]interface{})
	overrides := make(map[string]interface{})

	if len(overridesJSON.Raw) == 0 {
		return templateResults, nil
	}

	// Unmarshal overlay of type []byte to map[string]interface{}
	if err := json.Unmarshal(overridesJSON.Raw, &overrides); err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(templateResults, &renderedTemplateMap); err != nil {
		return nil, err
	}

	// Override generated config with extra config
	if err := mergo.Merge(&renderedTemplateMap, overrides, mergo.WithOverride); err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(renderedTemplateMap)
	if err != nil {
		return nil, err
	}
	return data, nil
}
