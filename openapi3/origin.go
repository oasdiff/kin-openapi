package openapi3

// Origin records where each element of a document came from: the position of
// the key that heads a collection, of each of its fields, and of the scalar
// items in its sequence-valued fields.
//
// The positions are read from the nodes as the document decodes. A node knows
// where it starts, so the only piece it cannot supply is Key -- the key above
// it belongs to the parent, which stamps it.

import (
	"reflect"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

var originPtrType = reflect.TypeFor[*Origin]()

// Origin contains the origin of a collection.
// Key is the location of the collection itself.
// Fields is a map of the location of each scalar field in the collection.
// Sequences is a map of the location of each item in sequence-valued fields.
type Origin struct {
	Key       *Location             `json:"key,omitempty" yaml:"key,omitempty"`
	Fields    map[string]Location   `json:"fields,omitempty" yaml:"fields,omitempty"`
	Sequences map[string][]Location `json:"sequences,omitempty" yaml:"sequences,omitempty"`
}

// Location is a struct that contains the location of a field.
type Location struct {
	File   string `json:"file,omitempty" yaml:"file,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
	Column int    `json:"column,omitempty" yaml:"column,omitempty"`
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`

	// EndLine and EndColumn mark the end of the block this location heads (set
	// only on Origin.Key). For an operation or schema this spans the whole
	// block, so a consumer can extract the entire element from its source.
	// Both are zero when the underlying YAML carried no end information.
	EndLine   int `json:"endLine,omitempty" yaml:"endLine,omitempty"`
	EndColumn int `json:"endColumn,omitempty" yaml:"endColumn,omitempty"`
}

// originTree aliases the decoder-side origin tree, so the loader and marsh can
// carry it without referencing the yaml package directly.
type originTree = yaml.Node

// originFileVar is the file stamped into origins for the decode in progress.
// UnmarshalYAML receives a node and nothing else, so the file cannot be passed
// through the call. One decode at a time per process, as with IncludeOrigin.
var originFileVar string

// originEnabledVar mirrors the includeOrigin argument unmarshal receives, which
// comes from the Loader. The package-level IncludeOrigin only seeds NewLoader,
// so a caller that set it on its Loader alone would be missed.
var originEnabledVar bool

// Shared machinery for the UnmarshalYAML methods: extension collection, and
// origins read from the node being decoded.
//
// Origins record where an element starts, not where it ends. A consumer that
// needs the extent of a block derives it from the next key or sequence item at
// the same or shallower indentation.

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
		setOriginKey(child, keyNode, valNode, file)
		recordScalarMapKeys(v, child, keyNode, valNode, file)

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
					setOriginKey(c.Index(j), item.Content[0], item, file)
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
func setOriginKey(child reflect.Value, keyNode, valNode *yaml.Node, file string) {
	if !originEnabledVar {
		return
	}
	keyNode, valNode = resolveAlias(keyNode, valNode)
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
		descendToWrapped(child, keyNode, valNode, file)
		return
	}
	if f.IsNil() {
		f.Set(reflect.ValueOf(&Origin{}))
	}
	key := withEnd(Location{
		File:   file,
		Line:   keyNode.Line,
		Column: keyNode.Column,
		Name:   keyNode.Value,
	}, valNode)
	f.Interface().(*Origin).Key = &key
	// A wrapper and the thing it holds occupy the same node, so both carry
	// that node's origin. Value is the $ref wrappers; Schema is BoolSchema,
	// which holds either a bool or a schema.
	descendToWrapped(child, keyNode, valNode, file)
}

// descendToWrapped stamps the thing a wrapper holds, which occupies the same
// node. Value is the $ref wrappers; Schema is BoolSchema, which holds either a
// bool or a schema.
func descendToWrapped(child reflect.Value, keyNode, valNode *yaml.Node, file string) {
	if child.Kind() != reflect.Struct {
		return
	}
	for _, name := range [...]string{"Value", "Schema"} {
		if inner := child.FieldByName(name); inner.IsValid() {
			setOriginKey(inner, keyNode, valNode, file)
		}
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
	key := withEnd(Location{File: nativeOriginFile(), Line: node.Line, Column: node.Column}, node)
	o.Key = &key
}

// recordScalarMapKeys records where each key of a scalar-valued map sits.
//
// A map[string]string -- scopes on an OAuth flow, say -- decodes to a plain Go
// map with nowhere to hang an Origin of its own, so its keys are recorded on
// the enclosing struct's Origin under the field name, sorted by key.
func recordScalarMapKeys(container, child reflect.Value, keyNode, valNode *yaml.Node, file string) {
	if child.Kind() != reflect.Map || valNode == nil || valNode.Kind != yaml.MappingNode {
		return
	}
	// A map of structs or pointers carries origins on its values instead.
	switch child.Type().Elem().Kind() {
	case reflect.Struct, reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return
	}
	f := container.FieldByName("Origin")
	if !f.IsValid() || f.Type() != originPtrType || !f.CanSet() {
		return
	}
	if f.IsNil() {
		f.Set(reflect.ValueOf(&Origin{}))
	}
	var locs []Location
	for i := 0; i+1 < len(valNode.Content); i += 2 {
		k := valNode.Content[i]
		locs = append(locs, Location{File: file, Line: k.Line, Column: k.Column, Name: k.Value})
	}
	if len(locs) == 0 {
		return
	}
	sort.Slice(locs, func(i, j int) bool { return locs[i].Name < locs[j].Name })
	o := f.Interface().(*Origin)
	if o.Sequences == nil {
		o.Sequences = make(map[string][]Location)
	}
	o.Sequences[keyNode.Value] = locs
}
