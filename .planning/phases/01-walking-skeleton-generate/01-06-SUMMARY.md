---
phase: 01-walking-skeleton-generate
plan: 06
subsystem: ollama-live-ci
tags:
  - ollama
  - nightly-ci
  - testcontainers
  - build-tags
dependency_graph:
  requires:
    - 01-05
  provides:
    - nightly live Ollama conformance coverage
    - build-tagged real-container generate verification
  affects:
    - 01-07 provider authoring guide
    - Phase 2 streaming groundwork
tech_stack:
  added:
    - github.com/testcontainers/testcontainers-go v0.42.0
    - github.com/testcontainers/testcontainers-go/modules/ollama v0.42.0
  patterns:
    - build-tagged live integration test
    - cron-only GitHub Actions workflow
    - pinned container image and model
key_files:
  created:
    - /tmp/llm-agent-providers/internal/contract/ollama_live_test.go
    - /tmp/llm-agent-providers/.github/workflows/nightly-ollama-live.yml
  modified:
    - /tmp/llm-agent-providers/go.mod
    - /tmp/llm-agent-providers/go.sum
decisions:
  - "Live Ollama coverage stays behind `//go:build ollama_live` so PR CI remains fast and Docker-free."
  - "The nightly workflow pins `ollama/ollama:0.5.7` and `llama3.1:8b-instruct-q4_K_M` to catch upstream drift without depending on `:latest`."
  - "Real-model output is treated as nondeterministic; the live test reuses the shared happy-path fixture only for structural assertions."
metrics:
  completed: 2026-05-10
  files_created: 2
  files_modified: 2
---

# Phase 1 Plan 06: Nightly Ollama Live CI Summary

**One-liner:** Added a build-tagged live Ollama conformance test plus a nightly GitHub Actions workflow, so the Generate-only adapter is exercised against a real local model without slowing PR CI.

## Tasks Completed

| # | Name | Key Files |
|---|------|-----------|
| 1 | Add build-tagged real-container Ollama generate test | `internal/contract/ollama_live_test.go` |
| 2 | Add nightly cron + manual workflow wrapper | `.github/workflows/nightly-ollama-live.yml` |
| 3 | Add test-only container dependencies | `go.mod`, `go.sum` |

## Verification Results

- `GOCACHE=/tmp/go-build go vet -tags ollama_live ./internal/contract/...` — PASS
- `GOCACHE=/tmp/go-build go test -count=1 ./internal/contract/...` — PASS
- `grep -c 'tcollama.Run\|container.Exec\|ConnectionString' internal/contract/ollama_live_test.go` — PASS (`5`)

Notes:

- The live test uses `tcollama.Run`, `container.Exec("ollama pull ...")`, and `ConnectionString()` from `testcontainers-go/modules/ollama`.
- The workflow runs only on `schedule` and `workflow_dispatch`, never on `pull_request`.
- Live `workflow_dispatch` verification remains deferred until the workflow file is merged, matching the existing Phase 1 deferred-item policy.

## What Comes Next

- `01-07`: Provider Author Guide v0.1 in the core repo
- After `01-07`, Phase 1 can close and Phase 2 planning can begin
