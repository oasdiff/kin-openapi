package openapi3

// UnmarshalYAML for the $ref wrappers. A node holding a $ref carries the
// reference, and may carry summary, description and extensions alongside it;
// anything else is the value.

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

func (x *CallbackRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *ExampleRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *HeaderRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *LinkRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *ParameterRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *RequestBodyRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *ResponseRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

func (x *SecuritySchemeRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions) {
		return nil
	}
	return node.Decode(&x.Value)
}

// SchemaRef takes no summary or description. OAS 3.1 allows schema keywords
// alongside a $ref, which are held on sibling until the reference resolves and
// they can be merged into the resolved value.
func (x *SchemaRef) UnmarshalYAML(node *yaml.Node) error {
	x.Origin = originFromNode(node, nativeOriginFile())
	if !unmarshalRefYAML(node, &x.Ref, nil, nil, &x.Extensions) {
		return node.Decode(&x.Value)
	}
	var siblings []string
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i].Value
		if k == "$ref" {
			continue
		}
		x.extra = append(x.extra, k)
		if !strings.HasPrefix(k, "x-") {
			siblings = append(siblings, k)
		}
	}
	if len(siblings) > 0 {
		var sibling Schema
		if err := node.Decode(&sibling); err == nil {
			x.sibling = &sibling
		}
	}
	return nil
}
