---
phase: 34-umbrella-coherence-gate-and-milestone-close
plan: 05
status: complete
completed_at: 2026-05-20
files_modified:
  - go.mod
  - go.sum
  - "(git: llm-agent-providers PR #8 merged @ efdef5a; tag v0.2.1 pushed)"
requirements: [ECO-05]
---

# Wave 5 SUMMARY — llm-agent-providers transitive cascade

## What shipped

- `llm-agent-providers/go.mod` bumped: `llm-agent v0.5.0 → v0.5.1`.
- No rag dep introduced (providers depends on core only).
- `llm-agent-providers` tagged **v0.2.1** on merge commit `efdef5a`, pushed.

## Why this slice exists

Discovered during Wave 5 planning: after Wave 2 retagged core to `v0.5.1`,
`providers/go.mod` (pinned at `core v0.5.0`) became stale by the
strict-equality dep-currency gate's rule — even though providers itself
was unchanged by the original cascade (providers has no rag dep). Operator
chose `"冲到底(加 Wave 4.5)"` to bump providers transitively.

## Flow

PR-merge flow (providers has identical branch protection to otel/cs):

1. Branch `chore/v1.1-cascade-bump`; bumped via `go get ...@v0.5.1`; `go mod tidy`.
2. Local: vet + build + test green across all 5 adapter packages (deepseek, minimax, openai, anthropic, ollama).
3. PR #8 opened. CI: `governance` 4s ✓, `go` 59s ✓, `auto-merge-owner` 6s ✓ (bot auto-merged).
4. Merge commit `efdef5a`; branch deleted; local main fast-forwarded.
5. Tag `v0.2.1` annotated; operator authorized push 2026-05-20.

## Verify

| Check | Result |
|---|---|
| BUMP-OK (core v0.5.1) | ✓ |
| NO-REPLACE | ✓ |
| NO-RAG-DEP | ✓ (none introduced) |
| PROVIDERS-GREEN (vet+build+test) | ✓ |
| TAG-OK + TAG-PUSHED | ✓ |

## Note

After this slice, `customer-support/go.mod`'s `providers v0.2.0` pin
became stale (the topological-order bug fixed in Wave 6, the cascade
follow-up).
