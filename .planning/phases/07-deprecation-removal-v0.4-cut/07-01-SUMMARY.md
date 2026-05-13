# Phase 7 Plan 01 Summary

Status: in progress

Purpose:

- document the explicit early gate override for Phase 7
- audit remaining internal usage of the legacy `llm.Client` surface
- prepare the execution order for `DEPRC-01`

Completed in this audit pass:

- active planning files updated to mark Phase 7 as open
- repo-wide scan of remaining legacy-surface dependencies
- fresh baseline `go test ./...` run in the core repo

## Audit Result

`DEPRC-01` is **not** yet satisfied. Remaining legacy-surface usage is real and
still spans production code, tests/examples, and docs.

### Production/runtime code still using the legacy surface

- `rag/`
  - `rag/rag.go`
  - `rag/advanced.go`
- `bench/`
  - `bench/judge.go`
  - `bench/winrate.go`
- `context/`
  - `context/builder.go`
  - `context/compress.go`
- `rl/`
  - `rl/trainer_proxy.go`

### Test-only or shim usage

- `rag/rag_test.go`
- `rag/tool_test.go`
- `bench/bench_test.go`
- `context/context_test.go`
- `llm/llm_test.go`

### Examples and user-facing docs still teaching the old surface

- `README.md`
- `examples/README.md`
- `examples/scriptedllm/scriptedllm.go`
- `docs/migration-v0.2-to-v0.3.md`
- `DEPRECATIONS.md`
- `CHANGELOG.md`
- `docs/api-snapshot.txt`

### Deprecated symbol definitions themselves

- `llm/legacy.go`
- related compatibility commentary in:
  - `llm/doc.go`
  - `llm/types.go`

## Fresh Verification

- `go test ./...`

Result: PASS

## Recommended Execution Order

1. Migrate runtime packages off `llm.Client` first:
   - `rag/`
   - `context/`
   - `bench/`
   - `rl/`
2. Update tests/examples to the new `llm.ChatModel` / `llm.Request` /
   `llm.Response` surface.
3. Remove the deprecated symbols in `llm/legacy.go`.
4. Rewrite migration/changelog/deprecations docs for the completed v0.4 break.
5. Coordinate sister-repo bumps and tags after the core API actually lands.

## Next Slice

The next concrete plan should target runtime migration only, not symbol
deletion yet:

- convert `rag/`, `context/`, `bench/`, and `rl/` from `llm.Client` to
  `llm.ChatModel`
- keep behavior stable
- leave docs/examples cleanup and actual `llm/legacy.go` removal for the
  following bounded slice
