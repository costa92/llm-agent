---
phase: 23-community-detection-and-graph-store-persistence
plan: 03
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-02]
---

# Summary: 23-03 postgres CommunityStore + Import community-detection stage

## Objective

Implement `store.CommunityStore` for the postgres store, and wire community
detection into `rag.Import` as a post-persist stage with re-detection on a
`ReplaceSource` re-ingest. Completes RAG-GRAPH3-02 and Phase 23.

## Delivered

- `postgres/community.go` — `*Store` now implements `store.CommunityStore`
  (`var _ store.CommunityStore = (*Store)(nil)`):
  - `GraphSnapshot` — full `SELECT` of `<table>_entities` and
    `<table>_relations` for the namespace, `ORDER BY id`, rebuilt into a
    `graph.Graph`. Reuses the existing `queryEntities` / `queryRelations`
    helpers from `postgres/graph.go`, so the scan/codec path is shared with
    `Neighborhood` / `FindEntities`.
  - `UpsertCommunities` — replace-all in one transaction: `DELETE` the
    namespace's rows then bulk-`INSERT` the new set. An empty set just
    clears the namespace.
  - `Communities` — `SELECT` the namespace's rows ordered by
    `community_id`; an unknown namespace yields `nil`, no error.
- `postgres/postgres.go` `Migrate()` — adds the idempotent
  `<table>_communities` table (`CREATE TABLE IF NOT EXISTS`, the table
  identifier still `isSafeIdent`-guarded at `New()` time, `_communities` a
  constant suffix): columns `namespace text`, `community_id text`,
  `level int`, `parent_id text`, `entity_ids text[]`, `relation_ids text[]`,
  primary key `(namespace, community_id)`. The `Migrate` doc comment now
  also names the entities/relations/communities tables.
- `rag/options.go` — `Options.CommunityDetector graph.CommunityDetector`.
- `rag/system.go` — carried as `System.communityDetector`, wired in `New`.
- `rag/import.go` — after the existing `if persistGraph` block: if the
  store is a `store.CommunityStore` **and** `s.communityDetector != nil`,
  `GraphSnapshot(ns)` → `Detect` → `UpsertCommunities(ns, …)`, then
  `res.Graph.Communities = communities` when `res.Graph != nil`. Errors
  wrap as `fmt.Errorf("rag: detect communities: %w", err)`. Because the
  snapshot is read **after** `UpsertGraph` (and after `RemoveGraphBySource`
  on a `ReplaceSource` re-ingest), a re-ingest re-detects the whole
  namespace automatically — replace-all `UpsertCommunities` reconciles
  (keystone KG3-7). A store that is not a `CommunityStore`, or no detector
  configured, degrades gracefully — no detection, no error.
- `postgres/postgres_conformance_test.go` — `TestPostgresCommunityConformance`
  wires `storetest.RunCommunityConformance` env-gated exactly like the
  existing `RunGraphConformance` wiring; the per-subtest table-drop cleanup
  now also drops `<table>_communities`.
- `rag/community_test.go` — four tests on the in-memory store +
  `LouvainDetector`: communities are detected and the store matches the
  `ImportResult`; a `ReplaceSource` re-ingest re-detects (the stale entity
  set does not persist or accumulate); graceful degradation with no
  detector and with a non-`CommunityStore` store.

## Files

- `postgres/community.go` — new.
- `postgres/postgres.go` — `_communities` table in `Migrate()`; doc comment.
- `postgres/postgres_conformance_test.go` — `TestPostgresCommunityConformance`;
  cleanup drops `_communities`.
- `rag/options.go` — `Options.CommunityDetector`.
- `rag/system.go` — `System.communityDetector`, wired in `New`.
- `rag/import.go` — post-persist community-detection stage.
- `rag/community_test.go` — new.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./rag/... ./postgres/... -count=1`
  — both ok (4 new community tests in `rag` PASS)
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all packages
  ok, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet
  ./rag/... && go test ./rag/...` — VET OK, TEST OK

## Deviations from plan

None — plan executed exactly as written. `storetest.RunCommunityConformance`
and the in-memory `CommunityStore` were already in place from 23-02, so the
postgres conformance wiring was a straight `t.Run` addition mirroring
`TestPostgresGraphConformance`.

## Notes

- The postgres community path is env-gated like the rest of the postgres
  store: `TestPostgresCommunityConformance` skips cleanly when
  `LLM_AGENT_RAG_PG_URL` is unset. It compiles and is exercised under
  `go test ./postgres/...`; live-Postgres verification is carried-forward
  debt, the same caveat as `RunGraphConformance`.
- `text[]` columns round-trip directly through pgx — `entity_ids` /
  `relation_ids` scan straight into `[]string`, no JSON layer needed.
- No new module dependency — `postgres` already depends on `pgx/v5`;
  nothing added.

## Phase 23 status

All three slices complete:

- **23-01** — `graph` package: `Community` type, `Graph.Communities` field,
  the `CommunityDetector` seam, deterministic stdlib `LouvainDetector` +
  `LabelPropagationDetector`, golden-output tests. (RAG-GRAPH3-01)
- **23-02** — `store.CommunityStore` capability, the pure-stdlib in-memory
  implementation, the shared `storetest.RunCommunityConformance` suite.
  (RAG-GRAPH3-02)
- **23-03** — postgres `CommunityStore` (`_communities` table,
  `GraphSnapshot` over `_entities`/`_relations`), the `rag.Import`
  community-detection stage with re-detection on re-ingest (KG3-7), the
  env-gated postgres conformance wiring. (RAG-GRAPH3-02)

**RAG-GRAPH3-01 and RAG-GRAPH3-02 are delivered.** Phase 23 — the first
v0.8 GraphRAG-tier-3 phase, community detection and graph-store persistence
— is complete. Community summaries / reports and global search are Phase 24.
