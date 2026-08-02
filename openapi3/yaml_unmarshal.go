package openapi3

import (
	"reflect"
	"strings"
	"sync"

	yaml "github.com/oasdiff/yaml3"
)

// Native YAML unmarshaling, a spike. See the benchmark in yaml_unmarshal_test.go.
//
// The UnmarshalJSON methods on these types all share one shape: decode into a
// shadow struct, decode the same bytes again into a map, then delete every
// known field name from that map so what remains is the extensions. Schema's
// version is 91 lines, of which about 60 are delete calls.
//
// The YAML equivalents below are three lines each, because a *yaml.Node lets
// the unknown keys be picked out directly instead of built and subtracted, and
// because the known set is read off the struct tags rather than restated. That
// removes a standing bug: a field added to a struct but forgotten in the
// delete list silently lands in Extensions today.

var knownYAMLFieldsCache sync.Map // reflect.Type -> map[string]struct{}

// knownYAMLFields returns the yaml keys a struct type declares, skipping "-".
func knownYAMLFields(t reflect.Type) map[string]struct{} {
	if v, ok := knownYAMLFieldsCache.Load(t); ok {
		return v.(map[string]struct{})
	}
	known := make(map[string]struct{}, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		if name == "-" {
			continue
		}
		known[name] = struct{}{}
	}
	knownYAMLFieldsCache.Store(t, known)
	return known
}

// decodeStructWithExtensions decodes node into out, which must point at a
// struct, and returns the mapping keys out does not declare. Returns nil
// rather than an empty map when there are none, matching the JSON path.
func decodeStructWithExtensions(node *yaml.Node, out any) (map[string]any, error) {
	if err := node.Decode(out); err != nil {
		return nil, err
	}
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}
	known := knownYAMLFields(reflect.TypeOf(out).Elem())

	var ext map[string]any
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if _, ok := known[key]; ok {
			continue
		}
		var v any
		if err := node.Content[i+1].Decode(&v); err != nil {
			return nil, err
		}
		if ext == nil {
			ext = make(map[string]any)
		}
		ext[key] = v
	}
	return ext, nil
}

// mappingValue returns the value node for key, or nil.
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

// --- shadow-struct types -----------------------------------------------------

func (response *Response) UnmarshalYAML(node *yaml.Node) error {
	type ResponseBis Response
	var x ResponseBis
	ext, err := decodeStructWithExtensions(node, &x)
	if err != nil {
		return err
	}
	x.Extensions = ext
	x.Origin = originFromNode(node, yamlOriginFile)
	*response = Response(x)
	setChildOriginKeys(node, response, yamlOriginFile)
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
	x.Origin = originFromNode(node, yamlOriginFile)
	*mediaType = MediaType(x)
	setChildOriginKeys(node, mediaType, yamlOriginFile)
	return nil
}

// --- ref wrappers ------------------------------------------------------------

// unmarshalRefYAML fills the reference half of a $ref wrapper. It reports
// whether the node was a reference; when false the caller decodes the value.
//
// The JSON version parses the same bytes up to four times (the ref, the extra
// keys, a sibling schema, then the value). Here each is read from the node
// that is already parsed.
func unmarshalRefYAML(node *yaml.Node, ref *string, summary, description **string, extensions *map[string]any) (bool, error) {
	refNode := mappingValue(node, "$ref")
	if refNode == nil || refNode.Value == "" {
		return false, nil
	}
	*ref = refNode.Value

	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i].Value, node.Content[i+1]
		switch {
		case k == "$ref":
		case k == "summary":
			var s string
			if err := v.Decode(&s); err != nil {
				return false, err
			}
			*summary = &s
		case k == "description":
			var s string
			if err := v.Decode(&s); err != nil {
				return false, err
			}
			*description = &s
		case strings.HasPrefix(k, "x-"):
			var a any
			if err := v.Decode(&a); err != nil {
				return false, err
			}
			if *extensions == nil {
				*extensions = make(map[string]any)
			}
			(*extensions)[k] = a
		}
	}
	return true, nil
}

func (x *ResponseRef) UnmarshalYAML(node *yaml.Node) error {
	isRef, err := unmarshalRefYAML(node, &x.Ref, &x.Summary, &x.Description, &x.Extensions)
	if err != nil {
		return err
	}
	x.Origin = originFromNode(node, yamlOriginFile)
	if isRef {
		return nil
	}
	return node.Decode(&x.Value)
}

// --- maplike types -----------------------------------------------------------

// The JSON version of this re-marshals every child value back to JSON and
// re-parses it, once per entry. Here the child node is handed to the child
// decoder directly, which is where most of the saving comes from.
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
		// This parent has the key node in hand, so no reflection is needed
		// here: stamp directly. setChildOriginKeys exists for the parents that
		// delegate to the generic decoder instead of iterating.
		setOriginKey(reflect.ValueOf(&vv), node.Content[i], v, yamlOriginFile)
	}
	*responses = x
	return nil
}
