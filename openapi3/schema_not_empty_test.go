package openapi3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getkin/kin-openapi/openapi3"
)

// The empty schema matches every instance, so `not: {}` matches none. Judging
// the `not` by whether its own value is empty dropped it from IsEmpty, and
// visitJSON skips an empty schema, so the schema accepted every value instead.
func TestNot_EmptySubschemaRejectsEverything(t *testing.T) {
	const spec = `
openapi: 3.0.0
info:
  title: t
  version: "1.0"
paths: {}
components:
  schemas:
    Nothing:
      not: {}
    NothingNested:
      properties:
        a:
          not: {}
`
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(spec))
	require.NoError(t, err)
	require.NoError(t, doc.Validate(loader.Context))

	nothing := doc.Components.Schemas["Nothing"].Value
	require.False(t, nothing.IsEmpty(), "a not is a constraint whatever it holds")

	for _, value := range []any{"a string", float64(1), true, []any{}, map[string]any{}} {
		require.Error(t, nothing.VisitJSON(value))
	}

	// The same through a property, so the fix is not limited to a root schema.
	nested := doc.Components.Schemas["NothingNested"].Value
	require.Error(t, nested.VisitJSON(map[string]any{"a": "anything"}))
	require.NoError(t, nested.VisitJSON(map[string]any{}))
}

// A non-empty `not` was already rejected and stays rejected.
func TestNot_NonEmptySubschemaUnchanged(t *testing.T) {
	const spec = `
openapi: 3.0.0
info:
  title: t
  version: "1.0"
paths: {}
components:
  schemas:
    NotAString:
      not:
        type: string
`
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(spec))
	require.NoError(t, err)

	s := doc.Components.Schemas["NotAString"].Value
	require.Error(t, s.VisitJSON("a string"))
	require.NoError(t, s.VisitJSON(float64(1)))
}
