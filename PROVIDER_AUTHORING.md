# Provider Author Guide

**Version:** v0.1 (Phase 1 - Generate-only contract)  
**Applies to:** `github.com/costa92/llm-agent` v0.3.0+

> **What this is:** the contract a Go provider adapter must satisfy to claim
> conformance with `llm.ChatModel`. v0.1 covers Generate (sync) only.
> Streaming, tool calling, embeddings, and OTel-specific guidance are added in
> later milestone phases.

## 1. Audience and Scope

This document is for authors building a Go adapter around an LLM provider API
and exposing it as `llm.ChatModel`.

Canonical Phase 1 examples live in the sister repo:

- `github.com/costa92/llm-agent-providers/openai`
- `github.com/costa92/llm-agent-providers/anthropic`
- `github.com/costa92/llm-agent-providers/ollama`

All three are Generate-only adapters in Phase 1. New adapters should
structurally mirror them.

Phase 1 scope:

- one-shot `Generate`
- stable `Info()`
- typed provider errors
- shared conformance via `internal/contract`

Out of scope for v0.1:

- streaming
- native tool calling
- embeddings
- structured outputs
- three-state cost record beyond `reported` / `unknown`
- retry state machine
- OTel decorators

## 2. Contract

Every provider implements `llm.ChatModel`:

```go
type ChatModel interface {
	Generate(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (StreamReader, error)
	Info() ProviderInfo
}
```

Source: `llm/chatmodel.go`.

Phase 1 boundary: `Stream` exists on the interface, but provider adapters may
return a not-implemented error until Phase 2. The method is present now so the
type shape stays stable across the walking skeleton.

Important supporting types:

- `llm.Request`
- `llm.Response`
- `llm.Message`
- `llm.Usage`
- `llm.ProviderInfo`
- `llm.Capabilities`
- `llm.AuthError`
- `llm.RateLimitError`
- `llm.InvalidRequestError`
- `llm.TransientError`

All implementations must be safe for concurrent use. Concurrent `Generate` and
`Stream` calls on the same value are part of the contract.

## 3. Generate Contract

`Generate(ctx, req)` must satisfy the following behavior:

| Aspect | Required behavior |
|---|---|
| `req.Messages` | Preserve user/assistant/tool turn order and map to the provider wire format |
| `req.SystemPrompt` | Lift into the provider's system-prompt home |
| `req.MaxOutputTokens > 0` | Pass through to the provider max-tokens field |
| `req.Temperature != nil` | Pass through when the provider supports it |
| `Response.Provider` | Canonical provider name such as `openai`, `anthropic`, `ollama` |
| `Response.Model` | The model bound at construction time |
| `Response.Usage.Source` | `llm.UsageReported` when token counts are present; otherwise `llm.UsageUnknown` |
| `Response.FinishReason` | Normalize to the existing finish-reason constants |
| Errors | Wrap into the typed taxonomy in section 5 while preserving the SDK error in `Wrapped` |

Provider-specific notes from the three Phase 1 adapters:

- OpenAI and Ollama derive a system-role message from `req.SystemPrompt`.
- Anthropic lifts `req.SystemPrompt` into the top-level `system` field.
- Phase 1 adapters do not honor tools; `Capabilities.Tools` remains `false`.
- If the provider does not report token usage for a response, do not guess in
  Phase 1. Return `Usage.Source = llm.UsageUnknown`.

## 4. Constructor Pattern

Phase 1 uses construction-time model binding plus functional options.

Required constructor shape:

```go
func New(opts ...Option) (*X, error)
```

Required option pattern:

```go
type Option func(*config)
```

Canonical expectations:

| Option | Purpose | Notes |
|---|---|---|
| `WithModel(string)` | Bind the model | Required; `New` must fail if empty |
| `WithAPIKey(string)` | Override env fallback | Not needed for keyless providers |
| `WithBaseURL(string)` | Point at custom endpoint | Used for tests, proxies, local hosts |
| `WithHTTPClient(*http.Client)` | Inject custom transport | Retry wrappers, tracing, mocks |
| `WithTimeout(time.Duration)` | Default request timeout | Distinct from per-call `ctx` |

Canonical sketch:

```go
type Option func(*config)

func WithModel(m string) Option   { return func(c *config) { c.model = m } }
func WithAPIKey(k string) Option  { return func(c *config) { c.apiKey = k } }
func WithBaseURL(u string) Option { return func(c *config) { c.baseURL = u } }

func New(opts ...Option) (*OpenAI, error) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.model == "" {
		return nil, errors.New("openai: WithModel is required")
	}
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAI{/* ... */}, nil
}
```

This is the same construction pattern used by the Phase 1 OpenAI,
Anthropic, and Ollama adapters.

## 5. Error Taxonomy

Every adapter should wrap transport or SDK failures into one of the typed
errors in `llm/errors.go`, preserving the original SDK error in `Wrapped` so
callers can still use `errors.As` on provider-specific error types.

Recommended HTTP-status -> typed-error mapping table:

| HTTP / Cause | Typed error |
|---|---|
| 401, 403 | `*llm.AuthError` |
| 429 | `*llm.RateLimitError` |
| 4xx other (400, 404, 422, etc.) | `*llm.InvalidRequestError` |
| 5xx | `*llm.TransientError` |
| network I/O, EOF, connection reset | `*llm.TransientError` |
| `context.DeadlineExceeded` | `*llm.TransientError` |
| `context.Canceled` | return as-is; do not wrap |

Additional guidance:

- Populate `Provider` on every typed error.
- Populate `RetryAfter` on `RateLimitError` when the provider exposes it.
- Populate `Reason` on `RateLimitError` when the provider surfaces a stable
  machine-readable quota discriminator.
- Preserve provider detail in `Wrapped`; do not stringify and discard it.

Example detection:

```go
var authErr *llm.AuthError
if errors.As(err, &authErr) {
	// credentials or permission failure
}
```

## 6. Conformance Test Pattern

Phase 1 provider adapters are expected to validate themselves against the
shared conformance harness in the sister repo:

- `github.com/costa92/llm-agent-providers/internal/contract`

The pattern is:

1. Create provider-specific fixture JSON files for happy-path and error cases.
2. Add an adapter factory that returns your `llm.ChatModel`.
3. Reuse `LoadFixture(...)` and `AssertGenerate(...)`.
4. Keep provider mapping logic in small `map.go` or `errors.go` helpers so the
   fixture matrix stays readable.

For local-only providers such as Ollama, Phase 1 also permits a build-tagged
live test. The nightly workflow runs it outside PR CI so real-container drift
does not block normal development.

## 7. Phase 1 Boundary

This guide is intentionally narrow. A provider can claim v0.1 conformance only
for Generate-only behavior.

Not done yet:

- no streaming contract guidance
- no `StreamEvent` validation rules
- no native `ToolCaller` guidance
- no `Embedder` guidance
- no structured-output guidance
- no estimated token accounting
- no retry state machine
- no OTel wrapper or semconv rules

Do not pre-invent these behaviors in a Phase 1 adapter. Match the current
contract first, then extend in the milestone phase that defines the behavior.

## 8. Cross-References

Core repo:

- `llm/chatmodel.go`
- `llm/types.go`
- `llm/info.go`
- `llm/errors.go`
- `llm/scripted.go`
- `llm/chat_only_mock.go`

Canonical sister-repo examples:

- `github.com/costa92/llm-agent-providers/openai`
- `github.com/costa92/llm-agent-providers/anthropic`
- `github.com/costa92/llm-agent-providers/ollama`
- `github.com/costa92/llm-agent-providers/internal/contract`

Versioning note:

- v0.1 of this guide corresponds to Phase 1 of the v0.3 roadmap.
- v0.2 will add streaming guidance after Phase 2 lands.
- v0.3 will add tools and embeddings guidance after Phases 3 and 4 land.
