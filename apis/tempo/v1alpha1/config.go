package v1alpha1

import (
	"bytes"
	"encoding/json"
)

// Values represent parts of the config.
type Values map[string]interface{}

// DeepCopy make a deep copy of the Values structure.
func (v *Values) DeepCopy() *Values {
	out := make(Values, len(*v))
	for key, val := range *v {
		switch val := val.(type) {
		case string:
			out[key] = val

		case []string:
			out[key] = append([]string(nil), val...)
		default:
			out[key] = val
		}
	}
	return &out
}

// Config encapsulates arbitrary config.
type Config struct {
	Raw Values `json:"-"`
}

var _ json.Marshaler = &Config{}
var _ json.Unmarshaler = &Config{}

// UnmarshalJSON implements an alternative parser for this field.
func (c *Config) UnmarshalJSON(b []byte) error {
	var entries Values
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := d.Decode(&entries); err != nil {
		return err
	}
	c.Raw = entries
	return nil
}

// MarshalJSON specifies how to convert this object into JSON.
func (c *Config) MarshalJSON() ([]byte, error) {
	if len(c.Raw) == 0 {
		return []byte("{}"), nil
	}

	return json.Marshal(c.Raw)
}

// ConfigLayers represent the different layers can be applied to different config files.
type ConfigLayers map[string]Config
