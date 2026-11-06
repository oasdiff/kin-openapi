// Package yamlconv converts between YAML and types that describe themselves
// with json tags and MarshalJSON/UnmarshalJSON methods.
//
// YAML reaches such a type through JSON, so its methods run and extensions are
// carried. Two things have to be reconciled first, both of which YAML resolves
// and JSON cannot represent.
package yamlconv

import (
	"encoding/json"

	"go.yaml.in/yaml/v4"
)

// PrepareForJSON retags the scalars that would not survive the conversion.
//
// A date-shaped scalar resolves to a timestamp, which has no JSON form, so it
// is retagged and keeps the text the document gave it.
//
// A mapping key resolves too: the unquoted 200: written in most specs for a
// status code is an integer, 18_24 is 1824 and .0 is the number zero.
// Retagging the key names the member by the text that spells it, which is the
// name the typed decode reads. JSON has no other name to offer: encoding/json
// refuses a map[any]any, and encoding/json/v2 refuses one holding a float key
// and renumbers an integer one, 18_24 to 1824 and 0x2A to 42.
//
// An explicitly tagged value is left alone, being a deliberate request for
// that type.
//
// Interim: https://github.com/yaml/go-yaml/issues/452 asks for an option to
// leave a timestamp-shaped scalar a string, which removes the first retag. The
// second needs the same for a key's tag.
func PrepareForJSON(n *yaml.Node) {
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
		PrepareForJSON(c)
	}
}

// Unmarshal decodes YAML into v via JSON, so v's UnmarshalJSON runs.
func Unmarshal(data []byte, v any) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	PrepareForJSON(&root)
	var generic any
	if err := root.Decode(&generic); err != nil {
		return err
	}
	j, err := json.Marshal(generic)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, v)
}

// Marshal renders v as YAML via JSON, so v's MarshalJSON runs.
func Marshal(v any) ([]byte, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(j, &generic); err != nil {
		return nil, err
	}
	return yaml.Marshal(generic)
}
