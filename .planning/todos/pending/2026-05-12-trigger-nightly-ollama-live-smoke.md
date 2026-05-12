---
created: 2026-05-12T12:41:00+08:00
title: Trigger nightly ollama live smoke
area: planning
files:
  - .planning/STATE.md:92
  - .planning/PROJECT.md:77
  - .planning/phases/01-walking-skeleton-generate/01-06-SUMMARY.md:53
---

## Problem

The `nightly-ollama-live` workflow exists in `llm-agent-providers`, but the
first post-merge manual `workflow_dispatch` smoke run was deferred. That means
the code path is defined, but there is no recorded first-run proof from GitHub
Actions after landing.

## Solution

Manually trigger `nightly-ollama-live` once in GitHub Actions for
`llm-agent-providers` and confirm:

- workflow starts successfully
- Docker/testcontainers path works in GitHub-hosted runners
- pinned Ollama model bootstraps as expected

Record the result in planning state and close the todo if green.
