# kin-openapi Development Guide

## Pre-commit steps (REQUIRED before every commit)

Run these in order:
1. `go fmt ./...` — format all Go code
2. `go vet ./...` — check for common errors
3. `./docs.sh` — regenerate `.github/docs/*.txt` and verify that any removed public API items are mentioned in README.md under "breaking API changes"

If `docs.sh` fails, it means a public API item was removed or changed without a changelog entry. Add the item to README.md under `## CHANGELOG: Sub-v1 breaking API changes` in the appropriate version section, then re-run `./docs.sh`.

## Repository structure

- `openapi2/` — OpenAPI 2.0 types
- `openapi2conv/` — Convert OpenAPI 2.0 → 3.0
- `openapi3/` — OpenAPI 3.0 types, loader, and validator
- `openapi3filter/` — Request/response validation
- `openapi3gen/` — Go struct → OpenAPI schema generation
- `routers/` — Request routing

## Origin tracking

`__origin__` is a special key injected by `oasdiff/yaml3` into every YAML map to track source file locations. When unmarshaling typed maps (e.g., `map[string]*Encoding`), this key must be stripped. Use `unmarshalStringMapP[V]` (in `openapi3/stringmap.go`) to unmarshal typed string maps — it automatically strips `__origin__`. Never use a raw `map[string]*V` as a struct field type for maps that come from YAML; always define a named type with `UnmarshalJSON` using `unmarshalStringMapP`.
