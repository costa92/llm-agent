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
- `llm-agent v0.3.0-pre.2` is tag-resolvable
- the missing sister-repo release publication prerequisite was closed later on
  2026-05-12 by publishing:
  - `llm-agent-otel v0.1.0`
  - `llm-agent-providers v0.1.0`
- post-push remote tag re-check via `git ls-remote` could not be repeated from
  this sandbox because DNS resolution to `github.com` later failed, but the
  preceding `git push origin main` + `git push origin v0.1.0` commands for both
  repos had already succeeded

Rerun evidence captured later on 2026-05-12 narrowed the blocker further:

- `docker compose -f compose/compose.yaml build app` now fails quickly instead
  of stalling
- failure point: `RUN go mod download`
- exact error:
  `github.com/costa92/llm-agent-otel@v0.1.0 ... invalid version: unknown revision v0.1.0`
- local sibling repos do contain the `v0.1.0` tags, so the remaining gap is
  not local release hygiene; it is container-side fetchability of those modules
  through the public Go module path / checksum path being used in the image

So the remaining compose-native app proof blocker is module accessibility from
inside the container build, then the container runtime path itself.

## Solution

When Docker/network conditions are favorable, rerun the full
`llm-agent-customer-support` compose stack with the app container included and
capture:

- `readyz` success
- `POST /chat` success
- `X-Trace-Id` correlation
- tail-sampling behavior or equivalent observability confirmation

The release publication prerequisite is now satisfied. The next meaningful
attempt should focus on:

- making `llm-agent-otel v0.1.0` and `llm-agent-providers v0.1.0` fetchable to
  the app image build, either by public visibility or private-module auth /
  `GOPRIVATE`-style wiring inside the build
- then re-running containerized `go mod download` / build behavior inside the
  app image
- full app-container startup
- `readyz`, `/chat`, `X-Trace-Id`, and observability proof capture

If successful, append the stronger proof to Phase 6 verification artifacts.
