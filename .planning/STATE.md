# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-15)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** v0.6 milestone — production-grade retrieval quality and safety. Six phases (14-19), each deepening one 🟡 Partial seam of `llm-agent-rag`: retrieval, reranking, evaluation, observability, content safety, agentic retrieval.

## Current Position

Phase: 19 — agentic retrieval (decomposition + self-correction) — complete 2026-05-17
Previous phase: 18 — content safety (PII redaction + injection defense) — complete 2026-05-17
Plan: v0.6 milestone opened 2026-05-15. Phases 14-16 executed:
Phase 14 — Okapi BM25 lexical retrieval (in-memory + Postgres `tsvector`),
`store.LexicalSearcher`, configurable + attributable RRF fusion.
Phase 15 — rerank explainability (`RerankScore`/`Trace.Scores` via
`Diagnostics.RerankScores`); `ScoringModel` seam + `ModelReranker` +
`HTTPScoringModel`.
Phase 16 — `eval.Judge` seam + `LLMJudge` (LLM-as-judge); `TriadEvaluator`
assembling retrieval + generation metrics into `TriadResult`, `WriteJSONL`
report + `Summary`, RAG-Triad CI gate.
Phase 17 — `obs` package (`Metrics`/`Counter`) + per-stage durations and
embed/generate call counts in `Diagnostics`/`retrieve.Trace`/`ImportResult`/
`ImportTrace`; `generate.Usage` + token accounting; `otelrag` RED + cost
metrics.
Phase 18 — new `guard` package: `PIIRedactor` (`Redactor` seam, configurable
rules) wired into `Import` to redact PII before chunking; `PatternScanner`
(`InjectionScanner` seam) + `SanitizeMode` (Neutralize/Drop) wired into `Ask`
to screen retrieved chunks before prompt assembly; `Redactions` on
`ImportResult`/`ImportTrace`, `InjectionFindings` on `Diagnostics`.
Phase 19 — `retrieve.MultiHopRetriever` (compound-query decomposition +
merge) with `QueryDecomposer` (`Heuristic`/`LLM`) and `Trace.Hops`; new
`agentic` package `CorrectiveAsker` — grounding-driven self-correcting
retry loop over `eval.Judge`, bounded by `MaxRetries`.
Phases 15-19 (llm-agent-rag side) added NO new dependency — all stdlib.
Status: milestone `v0.6` — all six phases (14-19) executed and verified
green; milestone audit PASS (`.planning/v0.6-MILESTONE-AUDIT.md`, 12/12
requirements delivered). Pending milestone-close: commit the v0.6 tree,
re-tag `llm-agent-rag`, bump `llm-agent-otel`'s `require`, transition.
Next step is the v0.6 milestone-close (commit on operator ask, then re-tag
+ `/gsd-transition`).
Last activity: 2026-05-18 — v0.6 milestone audit: re-ran the full gate
(18 `llm-agent-rag` packages + `otelrag` via `go.work` + core facade — all
green), wrote `v0.6-MILESTONE-AUDIT.md`, marked all 12 REQUIREMENTS
Delivered.

Progress: [██████████████] 6 of 6 v0.6 phases complete (Phases 14-19 all done)

## Performance Metrics

**Velocity:**
- Total plans completed: 40 (through v0.5)
- Average duration: -
- Total execution time: -

**By Phase:**

| Phase | Plans | Status |
|-------|-------|--------|
| 8 | 4 | complete |
| 9 | 3 | complete |
| 10 | 4 | complete |
| 11 | 13 | complete |
| 12 | 3 | complete |
| 13 | 4 | complete |
| 14 | 3 | complete |
| 15 | 2 | complete |
| 16 | 2 | complete |
| 17 | 3 | complete |
| 18 | 2 | complete |
| 19 | 2 | complete |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- 2026-05-15: `v0.5` shipped — `llm-agent-rag` tagged `v0.2.0`,
  `llm-agent-otel` consumes it (`replace` removed), core `llm-agent/rag`
  facade aligned.
- 2026-05-15: v0.6 scope deepens six 🟡 Partial seams (retrieval, rerank,
  eval, observability, security, agentic); deployment layer (HTTP/CLI/cache)
  deferred past v0.6.
- 2026-05-15: new non-stdlib deps for v0.6 are allowed in `llm-agent-rag`
  only, isolated behind a subpackage/build tag like `postgres`. Core
  `llm-agent` stays stdlib-only.

### Pending Todos

- Live-Postgres CI wiring (testcontainers-go or GH Actions services) —
  carried forward from v0.5.
- Keep standalone `llm-agent-rag` and core `llm-agent/rag` compatibility in
  lockstep as v0.6 retrieval changes land.
- v0.5 work across three repos is committed, pushed, and tagged
  (`llm-agent-rag v0.2.0`). Uncommitted: the `.planning/` v0.6 milestone
  setup (this change) plus v0.5 slice PLAN/SUMMARY files — commit on the
  operator's explicit ask.

### Blockers/Concerns

No immediate implementation blocker. The standing constraint is still to
preserve the zero-dependency core value while pushing retrieval capability
into the standalone `llm-agent-rag` module first.

## Session Continuity

Last session: 2026-05-17
Stopped at: v0.6 milestone code-complete — all six phases (14-19) executed
and verified green. Phase 14 (BM25 lexical, Postgres `tsvector`,
configurable/attributable RRF), Phase 15 (rerank explainability,
`ModelReranker` + `HTTPScoringModel`), Phase 16 (`LLMJudge`,
`TriadEvaluator` + RAG-Triad CI gate), Phase 17 (`obs` cost/latency
package, durations + call counts, `generate.Usage`, `otelrag` RED + cost
metrics), Phase 18 (`guard` — `PIIRedactor` at ingest, `PatternScanner`
injection defense), and Phase 19 (`MultiHopRetriever` decomposition, the
`agentic` `CorrectiveAsker` self-correcting loop) all executed. The
Postgres `tsvector` path remains unverified against a live DB. v0.6
planning + Phase 14-19 code + all PLAN/SUMMARY files are uncommitted across
`llm-agent-rag`, `llm-agent-otel`, and the `.planning/` tree — awaiting an
explicit commit instruction.
Next step: v0.6 milestone-close — milestone audit is done
(`.planning/v0.6-MILESTONE-AUDIT.md`, PASS). Remaining: commit the v0.6
tree (on operator ask), re-tag `llm-agent-rag`, bump `llm-agent-otel`'s
`require`, then `/gsd-transition`.
Carry-forward for 17-03: `otelrag` consumes untagged `llm-agent-rag`
working-tree fields — verified locally via `go.work`; the `otelrag/go.mod`
`require` bump waits on an `llm-agent-rag` re-tag at v0.6 close (a plain
`GOWORK=off` build of `llm-agent-otel` is red against `v0.2.0` until then).
Resume file: .planning/ROADMAP.md
