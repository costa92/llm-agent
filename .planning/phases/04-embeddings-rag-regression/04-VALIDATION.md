---
phase: 4
phase_slug: embeddings-rag-regression
date: 2026-05-12
---

# Phase 4 Validation Strategy

> Reconstructed after milestone close from Phase 4 PLAN/SUMMARY artifacts to
> backfill the Nyquist validation record.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` across `llm-agent-providers` and core `llm-agent` |
| Config file | None |
| Quick run | `GOCACHE=/tmp/go-build go test ./openai/... ./ollama/... ./anthropic/... ./internal/contract/... -count=1` |
| Full suite | `GOCACHE=/tmp/go-build go test ./... -count=1` |
| Supporting checks | `GOCACHE=/tmp/go-build go build ./...`, `GOCACHE=/tmp/go-build go vet ./...` where recorded |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| OAI-04 | OpenAI embeddings return vectors and usage data | unit | `go test ./openai/... -count=1` |
| OAI-08 | OpenAI passes the completed conformance suite at embeddings gate | conformance | `go test ./internal/contract/... -count=1` |
| ANT-04 | Anthropic embedding gap returns `ErrNotSupported` honestly | unit | `go test ./anthropic/... -count=1` |
| ANT-07 | Anthropic documented-gap path passes the suite | conformance | `go test ./internal/contract/... -count=1` |
| OLL-04 | Ollama embeddings return vectors for supported models | unit | `go test ./ollama/... -count=1` |
| OLL-07 | Ollama passes the completed conformance suite at embeddings gate | conformance | `go test ./internal/contract/... -count=1` |
| CONF-06 | Shared embedding conformance asserts dimensions, batch shape, and Anthropic gap behavior | conformance | `go test ./internal/contract/... -count=1` |
| CORE-11 | Provider Author Guide includes Embedder and documented-gap guidance | doc regression | `go test ./... -count=1` in core repo plus doc update evidence in summaries |

## Sampling Rate

- After each provider embedding task: run the touched package tests.
- After conformance extension: run `go test ./internal/contract/...`.
- After RAG regression plan: run the core repo full suite.
- Before phase close: provider and core suites must both be green.

## Plan -> Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 04-01 | OpenAI embedding tests |
| 04-02 | Ollama embedding tests |
| 04-03 | Anthropic `ErrNotSupported` regression tests |
| 04-04 | Shared embedding conformance matrix |
| 04-05 | Core RAG regression and `PROVIDER_AUTHORING.md` update |

## Manual-Only Verifications

No manual-only Nyquist gap remains after this backfill. The phase evidence is
recorded through provider tests, conformance, and the core regression suite.

## Evidence Carried Forward

- `04-04-SUMMARY.md`: shared embedding conformance landed in
  `internal/contract`
- `04-05-SUMMARY.md`: `rag.RAGSystem` regression and author-guide update
  shipped in the core repo

## Phase-Level Sign-Off

- The embeddings gate is covered at both adapter and shared-conformance level.
- The RAG-facing core regression is explicitly part of the validation surface.
- This phase now has a validation artifact on disk for future audits.

---

*Validation strategy backfilled on 2026-05-12 from existing Phase 4 artifacts.*
