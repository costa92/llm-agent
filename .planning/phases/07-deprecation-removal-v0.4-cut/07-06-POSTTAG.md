# Phase 7 Post-Tag Checklist

Date: 2026-05-13
Trigger: publish `github.com/costa92/llm-agent v0.4.0`

Once the core `v0.4.0` tag exists remotely, run the following in order.

## 1. Update sister-repo `go.mod`

Repos:

- `/tmp/llm-agent-providers`
- `/tmp/llm-agent-otel`
- `/tmp/llm-agent-customer-support`

Set:

- `github.com/costa92/llm-agent v0.3.0-pre.2`

to:

- `github.com/costa92/llm-agent v0.4.0`

## 2. Verify each repo against the released tag

Existing local workspace:

- `/tmp/phase7-v04-audit/go.work`

Commands:

```bash
cd /tmp/llm-agent-providers && GOWORK=/tmp/phase7-v04-audit/go.work GOCACHE=/tmp/go-build go test ./...
cd /tmp/llm-agent-otel && GOWORK=/tmp/phase7-v04-audit/go.work GOCACHE=/tmp/go-build go test ./...
cd /tmp/llm-agent-customer-support && GOWORK=/tmp/phase7-v04-audit/go.work GOCACHE=/tmp/go-build go test ./...
```

## 3. Commit in sister repos

Suggested commit shape:

- `chore: bump llm-agent to v0.4.0`

## 4. Push and tag coordinated releases

Repos:

- `llm-agent-providers`
- `llm-agent-otel`
- `llm-agent-customer-support`

## 5. Final closeout

After the bumps are published:

- mark `DEPRC-04` complete in `.planning/REQUIREMENTS.md`
- update `.planning/STATE.md` to Phase 7 complete
- archive/transition the milestone
