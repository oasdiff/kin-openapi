package openapi3

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodingJSON(t *testing.T) {
	t.Log("Marshal *openapi3.Encoding to JSON")
	data, err := json.Marshal(encoding())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	t.Log("Unmarshal *openapi3.Encoding from JSON")
	docA := &Encoding{}
	err = json.Unmarshal(encodingJSON, &docA)
	require.NoError(t, err)
	require.NotEmpty(t, docA)

	t.Log("Validate *openapi3.Encoding")
	err = docA.Validate(context.Background())
	require.NoError(t, err)

	t.Log("Ensure representations match")
	dataA, err := json.Marshal(docA)
	require.NoError(t, err)
	require.JSONEq(t, string(data), string(encodingJSON))
	require.JSONEq(t, string(data), string(dataA))
}

var encodingJSON = []byte(`
{
  "contentType": "application/json",
  "headers": {
    "someHeader": {}
  },
  "style": "form",
  "explode": true,
  "allowReserved": true
}
`)

func encoding() *Encoding {
	return &Encoding{
		ContentType: "application/json",
		Headers: map[string]*HeaderRef{
			"someHeader": {
				Value: &Header{},
			},
		},
		Style:         "form",
		Explode:       Ptr(true),
		AllowReserved: true,
	}
}

func TestEncodingSerializationMethod(t *testing.T) {
	testCases := []struct {
		name string
		enc  *Encoding
		want *SerializationMethod
	}{
		{
			name: "default",
			want: &SerializationMethod{Style: SerializationForm, Explode: true},
		},
		{
			name: "encoding with style",
			enc:  &Encoding{Style: SerializationSpaceDelimited},
			want: &SerializationMethod{Style: SerializationSpaceDelimited, Explode: true},
		},
		{
			name: "encoding with explode",
			enc:  &Encoding{Explode: Ptr(true)},
			want: &SerializationMethod{Style: SerializationForm, Explode: true},
		},
		{
			name: "encoding with no explode",
			enc:  &Encoding{Explode: Ptr(false)},
			want: &SerializationMethod{Style: SerializationForm, Explode: false},
		},
		{
			name: "encoding with style and explode ",
			enc:  &Encoding{Style: SerializationSpaceDelimited, Explode: Ptr(false)},
			want: &SerializationMethod{Style: SerializationSpaceDelimited, Explode: false},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.enc.SerializationMethod()
			require.EqualValues(t, tc.want, got, "got %#v, want %#v", got, tc.want)
		})
	}
}
