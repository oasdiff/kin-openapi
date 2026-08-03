package openapi3

// Shared by the generated $ref wrapper UnmarshalYAML methods in refs.go.

import (
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// unmarshalRefYAML fills the reference half of a wrapper and reports whether
// the node was a reference. When false the caller decodes the value.
func unmarshalRefYAML(node *yaml.Node, ref *string, summary, description **string, extensions *map[string]any) bool {
	refNode := mappingValue(node, "$ref")
	if refNode == nil || refNode.Value == "" {
		return false
	}
	*ref = refNode.Value
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i].Value, node.Content[i+1]
		switch {
		case k == "$ref":
		case k == "summary" && summary != nil:
			var s string
			if v.Decode(&s) == nil {
				*summary = &s
			}
		case k == "description" && description != nil:
			var s string
			if v.Decode(&s) == nil {
				*description = &s
			}
		case strings.HasPrefix(k, "x-"):
			var a any
			if v.Decode(&a) != nil {
				continue
			}
			if *extensions == nil {
				*extensions = make(map[string]any)
			}
			(*extensions)[k] = a
		}
	}
	return true
}
