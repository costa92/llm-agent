---
phase: 01-walking-skeleton-generate
plan: 05
subsystem: contract
tags:
  - conformance
  - httptest
  - fixtures
  - goleak
dependency_graph:
  requires:
    - 01-02
    - 01-03
    - 01-04
  provides:
    - internal/contract/ shared generate conformance harness
    - fixture-driven provider validation
  affects:
    - 01-06 nightly ollama live test
    - 01-07 provider authoring guide
tech_stack:
  added:
    - go.uber.org/goleak v1.3.0
  patterns:
    - fixture-driven httptest replay
    - adapter-factory registry
    - parallel table-driven conformance
key_files:
  created:
    - /tmp/llm-agent-providers/internal/contract/contract.go
    - /tmp/llm-agent-providers/internal/contract/generate_test.go
    - /tmp/llm-agent-providers/internal/contract/main_test.go
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/generate_happy_gpt-4o-mini.json
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/generate_401_invalid_api_key.json
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/generate_429_rate_limit.json
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/generate_429_quota_exhausted.json
    - /tmp/llm-agent-providers/internal/contract/testdata/openai/generate_500_server_error.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/generate_happy_claude-3-5-haiku.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/generate_400_invalid_request.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/generate_401_invalid_api_key.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/generate_429_rate_limit.json
    - /tmp/llm-agent-providers/internal/contract/testdata/anthropic/generate_529_overloaded.json
    - /tmp/llm-agent-providers/internal/contract/testdata/ollama/generate_happy_llama3.1-8b.json
    - /tmp/llm-agent-providers/internal/contract/testdata/ollama/generate_404_model_not_pulled.json
    - /tmp/llm-agent-providers/internal/contract/testdata/ollama/generate_500_oom.json
    - /tmp/llm-agent-providers/scripts/capture-fixtures-openai.sh
    - /tmp/llm-agent-providers/scripts/capture-fixtures-anthropic.sh
    - /tmp/llm-agent-providers/scripts/capture-fixtures-ollama.sh
  modified:
    - /tmp/llm-agent-providers/.gitignore
    - /tmp/llm-agent-providers/go.mod
    - /tmp/llm-agent-providers/go.sum
decisions:
  - "Shared conformance runs through adapter factories only; adding a provider now means adding one factory and fixtures."
  - "Phase 1 ships `goleak.VerifyTestMain` without ignore rules because sync Generate produced no false positives under current Go/runtime."
  - "Fixture assertions stay intentionally permissive substring checks in Phase 1; strict JSON-path validation is deferred."
metrics:
  completed: 2026-05-10
  fixture_count: 13
---

# Phase 1 Plan 05: Shared Conformance Harness Summary

**One-liner:** Built the shared `internal/contract` fixture-driven conformance harness for OpenAI, Anthropic, and Ollama, including 13 replay fixtures, a parallel factory registry, goleak coverage, and local capture scripts.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add shared conformance helpers and adapter factory contract | `internal/contract/contract.go` |
| 2 | Add parallel table-driven generate conformance and goleak TestMain | `internal/contract/generate_test.go`, `main_test.go` |
| 3 | Add 13 replay fixtures spanning all 3 providers | `internal/contract/testdata/**` |
| 4 | Add local fixture capture scripts and repo ignore rules | `scripts/capture-fixtures-*.sh`, `.gitignore` |

## Verification Results

- `GOCACHE=/tmp/go-build go vet ./internal/contract/...` — PASS
- `GOCACHE=/tmp/go-build go build ./internal/contract/...` — PASS
- `GOCACHE=/tmp/go-build go test ./internal/contract/...` — PASS

Coverage in `TestGenerate_Conformance`:

- OpenAI: happy, `401`, `429 rate_limit`, `429 insufficient_quota`, `500`
- Anthropic: happy, `400 invalid_request`, `401 authentication_error`, `429 rate_limit_error`, `529 overloaded_error`
- Ollama: happy, `404 model not pulled`, `500`

Additional coverage:

- `TestErrorString_NoSecretLeak` documents current typed-error transparency and keeps unwrap-chain behavior under test.
- `TestMain` runs `goleak.VerifyTestMain`, establishing the no-leak baseline before Phase 2 streaming work.

## What Comes Next

- `01-06`: nightly Ollama live CI using build-tagged integration coverage
- `01-07`: Provider Author Guide v0.1 in core repo
