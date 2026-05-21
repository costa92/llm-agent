> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 12-01 Summary

Date: 2026-05-14
Repo: `llm-agent-rag`
Plan: [12-01-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/12-persistence-tracing-and-backend-conformance/12-01-PLAN.md)

## Objective

First persistent vector backend for the standalone SDK: PostgreSQL +
pgvector behind the existing `store.Store` interface.

## Delivered

- new package `postgres/` (`/tmp/llm-agent-rag/postgres/`)
  - `postgres.Store` satisfies `store.Store` (compile-time
    `var _ store.Store = (*Store)(nil)`)
  - `New(*pgxpool.Pool, Config) (*Store, error)` constructor
    (caller owns the pool)
  - `Config{Table, Dimension}` — explicit dimension at construction
    time; `Migrate(ctx)` creates the table with `vector(N)`
  - all 7 `store.Store` methods implemented: `Upsert`, `Search`,
    `List`, `Get`, `Remove`, `RemoveByFilter`, `Stats`
  - dimension mismatch returns `store.ErrDimensionMismatch`;
    missing rows return `store.ErrNotFound`; both honored by
    `errors.Is`
  - cosine-distance search via pgvector's `<=>` operator; score
    surfaced as `1 - distance` so larger = closer (consistent with
    other stores)
  - metadata round-trip via JSONB; filter / security-filter matched
    with `metadata @> $N` so subset matching works
  - `RegisterTypes(ctx, *pgx.Conn)` helper for use in
    `pgxpool.Config.AfterConnect` so each pooled connection knows
    the vector codec
  - guards: configurable `Table` name validated as a safe ASCII
    identifier (so `fmt.Sprintf` embed cannot inject SQL)
- `go.mod` gains its first non-stdlib deps (sister-repo only; core
  `llm-agent` stays stdlib-only):
  - `github.com/jackc/pgx/v5 v5.9.2`
  - `github.com/pgvector/pgvector-go v0.3.0`
  - resolved transitives: `pgpassfile`, `pgservicefile`,
    `jackc/puddle/v2`, `x448/float16`, `golang.org/x/sync`,
    `golang.org/x/text`
  - `go mod tidy` also added `github.com/costa92/llm-agent v0.4.0`
    as a direct dep — this is **not** new code from this slice;
    it's the existing build-tagged `adapter/llmagent` package
    finally getting its deps resolved (see Notes)
- `postgres/postgres_test.go` with three cases:
  - `TestStoreNewRejectsBadInputs` — runs always; covers nil pool,
    zero dimension, bad identifier
  - `TestPostgresStore_LiveSmoke` — gated on
    `LLM_AGENT_RAG_PG_URL`; skips cleanly when unset. Upsert two
    vectors, assert nearest-first ranking, metadata round-trip,
    `Get`/`Remove`/`Stats` flow
  - `TestPostgresStore_LiveMetadataFilter` — also env-gated;
    asserts JSON-subset metadata filter narrows results

## Files

- `/tmp/llm-agent-rag/postgres/postgres.go` (new, ~340 LOC)
- `/tmp/llm-agent-rag/postgres/postgres_test.go` (new, ~170 LOC)
- `/tmp/llm-agent-rag/go.mod` + `/tmp/llm-agent-rag/go.sum` (new)

## Verification

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go build ./...
GOWORK=off GOCACHE=/tmp/go-build go vet ./...
GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1

cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off go vet ./rag/...
GOWORK=off go test ./rag/... -count=1
```

Result:

- `go build ./...` (standalone): pass
- `go vet ./...` (standalone): pass
- `go test ./...` (standalone, 13 packages including new `postgres`):
  pass — live Postgres tests skip cleanly without `LLM_AGENT_RAG_PG_URL`
- `go vet ./rag/...` / `go test ./rag/...` (core compat): pass

Live-Postgres verification (when you have a local PG with pgvector):

```bash
cd /tmp/llm-agent-rag
LLM_AGENT_RAG_PG_URL=postgres://localhost/llm_agent_rag_test?sslmode=disable \
  GOWORK=off GOCACHE=/tmp/go-build go test ./postgres -count=1 -v
```

## Notes & Deviations

- **Plan said variable dim, ship says fixed dim.** The plan
  proposed storing the column as `vector` (no dim) and validating
  per-namespace on Upsert; ship picks `vector(Dimension)` with
  `Dimension` as a required `Config` field. Reason: matches
  industry practice (one embedding model per deployment), gives
  pgvector a hint for index choice, and Upsert validation is
  simpler — checks against a constant, not against a probe.
- **Pre-existing latent failure surfaced.** The repo had a
  build-tag-gated package `adapter/llmagent` that imports core
  `llm-agent`. Before this slice, the repo had no `go.sum`, so
  building under `-tags llmagent` was broken (missing deps).
  After `go mod tidy`, `go build -tags llmagent ./adapter/...`
  works, and `go test -tags llmagent ./adapter/...` runs but
  surfaces one failing test:
  `TestAsToolNamespaceIsolation` (`alpha search missing alpha
  content: []`). This is **not** introduced by 12-01 — the test
  was previously unreachable. Default builds (no `llmagent` tag)
  are unaffected. Flagging here so it lands on someone's
  triage list; fix belongs in a separate slice.
- **Plan said pool-owned-by-caller; ship matches.** New takes
  `*pgxpool.Pool` and the caller is responsible for closing it.
  AfterConnect is the right hook for `RegisterTypes`; the smoke
  test wires this end-to-end as the reference pattern.
- **Plan said no facade rewiring; ship matches.** `rag.New`
  already accepts any `store.Store` via `Options.Store`, so users
  can swap in `postgres.New(...)` without code changes in the rag
  layer. Doc / example wiring can land in 12-02 once the
  conformance suite is in place.

## Next slice

`12-02` is now unblocked and should:

1. add a `store.ConformanceSuite(t, factory)` helper that all
   `store.Store` implementations can run against — including the
   in-memory store and `postgres.Store`
2. add tracing hooks (`Trace`) for `Import`, `Retrieve`, `Pack`,
   `Ask` covering RAG-OPS-02
3. optionally wire the live-Postgres smoke into CI via
   testcontainers-go or `services:` in the GitHub Actions config
