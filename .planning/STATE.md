# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-18)

**Core value:** The core `llm-agent` module stays stdlib-only and zero-dep — anyone can `go get` it and read every line. Providers, telemetry, and reference services live in sister repos so users opt into deps one package at a time.
**Current focus:** between milestones — `v0.6` shipped 2026-05-18. Run `/gsd-new-milestone` to scope the next one.

## Current Position

Phase: none active — `v0.6` shipped 2026-05-18; the milestone is closed and
archived.
Previous milestone: `v0.6` — production-grade retrieval quality and safety,
phases 14-19, all executed and verified green; milestone audit PASS (12/12
requirements). Delivered: BM25 lexical retrieval + RRF fusion, model-based
reranking with explainability, the RAG Triad evaluation, cost/latency
observability (`obs` + `otelrag` RED metrics), content safety (`guard` PII
redaction + injection defense), and agentic retrieval (`MultiHopRetriever`
+ `CorrectiveAsker`). `llm-agent-rag` gained no new dependency — all stdlib.
Plan: no milestone is active. v0.6 is archived to
`.planning/milestones/v0.6-ROADMAP.md` + `v0.6-REQUIREMENTS.md`; audit at
`.planning/v0.6-MILESTONE-AUDIT.md`.
Status: committed, merged, and tagged — `llm-agent-rag` `master` `798bf3f`,
tag `v0.3.0`; `llm-agent` `main` `03ff8a7`; `llm-agent-otel`
`feat/otelrag-wrap-rag-system` `12b647e`. All three pushed to origin. The
one open v0.6 thread is the `llm-agent-otel` `require` bump to
`llm-agent-rag v0.3.0` — blocked on sandbox network (see Blockers).
Next step: `/gsd-new-milestone` to scope the next milestone.
Last activity: 2026-05-18 — v0.6 milestone-close: audited (PASS),
committed/merged/tagged (`v0.3.0`) and pushed all three repos, then ran the
milestone transition — archived v0.6 ROADMAP/REQUIREMENTS to
`.planning/milestones/`, updated PROJECT/ROADMAP/STATE.

Progress: v0.6 milestone complete (6/6 phases, 12/12 requirements) and
closed. No milestone currently active.

## Performance Metrics

**Velocity:**
- Total plans completed: 54 (40 through v0.5 + 14 in v0.6 phases 14-19)
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
  carried forward from v0.5; the Phase 14 `tsvector` path is still
  unverified against a live DB.
- v0.6 milestone-close, remaining: (1) bump `llm-agent-otel`'s `require
  github.com/costa92/llm-agent-rag` to `v0.3.0` + `go mod tidy` + confirm
  `GOWORK=off` builds green — **BLOCKED on sandbox network** (see
  Blockers); clears the 17-03 cross-repo debt. (2) `/gsd-transition` to the
  next milestone.
- `CHANGELOG.md` in `llm-agent-rag` has its `v0.3.0` entry — committed on
  `master` as `e97bd3d` (`docs: changelog for v0.3.0`), one commit past the
  `v0.3.0` tag (the tag stays on `798bf3f`; not force-moved).
- `e97bd3d` (changelog) is committed locally but not pushed — `master` is 2
  commits ahead of `origin/master`.

### Blockers/Concerns

- **`llm-agent-otel` `require` bump blocked on sandbox network.** Bumping
  `require github.com/costa92/llm-agent-rag` to `v0.3.0` needs `go mod
  tidy` to fetch the module — `go mod` uses HTTPS (github.com:443), which
  is unreachable in this sandbox (only SSH/22 is open; that is why the
  `git push`es succeeded). The `v0.3.0` tag IS pushed to GitHub, so the
  bump is a ~2-minute step in any environment with HTTPS: edit `go.mod`
  (`v0.2.0` → `v0.3.0`), `go mod tidy`, confirm `GOWORK=off go build/test
  ./...` green, commit, push. This is the same recurring limitation prior
  sessions hit during v0.5 otel cleanup — not a design blocker.
- Until that bump lands, `llm-agent-otel`'s committed state still requires
  `llm-agent-rag v0.2.0`, so `otelrag` only builds against the v0.6 RAG
  SDK via a local `go.work` (the 17-03 carry-forward debt).

## Session Continuity

Last session: 2026-05-18
Stopped at: `v0.6` milestone shipped and closed. All six phases (14-19)
executed and verified green; milestone audit PASS (12/12 requirements,
`.planning/v0.6-MILESTONE-AUDIT.md`). Committed, merged to `master`, tagged
`llm-agent-rag v0.3.0`, and pushed all three repos to origin. Milestone
transition done: v0.6 ROADMAP/REQUIREMENTS archived to
`.planning/milestones/`, PROJECT/ROADMAP/STATE updated to "between
milestones."
Open thread: `llm-agent-otel`'s `require` is still pinned to `llm-agent-rag
v0.2.0` — the bump to `v0.3.0` is blocked on sandbox network (HTTPS to
github.com unreachable; SSH works). It is a ~2-minute step in any
HTTPS-capable environment — see Blockers/Concerns for the exact commands.
Next step: `/gsd-new-milestone` to scope the next milestone. (Optionally
first finish the `llm-agent-otel` `require` bump where HTTPS is available.)
Resume file: .planning/ROADMAP.md
