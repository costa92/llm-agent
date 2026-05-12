---
created: 2026-05-12T12:43:00+08:00
closed: 2026-05-12T18:58:00+08:00
title: Rerun refsvc compose native proof
area: planning
files:
  - .planning/STATE.md:94
  - .planning/phases/06-reference-customer-support/06-VERIFICATION.md:14
  - .planning/v0.3-MILESTONE-AUDIT.md:20
---

## Outcome

Closed on 2026-05-12.

The stronger Phase 6 app-container proof now exists.

Verified facts captured on 2026-05-12:

- `docker compose -f compose/compose.yaml build app` succeeded in
  `/tmp/llm-agent-customer-support`, proving the published module path is now
  resolvable from the compose image build after the local compatibility fix and
  dependency metadata refresh.
- a compose-built `app` container then returned:
  - `GET /readyz` → `200 OK` with
    `X-Trace-Id: ee4f066c282b514061bbd6e8ce974805`
  - `POST /chat` → `200 OK` with
    `X-Trace-Id: 94f59f0a338bbbae1f3103076a5e85da`
    `X-Session-Id: beb6dd13-cabd-4275-a19b-c156cc7010ea`
    and answer
    `refund_policy: Refund guidance for order 123: Orders cancelled within 24h are eligible for a full refund.`

Environment-specific compose demo issues were also observed on the same date,
but they are no longer milestone blockers:

- stock `docker compose ... up -d app` hit a host-port conflict on `11434`
- `ollama-init` later failed with `curl: (6) Could not resolve host: ollama`
  in this host's compose environment

To isolate the app-container proof from those host/demo issues, the rerun used
temporary verification-only compose overrides in the reference-service
workspace:

- `compose/compose.verify.yaml`
- `compose/compose.runtime-proof.yaml`

This closes the remaining Phase 6 closeout todo in the planning repo. The
remaining compose demo sensitivity is archived as non-blocking environment tech
debt rather than an open milestone task.
