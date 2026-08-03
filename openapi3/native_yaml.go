package openapi3

//go:generate go run nativeyamlgenerator.go

import (
	"reflect"
	"strings"
	"sync"

	yaml "go.yaml.in/yaml/v3"
)

// Shared machinery for the UnmarshalYAML methods: extension collection, and
// origins read from the node being decoded.
//
// Origins record where an element starts, not where it ends. A consumer that
// needs the extent of a block derives it from the next key or sequence item at
// the same or shallower indentation.

// originFileVar is the file stamped into origins for the decode in progress.
// UnmarshalYAML receives a node and nothing else, so the file cannot be passed
// through the call. One decode at a time per process, as with IncludeOrigin.
var originFileVar string

// originEnabledVar mirrors the includeOrigin argument unmarshal receives, which
// comes from the Loader. The package-level IncludeOrigin only seeds NewLoader,
// so a caller that set it on its Loader alone would be missed.
var originEnabledVar bool

func nativeOriginFile() string { return originFileVar }

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
// which the decoder fills directly (Example, Default and the like).
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
		if f.Kind() == reflect.Interface && f.CanSet() && !f.IsNil() {
			f.Set(reflect.ValueOf(normalizeNumbers(f.Interface())))
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

// originFromNode builds the origin data a mapping can see for itself: where
// each of its field keys is, and where the scalar items of its sequence-valued
// fields are.
//
// Origin.Key is not set here -- it is the location of the key heading this
// mapping in its parent, which a node does not know. See setChildOriginKeys.
func originFromNode(node *yaml.Node, file string) *Origin {
	// Origins are opt-in: without this every decode pays for them.
	if !originEnabledVar {
		return nil
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	o := &Origin{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if o.Fields == nil {
			o.Fields = make(map[string]Location, len(node.Content)/2)
		}
		o.Fields[k.Value] = Location{File: file, Line: k.Line, Column: k.Column, Name: k.Value}

		if v.Kind != yaml.SequenceNode {
			continue
		}
		var locs []Location
		for _, item := range v.Content {
			if item.Kind == yaml.ScalarNode {
				locs = append(locs, Location{File: file, Line: item.Line, Column: item.Column, Name: item.Value})
			}
		}
		if len(locs) > 0 {
			if o.Sequences == nil {
				o.Sequences = make(map[string][]Location)
			}
			o.Sequences[k.Value] = locs
		}
	}
	if o.Fields == nil && o.Sequences == nil {
		return nil
	}
	return o
}

// setChildOriginKeys sets Origin.Key on the immediate children of a mapping,
// from the key node heading each one.
//
// This is the only origin data a node cannot supply for itself: UnmarshalYAML
// receives the value node, and Key is the position of the key above it. Each
// child sets its own children's keys in turn, so one level per call covers the
// tree.
func setChildOriginKeys(node *yaml.Node, container any, file string) {
	if !originEnabledVar {
		return
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	v := reflect.ValueOf(container)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode, valNode := node.Content[i], node.Content[i+1]
		child := childByKey(v, keyNode.Value)
		if !child.IsValid() {
			continue
		}
		setOriginKey(child, keyNode, file)

		switch c := deref(child); c.Kind() {
		case reflect.Map:
			// A map-valued field (Content, Headers, Links) holds children of
			// its own, keyed in valNode. The generic map decoder gives them no
			// hook of their own, so descend.
			if c.CanInterface() {
				setChildOriginKeys(valNode, c.Interface(), file)
			}
		case reflect.Slice:
			// A sequence item has no key above it, so it takes its own first
			// key as its Key.
			if valNode.Kind != yaml.SequenceNode {
				continue
			}
			for j := 0; j < len(valNode.Content) && j < c.Len(); j++ {
				item := valNode.Content[j]
				if item.Kind == yaml.MappingNode && len(item.Content) > 0 {
					setOriginKey(c.Index(j), item.Content[0], file)
				}
			}
		}
	}
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

// childByKey finds the struct field or map entry a mapping key decoded into.
func childByKey(v reflect.Value, key string) reflect.Value {
	switch v.Kind() {
	case reflect.Map:
		if v.IsNil() {
			return reflect.Value{}
		}
		return v.MapIndex(reflect.ValueOf(key))
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if name, _, _ := strings.Cut(f.Tag.Get("yaml"), ","); name == key {
				return v.Field(i)
			}
		}
	}
	return reflect.Value{}
}

// setOriginKey stamps Key on a child carrying an *Origin, from the key's own
// position. The extent of what the key heads is the consumer's to derive.
func setOriginKey(child reflect.Value, keyNode *yaml.Node, file string) {
	if !originEnabledVar {
		return
	}
	for child.Kind() == reflect.Pointer || child.Kind() == reflect.Interface {
		if child.IsNil() {
			return
		}
		child = child.Elem()
	}
	if child.Kind() != reflect.Struct {
		return
	}
	f := child.FieldByName("Origin")
	if !f.IsValid() || f.Type() != originPtrType || !f.CanSet() {
		// No origin of its own; it may still wrap something that has one.
		descendToWrapped(child, keyNode, file)
		return
	}
	if f.IsNil() {
		f.Set(reflect.ValueOf(&Origin{}))
	}
	f.Interface().(*Origin).Key = &Location{
		File:   file,
		Line:   keyNode.Line,
		Column: keyNode.Column,
		Name:   keyNode.Value,
	}
	// A wrapper and the thing it holds occupy the same node, so both carry
	// that node's origin. Value is the $ref wrappers; Schema is BoolSchema,
	// which holds either a bool or a schema.
	descendToWrapped(child, keyNode, file)
}

// descendToWrapped stamps the thing a wrapper holds, which occupies the same
// node. Value is the $ref wrappers; Schema is BoolSchema, which holds either a
// bool or a schema.
func descendToWrapped(child reflect.Value, keyNode *yaml.Node, file string) {
	if child.Kind() != reflect.Struct {
		return
	}
	for _, name := range [...]string{"Value", "Schema"} {
		if inner := child.FieldByName(name); inner.IsValid() {
			setOriginKey(inner, keyNode, file)
		}
	}
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

// stampRootOrigin gives a document root the position of the document itself.
//
// Origin.Key is normally the key heading a mapping in its parent, stamped by
// that parent. A root has none -- an externally $ref'd file may be a bare
// schema -- so it takes the root node's own position and an empty name.
// Applied only when nothing has already set Key, so a type that supplies its
// own keeps it.
func stampRootOrigin(v any, node *yaml.Node) {
	if !originEnabledVar || node == nil {
		return
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName("Origin")
	if !f.IsValid() || f.Type() != originPtrType || !f.CanSet() || f.IsNil() {
		return
	}
	o := f.Interface().(*Origin)
	if o.Key != nil {
		return
	}
	o.Key = &Location{File: nativeOriginFile(), Line: node.Line, Column: node.Column}
}
