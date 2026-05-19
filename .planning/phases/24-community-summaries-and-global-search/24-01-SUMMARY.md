---
phase: 24-community-summaries-and-global-search
plan: 01
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-03]
---

# Summary: 24-01 Community summarization + report persistence

## Objective

Add community summarization: a `CommunityReport` type, a
`CommunitySummarizer` seam, an `LLMCommunitySummarizer`, a deterministic
`CommunityContentHash`, and report persistence behind the
`store.CommunityStore` capability (in-memory + postgres). RAG-GRAPH3-03.

## Delivered

- `graph/summary.go` (new):
  - `CommunityReport{CommunityID, Title, Summary, ContentHash}` — an
    LLM-written summary of one community; `ContentHash` records the
    membership the report was built from so a stale report is detectable.
  - `CommunityContentHash(c Community) string` — a deterministic hex
    SHA-256 over the sorted `EntityIDs` then the sorted `RelationIDs`, with
    fixed unit (`0x1f`) and record (`0x1e`) separators. Member order does
    not matter (it sorts internally); any membership change flips the hash;
    the entity/relation boundary is honored so moving an ID across it is a
    real change. `crypto/sha256` is stdlib — no new dependency.
  - `CommunitySummarizer` seam — `Summarize(ctx, Community, Graph)
    (CommunityReport, error)` — the vendor-neutral seam pattern, a sibling
    of `EntityExtractor`.
  - `LLMCommunitySummarizer{Model generate.Model}` — builds a deterministic
    prompt from the community's member entity names+types+descriptions and
    relation descriptions (looked up in the namespace `Graph` by ID, emitted
    in the community's already-sorted ID order, unknown IDs skipped), asks
    for a `Title:` line + a paragraph summary, and parses leniently. A nil
    `Model` returns the sentinel `ErrCommunitySummarizerModelRequired`; a
    model error is propagated. `Summarize` always sets `CommunityID` and
    `ContentHash = CommunityContentHash(c)`.
  - `parseCommunityReport` — lenient parse mirroring `LLMEntityExtractor`:
    the first `Title:` line (case-insensitive) becomes the title; all other
    non-fence lines join into the summary; code fences are stripped; a
    response with no title line still parses (empty title, whole body the
    summary).
- `graph/summary_test.go` (new): `LLMCommunitySummarizer` against the
  scripted model on a fixed two-entity/one-relation graph — a canned report
  parses; malformed output (no title line, preamble, code fences) parses
  leniently without error and still carries `ContentHash`; nil-model and
  model-error cases; `CommunityContentHash` determinism (stable, order-
  independent) and membership sensitivity (added entity, added relation,
  entity/relation boundary not conflated).
- `store/store.go`: `CommunityStore` extended with `PutCommunityReport` and
  `CommunityReport` — the report cache's write and read sides.
- `store/inmemory.go` + `store/community.go`: `InMemoryStore` gained a
  `reports map[string]map[string]graph.CommunityReport` field (namespace →
  communityID → report), initialized in `NewInMemoryStore`. `CommunityReport`
  returns `found=false`/no-error for an unknown ID (a cache miss).
- `postgres/postgres.go`: `Migrate()` gained an idempotent
  `<table>_community_reports` table (`namespace, community_id, title,
  summary, content_hash`; PK `(namespace, community_id)`); the table name
  derives from the `isSafeIdent`-validated `cfg.Table`.
- `postgres/community.go`: `PutCommunityReport` (upsert via `ON CONFLICT`)
  and `CommunityReport` (`pgx.ErrNoRows` → `found=false`/no-error) on
  `*Store`.
- `store/storetest/storetest.go`: `RunCommunityConformance` extended with
  four report subtests — round-trip, unknown-id miss, overwrite, and
  namespace isolation.
- `postgres/postgres_conformance_test.go`: the per-subtest cleanup `DROP`
  now also drops `<table>_community_reports`.

## Files

- `graph/summary.go` — new: `CommunityReport`, `CommunityContentHash`,
  `CommunitySummarizer`, `LLMCommunitySummarizer`, lenient parse.
- `graph/summary_test.go` — new: scripted-model + content-hash tests.
- `store/store.go` — `CommunityStore` extended with the two report methods.
- `store/inmemory.go` — `InMemoryStore.reports` field + init.
- `store/community.go` — in-memory `PutCommunityReport`/`CommunityReport`.
- `store/community_test.go` — unchanged (already drives
  `RunCommunityConformance`, which now covers reports).
- `store/storetest/storetest.go` — `RunCommunityConformance` report
  round-trip subtests + `sampleReport` helper.
- `postgres/postgres.go` — `_community_reports` table in `Migrate()`.
- `postgres/community.go` — postgres `PutCommunityReport`/`CommunityReport`.
- `postgres/postgres_conformance_test.go` — cleanup drops the new table.

## Verification

All `<verify>` commands run, all green:

- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go build ./...`
  — BUILD OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go vet ./...`
  — VET OK
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./graph
  ./store/... ./postgres/... -count=1` — `ok` for `graph`, `store`,
  `postgres`; `store/storetest` has no test files
- `cd /tmp/llm-agent-rag && GOWORK=off GOCACHE=/tmp/go-build go test ./...
  -count=1` — all packages `ok`, no FAIL
- `cd /tmp/llm-agent-rag && git diff --stat go.mod go.sum` — empty (no new
  dependency)
- core facade (from the core repo `llm-agent`):
  `GOWORK=off go vet ./rag/... && go test ./rag/...` — VET OK, `ok`

Additionally confirmed `TestPostgresCommunityConformance` skips cleanly with
no `LLM_AGENT_RAG_PG_URL` set (`--- SKIP`, `PASS`) — the postgres path
compiles and is env-gated.

## Notes / deviations

- No deviations — the plan was executed exactly as written. The
  `files_modified` list matches one-to-one.
- `CommunityContentHash` uses `0x1f` (unit separator) between member IDs
  and `0x1e` (record separator) at the entity/relation boundary rather than
  a printable separator: these control bytes cannot appear in an entity or
  relation ID, so the hash cannot be made to collide by an ID that contains
  the separator string. A test asserts the boundary is honored.
- The lenient parser strips lines beginning with ```` ``` ```` so a model
  that wraps its output in a markdown fence still parses; a missing title
  line yields an empty `Title` and the whole body as `Summary`, never an
  error — the `LLMEntityExtractor` "malformed is never fatal" precedent.
- Postgres derived table names (`<table>_community_reports`) inherit the
  `isSafeIdent` guarantee: `postgres.New` rejects an unsafe `cfg.Table`, and
  the suffix is a constant ASCII identifier, so the composed name is always
  a safe identifier. No separate guard needed — same pattern as
  `communitiesTable()`.
- Extending the unreleased v0.8 `store.CommunityStore` interface forced both
  impls to gain the new methods in this slice (compile-time
  `var _ store.CommunityStore = ...` assertions exist in both
  `store/community.go` and `postgres/community.go`); done, build stays
  green.
- No new module dependency — `graph` stays a stdlib + `generate`-seam leaf
  package (`crypto/sha256`, `encoding/hex` are stdlib).
- Out of scope as planned: `AskGlobal` / global search (24-02);
  `PrewarmCommunityReports` and the worked example (24-03).

## Self-Check: PASSED

- `graph/summary.go`, `graph/summary_test.go` present in the working tree.
- `store/store.go`, `store/inmemory.go`, `store/community.go`,
  `store/storetest/storetest.go`, `postgres/postgres.go`,
  `postgres/community.go`, `postgres/postgres_conformance_test.go` all
  modified in the working tree.
- No commits made — per operator instruction, all changes left uncommitted
  for separate commit.
