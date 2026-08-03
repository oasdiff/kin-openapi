package openapi3

// UnmarshalYAML for the maplike collections and the union-typed scalars, which
// do not follow the shadow-struct shape.

import (
	"reflect"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// unmarshalMaplikeYAML decodes a mapping whose entries are components and whose
// x- keys are extensions, stamping each entry's origin from the key that heads
// it. The JSON version re-marshals every entry back to JSON and re-parses it,
// once per entry; here the child node goes straight to the child decoder.
func unmarshalMaplikeYAML[V any](node *yaml.Node, ext *map[string]any, out *map[string]*V) error {
	if node.Kind != yaml.MappingNode {
		return node.Decode(out)
	}
	*ext = make(map[string]any)
	*out = make(map[string]*V, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i].Value, node.Content[i+1]
		if strings.HasPrefix(k, "x-") {
			var a any
			if err := v.Decode(&a); err != nil {
				return err
			}
			(*ext)[k] = a
			continue
		}
		var vv V
		if err := v.Decode(&vv); err != nil {
			return err
		}
		(*out)[k] = &vv
		// This parent iterates, so it holds the key node and needs no
		// reflection to stamp it.
		setOriginKey(reflect.ValueOf(&vv), node.Content[i], nativeOriginFile())
	}
	return nil
}

func (responses *Responses) UnmarshalYAML(node *yaml.Node) error {
	var x Responses
	if err := unmarshalMaplikeYAML(node, &x.Extensions, &x.m); err != nil {
		return err
	}
	*responses = x
	return nil
}

func (callback *Callback) UnmarshalYAML(node *yaml.Node) error {
	var x Callback
	if err := unmarshalMaplikeYAML(node, &x.Extensions, &x.m); err != nil {
		return err
	}
	*callback = x
	return nil
}

func (paths *Paths) UnmarshalYAML(node *yaml.Node) error {
	var x Paths
	if err := unmarshalMaplikeYAML(node, &x.Extensions, &x.m); err != nil {
		return err
	}
	*paths = x
	return nil
}

// Header embeds Parameter and defers to it, as the JSON version does.
func (header *Header) UnmarshalYAML(node *yaml.Node) error {
	return header.Parameter.UnmarshalYAML(node)
}

// Types is a string or a list of strings.
func (types *Types) UnmarshalYAML(node *yaml.Node) error {
	var list []string
	if err := node.Decode(&list); err != nil {
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		list = []string{s}
	}
	*types = list
	return nil
}

// BoolSchema is `true`/`false` or a schema.
func (bs *BoolSchema) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		if node.Tag == "!!null" {
			return nil
		}
		var b bool
		if err := node.Decode(&b); err == nil {
			bs.Has = &b
			return nil
		}
	}
	var sr SchemaRef
	if err := node.Decode(&sr); err != nil {
		return err
	}
	bs.Schema = &sr
	return nil
}

// ExclusiveBound is a bool in OAS 3.0 (a modifier for min/max) or a number in
// 3.1 (the bound itself).
func (eb *ExclusiveBound) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag == "!!null" {
		return nil
	}
	var b bool
	if err := node.Decode(&b); err == nil {
		eb.Bool = &b
		return nil
	}
	var f float64
	if err := node.Decode(&f); err != nil {
		return err
	}
	eb.Value = &f
	return nil
}
