package openapi3

import (
	"reflect"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// UnmarshalYAML implementations for one connected subtree, decoding from the
// node instead of via JSON text. Each is three lines of real work because the
// known field set comes off the struct tags.
//
// nativeOriginFile is the file stamped into origins; the loader supplies it
// per document in the real thing.
const nativeOriginFile = ""

func (response *Response) UnmarshalYAML(node *yaml.Node) error {
	type ResponseBis Response
	var x ResponseBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile)
	*response = Response(x)
	setChildOriginKeys(node, response, nativeOriginFile)
	return nil
}

func (mediaType *MediaType) UnmarshalYAML(node *yaml.Node) error {
	type MediaTypeBis MediaType
	var x MediaTypeBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, nativeOriginFile)
	*mediaType = MediaType(x)
	setChildOriginKeys(node, mediaType, nativeOriginFile)
	return nil
}

// UnmarshalYAML for a $ref wrapper. The JSON version parses the same bytes up
// to four times -- the ref, the extra keys, a sibling schema, the value. Here
// each is read from the node that is already parsed.
func (x *ResponseRef) UnmarshalYAML(node *yaml.Node) error {
	refNode := mappingValue(node, "$ref")
	x.Origin = originFromNode(node, nativeOriginFile)
	if refNode == nil || refNode.Value == "" {
		return node.Decode(&x.Value)
	}
	x.Ref = refNode.Value
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i].Value, node.Content[i+1]
		switch {
		case k == "$ref":
		case k == "summary":
			var s string
			if err := v.Decode(&s); err != nil {
				return err
			}
			x.Summary = &s
		case k == "description":
			var s string
			if err := v.Decode(&s); err != nil {
				return err
			}
			x.Description = &s
		case strings.HasPrefix(k, "x-"):
			var a any
			if err := v.Decode(&a); err != nil {
				return err
			}
			if x.Extensions == nil {
				x.Extensions = make(map[string]any)
			}
			x.Extensions[k] = a
		}
	}
	return nil
}

// The JSON version re-marshals every child back to JSON and re-parses it, once
// per entry. Here the child node goes straight to the child decoder.
func (responses *Responses) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return node.Decode(&responses.m)
	}
	x := Responses{
		Extensions: make(map[string]any),
		m:          make(map[string]*ResponseRef, len(node.Content)/2),
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i].Value, node.Content[i+1]
		if strings.HasPrefix(k, "x-") {
			var a any
			if err := v.Decode(&a); err != nil {
				return err
			}
			x.Extensions[k] = a
			continue
		}
		var vv ResponseRef
		if err := v.Decode(&vv); err != nil {
			return err
		}
		x.m[k] = &vv
		// This parent iterates, so it has the key node and needs no reflection.
		setOriginKey(reflect.ValueOf(&vv), node.Content[i], nativeOriginFile)
	}
	*responses = x
	return nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
