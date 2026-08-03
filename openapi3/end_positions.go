package openapi3

import (
	"bytes"

	yaml "go.yaml.in/yaml/v3"
)

// endIndex answers where the block headed by a key ends.
//
// The parser reports where every node starts and nothing about where it stops,
// so the end is derived: a block ends on the last line any part of it occupies,
// which is the largest line among the node's descendants.
//
// Reading it off the tree rather than from indentation is what makes a sequence
// item work. A parameter's key location is the item's first key, which sits at
// the same column as the keys following it, so no column comparison can tell
// the end of the item from the start of its own second field. Its subtree ends
// where the item ends either way.
//
// Trailing blank and comment lines fall outside, carrying no node.
type endIndex struct {
	end map[*yaml.Node]int
	// anchorKey is the key heading each anchored node, so an alias can report
	// where the content it points at is defined.
	anchorKey map[*yaml.Node]*yaml.Node
	// lineLen is each line's length, giving the column a block's last line
	// stops at.
	lineLen []int
}

// originEndsVar is the index for the decode in progress, alongside
// originFileVar. Ends are derived per file: a $ref into another file is decoded
// separately, against its own text.
var originEndsVar *endIndex

// newEndIndex indexes data's node tree. Returns nil when origins are off, in
// which case no end is ever asked for.
func newEndIndex(root *yaml.Node, data []byte) *endIndex {
	if !originEnabledVar || root == nil {
		return nil
	}
	ei := &endIndex{end: map[*yaml.Node]int{}, anchorKey: map[*yaml.Node]*yaml.Node{}}
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		ei.lineLen = append(ei.lineLen, len(bytes.TrimRight(line, "\r")))
	}
	ei.measure(root)
	return ei
}

// measure records each node's last line, bottom up, and returns it.
func (ei *endIndex) measure(n *yaml.Node) int {
	if n == nil {
		return 0
	}
	last := n.Line
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if v := n.Content[i+1]; v.Anchor != "" {
				ei.anchorKey[v] = n.Content[i]
			}
		}
	}
	for _, c := range n.Content {
		if l := ei.measure(c); l > last {
			last = l
		}
	}
	ei.end[n] = last
	return last
}

// endOf returns the last line and column of the block node occupies.
func (ei *endIndex) endOf(node *yaml.Node) (int, int) {
	if ei == nil || node == nil {
		return 0, 0
	}
	line, ok := ei.end[node]
	if !ok || line < 1 || line > len(ei.lineLen) {
		return 0, 0
	}
	// The column just past the last character, matching how a parser reports
	// the position it stopped at.
	return line, ei.lineLen[line-1] + 1
}

// withEnd returns loc carrying the extent of the block node heads.
func withEnd(loc Location, node *yaml.Node) Location {
	loc.EndLine, loc.EndColumn = originEndsVar.endOf(node)
	return loc
}

// resolveAlias follows an alias to the node it points at, together with the key
// that node was defined under. An aliased schema is the anchored one, so its
// origin is where that content was written, not where it was referred to.
func resolveAlias(keyNode, valNode *yaml.Node) (*yaml.Node, *yaml.Node) {
	if valNode == nil || valNode.Kind != yaml.AliasNode || valNode.Alias == nil {
		return keyNode, valNode
	}
	if originEndsVar == nil {
		return keyNode, valNode.Alias
	}
	if k, ok := originEndsVar.anchorKey[valNode.Alias]; ok {
		return k, valNode.Alias
	}
	return keyNode, valNode.Alias
}
