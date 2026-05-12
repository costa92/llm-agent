---
created: 2026-05-12T12:40:00+08:00
title: Enable sister repo branch protection
area: planning
files:
  - .planning/STATE.md:91
  - .planning/PROJECT.md:75
  - .planning/phases/00-keystone-interfaces/00-VERIFICATION.md:272
---

## Problem

The 3 sister repos were created and used during `v0.3`, but manual GitHub
branch protection on `main` was never confirmed as complete. That leaves the
multi-repo release discipline partially enforced in code and CI, but not fully
enforced at the repository policy layer.

## Solution

Use the GitHub web UI to enable branch protection on `main` for:

- `llm-agent-providers`
- `llm-agent-otel`
- `llm-agent-customer-support`

Require PR-based merges and required status checks at minimum. After the manual
step, update `STATE.md` and close this todo.
