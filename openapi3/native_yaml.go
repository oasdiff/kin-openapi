package openapi3

//go:generate go run nativeyamlgenerator.go

import (
	"reflect"
	"strings"
	"sync"

	yaml "go.yaml.in/yaml/v3"
)

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

// normalizeNumbers converts integers to float64 inside a decoded any.
//
// JSON has one number type, so a value reaching an any-typed field carries a
// float64 whichever notation the source used. YAML resolves 42, 0x2A and 4_2
// to an int, which would make a consumer's type switch depend on notation.
//
// Applied only to any-typed values. A declared integer field keeps its own
// type and its full range, which a blanket conversion would cost beyond 2^53.
func normalizeNumbers(v any) any {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint64:
		return float64(t)
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeNumbers(e)
		}
	case []any:
		for i, e := range t {
			t[i] = normalizeNumbers(e)
		}
	}
	return v
}

// normalizeAnyFields applies normalizeNumbers to a struct's any-typed fields,
// which the decoder fills directly: Example and Default, but also Enum, whose
// element type is any. Missing the slice case left an integer example as a
// float64 and the enum it must match as an int, so a schema failed against its
// own allowed values.
func normalizeAnyFields(out any) {
	v := reflect.ValueOf(out)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := range v.NumField() {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Interface:
			if !f.IsNil() {
				f.Set(reflect.ValueOf(normalizeNumbers(f.Interface())))
			}
		case reflect.Slice, reflect.Map:
			// []any and map[string]any: normalise the elements in place.
			if f.Type().Elem().Kind() != reflect.Interface || f.IsNil() {
				continue
			}
			if f.Kind() == reflect.Slice {
				for j := range f.Len() {
					e := f.Index(j)
					if !e.IsNil() {
						e.Set(reflect.ValueOf(normalizeNumbers(e.Interface())))
					}
				}
				continue
			}
			for _, k := range f.MapKeys() {
				e := f.MapIndex(k)
				if !e.IsNil() {
					f.SetMapIndex(k, reflect.ValueOf(normalizeNumbers(e.Interface())))
				}
			}
		}
	}
}

// decodeMapping decodes node into a method-less view of the target, supplied by
// the caller as a locally-declared shadow type, and returns the keys the target
// does not declare.
func decodeMapping[S any](node *yaml.Node, shadow *S) (map[string]any, error) {
	return decodeStructWithExtensions(node, shadow)
}

// decodeStructWithExtensions decodes node into out and returns the mapping keys
// out does not declare, which are the extensions. Returns nil rather than an
// empty map when there are none.
//
// The declared set comes from out's yaml tags, so adding a field to a struct is
// enough to stop it being collected as an extension.
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
		ext[key] = normalizeNumbers(v)
	}
	normalizeAnyFields(out)
	return ext, nil
}

// stripTimestamps retags implicitly-resolved date-shaped scalars as strings.
//
// YAML 1.1 resolves an untagged scalar such as 2020-06-11T16:32:50-03:00 to a
// timestamp, which would make an OpenAPI `example` of that shape decode to a
// time.Time and fail validation as an unhandled type. An explicit !!timestamp
// tag is a deliberate request for a time.Time and is left alone.
func stripTimestamps(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!timestamp" && n.Style != yaml.TaggedStyle {
		n.Tag = "!!str"
	}
	for _, c := range n.Content {
		stripTimestamps(c)
	}
}
