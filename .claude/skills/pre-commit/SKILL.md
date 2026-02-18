---
name: pre-commit
description: Run pre-commit checks for kin-openapi before committing. Use when about to commit, after making code changes, or when the user asks to validate changes.
disable-model-invocation: true
allowed-tools: Bash
---

# Pre-commit checks for kin-openapi

Before committing changes to kin-openapi, run these steps in order:

## 1. Regenerate docs

```bash
./docs.sh
```

This regenerates `.github/docs/*.txt` from `go doc` output. CI checks that these files match and will fail if they are stale.

If `docs.sh` fails with missing mentions, it means a public API symbol was changed or removed. Add an entry to the `## CHANGELOG: Sub-v1 breaking API changes` section in `README.md` describing the change. The script uses `grep -F` to find the symbol name, so the mention must contain the exact symbol text.

Stage any updated `.github/docs/*.txt` and `README.md` files.

## 2. Run tests

```bash
go test ./...
```

## 3. Vet and format

```bash
go vet ./...
go fmt ./...
```

## CI lint rule

Never use `require.Contains(t, err.Error(), ...)` in tests. Use `require.ErrorContains(t, err, ...)` instead. CI greps for `require[.].+err.Error` and fails.
