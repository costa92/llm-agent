---
created: 2026-05-12T14:25:00+08:00
title: Publish sister repo release tags
area: planning
files:
  - /tmp/llm-agent-customer-support/go.mod:7
  - /tmp/llm-agent-customer-support/go.mod:8
  - /tmp/llm-agent-customer-support/compose/Dockerfile:6
  - .planning/todos/pending/2026-05-12-rerun-refsvc-compose-native-proof.md:16
---

## Problem

The compose-native Phase 6 proof cannot succeed until the app container can
build without local sibling `replace` directives. `compose/Dockerfile` drops
those `replace` lines before `go mod download`, so the build depends on remote,
fetchable release versions of the sister repos.

As verified on 2026-05-12:

- `github.com/costa92/llm-agent v0.3.0-pre.2` is tag-resolvable
- `github.com/costa92/llm-agent-otel v0.1.0` has no confirmed release tag
  evidence in this session
- `github.com/costa92/llm-agent-providers v0.1.0` has no confirmed release tag
  evidence in this session

Without those published tags, the compose-native app build is blocked before
runtime verification even starts.

## Solution

Publish and verify the missing sister-repo releases:

- tag and push `llm-agent-otel v0.1.0`
- tag and push `llm-agent-providers v0.1.0`
- verify the tags are fetchable remotely (`git ls-remote --tags` and/or
  `go mod download` without local `replace`)

Once those release prerequisites are true, retry the compose-native proof todo.
