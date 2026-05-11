# Phase 06-03 Summary

Date: 2026-05-11
Repo: `llm-agent-customer-support`
Plan: [06-03-PLAN.md](/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/06-reference-customer-support/06-03-PLAN.md)

## Objective

Split chat-provider and embedding-provider selection so the reference service
can run truthful mixed-provider combinations without changing the binary.

## Delivered

- Added `EmbeddingProvider` and `EmbeddingModel` to service config.
- Added provider-aware defaults for embedding selection:
  - default Ollama embeddings for Ollama chat
  - default OpenAI embeddings for Anthropic chat
- Added `internal/providers` with explicit chat-model and embedder factories.
- Added `DefaultEmbedderFactory(...)` in `internal/app`.
- Threaded the embedder into `App` state and exposed `EmbeddingInfo()`.
- Added tests covering:
  - config defaults and invalid embedding provider handling
  - OpenAI and Ollama embedding factory selection
  - explicit rejection of Anthropic as an embedding provider
  - mixed chat/embedder bootstrap through `app.New(...)`
- Updated the README to describe the independent provider-selection model.

## Files

- `/tmp/llm-agent-customer-support/internal/config/config.go`
- `/tmp/llm-agent-customer-support/internal/config/config_test.go`
- `/tmp/llm-agent-customer-support/internal/app/app.go`
- `/tmp/llm-agent-customer-support/internal/app/app_test.go`
- `/tmp/llm-agent-customer-support/internal/providers/providers.go`
- `/tmp/llm-agent-customer-support/README.md`

## Verification

Executed against the local 4-repo workspace:

```bash
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go test ./internal/... -count=1
GOWORK=/tmp/go.work GOCACHE=/tmp/go-build go build ./...
```

Result:

- `go test ./internal/...`: pass
- `go build ./...`: pass

## Notes

- Anthropic remains a truthful non-embedder in v0.3; this plan makes the
  supported workaround explicit instead of implicit.
- The embedder is now stored on `App` for later RAG/session-flow plans, but is
  not yet consumed by the HTTP transport layer.
