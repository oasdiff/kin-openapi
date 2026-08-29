package openapi3

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/oasdiff/yaml"
)

func unmarshalError(jsonUnmarshalErr error) error {
	if before, after, found := strings.Cut(jsonUnmarshalErr.Error(), "Bis"); found && before != "" && after != "" {
		before = strings.ReplaceAll(before, " Go struct ", " ")
		return fmt.Errorf("%s%s", before, strings.ReplaceAll(after, "Bis", ""))
	}
	return jsonUnmarshalErr
}

// unmarshal decodes data into v. It returns the document origin tree when the
// data took the yaml path, so the caller can retain it (see Loader.originTrees).
//
// Only the yaml path records positions, so the order of the two attempts
// depends on includeOrigin: without it the json fast path runs first, with it
// the yaml path does, which is what gives json documents origins. Either way
// both parsers are tried before failing, so anything that loaded before still
// loads.
func unmarshal(data []byte, v any, includeOrigin bool, location *url.URL) (*originTree, error) {
	var jsonErr, yamlErr error

	// See https://github.com/getkin/kin-openapi/issues/680
	if !includeOrigin {
		if jsonErr = json.Unmarshal(data, v); jsonErr == nil {
			return nil, nil
		}
	}

	// UnmarshalStrict(data, v) TODO: investigate how ymlv3 handles duplicate map keys
	var file string
	if location != nil {
		file = location.String()
	}
	if tree, err := yaml.Unmarshal(data, v, yaml.DecodeOpts{
		Origin:            yaml.OriginOpt{Enabled: includeOrigin, File: file},
		DisableTimestamps: true,
	}); err == nil {
		applyOrigins(v, tree)
		return tree, nil
	} else {
		yamlErr = err
	}

	// The yaml path was tried first and failed, so json has not run yet. It
	// accepts documents yaml rejects (duplicate keys, which json resolves
	// last-one-wins), so this keeps them loading, without origins.
	if includeOrigin {
		if jsonErr = json.Unmarshal(data, v); jsonErr == nil {
			return nil, nil
		}
	}

	// If both unmarshaling attempts fail, return a new error that includes both errors
	return nil, fmt.Errorf("failed to unmarshal data: json error: %v, yaml error: %v", jsonErr, yamlErr)
}
