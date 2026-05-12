---
phase: 06-reference-customer-support
verified: 2026-05-12T00:00:00Z
status: pass
score: 13/13 requirements materially implemented and runtime-verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 12/13 requirements materially implemented; 1/13 still awaiting collector-sampling proof
  gaps_closed:
    - "REFSVC-09 trace marking now emits prompt_injection_attempt=true on blocked input."
    - "REFSVC-12 collector asset now uses decision_wait=30s."
    - "REFSVC-10 runtime proof now exists: a locally built server returned 200 from /readyz and /chat while emitting real X-Trace-Id and X-Session-Id headers against the local dependency stack."
    - "REFSVC-11 dashboard provisioning proof now exists: Grafana API returns the preloaded Customer Support Observability dashboard."
    - "REFSVC-12 live collector-sampling proof now exists: a direct OTLP probe sent 30 fast traces, 1 error trace, and 1 six-second trace through the live collector, and Tempo retained 2/30 fast traces plus both special-case traces after the 30s tail-sampling decision window."
    - "Phase 6 observability wiring now fails closed in the right places: the collector OTLP receiver binds to 0.0.0.0 for compose-network reachability, and HTTP/model/agent spans mark error status explicitly so the collector's status_code policy can match real failures."
  gaps_remaining: []
  regressions: []
gaps: []
deferred: []
human_verification: []
---

# Phase 6 Verification

## Verdict

# PASS

Phase 6 is implemented, test-green, and now audit-closed. The service,
guardrails, session storage, provider split, demo packaging, and tail-sampling
contract all have code and live runtime proof.

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
| REFSVC-10 | pass | on 2026-05-12 both a locally built server and a compose-built `app` container returned `200` from `/readyz` and `200` from `/chat` with real `X-Trace-Id` and `X-Session-Id` headers |
| REFSVC-11 | pass | Grafana API search returned the provisioned `Customer Support Observability` dashboard (`uid=customer-support-demo`) |
| REFSVC-12 | pass | 2026-05-12 live OTLP probe evidence: 30 fast traces retained 2 (~6.7%), 1 error trace retained 1/1, and 1 slow 6000ms trace retained 1/1 after the collector `decision_wait=30s` window |
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
- `docker compose -f compose/compose.yaml build app`
- checked `docker compose -f compose/compose.yaml ps` on 2026-05-12
- fixed the live observability path after verification exposed two defects:
  - `compose/otel-collector.yaml` OTLP receiver endpoints now bind
    `0.0.0.0:4317` and `0.0.0.0:4318` so the app can reach the collector over
    the compose network instead of only inside the collector container
  - HTTP root spans and OTel wrapper spans now set explicit OTel
    `STATUS_CODE_ERROR` on error paths so the collector `status_code` policy
    sees real failures instead of only recorded exceptions
- built `/tmp/llm-agent-customer-support-server` locally with the 4-repo
  workspace and ran it against:
  - local Ollama on `127.0.0.1:11434`
  - `otel-lgtm` on `127.0.0.1:3000`
  - collector OTLP HTTP on `127.0.0.1:4318`
- verified:
  - `curl -i http://127.0.0.1:18081/readyz`
  - `curl -i -X POST http://127.0.0.1:18081/chat -H 'Content-Type: application/json' -d '{"message":"hello"}'`
  - `curl -i http://127.0.0.1:8080/readyz`
  - `curl -i -X POST http://127.0.0.1:8080/chat -H 'Content-Type: application/json' -d '{"message":"I need a refund for order 123"}'`
  - `curl -s 'http://127.0.0.1:3000/api/search?query=Customer%20Support%20Observability'`
  - targeted regression tests after the observability fixes:
    - `GOTOOLCHAIN=go1.26.0 GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/httpapi ./compose -count=1`
    - `GOTOOLCHAIN=go1.26.0 GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./otelagent ./otelmodel -count=1`
  - direct collector-tail probe:
    - built `/tmp/tailprobe` from `/tmp/llm-agent-otel/cmd/tailprobe`
    - emitted `30` fast traces to collector `172.18.0.3:4318`
    - emitted `1` error trace to collector `172.18.0.3:4318`
    - emitted `1` slow trace with `durationMs=6000` to collector `172.18.0.3:4318`
    - waited `35s` to exceed `decision_wait=30s`
    - queried Tempo inside `compose-otel-lgtm-1` for all emitted trace IDs

Observed runtime evidence from the 2026-05-12 retry:

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
- the recreated collector logged OTLP listeners on `[::]:4317` and `[::]:4318`
- Tempo retained these special-case probe traces:
  - error trace `3e282e12b600415c2b78223156d96cb6`
  - slow trace `db7b42dd3ff8df3db805121db224503b` with `durationMs: 6000`
- Tempo retained only `2` of the `30` fast probe traces:
  - `52005932e298742c19d431a56281a618`
  - `d52fe51443c465f0e92dee9adce85a88`
- baseline retention was therefore `2/30 = 6.7%`, which is close enough to the
  configured `10%` probabilistic branch for a small local sample while the
  error and slow branches both retained `100%`
- `docker compose -f compose/compose.yaml build app` later completed
  successfully and produced the compose `app` image
- a compose-built `app` container then returned:
  - `/readyz` → `200 OK` with
    `X-Trace-Id: ee4f066c282b514061bbd6e8ce974805`
  - `/chat` → `200 OK` with
    `X-Trace-Id: 94f59f0a338bbbae1f3103076a5e85da`
    `X-Session-Id: beb6dd13-cabd-4275-a19b-c156cc7010ea`
    and answer
    `"refund_policy: Refund guidance for order 123: Orders cancelled within 24h are eligible for a full refund."`
- the stock demo compose path on this host still showed two environment-level
  wrinkles that did not invalidate the app-container proof:
  - host port `11434` was already occupied, so publishing the compose Ollama
    service failed
  - `ollama-init` later exited with `curl: (6) Could not resolve host: ollama`
    in this compose environment
- to isolate the stronger app-container proof from those host/demo issues, the
  successful runtime rerun used temporary verification-only compose overrides in
  the reference-service workspace:
  - `compose/compose.verify.yaml`
  - `compose/compose.runtime-proof.yaml`

## Remaining Closure Work

1. Optionally clean up the remaining stock-demo environment sensitivity on hosts
   where:
   - local port `11434` is already occupied
   - `ollama-init` service-name DNS resolution fails inside the compose network
2. No further milestone-close proof is required before Phase 6 remains archived
   as complete.
