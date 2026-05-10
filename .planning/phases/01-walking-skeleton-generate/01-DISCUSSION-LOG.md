# Phase 1: Walking Skeleton — Generate (sync) only - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-10
**Phase:** 1-walking-skeleton-generate
**Areas discussed:** OpenAI API surface, Provider constructor, Error mapping, Conformance fixture format

---

## OpenAI API Surface

| Option | Description | Selected |
|--------|-------------|----------|
| Chat Completions only | Standard, broadest model coverage, deepest docs. Use `client.Chat.Completions.New(...)`. | ✓ |
| Responses API only | OpenAI's recommended new path. Stateful-friendly. Smaller model coverage. Doc churn. | |
| Both via `WithAPIVersion` option | Maximum flexibility. Adapter complexity, 2 fixture sets. Defer past Phase 1. | |

**User's choice:** Chat Completions (D-01)
**Notes:** Phase 1 only ships sync `Generate(ctx, Request) Response`. The Stateful-conversation advantage of Responses API is irrelevant for sync calls. Chat Completions is the proven, widely-documented path. Responses API can come in Phase 2 (streaming) or Phase 3 (tools) when its content-block model and built-in tool runner add real value, gated behind `WithAPIVersion(openai.APIChat | openai.APIResponses)`.

---

## Provider Constructor Pattern

| Option | Description | Selected |
|--------|-------------|----------|
| Functional options | `openai.New(openai.WithModel(...), openai.WithAPIKey(...))`. Go-idiomatic. | ✓ |
| Config struct | `openai.New(openai.Config{Model: ..., APIKey: ...})`. Simplest, JSON-friendly. | |
| Hybrid: Required struct + Options | `openai.New(openai.Required{Model: ...}, opts...)`. Mark required visually. | |

**User's choice:** Functional options (D-02)
**Notes:** Matches stdlib (`log/slog`), gRPC, and openai-go v3 itself. Default values clean (omit option = default). Adding options later is BC-additive (no struct field churn). Cost is verbose call sites — one-time cost per process. Universal options: `WithModel`, `WithAPIKey`, `WithHTTPClient`, `WithBaseURL`, `WithTimeout`. Per-provider extras with provider-prefixed names (`openai.WithOrganization`, `anthropic.WithBetaHeader`, `ollama.WithHost`). API key default: env var fallback (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, none for Ollama). No default model — explicit `WithModel(...)` required.

---

## Error Classification Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Per-adapter mapping + Author Guide table | Each adapter's `errors.go` wraps SDK errors into core `*llm.AuthError` etc. PROVIDER_AUTHORING.md documents recommended HTTP→type table. | ✓ |
| Central `errors.MapHTTPStatus(int) error` helper | Consistent, no duplication. Inflexible: can't map Anthropic `overloaded_error` (529 for "rate-limit-like" semantics) without coupling. | |
| Transparent passthrough; users `errors.As` raw SDK error | Minimal abstraction. Fails consistency goal — agents can't `errors.As(&llm.RateLimitError{}, ...)` portably. | |

**User's choice:** Per-adapter + Author Guide (D-03)
**Notes:** Recommended mapping table in PROVIDER_AUTHORING.md: 401/403 → AuthError, 429 → RateLimitError, 4xx other → InvalidRequestError, 5xx → TransientError, ctx.DeadlineExceeded → TransientError, network errors → TransientError. Provider overrides allowed: Anthropic `overloaded_error` 529 → RateLimitError (rate-limit-like), Ollama "model not pulled" 404 → InvalidRequestError. Original SDK error preserved via `errors.Unwrap`. Conformance suite asserts that 401/429/500 fixtures produce the expected typed errors regardless of provider.

---

## Conformance Fixture Format

| Option | Description | Selected |
|--------|-------------|----------|
| testdata/*.json + httptest.Server | Version-controlled, diff-friendly, no API keys in CI. One JSON file per scenario. Capture script per provider. | ✓ |
| Inline JSON in test code | Quick to write, illegible in source, refactor breaks compile. | |
| go-vcr / SDK record-replay | Real-API recording. Adds dep; works fine in sister repo. Heavier than needed for Phase 1. | |

**User's choice:** testdata/*.json + httptest (D-04)
**Notes:** Layout: `internal/contract/testdata/<provider>/<scenario>.json` (e.g., `openai/generate_429_with_retry_after.json`). Each JSON file has `{request: {body_assertions, ...}, response: {status, headers, body}}`. Conformance test loads JSON, starts httptest.Server, configures adapter via `WithBaseURL(server.URL)`, calls Generate, asserts. `scripts/capture-fixtures-<provider>.sh` runs locally with real API keys to produce/refresh fixtures. CI replays — no keys needed. `goleak.VerifyTestMain` integrated to catch goroutine leaks early (Pitfall 3 prevention before streaming lands in Phase 2). Format extends naturally to streaming (SSE chunks in `body`) and tools (multi-block content) for Phases 2/3.

---

## Claude's Discretion

- **Package layout in `llm-agent-providers`:** subpackages `openai/`, `anthropic/`, `ollama/` at top level; `internal/contract/` for shared harness. Match core repo's flat-subpackage style.
- **Constructor invocation order convention:** users typically write `WithModel` first, then `WithAPIKey`, then transport options. Readability hint, not a rule.
- **Test naming:** `TestGenerate_<Provider>_<Scenario>` (e.g., `TestGenerate_OpenAI_Happy`). Provider middle, scenario last.
- **Nightly Ollama-live workflow:** `llm-agent-providers/.github/workflows/nightly-ollama-live.yml`, separate from `test.yml`. Cron at 03:00 UTC. Pin model to `llama3.1:8b-instruct-q4_K_M` for fast container start.

## Deferred Ideas

- **Streaming on all 3 providers** — Phase 2 (CONF-03; OAI-02, ANT-02, OLL-02).
- **Native tool calling** — Phase 3.
- **Embeddings** — Phase 4.
- **Responses API for OpenAI** — Phase 2 or 3 with `WithAPIVersion(...)`.
- **Anthropic prompt caching** — P2 / v0.4.
- **OpenTelemetry instrumentation** — Phase 5 (decorator wrap).
- **Ollama per-model strategy table** — Phase 3 (tool calling).
