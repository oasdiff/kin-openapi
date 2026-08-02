package openapi3

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kinyaml "github.com/oasdiff/yaml"
	yaml "github.com/oasdiff/yaml3"
)

// A payload exercising every branch the ported types have: a $ref entry with
// summary/description and an x- sibling, an inline entry, extensions at the
// collection level and on the objects, and an unknown key that must land in
// Extensions rather than be dropped.
const responsesYAML = `
"200":
  description: ok
  x-response-ext: keep-me
  unknown-key: also-kept
  content:
    application/json:
      x-media-ext: 1
      example: {a: 1}
"404":
  $ref: '#/components/responses/NotFound'
  summary: missing
  description: the thing was not there
  x-ref-ext: on-the-ref
x-collection-ext: at-the-top
`

// The spike is only interesting if it decodes to the same thing. Both paths
// are run on the same document and compared, because a faster decoder that
// quietly drops extensions or mis-handles a $ref is worse than the slow one.
func TestNativeYAML_MatchesJSONPath(t *testing.T) {
	jsonBytes, err := yamlToJSONForTest(responsesYAML)
	require.NoError(t, err)

	var viaJSON Responses
	require.NoError(t, json.Unmarshal(jsonBytes, &viaJSON))

	var viaYAML Responses
	require.NoError(t, yaml.Unmarshal([]byte(responsesYAML), &viaYAML))

	// Compare through JSON so unexported fields and map ordering do not
	// confuse the diff; both sides marshal with the same MarshalJSON.
	wantJSON, err := json.Marshal(&viaJSON)
	require.NoError(t, err)
	gotJSON, err := json.Marshal(&viaYAML)
	require.NoError(t, err)
	require.JSONEq(t, string(wantJSON), string(gotJSON))
}

// Spot-check the individual branches too, so a failure above points somewhere.
func TestNativeYAML_Branches(t *testing.T) {
	var r Responses
	require.NoError(t, yaml.Unmarshal([]byte(responsesYAML), &r))

	require.Equal(t, "at-the-top", r.Extensions["x-collection-ext"])

	ok := r.Value("200")
	require.NotNil(t, ok)
	require.NotNil(t, ok.Value)
	require.Equal(t, "ok", *ok.Value.Description)
	require.Equal(t, "keep-me", ok.Value.Extensions["x-response-ext"])
	// Not an x- key, but not a declared field either: the JSON path keeps it,
	// so this one must too.
	require.Equal(t, "also-kept", ok.Value.Extensions["unknown-key"])

	mt := ok.Value.Content["application/json"]
	require.NotNil(t, mt)
	// Note the Go type: see TestNativeYAML_ExtensionNumberTypesDiffer.
	require.EqualValues(t, 1, mt.Extensions["x-media-ext"])

	nf := r.Value("404")
	require.NotNil(t, nf)
	require.Equal(t, "#/components/responses/NotFound", nf.Ref)
	require.Equal(t, "missing", *nf.Summary)
	require.Equal(t, "the thing was not there", *nf.Description)
	require.Equal(t, "on-the-ref", nf.Extensions["x-ref-ext"])
	require.Nil(t, nf.Value)
}

// A field declared on the struct must never be collected as an extension --
// the failure mode the reflection-derived known set is meant to prevent.
func TestNativeYAML_DeclaredFieldsAreNotExtensions(t *testing.T) {
	var r Responses
	require.NoError(t, yaml.Unmarshal([]byte(responsesYAML), &r))
	resp := r.Value("200").Value
	for _, declared := range []string{"description", "headers", "content", "links"} {
		_, found := resp.Extensions[declared]
		require.False(t, found, "%q is a declared field and must not be an extension", declared)
	}
}

// A real difference between the two paths, and the one thing here that could
// break a consumer: a number inside an extension arrives as float64 from
// encoding/json and as int from the YAML decoder. Marshaling normalises both,
// which is why the equivalence test above does not catch it, but code doing
// Extensions["x-n"].(float64) would panic after a migration.
func TestNativeYAML_ExtensionNumberTypesDiffer(t *testing.T) {
	const src = "\"200\":\n  description: ok\n  x-n: 1\n"

	jsonBytes, err := yamlToJSONForTest(src)
	require.NoError(t, err)
	var viaJSON Responses
	require.NoError(t, json.Unmarshal(jsonBytes, &viaJSON))

	var viaYAML Responses
	require.NoError(t, yaml.Unmarshal([]byte(src), &viaYAML))

	require.IsType(t, float64(0), viaJSON.Value("200").Value.Extensions["x-n"])
	require.IsType(t, int(0), viaYAML.Value("200").Value.Extensions["x-n"])
}

func yamlToJSONForTest(src string) ([]byte, error) {
	var v any
	if err := yaml.Unmarshal([]byte(src), &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// --- benchmark ---------------------------------------------------------------

func benchResponsesDoc(n int) (yamlSrc, jsonSrc []byte) {
	var sb strings.Builder
	for i := range n {
		fmt.Fprintf(&sb, `"%d":
  description: response %d
  x-ext-%d: value
  content:
    application/json:
      example: {field: %d}
`, 200+i, i, i, i)
	}
	yamlSrc = []byte(sb.String())
	jsonSrc, _ = yamlToJSONForTest(sb.String())
	return
}

// The comparison the spike exists for: the same subtree decoded through the
// current path (JSON bytes -> UnmarshalJSON, with its per-entry re-marshal)
// and through the native one (node -> UnmarshalYAML).
func BenchmarkResponsesDecode(b *testing.B) {
	yamlSrc, jsonSrc := benchResponsesDoc(400)

	b.Run("json/UnmarshalJSON", func(b *testing.B) {
		b.SetBytes(int64(len(jsonSrc)))
		b.ReportAllocs()
		for b.Loop() {
			var r Responses
			if err := json.Unmarshal(jsonSrc, &r); err != nil {
				b.Fatal(err)
			}
			if r.Len() != 400 {
				b.Fatalf("incomplete: %d", r.Len())
			}
		}
	})

	// What kin actually runs for a YAML document today: the wrapper's round
	// trip to JSON text, and then the UnmarshalJSON hooks above.
	b.Run("roundtrip/today", func(b *testing.B) {
		b.SetBytes(int64(len(yamlSrc)))
		b.ReportAllocs()
		for b.Loop() {
			var r Responses
			if _, err := kinyaml.Unmarshal(yamlSrc, &r, kinyaml.DecodeOpts{}); err != nil {
				b.Fatal(err)
			}
			if r.Len() != 400 {
				b.Fatalf("incomplete: %d", r.Len())
			}
		}
	})

	b.Run("yaml/UnmarshalYAML", func(b *testing.B) {
		b.SetBytes(int64(len(yamlSrc)))
		b.ReportAllocs()
		for b.Loop() {
			var r Responses
			if err := yaml.Unmarshal(yamlSrc, &r); err != nil {
				b.Fatal(err)
			}
			if r.Len() != 400 {
				b.Fatalf("incomplete: %d", r.Len())
			}
		}
	})
}

// --- origins ------------------------------------------------------------------

// The migration's real question: can positions be read off the node instead of
// smuggled through __origin__ nodes and reapplied by a reflection walk? This
// decodes the same document both ways and compares the Origins.
//
// Origin.Key is the interesting one. It is the location of the key heading a
// mapping in its *parent* -- the `"200":` line above a response, not the
// response's own first line -- which is what lets a consumer slice a whole
// block out of source. A node does not know its own key, so this is the one
// piece the parent has to stamp.
func TestNativeYAML_OriginsMatchAppliedOrigins(t *testing.T) {
	const src = `"200":
  description: ok
  content:
    application/json:
      example: {a: 1}
`
	// Current path: origins ride through the round trip as data, then get
	// reapplied to the struct tree.
	var viaTree Responses
	tree, err := kinyaml.Unmarshal([]byte(src), &viaTree, kinyaml.DecodeOpts{
		Origin: kinyaml.OriginOpt{Enabled: true},
	})
	require.NoError(t, err)
	require.NotNil(t, tree)
	applyOrigins(&viaTree, tree)

	// Native path: origins read off the node during decode.
	var viaNode Responses
	require.NoError(t, yaml.Unmarshal([]byte(src), &viaNode))

	respTree := viaTree.Value("200").Value
	respNode := viaNode.Value("200").Value
	require.NotNil(t, respTree.Origin, "the current path should produce an origin to compare against")
	require.NotNil(t, respNode.Origin)

	// Key: the `"200":` line, stamped by the parent in both designs.
	require.NotNil(t, respTree.Origin.Key)
	require.NotNil(t, respNode.Origin.Key)
	require.Equal(t, respTree.Origin.Key.Line, respNode.Origin.Key.Line, "Key.Line")
	require.Equal(t, respTree.Origin.Key.Column, respNode.Origin.Key.Column, "Key.Column")
	require.Equal(t, respTree.Origin.Key.EndLine, respNode.Origin.Key.EndLine, "Key.EndLine")
	require.Equal(t, respTree.Origin.Key.Name, respNode.Origin.Key.Name, "Key.Name")

	// Fields: each field key's own position.
	for name, want := range respTree.Origin.Fields {
		got, ok := respNode.Origin.Fields[name]
		require.True(t, ok, "field %q missing from the native origin", name)
		require.Equal(t, want.Line, got.Line, "field %q line", name)
		require.Equal(t, want.Column, got.Column, "field %q column", name)
	}
	require.Equal(t, len(respTree.Origin.Fields), len(respNode.Origin.Fields), "field count")
}

// The block span is what review tooling actually consumes: start at the key
// line, end past the last content. Assert it directly rather than trusting the
// field-by-field comparison to imply it.
func TestNativeYAML_OriginKeySpansTheBlock(t *testing.T) {
	const src = `"200":
  description: ok
  content:
    application/json:
      example: {a: 1}
"404":
  description: gone
`
	var r Responses
	require.NoError(t, yaml.Unmarshal([]byte(src), &r))

	k := r.Value("200").Value.Origin.Key
	require.NotNil(t, k)
	require.Equal(t, 1, k.Line, "block starts at its own key line, not its first content line")
	require.Equal(t, 5, k.EndLine, "block ends at its last content line, not at the next sibling")
}
