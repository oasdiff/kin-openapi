package openapi3

import (
	"reflect"
	"strings"
	"sync"

	yaml "go.yaml.in/yaml/v3"
)

// Native YAML decoding against stock go.yaml.in/yaml/v3 -- no fork, no patches.
//
// Today a YAML document is decoded to map[string]any, re-serialized to JSON
// text and parsed again, because these types implement UnmarshalJSON rather
// than UnmarshalYAML. Positions cannot survive that, so they are smuggled
// through it as synthetic __origin__ nodes and reapplied afterwards by a
// reflection walk over a separately-built tree.
//
// Decoding from the node directly removes both. The node carries Line and
// Column, which stock go-yaml has always had, so origins are read off it.
//
// End positions are deliberately not used. A block's extent is recoverable
// from start positions alone -- it runs to the line before the next key or
// sequence item at the same or shallower indentation -- which was measured to
// agree with recorded end positions on ~99.98% of ~11.9M spans, the remainder
// being a trailing blank-or-comment boundary convention. That is what lets
// this run on the stock parser.

// originFileVar is the file stamped into origins for the decode in progress.
//
// UnmarshalYAML receives a node and nothing else, so the file cannot be
// threaded through the call. This follows the precedent of IncludeOrigin,
// which is already a package-level decode setting, and inherits its
// concurrency characteristics: one decode at a time per process. Making both
// per-Loader is worth doing, but is a separate change to a public API.
var originFileVar string

// originEnabledVar mirrors the includeOrigin argument unmarshal receives, which
// comes from the Loader rather than from the package-level IncludeOrigin.
// Gating on the global would miss a caller that set it only on its Loader.
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

// decodeStructWithExtensions decodes node into out and returns the mapping keys
// out does not declare. nil rather than an empty map when there are none,
// matching the JSON path.
//
// The JSON versions of this build the whole object as a map and then delete
// every known name from it -- Schema.UnmarshalJSON is 91 lines, about 60 of
// them deletes. Reading the known set off the struct tags means the list
// cannot drift from the struct, which it silently can today.
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

// originFromNode builds the origin data a mapping can see for itself: where
// each of its field keys is, and where the scalar items of its sequence-valued
// fields are.
//
// Origin.Key is not set here -- it is the location of the key heading this
// mapping in its parent, which a node does not know. See setChildOriginKeys.
func originFromNode(node *yaml.Node, file string) *Origin {
	// Origins are opt-in. Without this every decode pays for them and every
	// consumer sees positions it did not ask for.
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
// from the key node heading each one. This is the only origin data that cannot
// be read locally: UnmarshalYAML receives the value node, not the key above it.
//
// One field, one level. Children stamp their own children, so the tree is
// covered without anyone walking it -- unlike applyOrigins, which rebuilds the
// whole tree in parallel with a separately-built OriginTree.
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
			// its own, each keyed in valNode. They are decoded by the generic
			// map decoder, which has no hook to stamp them, so descend here.
			if c.CanInterface() {
				setChildOriginKeys(valNode, c.Interface(), file)
			}
		case reflect.Slice:
			// A sequence item has no key above it, so it takes its own first
			// key as its Key -- the same choice the existing origin code makes
			// ("in case of a sequence, we use the first element as the key").
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

// setOriginKey stamps Key on a child carrying an *Origin. Only the key's own
// position: the extent of what it heads is the consumer's to derive from the
// next boundary, which is what removes the need for a patched parser.
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
	// A $ref wrapper and the value it holds occupy the same node, so both
	// carry that node's origin -- which is what applyOrigins produces today.
	if inner := child.FieldByName("Value"); inner.IsValid() {
		setOriginKey(inner, keyNode, file)
	}
}

// stripTimestamps retags date-shaped scalars as strings.
//
// YAML 1.1 resolves an untagged scalar like 2020-06-11T16:32:50-03:00 to a
// timestamp, so an OpenAPI `example` of that shape decodes to a time.Time and
// then fails validation as an unhandled type. The previous decode path avoided
// this with a DisableTimestamps option on our yaml fork; stock go-yaml has no
// such option, so the same effect is had by retagging before decoding.
//
// Explicit !!timestamp tags in the source are left alone: those are a
// deliberate request for a time.Time, which is what the fork's option did too.
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
