package openapi3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getkin/kin-openapi/openapi3"
)

// A JSON document gets origins like a YAML one: only the yaml parser records
// positions, so the loader runs it first when origins are requested.
func TestOrigin_JSONSpec(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IncludeOrigin = true
	loader.Context = t.Context()

	doc, err := loader.LoadFromFile("testdata/origin/simple.json")
	require.NoError(t, err)

	require.NotNil(t, doc.Origin)
	require.Equal(t,
		openapi3.Location{
			File:   "testdata/origin/simple.json",
			Line:   2,
			Column: 3,
			Name:   "openapi",
		},
		doc.Origin.Fields["openapi"])

	require.NotNil(t, doc.Info.Origin)
	require.Equal(t,
		&openapi3.Location{
			File:      "testdata/origin/simple.json",
			Line:      3,
			Column:    3,
			Name:      "info",
			EndLine:   6,
			EndColumn: 4,
		},
		doc.Info.Origin.Key)

	pathItem := doc.Paths.Find("/partner-api/test/some-method")
	require.NotNil(t, pathItem.Origin)
	require.Equal(t,
		&openapi3.Location{
			File:      "testdata/origin/simple.json",
			Line:      8,
			Column:    5,
			Name:      "/partner-api/test/some-method",
			EndLine:   19,
			EndColumn: 6,
		},
		pathItem.Origin.Key)

	require.NotNil(t, pathItem.Get.Origin)
	require.Equal(t,
		&openapi3.Location{
			File:      "testdata/origin/simple.json",
			Line:      9,
			Column:    7,
			Name:      "get",
			EndLine:   18,
			EndColumn: 8,
		},
		pathItem.Get.Origin.Key)

	response := pathItem.Get.Responses.Value("200")
	require.NotNil(t, response.Value.Origin)
	require.Equal(t,
		openapi3.Location{
			File:   "testdata/origin/simple.json",
			Line:   15,
			Column: 13,
			Name:   "description",
		},
		response.Value.Origin.Fields["description"])
}

// Origins are off, so the json fast path runs first and no positions are
// recorded. Pins that requesting origins is what changes the parser order.
func TestOrigin_JSONSpecWithoutIncludeOrigin(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.Context = t.Context()

	doc, err := loader.LoadFromFile("testdata/origin/simple.json")
	require.NoError(t, err)
	require.Nil(t, doc.Origin)
	require.Equal(t, "Test API", doc.Info.Title)
}

// json permits duplicate keys and resolves them last-one-wins; yaml rejects
// them. Such a document must keep loading when origins are requested, since
// the loader falls back to json, and it simply has no origins.
func TestOrigin_JSONSpecDuplicateKeys(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IncludeOrigin = true
	loader.Context = t.Context()

	doc, err := loader.LoadFromFile("testdata/origin/duplicate_keys.json")
	require.NoError(t, err)
	require.Equal(t, "Second", doc.Info.Title)
	require.Nil(t, doc.Origin)
}
