package openapi3

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	kinyaml "github.com/oasdiff/yaml"
	goyaml "go.yaml.in/yaml/v3"
)

const nativeSrc = `"200":
  description: ok
  x-tags:
    - alpha
    - beta
  content:
    application/json:
      x-media: 1
"404":
  $ref: '#/components/responses/NotFound'
  summary: missing
x-collection: top
`

// Decoding from the node, on the stock parser, must produce the same document
// as the JSON round trip does.
func TestNativeStock_MatchesJSONPath(t *testing.T) {
	var viaJSON Responses
	jsonBytes, err := yamlToJSON(nativeSrc)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(jsonBytes, &viaJSON))

	var viaNode Responses
	require.NoError(t, goyaml.Unmarshal([]byte(nativeSrc), &viaNode))

	want, err := json.Marshal(&viaJSON)
	require.NoError(t, err)
	got, err := json.Marshal(&viaNode)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
}

// And the origins must match what the current path reconstructs -- except for
// end positions, which this design deliberately does not record. The extent of
// a block is derivable from the next boundary, which is what lets this run on
// an unpatched parser.
func TestNativeStock_OriginsMatchExceptEnds(t *testing.T) {
	defer func(v bool) { originEnabledVar = v }(originEnabledVar)
	originEnabledVar = true

	var viaTree Responses
	tree, err := kinyaml.Unmarshal([]byte(nativeSrc), &viaTree, kinyaml.DecodeOpts{
		Origin: kinyaml.OriginOpt{Enabled: true},
	})
	require.NoError(t, err)
	applyOrigins(&viaTree, tree)

	var viaNode Responses
	require.NoError(t, goyaml.Unmarshal([]byte(nativeSrc), &viaNode))

	want := viaTree.Value("200").Value.Origin
	got := viaNode.Value("200").Value.Origin
	require.NotNil(t, want)
	require.NotNil(t, got)

	// Key: the `"200":` line.
	require.Equal(t, want.Key.Line, got.Key.Line, "Key.Line")
	require.Equal(t, want.Key.Column, got.Key.Column, "Key.Column")
	require.Equal(t, want.Key.Name, got.Key.Name, "Key.Name")

	// Fields and Sequences, in full.
	require.NotEmpty(t, want.Fields)
	require.Equal(t, len(want.Fields), len(got.Fields), "field count")
	for name, w := range want.Fields {
		g, ok := got.Fields[name]
		require.True(t, ok, "field %q", name)
		require.Equal(t, w.Line, g.Line, "field %q line", name)
		require.Equal(t, w.Column, g.Column, "field %q column", name)
	}
	require.NotEmpty(t, want.Sequences)
	require.Equal(t, len(want.Sequences), len(got.Sequences), "sequence count")
	for f, wl := range want.Sequences {
		gl, ok := got.Sequences[f]
		require.True(t, ok, "sequence %q", f)
		require.Equal(t, len(wl), len(gl))
		for i := range wl {
			require.Equal(t, wl[i].Line, gl[i].Line, "%s[%d]", f, i)
			require.Equal(t, wl[i].Name, gl[i].Name, "%s[%d]", f, i)
		}
	}

	// End positions are absent by design: the stock parser does not record
	// them and the consumer derives extents from the next boundary.
	require.Zero(t, got.Key.EndLine, "the stock parser records no end position")
}

// The origin has to reach the nested media type, not just the top level.
func TestNativeStock_OriginsAtDepth(t *testing.T) {
	defer func(v bool) { originEnabledVar = v }(originEnabledVar)
	originEnabledVar = true

	var r Responses
	require.NoError(t, goyaml.Unmarshal([]byte(nativeSrc), &r))
	mt := r.Value("200").Value.Content["application/json"]
	require.NotNil(t, mt)
	require.NotNil(t, mt.Origin, "nested media type should carry an origin")
	require.NotNil(t, mt.Origin.Key)
	require.Equal(t, "application/json", mt.Origin.Key.Name)
	require.Equal(t, 7, mt.Origin.Key.Line)
}

func yamlToJSON(src string) ([]byte, error) {
	var v any
	if err := goyaml.Unmarshal([]byte(src), &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
