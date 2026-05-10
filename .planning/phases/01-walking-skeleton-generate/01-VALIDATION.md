---
phase: 1
phase_slug: walking-skeleton-generate
date: 2026-05-10
---

# Phase 1 Validation Strategy

> Lifted from `01-RESEARCH.md` §"Validation Architecture" to satisfy the Nyquist gate (`workflow.nyquist_validation: true`). RESEARCH.md remains the authoritative source — edits here mirror back.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26) + `go.uber.org/goleak` v1.3.0+ for goroutine assertions |
| Config file | None (Go's `go test` is config-free); `goleak` invoked from `internal/contract/main_test.go` |
| Quick run | `cd llm-agent-providers && go test -short ./...` (≤ 30s; runs unit tests + conformance mock-only) |
| Full suite | `cd llm-agent-providers && go test ./...` (~1 min; same as quick — Phase 1 has no slow PR tests) |
| Nightly Ollama-live | `cd llm-agent-providers && go test -tags ollama_live -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live` (run only by nightly workflow) |

## Phase Requirements → Test Map

| Req ID | Behavior | Test type | Command |
|--------|----------|-----------|---------|
| OAI-01 | OpenAI Generate happy path; FinishReason=stop | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_Happy` |
| OAI-01 | OpenAI Generate with system message lifts to messages[0] role=system | unit | `go test ./openai/... -run TestGenerate_OpenAI_SystemPrompt` |
| OAI-05 | OpenAI 401 → `*llm.AuthError` | unit | `go test ./openai/... -run TestGenerate_OpenAI_401` |
| OAI-05 | OpenAI 429 → `*llm.RateLimitError`; insufficient_quota sets Reason | unit | `go test ./openai/... -run TestGenerate_OpenAI_429` |
| OAI-05 | OpenAI 5xx → `*llm.TransientError` | unit | `go test ./openai/... -run TestGenerate_OpenAI_5xx` |
| OAI-05 | OpenAI 4xx other (400/404/422) → `*llm.InvalidRequestError` | unit | `go test ./openai/... -run TestGenerate_OpenAI_4xxOther` |
| ANT-01 | Anthropic Generate happy path with claude-3-5-haiku | unit | `go test ./anthropic/... -run TestGenerate_Anthropic_Happy` |
| ANT-01 | SystemPrompt lifts to top-level System []TextBlockParam (Pitfall C) | unit | `go test ./anthropic/... -run TestGenerate_Anthropic_SystemTopLevel` |
| ANT-05 | Anthropic 529 overloaded → `*llm.RateLimitError` | unit | `go test ./anthropic/... -run TestGenerate_Anthropic_529` |
| ANT-05 | Anthropic 400 invalid_request_error → `*llm.InvalidRequestError` | unit | `go test ./anthropic/... -run TestGenerate_Anthropic_400` |
| OLL-01 | Ollama Generate happy path against llama3.1:8b | unit | `go test ./ollama/... -run TestGenerate_Ollama_Happy` |
| OLL-01 | Ollama with bound model returns Response.Model matching | unit | `go test ./ollama/... -run TestGenerate_Ollama_ModelEcho` |
| OLL-05 | Ollama 404 model-not-pulled → `*llm.InvalidRequestError` | unit | `go test ./ollama/... -run TestGenerate_Ollama_404ModelNotPulled` |
| OLL-05 | Ollama no daemon reachable → `*llm.TransientError` | unit | `go test ./ollama/... -run TestGenerate_Ollama_NoDaemon` |
| OLL-08 | Nightly Ollama-live container Generate succeeds | integration (testcontainers, build-tag) | `go test -tags ollama_live -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live` |
| CONF-01 | Shared httptest harness loads fixtures + starts server | unit (LoadFixture / NewMockServer round-trip) | `go test ./internal/contract/... -run TestContractHelpers` |
| CONF-02 | Same fixture matrix produces identical normalized output across 3 adapters | conformance (table-driven) | `go test ./internal/contract/... -run TestGenerate_Conformance` |
| CONF-07 | Capture script per provider produces a Fixture JSON | manual smoke run | `bash scripts/capture-fixtures-openai.sh` (requires real API key; verify resulting JSON parses) |
| CONF-08 | `goleak.VerifyTestMain` reports zero leaks | TestMain | `go test ./internal/contract/...` (goleak is automatic via `main_test.go`) |
| CORE-11 | PROVIDER_AUTHORING.md v0.1 exists in core repo | manual-only | `test -f PROVIDER_AUTHORING.md && wc -l PROVIDER_AUTHORING.md` |

## Sampling Rate (per `nyquist_validation: true`)

- **Per task commit:** `cd llm-agent-providers && go test -short ./<package>/...` for the package the task touched (≤ 5s typical).
- **Per wave merge:** `cd llm-agent-providers && go test ./...` (full mock suite; ~1 min; goleak runs at end).
- **Phase gate:** Full suite green + nightly-ollama-live green at least once before `/gsd-verify-work`. If the nightly hasn't run yet because the workflow file is brand-new, manually trigger it (`workflow_dispatch`).

## Plan → Validation Map

| Plan | Validation deliverable |
|------|------------------------|
| 01-01 — Typed errors in core | `llm/errors_test.go` round-trip + `errors.As`/`errors.Is` coverage; tag `v0.3.0-pre.2` resolves |
| 01-02 — OpenAI adapter | `openai/openai_test.go` covering 7 test cases above; httptest fixtures from `testdata/openai/` |
| 01-03 — Anthropic adapter | `anthropic/anthropic_test.go` covering 4 test cases; httptest with PATH A or PATH B (RoundTripper) status capture |
| 01-04 — Ollama adapter | `ollama/ollama_test.go` covering 4 test cases; statusCapturingTransport pattern (Q3) |
| 01-05 — Conformance harness | `internal/contract/{contract,generate_test,main_test}.go`; `testdata/{openai,anthropic,ollama}/*.json` (13 fixtures); `scripts/capture-fixtures-*.sh` (3 scripts); `goleak.VerifyTestMain` |
| 01-06 — Nightly Ollama-live | `.github/workflows/nightly-ollama-live.yml` triggers `go test -tags ollama_live ./internal/contract/...`; testcontainers Ollama with pinned model + image |
| 01-07 — Provider Author Guide | `PROVIDER_AUTHORING.md` exists with 8 sections; conformance harness named as the authoritative contract |

## Wave 0 Gaps (must land before per-requirement tests can run)

These setup items are deliverables of Wave 0 (plan 01-01) — they are not test cases themselves:

- Core-repo: extend `llm/errors.go` with the 4 typed-error structs; cut tag `v0.3.0-pre.2`.
- Sister-repo: `go get github.com/costa92/llm-agent@v0.3.0-pre.2` after the tag exists.
- Sister-repo: `internal/contract/{contract,main_test,generate_test}.go` skeletons (Plan 01-05 Wave 2).

## Phase-Level Acceptance

Phase 1 is GREEN when:

1. Every test command in the Phase Requirements → Test Map exits 0.
2. `goleak.VerifyTestMain` reports zero leaks across the entire `internal/contract` package.
3. Nightly Ollama-live workflow has had at least one successful run (manual `workflow_dispatch` permitted for the first invocation since cron schedule is brand-new).
4. Core repo's `go.mod` has zero `require` lines outside `module` and `go` directives (stdlib-only invariant intact).
5. Sister repo's `go.mod` has at most these direct requires: `github.com/costa92/llm-agent v0.3.0-pre.2`, `github.com/openai/openai-go/v3`, `github.com/anthropics/anthropic-sdk-go`, `github.com/ollama/ollama`, `go.uber.org/goleak`, `github.com/testcontainers/testcontainers-go`, plus their transitive deps (in `go.sum`).
6. Pitfalls A-F (sister-repo `Pitfalls.md` cross-reference) all have a corresponding asserter test.

---

*Validation Strategy synthesized 2026-05-10 from `01-RESEARCH.md` §"Validation Architecture" + §"Sampling Rate" + §"Wave 0 Gaps".*
