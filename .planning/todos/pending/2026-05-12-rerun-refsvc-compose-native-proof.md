---
created: 2026-05-12T12:43:00+08:00
title: Rerun refsvc compose native proof
area: planning
files:
  - .planning/v0.3-MILESTONE-AUDIT.md:64
  - .planning/phases/06-reference-customer-support/06-VERIFICATION.md:119
  - .planning/milestones/v0.3-ROADMAP.md:252
---

## Problem

The `REFSVC-12` and overall Phase 6 closeout proof is good enough to archive,
but the strongest runtime evidence still used a compose dependency stack plus a
locally built app binary. A fully compose-native app-container rerun remained
environment-sensitive during closeout.

## Solution

When Docker/network conditions are favorable, rerun the full
`llm-agent-customer-support` compose stack with the app container included and
capture:

- `readyz` success
- `POST /chat` success
- `X-Trace-Id` correlation
- tail-sampling behavior or equivalent observability confirmation

If successful, append the stronger proof to Phase 6 verification artifacts.
