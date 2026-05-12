---
phase: 06-reference-customer-support
verified: 2026-05-12T00:00:00Z
status: gaps_found
score: 12/13 requirements materially implemented; 1/13 still awaiting collector-sampling proof
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 10/13 requirements materially implemented; 3/13 awaiting cold-stack verification evidence
  gaps_closed:
    - "REFSVC-09 trace marking now emits prompt_injection_attempt=true on blocked input."
    - "REFSVC-12 collector asset now uses decision_wait=30s."
    - "REFSVC-10 runtime proof now exists: a locally built server returned 200 from /readyz and /chat while emitting real X-Trace-Id and X-Session-Id headers against the local dependency stack."
    - "REFSVC-11 dashboard provisioning proof now exists: Grafana API returns the preloaded Customer Support Observability dashboard."
  gaps_remaining:
    - "REFSVC-12 live collector-sampling proof is still not recorded."
  regressions: []
gaps:
  - requirement: REFSVC-12
    severity: medium
    evidence: "Tail-sampling config asset matches contract and the collector metrics endpoint is live on :8889, but request-class-specific 100% error / 100% >5s / ~10% baseline sampling behavior has still not been demonstrated with explicit metric evidence."
deferred:
  - "Explicit collector-sampling verification with crafted fast/slow/error requests and captured metrics."
human_verification: []
---

# Phase 6 Verification

## Verdict

# GAPS FOUND

Phase 6 is implemented and test-green, but it is not fully audit-closed. The
core service, guardrails, session storage, provider split, and demo packaging
all exist in code. The remaining gap is **runtime proof**, not missing
implementation.

## Requirements Check

| Requirement | Status | Evidence |
|-------------|--------|----------|
| REFSVC-01 | pass | `cmd/server`, `internal/app`, `internal/config` shipped in `06-01`; tests and `go build` green |
| REFSVC-02 | pass | `/chat`, `/chat/stream`, `/healthz`, `/readyz` shipped in `06-02`; HTTP tests green |
| REFSVC-03 | pass | `X-Trace-Id` propagation shipped in `06-02`; verified by transport tests |
| REFSVC-04 | pass | independent chat/embedding provider selection shipped in `06-03` |
| REFSVC-05 | pass | support flow with RAG + `StateGraph` + tools shipped in `06-04` |
| REFSVC-06 | pass | SQLite/Postgres-backed session storage shipped in `06-05` |
| REFSVC-07 | pass | request/tool/token guardrails shipped in `06-06` |
| REFSVC-08 | pass | `DISABLE_LLM=1` panic switch shipped in `06-06` |
| REFSVC-09 | pass | prompt-injection filter, safe fallback, tool identity hardening, untrusted-RAG marking, and trace attribute proof shipped in `06-07` plus closeout fix |
| REFSVC-10 | pass | on 2026-05-12 a locally built server returned `200` from `/readyz` and `200` from `/chat` against the live local dependency stack, with real `X-Trace-Id` and `X-Session-Id` headers |
| REFSVC-11 | pass | Grafana API search returned the provisioned `Customer Support Observability` dashboard (`uid=customer-support-demo`) |
| REFSVC-12 | gap | `decision_wait=30s` and policies are in config/tests, but live sampling behavior is not archived |
| REFSVC-13 | pass | README demo-only hardening banner shipped in `06-08` |

## Verification Executed

- `GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./... -count=1`
- targeted closeout tests:
  - `go test ./compose -run TestDemoAssetsExistAndDocumentObservability -count=1`
  - `go test ./internal/supportflow -run TestFlow_FlaggedInputMarksTraceAttribute -count=1`
- `docker compose -f compose/compose.yaml config`
- attempted `docker compose -f compose/compose.yaml up --build`
- re-attempted `docker compose -f compose/compose.yaml up --build -d` on
  2026-05-12 with elevated Docker access
- checked `docker compose -f compose/compose.yaml ps` on 2026-05-12
- built `/tmp/llm-agent-customer-support-server` locally with the 4-repo
  workspace and ran it against:
  - local Ollama on `127.0.0.1:11434`
  - `otel-lgtm` on `127.0.0.1:3000`
  - collector OTLP HTTP on `127.0.0.1:4318`
- verified:
  - `curl -i http://127.0.0.1:18081/readyz`
  - `curl -i -X POST http://127.0.0.1:18081/chat -H 'Content-Type: application/json' -d '{"message":"hello"}'`
  - `curl -s 'http://127.0.0.1:3000/api/search?query=Customer%20Support%20Observability'`

Observed runtime evidence from the 2026-05-12 retry:

- the stack remained in image-pull progress for several minutes
- the two large image layers advanced roughly to `934MB` and `676MB`
- `docker compose ps` still returned only the header row, meaning no containers
  had been created yet
- after switching to a locally built app binary plus the compose dependency
  stack, `/readyz` returned `200 OK` with
  `X-Trace-Id: fa2fd77bd21fa4b698d0aecb9aab5a76`
- `/chat` returned `200 OK` with
  `X-Trace-Id: 44257ff1fe822996556b996d6a61f7e9`,
  `X-Session-Id: 476f7405-4571-488d-afa1-f5751d3136ef`, and answer
  `"Please share your order ID so I can check the refund policy."`
- Grafana search API returned the provisioned
  `Customer Support Observability` dashboard with
  `uid=customer-support-demo`

## Remaining Closure Work

1. Capture explicit collector metrics proving the tail-sampling policy keeps
   error traces at 100%, >5s traces at 100%, and clean baseline traffic at
   roughly 10%.
2. Optionally replace the host-run app workaround with a full compose-native
   app container proof after the environment/GitHub build constraints are
   removed.
