package openapi3

import (
	"reflect"

	yaml "github.com/oasdiff/yaml3"
)

// yamlOriginFile is the file name stamped into origins. In the real thing the
// loader supplies it per document; the spike has one file.
const yamlOriginFile = ""

// Origins read straight off the node tree, part of the native-YAML spike.
//
// Today positions reach these types the long way round: yaml3 encodes them
// into synthetic __origin__ nodes, the decoder turns those into data, the yaml
// wrapper extracts them into an OriginTree, and applyOrigins walks that tree
// against the struct tree to put them back. That exists because the JSON round
// trip destroys the node tree, so positions have to survive as data.
//
// Under native decoding the node is right here, so almost all of it is just
// read off. One piece is not.

// originFromNode builds the origin data a mapping can see for itself: the
// location of each of its field keys, and of the scalar items in its
// sequence-valued fields.
//
// Origin.Key is deliberately not set here. It is the location of the key that
// heads this mapping in its *parent* -- the `/pets:` line above an operation,
// not the operation's own first line -- and a node does not know its own key.
// See setChildOriginKeys.
func originFromNode(node *yaml.Node, file string) *Origin {
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
// be read locally, because UnmarshalYAML receives the value node and not the
// key above it.
//
// It is a reflection walk, which is what applyOrigins is too, but a much
// smaller one: it sets one field, and it descends exactly one level. Every
// child calls it for its own children, so the tree is covered without anyone
// walking it. applyOrigins by contrast reconstructs the whole tree in parallel
// with a separately-built OriginTree, which is where its 136 lines go.
func setChildOriginKeys(node *yaml.Node, container any, file string) {
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
	}
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
			if name, _, _ := cutTag(f.Tag.Get("yaml")); name == key {
				return v.Field(i)
			}
		}
	}
	return reflect.Value{}
}

// setOriginKey stamps Key on a child that carries an *Origin, using the key
// node for the position and the value node for the block's extent -- so the
// span runs from the key line to the end of what it heads, which is what a
// consumer slicing source needs.
func setOriginKey(child reflect.Value, keyNode, valNode *yaml.Node, file string) {
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
		File:      file,
		Line:      keyNode.Line,
		Column:    keyNode.Column,
		Name:      keyNode.Value,
		EndLine:   valNode.EndLine,
		EndColumn: valNode.EndColumn,
	}

	// A $ref wrapper and the value it holds occupy the same node, so both
	// carry that node's origin -- which is what the current applyOrigins walk
	// produces, since it sets Origin on every Origin-bearing struct at a
	// position. Descend so the two designs agree.
	if inner := child.FieldByName("Value"); inner.IsValid() {
		setOriginKey(inner, keyNode, valNode, file)
	}
}

func cutTag(tag string) (name, rest string, found bool) {
	for i := range len(tag) {
		if tag[i] == ',' {
			return tag[:i], tag[i+1:], true
		}
	}
	return tag, "", false
}
