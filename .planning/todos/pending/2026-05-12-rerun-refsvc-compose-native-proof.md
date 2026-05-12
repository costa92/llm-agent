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

Additional blocking detail confirmed on 2026-05-12:

- the compose `app` image intentionally drops all local `replace` directives in
  `compose/Dockerfile`
- that means the container build requires published remote module versions for:
  - `github.com/costa92/llm-agent v0.3.0-pre.2`
  - `github.com/costa92/llm-agent-otel v0.1.0`
  - `github.com/costa92/llm-agent-providers v0.1.0`
- `llm-agent v0.3.0-pre.2` is tag-resolvable, but `llm-agent-otel` and
  `llm-agent-providers` currently have no local release tags and no verified
  remote `v0.1.0` release evidence in this session

So the compose-native app proof is blocked not only by Docker cold-start cost,
but by missing sister-repo release publication prerequisites.

## Solution

When Docker/network conditions are favorable, rerun the full
`llm-agent-customer-support` compose stack with the app container included and
capture:

- `readyz` success
- `POST /chat` success
- `X-Trace-Id` correlation
- tail-sampling behavior or equivalent observability confirmation

Before rerunning, ensure the compose build prerequisites are true:

- `llm-agent-otel v0.1.0` is tagged and fetchable remotely
- `llm-agent-providers v0.1.0` is tagged and fetchable remotely

If successful, append the stronger proof to Phase 6 verification artifacts.
