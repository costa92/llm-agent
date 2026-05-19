---
phase: 24-community-summaries-and-global-search
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH3-04]
---

# Summary: 24-02 rag.System.AskGlobal — map-reduce global search

## Objective

Add `rag.System.AskGlobal` — a map-reduce global-search answer path over
community reports, distinct from `Ask`. Community selection (coarsest level),
lazy report generation, per-community map, score-ranked reduce. `GlobalOptions`,
a `Diagnostics.Global` block. Completes RAG-GRAPH3-04.

## Delivered

- `rag.GlobalOptions` — `{ Namespace string; MaxCommunities int }`.
  `MaxCommunities <= 0` falls back to `defaultMaxCommunities` (8).
- `rag.Options.CommunitySummarizer graph.CommunitySummarizer` — the seam
  AskGlobal uses to generate community reports lazily; carried on `System` as
  `s.communitySummarizer`, wired in `New`.
- `rag.GlobalDiagnostics` — `{ CommunityIDs []string; MapScores
  map[string]int; MapCalls int; ReduceCalls int }`; added as the additive
  `Diagnostics.Global` field (the zero value for an ordinary `Ask`).
- `rag/global.go` — `func (s *System) AskGlobal(ctx, question string, opts
  GlobalOptions) (Answer, error)`, a SEPARATE answer path. It never calls
  `s.retrieve`, the reranker, or the packer. Flow:
  - `s.model == nil` → `ErrModelRequired`; installs a fresh `obs.Counter` and
    times each stage (`select`, `report`, `map`, `reduce`), mirroring `Ask`.
  - type-asserts `s.store` for `store.CommunityStore`; a store without it →
    empty `Answer`, no error (graceful, like a missing v0.7 graph signal).
  - `Communities(ns)`; none → empty `Answer`, no error. Selects the
    **coarsest** level (highest `Level`). When that level exceeds
    `MaxCommunities`, ranks by query-token overlap with member entity names
    (lowercase whitespace tokens — the `retrieve.LexicalEntityLinker` idiom),
    caps, then restores community-ID order — deterministic, tie-break by ID.
  - **lazy reports**: per selected community, `CommunityReport(ns, id)`;
    reused iff found AND `report.ContentHash == graph.CommunityContentHash(c)`;
    otherwise requires `s.communitySummarizer` (nil →
    `ErrCommunitySummarizerRequired`), `Summarize`, then `PutCommunityReport`.
    The namespace `GraphSnapshot` is loaded once and only on the first miss.
  - **map**: per report, one `s.model.Generate` with the report title+summary
    and the question; `parseGlobalMap` leniently extracts a `Score:` line
    (0-100, clamped; unparseable → 0) and the partial-answer text.
  - **reduce**: drops score-0 partials, sorts survivors by score desc
    (tie-break by community ID), runs `s.model.Generate` once to synthesize
    the final answer. No survivor → a graceful "No relevant community
    information…" text and `ReduceCalls == 0`.
  - returns `Answer{Text, Diagnostics{Global, Metrics}, Trace{Question,
    Namespace}}` — recording consulted community IDs, per-community map
    scores, and map/reduce call counts.
- `rag.ErrCommunitySummarizerRequired` — returned when a report must be
  generated but no `Options.CommunitySummarizer` was configured.

## Files

- `rag/global.go` — new; `AskGlobal`, `GlobalOptions` (in options.go),
  `selectCommunities`, lazy `communityReports`, the map/reduce prompt
  builders, `parseGlobalMap` and helpers (`queryTokens`, `communityOverlap`,
  `clampScore`, …).
- `rag/global_test.go` — new; a `globalScriptedModel` keyed by `SystemPrompt`
  so map and reduce are distinguishable, a `countingSummarizer` to prove
  cache hits vs. re-summarize, a four-triangle `globalTestGraph` detected
  with `graph.LouvainDetector` into a multi-level hierarchy. Tests:
  map-reduce happy path + cache reuse on the second call; stale `ContentHash`
  → re-summarize; non-`CommunityStore` store → empty answer; empty namespace
  → empty answer; `MaxCommunities` caps the consulted set; missing summarizer
  → `ErrCommunitySummarizerRequired`; nil model → `ErrModelRequired`;
  all-zero map scores → graceful no-survivor text, no reduce call.
- `rag/options.go` — added `GlobalOptions` and
  `Options.CommunitySummarizer`.
- `rag/system.go` — added `GlobalDiagnostics`, the additive
  `Diagnostics.Global` field, the `s.communitySummarizer` field, wired in
  `New`.
- `rag/errors.go` — added `ErrCommunitySummarizerRequired` (see Deviations).

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD-OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET-OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./rag/... -count=1` — `ok
  github.com/costa92/llm-agent-rag/rag`; all 9 new `TestAskGlobal*` tests PASS
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all 21
  packages `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — `ok github.com/costa92/llm-agent/rag`

## Deviations from plan

- The plan's `files_modified` lists `rag/global.go`, `rag/global_test.go`,
  `rag/options.go`, `rag/system.go`. One additional file was touched:
  `rag/errors.go` — `ErrCommunitySummarizerRequired` was added there, beside
  the other `rag` sentinel errors (`ErrModelRequired`, `ErrRetrieverRequired`,
  …), rather than inline in `global.go`. This matches the existing package
  convention (all `rag` errors live in `errors.go`) and is the natural home
  for a sentinel callers will compare against. No behavior change versus the
  plan; the error name and meaning are exactly as the plan's task 3 specified.
- No other deviations. `Ask` is untouched; `Diagnostics.Global` is additive
  (zero value for a plain `Ask`).

## Notes

- `AskGlobal` is a genuinely separate path: it has no `s.retrieve`,
  `s.reranker`, or `s.packer` reference. Determinism lives entirely in
  selection (coarsest level, then ID-ordered overlap ranking with ID
  tie-break), content hashing, and survivor ranking — the only LLM-backed
  steps (map, reduce, lazy summarize) are scripted-model tested.
- The map/reduce scripted model keys on `SystemPrompt` prefixes, so a single
  stub serves both steps and a test can assert exact map/reduce call counts.
- `GraphSnapshot` is loaded lazily inside `communityReports` — once, on the
  first cache miss — so an all-cache-hit `AskGlobal` makes no snapshot read.
- No new module dependency — `crypto/sha256` (already used by
  `graph.CommunityContentHash` from 24-01) is stdlib; map/reduce/summarize
  reuse `generate.Model`.
- Out of scope, per plan: `PrewarmCommunityReports`, local-search community
  attribution, the worked example — slice 24-03.

## Self-Check: PASSED

- `rag/global.go` — FOUND
- `rag/global_test.go` — FOUND
- `rag/options.go`, `rag/system.go`, `rag/errors.go` — FOUND (modified)
- All `<verify>` commands green; `go.mod`/`go.sum` diff empty.
