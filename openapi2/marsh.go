package openapi2

import (
	"encoding/json"
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

func unmarshalError(jsonUnmarshalErr error) error {
	if before, after, found := strings.Cut(jsonUnmarshalErr.Error(), "Bis"); found && before != "" && after != "" {
		before = strings.ReplaceAll(before, " Go struct ", " ")
		return fmt.Errorf("%s%s", before, strings.ReplaceAll(after, "Bis", ""))
	}
	return jsonUnmarshalErr
}

func unmarshal(data []byte, v any) error {
	var jsonErr, yamlErr error

	// See https://github.com/getkin/kin-openapi/issues/680
	if jsonErr = json.Unmarshal(data, v); jsonErr == nil {
		return nil
	}

	// YAML reaches these types through JSON, since they implement
	// UnmarshalJSON and not UnmarshalYAML. See prepareForJSON for what has to
	// be reconciled first.
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err == nil {
		prepareForJSON(&root)
		var generic any
		if err := root.Decode(&generic); err == nil {
			if j, err := json.Marshal(generic); err == nil {
				if yamlErr = json.Unmarshal(j, v); yamlErr == nil {
					return nil
				}
			} else {
				yamlErr = err
			}
		} else {
			yamlErr = err
		}
	} else {
		yamlErr = err
	}

	// If both unmarshaling attempts fail, return a new error that includes both errors
	return fmt.Errorf("failed to unmarshal data: json error: %v, yaml error: %v", jsonErr, yamlErr)
}

// prepareForJSON retags the two things YAML resolves that JSON cannot carry.
//
// A date-shaped scalar resolves to a timestamp, which has no JSON form. And a
// mapping key that is not a string -- the unquoted 200: written in most specs
// for a status code -- decodes to a map[any]any that json.Marshal rejects.
// Both are retagged as strings, an explicit tag being left alone.
func prepareForJSON(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!timestamp" && n.Style != yaml.TaggedStyle {
		n.Tag = "!!str"
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			if k := n.Content[i]; k.Kind == yaml.ScalarNode && k.Tag != "!!str" {
				k.Tag = "!!str"
			}
		}
	}
	for _, c := range n.Content {
		prepareForJSON(c)
	}
}

// UnmarshalFromData loads a document from swagger 2.0 bytes in either JSON or
// YAML. The v3 side has Loader for this; here the whole job is the decode.
func UnmarshalFromData(data []byte, doc *T) error {
	return unmarshal(data, doc)
}
