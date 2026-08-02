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
