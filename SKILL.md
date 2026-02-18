# kin-openapi Build Instructions

## Before committing

Always run `./docs.sh` before committing changes. CI checks that `.github/docs/*.txt` matches `go doc` output and will fail if they are out of date.

```bash
./docs.sh
```

### What docs.sh does

1. Regenerates `.github/docs/*.txt` files from `go doc -all` output for each package
2. Checks that any removed public API symbols are mentioned in README.md's "Sub-v1 breaking API changes" section

### If docs.sh fails

- **Stale docs:** The generated `.txt` files don't match current code. Stage the updated `.github/docs/*.txt` files in your commit.
- **Missing breaking change mention:** If you changed or removed a public API (struct field, function signature, type), add an entry to the `## CHANGELOG: Sub-v1 breaking API changes` section in README.md describing the change. The script uses `grep -F` to match the removed symbol text, so the mention must contain the exact symbol name.

### CI lint rule

Never use `require.Contains(t, err.Error(), ...)` in tests. Use `require.ErrorContains(t, err, ...)` instead. CI greps for `require[.].+err.Error` and fails.
