# Phase 1: Three-provider walking skeleton — Generate (sync) only — Research

**Researched:** 2026-05-10
**Domain:** Go provider adapters (OpenAI / Anthropic / Ollama), shared httptest+testdata conformance harness, testcontainers-go nightly Ollama-live CI, Provider Author Guide v0.1
**Confidence:** HIGH on SDK shapes (Context7-verified for all 3 SDKs + testcontainers + goleak), HIGH on conformance approach (D-04 locked + httptest is stdlib), HIGH on provider error mapping table (D-03 locked).

---

## Summary

Phase 1 implements three Generate-only sister-repo adapters against the locked `llm.ChatModel` contract from Phase 0. The contract is small (3 methods: Generate / Stream / Info), the wire formats are well-documented in their respective SDKs, and CONTEXT.md has resolved the 4 design questions that would otherwise dominate. The remaining uncertainty lives in **SDK ergonomics** (exact param-struct field names, error-type access patterns, base-URL injection) — all verifiable today via the official SDK docs. This research bottoms those out.

Three adapters land independently; their conformance is proved by a shared `internal/contract/` harness that loads `testdata/<provider>/<scenario>.json` files, starts an `httptest.Server`, configures the adapter with `WithBaseURL(server.URL)`, calls `Generate`, and asserts the normalized `llm.Response` matches expectation. The same harness is extended in Phases 2/3/4 — JSON files just gain SSE/streaming/tool fields. Pitfall 3 (goroutine leaks) is guarded via `go.uber.org/goleak.VerifyTestMain` even though Generate is sync — the harness lands here so Phase 2's streaming work inherits it.

**Primary recommendation:** Build the three adapters as parallel mechanically-similar packages — same constructor shape (`New(opts ...Option) (*X, error)` returning error if `WithModel` not provided), same error-mapping pattern (per-adapter `wrapErr(err) error` helper using `errors.As` against the SDK's typed error), same testdata-fixture conformance shape. Stream method on each adapter is a Phase-1 stub returning `errors.New("openai: streaming not implemented in Phase 1")` — keeps the `ChatModel` interface satisfied without dragging Phase 2 work in.

---

## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 (P1):** Phase 1 OpenAI adapter targets the **Chat Completions API only** (`client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{...})` via openai-go v3). Responses API is deferred to Phase 2 (streaming) or Phase 3 (tools). When Responses lands, it's gated via `WithAPIVersion(...)` constructor option — default stays Chat Completions for v0.3.

**D-02 (P1):** **Functional options across all three adapters.** Uniform constructor: `New(opts ...Option) (*X, error)`. Universal options for every adapter: `WithModel(string)` (required; constructor returns error if unset), `WithAPIKey(string)`, `WithHTTPClient(*http.Client)`, `WithBaseURL(string)`, `WithTimeout(time.Duration)`. Per-provider extras: `openai.WithOrganization`, `anthropic.WithBetaHeader`, `ollama.WithHost` (alias for `WithBaseURL` since Ollama uses `OLLAMA_HOST` env). API-key sourcing default: env var fallback per provider (`OPENAI_API_KEY` / `ANTHROPIC_API_KEY`); Ollama has no key — defaults base URL to `http://localhost:11434`. **No default model:** every adapter requires explicit `WithModel(...)`.

**D-03 (P1):** **Per-adapter mapping** of HTTP status → typed `llm.*Error`. PROVIDER_AUTHORING.md documents the recommended table; adapters override per provider quirk. Recommended table (canonical):

| HTTP / Cause | Typed error |
|--------------|-------------|
| 401, 403 | `*llm.AuthError` |
| 429 | `*llm.RateLimitError` |
| 4xx other (400, 404, 422, …) | `*llm.InvalidRequestError` |
| 5xx | `*llm.TransientError` |
| `errors.Is(err, context.DeadlineExceeded)` | `*llm.TransientError` |
| `errors.Is(err, context.Canceled)` | propagate as-is (NOT a typed `llm.*Error`) |
| network error (DNS, TCP reset) | `*llm.TransientError` |

Provider-specific overrides:
- OpenAI: `insufficient_quota` (429 with specific `code`) → `*llm.RateLimitError` with `Reason: "quota_exhausted"`
- Anthropic: `overloaded_error` (529) → `*llm.RateLimitError` (semantically rate-limit-like)
- Anthropic: `invalid_request_error` (400) → `*llm.InvalidRequestError`
- Ollama: model-not-pulled (404 with specific message body) → `*llm.InvalidRequestError`

Original SDK error preserved via `errors.Unwrap` chain — callers can `errors.As(err, &openaiErr)` for provider detail. **No shared `llm-agent-providers/internal/errors` helper** — each adapter wraps its own SDK's typed error independently.

**D-04 (P1):** **`testdata/*.json` files + `httptest.Server` loader.**
- Layout: `internal/contract/testdata/<provider>/<scenario>.json`
- Each fixture contains both **request** assertions (method, path, body assertions) and **response** (status, headers, verbatim body from real-API capture)
- Conformance test loads, starts `httptest.Server`, configures adapter via `WithBaseURL(server.URL)`, calls Generate, asserts response matches
- `scripts/capture-fixtures-<provider>.sh` — local-only one-shot real-API capture (never CI)
- `goleak.VerifyTestMain` in conformance suite (Pitfall 3 guard for Phase 2)
- **Not chosen:** `dnaeon/go-vcr` (extra dep with version ceremony) or inline JSON in test code (illegible, recompile-to-edit)

### Claude's Discretion

- **Package layout** in `llm-agent-providers`: subpackages `openai/`, `anthropic/`, `ollama/` at top level; `internal/contract/` for the shared harness. No `pkg/` or `cmd/` subdirs.
- **Default option ordering** in constructor invocations: `Model` first by readability convention.
- **Test naming convention:** `TestGenerate_OpenAI_Happy`, `TestGenerate_Anthropic_429`, `TestGenerate_Ollama_404ModelNotPulled`. Matches Go's `TestSubject_Variant`.
- **Nightly Ollama-live workflow location:** `llm-agent-providers/.github/workflows/nightly-ollama-live.yml` (separate from `test.yml`). Schedule `cron: '0 3 * * *'`. Pulls `llama3.1:8b-instruct-q4_K_M`.

### Deferred Ideas (OUT OF SCOPE)

- **Streaming on all 3 providers** — Phase 2 (CONF-03; OAI-02, ANT-02, OLL-02). Phase 1 stays sync-only; Stream method on each adapter returns a clear "not implemented in Phase 1" error.
- **Native tool calling** — Phase 3. Tools may pass through Request struct in Phase 1 but adapters NOT required to honor them (recommendation below: ignore `Request.Tools` entirely in Phase 1; capability `Tools: false`).
- **Embeddings** — Phase 4.
- **Responses API for OpenAI** — Phase 2 / 3. No `WithAPIVersion(...)` option in Phase 1.
- **Anthropic prompt caching** — P2 / v0.4.
- **Cost-table for token-cost estimation** — DIFF-04. Phase 1 just exposes `Usage` as-Reported.
- **OpenTelemetry instrumentation** — Phase 5. Phase 1 exposes `WithHTTPClient` so users CAN OTel-wrap later, but adapters do NOT add OTel themselves.
- **Per-model strategy table (Ollama)** — Phase 3 (OLL-03 tool calling). Phase 1 binds a model and calls `/api/chat` regardless.

---

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| OAI-01 | Implements `ChatModel.Generate` against `github.com/openai/openai-go/v3` (Chat Completions only per D-01) | §"OpenAI adapter shape" — verified `client.Chat.Completions.New` signature + `option.WithBaseURL` + error type `*openai.Error` w/ `StatusCode` |
| OAI-05 | Typed error taxonomy: `RateLimitError`, `AuthError`, `InvalidRequestError`, `TransientError` mapped from openai-go errors | §"Error mapping per adapter" — `errors.As(err, &apierr)` against `*openai.Error.StatusCode` |
| ANT-01 | Implements `ChatModel.Generate` against `github.com/anthropics/anthropic-sdk-go` | §"Anthropic adapter shape" — verified `client.Messages.New` non-beta + `System []TextBlockParam` top-level field |
| ANT-05 | Typed error taxonomy mapped from anthropic-sdk-go errors | §"Error mapping per adapter" — `*apierror.Error` w/ `StatusCode` and `RawJSON()` |
| OLL-01 | Implements `ChatModel.Generate` against `github.com/ollama/ollama/api` | §"Ollama adapter shape" — verified `api.NewClient(*url.URL, *http.Client)` + `client.Chat(ctx, req, fn)` callback shape |
| OLL-05 | Typed error taxonomy | §"Error mapping per adapter" — Ollama returns plain `error`; need custom HTTP transport to capture status |
| OLL-08 | Nightly testcontainers Ollama-live CI | §"CI YAML sketches" — `tcollama.Run(ctx, "ollama/ollama:0.5.7")` + `Exec(ctx, []string{"ollama", "pull", ...})` + `ConnectionString()` |
| CONF-01 | Shared httptest harness | §"Conformance harness shape" — `internal/contract/contract.go` LoadFixture + NewMockServer + AssertGenerate helpers |
| CONF-02 | Generate conformance: request shape, response shape, error taxonomy, finish-reason normalization | §"Conformance harness shape" — fixture-table-driven `generate_test.go` running adapter factories |
| CONF-07 | Capture script per provider | §"Capture scripts" — bash one-shots calling real APIs with key from env |
| CONF-08 | goleak integration in conformance suite | §"goleak integration" — `func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }` in `internal/contract/main_test.go` |
| CORE-11 | Provider Author Guide v0.1 in llm-agent core | §"PROVIDER_AUTHORING.md v0.1 outline" |

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Generate sync (OpenAI / Anthropic / Ollama) | Adapter package (sister repo) | Core `llm.ChatModel` interface | Wire-format-specific work — owned by adapter; the contract (shape of `Request`/`Response`/`ProviderInfo`) is owned by core and reused as-is |
| Error classification (HTTP → typed) | Adapter package (`<provider>/errors.go`) | Core `llm.*Error` types (`llm/errors.go` … `llm.AuthError`/`RateLimitError`/`InvalidRequestError`/`TransientError`) | D-03: per-adapter mapping; the typed error universe is shared (defined in core); the mapping logic is per-provider because SDK error types differ |
| Conformance harness (testdata + httptest) | `internal/contract/` package in sister repo | Adapters expose factories | Internal package (not importable by third parties) iterates adapter factories against the same fixture matrix |
| Nightly live CI (Ollama only) | Sister-repo `.github/workflows/nightly-ollama-live.yml` | testcontainers-go module | GitHub Actions runner has Docker pre-installed; testcontainers handles container lifecycle |
| PROVIDER_AUTHORING.md | Core repo (`llm-agent/PROVIDER_AUTHORING.md`) | — | The contract is in core; the guide describing how to write a provider lives where the contract lives |
| Sister-repo `require github.com/costa92/llm-agent v0.3.0-pre.1` | sister repo `go.mod` (already in place from Phase 0) | — | Verified live: tag pushed; resolves |

**Sanity-check note for the planner:** Phase 1 does NOT touch the core repo's `llm/` surface — Phase 0 locked it. The only core-repo deliverable is `PROVIDER_AUTHORING.md` (a markdown file). All code lands in `llm-agent-providers`. Plan tasks accordingly.

---

## Standard Stack

### Core (per CONTEXT.md and `.planning/research/STACK.md`)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/openai/openai-go/v3` | v3.35.0 (verified 2026-05-07) | OpenAI Chat Completions adapter | Official; full Chat Completions / Responses / Embeddings / streaming / tools coverage; replaces unofficial `sashabaranov/go-openai` for new code |
| `github.com/anthropics/anthropic-sdk-go` | v1.41.0 (verified 2026-05-06) | Anthropic Messages adapter | Official; v1+ stable; `Messages.New` returns `*anthropic.Message`; error type `*apierror.Error` |
| `github.com/ollama/ollama/api` | v0.23.2 (verified 2026-05-07) | Ollama `/api/chat` adapter | Official sub-module of `ollama/ollama` (used by the CLI); `api.NewClient(base *url.URL, http *http.Client) *Client` for clean httptest injection (verified — see [Ollama issue #2948](https://github.com/ollama/ollama/issues/2948)); pre-v1.0 (pin minor, accept some churn) |
| `github.com/testcontainers/testcontainers-go` | v0.33.x+ | Nightly Ollama-live container | Pre-installed Docker on GitHub Actions Linux runners; module `modules/ollama` provides `Run(ctx, image, opts...) (*OllamaContainer, error)` |
| `github.com/testcontainers/testcontainers-go/modules/ollama` | matches parent | Ollama-specific Run/Exec/ConnectionString | Modern API: `tcollama.Run(ctx, "ollama/ollama:0.5.7")` returns container; `container.Exec(ctx, []string{"ollama", "pull", "llama3.1:8b-instruct-q4_K_M"})` to pre-pull; `container.ConnectionString(ctx)` returns `http://host:port` |
| `go.uber.org/goleak` | v1.3.0+ | Goroutine-leak detection in conformance suite | Stdlib-only constraint applies to **core** only; `llm-agent-providers` may take this as a test dep (Pitfall 3 guard) |
| `net/http`, `net/http/httptest`, `encoding/json`, `testing` | stdlib (Go 1.26) | Conformance harness backbone | No external dep needed for the test-server stub |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/go-cmp/cmp` | v0.6.x+ | Diff-friendly assertion in conformance tests | Compare normalized `llm.Response` against expected without `reflect.DeepEqual`'s opacity. Optional — `t.Errorf` with field-by-field is sufficient for Phase 1's small assertion surface |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| openai-go/v3 Chat Completions | openai-go/v3 Responses API | D-01 locks Chat Completions for Phase 1 (broadest model coverage; deepest docs; sync `Generate` doesn't need Responses' stateful-conversation feature) |
| anthropic-sdk-go non-beta `client.Messages.New` | `client.Beta.Messages.New` | Beta path requires `Betas: []anthropic.AnthropicBeta{...}` and may rename. Phase 1 uses non-beta `MessageNewParams` for stability; beta features (prompt caching, structured outputs) are P2 / v0.4 |
| `api.NewClient(base, http)` (Ollama) | `api.ClientFromEnvironment()` | The latter only honors `OLLAMA_HOST` env; the former lets Phase 1 inject httptest-server URLs and a custom `*http.Client`. We use both: env-fallback when no `WithBaseURL`/`WithHTTPClient`/`WithHost` provided |
| `dnaeon/go-vcr` cassettes | `testdata/*.json` + httptest | D-04 chose JSON+httptest — version-controlled, diff-friendly, no dep ceremony, JSON inspectable in editor |
| `tcollama.RunContainer(ctx, req)` | `tcollama.Run(ctx, image, opts...)` | `RunContainer` is **deprecated** in current testcontainers-go releases — use `Run(ctx, image, opts...)` per [official docs](https://golang.testcontainers.org/modules/ollama/) |

**Installation (in `llm-agent-providers/go.mod`):**

```bash
# Already in providers' go.mod from Phase 0:
#   require github.com/costa92/llm-agent v0.3.0-pre.1

go get github.com/openai/openai-go/v3@v3.35.0
go get github.com/anthropics/anthropic-sdk-go@v1.41.0
go get github.com/ollama/ollama/api@v0.23.2
go get -t github.com/testcontainers/testcontainers-go
go get -t github.com/testcontainers/testcontainers-go/modules/ollama
go get -t go.uber.org/goleak
# go-cmp is optional; add only if conformance assertions need diff
```

**Version verification (planner: run before pinning in `go.mod`):**

```bash
# Confirm latest published versions:
go list -m -versions github.com/openai/openai-go/v3
go list -m -versions github.com/anthropics/anthropic-sdk-go
go list -m -versions github.com/ollama/ollama
go list -m -versions github.com/testcontainers/testcontainers-go
go list -m -versions github.com/testcontainers/testcontainers-go/modules/ollama
go list -m -versions go.uber.org/goleak
```

CONTEXT.md and STACK.md cite versions verified on **2026-05-10**; the planner should re-confirm at plan-time and document any drift.

---

## Architecture Patterns

### System Architecture Diagram

```
┌───────────────────────── llm-agent-providers (sister repo) ─────────────────────────┐
│                                                                                       │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                              │
│  │  openai/    │     │ anthropic/  │     │  ollama/    │   3 parallel adapter pkgs    │
│  │             │     │             │     │             │   each ~5 files (sketch §)   │
│  │ openai.go   │     │ anthropic.go│     │ ollama.go   │                              │
│  │ options.go  │     │ options.go  │     │ options.go  │                              │
│  │ errors.go   │     │ errors.go   │     │ errors.go   │                              │
│  │ map.go      │     │ map.go      │     │ map.go      │   SDK ↔ llm type mapping     │
│  │ *_test.go   │     │ *_test.go   │     │ *_test.go   │   per-adapter unit tests     │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘                              │
│         │                   │                   │                                     │
│         │ implements        │ implements        │ implements                          │
│         ▼                   ▼                   ▼                                     │
│  ┌──────────────────────────────────────────────────────┐                             │
│  │  github.com/costa92/llm-agent/llm.ChatModel          │  imported from CORE         │
│  │   • Generate(ctx, Request) (Response, error)         │  Phase 0 frozen surface     │
│  │   • Stream(ctx, Request)   (StreamReader, error)     │   (Phase 1: Stream returns  │
│  │   • Info() ProviderInfo                              │    "not implemented" err)   │
│  └──────────────────────────────────────────────────────┘                             │
│                                                                                       │
│         ▲ exercised by                                                                │
│         │                                                                             │
│  ┌──────┴──────────────────────────────────────────────┐                              │
│  │  internal/contract/  (the shared conformance harness)│                              │
│  │   • contract.go — LoadFixture / NewMockServer /     │                              │
│  │                   AssertGenerate helpers            │                              │
│  │   • generate_test.go — table-driven over (factory,  │                              │
│  │                       fixture) pairs                │                              │
│  │   • main_test.go — TestMain → goleak.VerifyTestMain │                              │
│  │   • testdata/                                       │                              │
│  │       openai/{generate_happy,401,429,500}.json      │                              │
│  │       anthropic/{generate_happy,400,529}.json       │                              │
│  │       ollama/{generate_happy,404_not_pulled,500}.json│                              │
│  └──────────────────────────────────────────────────────┘                              │
│                                                                                       │
│  scripts/                                                                             │
│   capture-fixtures-openai.sh                                                          │
│   capture-fixtures-anthropic.sh        local-only; never CI                           │
│   capture-fixtures-ollama.sh                                                          │
│                                                                                       │
│  .github/workflows/                                                                   │
│   test.yml                  (existing — Phase 0)                                      │
│   release-precheck.yml      (existing — Phase 0)                                      │
│   nightly-ollama-live.yml   (NEW — Phase 1)                                           │
│                                                                                       │
└───────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────── llm-agent (core) ──────────────────────────────────┐
│                                                                                  │
│  PROVIDER_AUTHORING.md  (NEW — Phase 1; CORE-11)                                 │
│   v0.1: Generate contract; functional-options pattern;                           │
│         HTTP→typed-error mapping table; conformance test pattern                 │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Recommended package layout (each adapter)

```
llm-agent-providers/openai/
├── openai.go        # type *OpenAI; New(opts...); Generate; Stream (stub); Info
├── options.go       # type Option func(*config); WithModel/WithAPIKey/WithBaseURL/WithHTTPClient/WithTimeout/WithOrganization
├── map.go           # request mapping (llm.Request → openai.ChatCompletionNewParams) + response (openai.ChatCompletion → llm.Response)
├── errors.go        # wrapErr(err) error — HTTP-status switch on *openai.Error.StatusCode
├── doc.go           # package doc + capability negotiation snippet
├── openai_test.go   # unit tests against httptest.Server (per-adapter)
└── README.md        # 1-screen: install + minimal Generate example + nightly-CI mention
```

(Same shape for `anthropic/` and `ollama/`.)

### Pattern 1: Functional-options constructor (D-02 canonical)

**What:** Uniform `New(opts ...Option) (*X, error)` — required `WithModel`, env-var-fallback for API key, optional override hooks.

**When to use:** Every adapter. Every Phase-1 plan that touches a constructor.

**Example (OpenAI; same shape for all 3):**

```go
// Source: D-02 (locked) + Context7 /openai/openai-go (option.WithAPIKey/WithBaseURL/WithHTTPClient verified)
package openai

import (
    "errors"
    "net/http"
    "os"
    "time"

    "github.com/costa92/llm-agent/llm"
    openai "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
)

type OpenAI struct {
    client *openai.Client
    info   llm.ProviderInfo
}

type config struct {
    apiKey       string
    model        string
    baseURL      string
    httpClient   *http.Client
    timeout      time.Duration
    organization string
}

type Option func(*config)

func WithAPIKey(k string) Option        { return func(c *config) { c.apiKey = k } }
func WithModel(m string) Option         { return func(c *config) { c.model = m } }
func WithBaseURL(u string) Option       { return func(c *config) { c.baseURL = u } }
func WithHTTPClient(h *http.Client) Option { return func(c *config) { c.httpClient = h } }
func WithTimeout(d time.Duration) Option { return func(c *config) { c.timeout = d } }
func WithOrganization(o string) Option  { return func(c *config) { c.organization = o } }

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
    var sdkOpts []option.RequestOption
    if cfg.apiKey != "" {
        sdkOpts = append(sdkOpts, option.WithAPIKey(cfg.apiKey))
    }
    if cfg.baseURL != "" {
        sdkOpts = append(sdkOpts, option.WithBaseURL(cfg.baseURL))
    }
    if cfg.httpClient != nil {
        sdkOpts = append(sdkOpts, option.WithHTTPClient(cfg.httpClient))
    }
    if cfg.organization != "" {
        sdkOpts = append(sdkOpts, option.WithHeader("OpenAI-Organization", cfg.organization))
    }
    if cfg.timeout > 0 {
        sdkOpts = append(sdkOpts, option.WithRequestTimeout(cfg.timeout))
    }
    client := openai.NewClient(sdkOpts...)
    return &OpenAI{
        client: &client,
        info: llm.ProviderInfo{
            Provider:     "openai",
            Model:        cfg.model,
            Capabilities: llm.Capabilities{Tools: false, Embeddings: false, StructuredOutputs: false, PromptCaching: false},
        },
    }, nil
}

func (o *OpenAI) Info() llm.ProviderInfo { return o.info }
```

### Pattern 2: SDK-error wrap with `errors.As` chain (D-03 canonical)

**What:** Single `wrapErr(err error) error` helper per adapter that uses `errors.As` against the SDK's typed-error. Returns the appropriate `*llm.AuthError` / `*llm.RateLimitError` / `*llm.InvalidRequestError` / `*llm.TransientError` with the SDK error preserved in the `Wrapped` field for `errors.Unwrap`.

**When to use:** Every adapter `Generate` method wraps SDK errors before returning.

**Example (OpenAI):**

```go
// Source: D-03 (locked) + Context7 /openai/openai-go (Error type with StatusCode/Type/Message; errors.As verified)
package openai

import (
    "context"
    "errors"
    "net"

    "github.com/costa92/llm-agent/llm"
    openai "github.com/openai/openai-go/v3"
)

func wrapErr(err error) error {
    if err == nil {
        return nil
    }
    // ctx.Canceled propagates as-is per D-03
    if errors.Is(err, context.Canceled) {
        return err
    }
    if errors.Is(err, context.DeadlineExceeded) {
        return &llm.TransientError{Provider: "openai", Wrapped: err}
    }

    var apiErr *openai.Error
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 401, 403:
            return &llm.AuthError{Provider: "openai", Wrapped: err}
        case 429:
            // OpenAI provider-specific override: insufficient_quota → RateLimit w/ reason
            reason := ""
            if apiErr.Type == "insufficient_quota" || apiErr.Code == "insufficient_quota" {
                reason = "quota_exhausted"
            }
            return &llm.RateLimitError{
                Provider:   "openai",
                RetryAfter: apiErr.Headers().Get("Retry-After"),
                Reason:     reason,
                Wrapped:    err,
            }
        case 500, 502, 503, 504:
            return &llm.TransientError{Provider: "openai", Wrapped: err}
        default:
            if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
                return &llm.InvalidRequestError{Provider: "openai", Wrapped: err}
            }
        }
    }

    // Network-level error (DNS, TCP reset)
    var netErr net.Error
    if errors.As(err, &netErr) {
        return &llm.TransientError{Provider: "openai", Wrapped: err}
    }
    return err
}
```

> **NOTE for planner:** the `Wrapped`/`RetryAfter`/`Reason` fields above are the **expected** shapes on `llm.AuthError` / `llm.RateLimitError` / `llm.InvalidRequestError` / `llm.TransientError`. Phase 0 ratified the existence of these typed errors but the current `llm/errors.go` only has the two sentinels (`ErrCapabilityNotSupported`, `ErrScriptExhausted`) — see [§"Open Questions"](#open-questions-resolved) Q1 RESOLVED for the recommended core-repo extension. **This is a small core-repo deliverable that needs to land at Phase 1 open** before adapters can compile.

### Pattern 3: Per-adapter SDK ↔ llm type mapping (each `map.go`)

**What:** Two functions per adapter:
- `toSDKRequest(llm.Request) <SDK request struct>` — extract `SystemPrompt` to provider-specific home (Anthropic top-level `System []TextBlockParam`; OpenAI `messages[0]` with `role: "system"`); flatten `Messages`; pass through `MaxOutputTokens` / `Temperature`
- `fromSDKResponse(<SDK response>) llm.Response` — extract `Text` from response message/content blocks; normalize `FinishReason` to `llm.FinishReason*` constants; populate `Usage` with `UsageReported`; populate `Provider` and `Model` from `Info()`

**Anti-Patterns to Avoid:**

- **Don't share a `mapping/` package between adapters.** Each provider's wire format diverges enough that "shared mapping" leaks accidental complexity. D-03 already extends this rule to `errors.go` — same here.
- **Don't merge `system` messages with user/assistant messages on Anthropic.** Anthropic's API rejects `role: "system"` in `messages`; system content must go to the top-level `System []TextBlockParam` field. The mapping function **MUST** lift `Request.SystemPrompt` into that top-level slot.
- **Don't ignore the SDK error in `Wrapped`.** Users need `errors.As(err, &apiErr)` to inspect provider-specific detail (insufficient_quota, model_not_found, etc.). Always preserve the chain.
- **Don't honor `Request.Tools` in Phase 1 even if the SDK supports it.** Recommendation per [§"Tool field passthrough"](#tool-field-passthrough): **ignore `Request.Tools` entirely**; capability `Tools: false` for all 3 adapters in Phase 1. Cleanest. Tools land in Phase 3 with full ToolCaller / per-tool-call indexing / dedupe.

### Pattern 4: `Stream` is a Phase-1 stub returning a "not implemented" error

**What:** The `ChatModel` interface requires `Stream(ctx, Request) (StreamReader, error)`. Phase 1 is sync-only, but the type system requires the method to exist. Each adapter's `Stream` returns `nil, errors.New("openai: streaming not implemented in Phase 1; use Generate")` (and parallel for Anthropic/Ollama).

**Why:** Avoids scope creep into Phase 2; makes the "Phase 1 = sync only" boundary visible at runtime; conformance suite has a `TestStream_Phase1NotImplemented` that asserts the error is returned. When Phase 2 lands, one PR per adapter replaces the stub.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP-level OpenAI Chat Completions wire format | Custom `net/http` POST + JSON unmarshal | `openai-go/v3` `client.Chat.Completions.New` | SDK tracks daily upstream; param/response struct tags + types are kept current; gets new fields (`stream_options`, `parallel_tool_calls`) for free |
| Anthropic Messages API wire format | Custom HTTP client | `anthropic-sdk-go` `client.Messages.New` | Same; plus the SDK already handles the `system` top-level lift, beta-header negotiation, and the `apierror.Error` shape |
| Ollama `/api/chat` wire format | Custom HTTP client | `github.com/ollama/ollama/api` `client.Chat(ctx, req, fn)` | The Ollama CLI itself uses this package — keeps wire format aligned with server upgrades. Pre-v1.0; pin minor (`v0.23.2`) and accept some churn (Pitfall 19 remembered for Phase 3) |
| Goroutine-leak detection | hand-rolled `runtime.NumGoroutine()` polling | `goleak.VerifyTestMain(m)` | Stdlib polling can't distinguish leaked-by-test from runtime/idle; goleak knows the runtime patterns and gives stack dumps |
| Container lifecycle for Ollama-live | hand-rolled `docker run` shell-out | `testcontainers-go/modules/ollama` `tcollama.Run(ctx, image)` | Owns container lifecycle (startup wait, port discovery, cleanup); `ConnectionString(ctx)` resolves host+port; `Exec(ctx, []string{"ollama", "pull", ...})` runs commands inside the container |
| Per-fixture replay-engine | bespoke recording library | `httptest.Server` + `testdata/*.json` (D-04 locked) | stdlib only; no version ceremony; fixtures are diff-friendly JSON; no API keys in PR CI |
| Fixture capture for replay | hand-craft fake JSON responses | `scripts/capture-fixtures-<provider>.sh` calling real API once | Fake JSON drifts from reality; real-API capture pins the actual wire format; commit JSON; refresh quarterly |

**Key insight:** Phase 1's value comes from **adapter conformance**, not from re-implementing wire formats. Every line of HTTP/JSON code an adapter writes is a line that drifts from upstream. The 3 SDKs + httptest + testcontainers + goleak together cover ≥ 95% of the work; the adapter code is glue (request/response mapping + error wrapping + option plumbing).

---

## Common Pitfalls

### Pitfall A: Honoring `Request.Tools` in Phase 1 — drift into Phase 3 scope

**What goes wrong:** OpenAI/Anthropic SDKs both accept a `Tools` field and will return tool-call content blocks. If the Phase 1 adapter passes `Request.Tools` through to the SDK and parses tool-call responses into `llm.ToolCall`, the test surface explodes (parallel-tool-call indexing, dedupe, capability-degrade fallback) — all Phase 3 concerns.

**Why it happens:** The SDKs make it easy. The temptation to "support tools too, while we're here" is strong.

**How to avoid:**
- **Ignore `Request.Tools` entirely in Phase 1.** Each adapter's `toSDKRequest` does NOT pass `Request.Tools` to the SDK.
- `Capabilities.Tools = false` for all 3 adapters' `Info()` return value — even though the SDK *can* do tools.
- PROVIDER_AUTHORING.md v0.1 documents this: "Phase 1 adapters MUST NOT honor Request.Tools; the field is reserved for Phase 3."
- Conformance test `TestGenerate_<Provider>_ToolsFieldIgnored`: pass `Request.Tools = []llm.Tool{...}`; assert that the request body sent to the httptest server does NOT contain a `tools` field (or contains an empty array).

**Warning signs:** PR diff for Phase 1 adapter touches tool_calls parsing; SDK request struct in `map.go` has any reference to `Tools`.

### Pitfall B: SDK error type lookup wrong (false-negative typed errors)

**What goes wrong:** Adapter calls `errors.As(err, &apiErr)` against the wrong SDK error type. `errors.As` returns false; `wrapErr` returns the raw error unwrapped; conformance test for `TestGenerate_OpenAI_401` fails because the returned error is a generic SDK error, not a `*llm.AuthError`.

**Why it happens:** SDK error types are nested in subpackages and easy to miss:
- OpenAI: `*openai.Error` (top-level `openai` package; verified via Context7 — has `StatusCode int`, `Type string`, `Code string`, `Message string`, `DumpRequest(true)`, `Headers() http.Header`)
- Anthropic: `*apierror.Error` from `github.com/anthropics/anthropic-sdk-go/internal/apierror` (verified via Context7 — `StatusCode int`, `RequestID string`, `RawJSON()`, `DumpRequest(true)`, `DumpResponse(true)`). **NOTE:** the package path includes `internal/`, which makes the type unimportable for type-assertion in our code. Recommended: use `apierror.Error` IF the SDK re-exports it from a public path (verify at plan-time); **fallback** is to inspect `err.Error()` for known prefixes OR HTTP-status-code-extraction via SDK helper. **See [§"Open Questions"](#open-questions-resolved) Q2 RESOLVED below.**
- Ollama: SDK returns plain `error` (it's just `fmt.Errorf` over the JSON-decoded API error). The HTTP status code is **not directly accessible** via the public `*api.Client.Chat` callback. Recommended: inject a custom `*http.Client` with a `RoundTripper` that records the last response's status code in a struct field; the adapter consults that field on error. **See [§"Open Questions"](#open-questions-resolved) Q3 RESOLVED.**

**How to avoid:** Pre-write per-adapter `wrapErr` with verified SDK error type + scenario-based unit tests for every status code in the recommended mapping table. Conformance suite asserts the typed-error contract end-to-end.

**Warning signs:** Conformance suite scenarios that should produce `*llm.AuthError` instead produce a generic error; `errors.As(err, &authErr)` returns false in tests.

### Pitfall C: Anthropic's `system` lifted to wrong place

**What goes wrong:** The adapter takes `Request.SystemPrompt` and prepends it as `Messages[0]` with `role: "system"`. Anthropic's API **rejects** `role: "system"` in `messages` — the system content must go to the **top-level** `System []TextBlockParam` field on `MessageNewParams`.

**Why it happens:** OpenAI accepts `role: "system"` in messages; the OpenAI mapping is naïvely copied to Anthropic.

**How to avoid:**
- Anthropic's `map.go` MUST lift `Request.SystemPrompt` to `MessageNewParams.System = []anthropic.TextBlockParam{{Text: req.SystemPrompt}}` (verified via Context7 `runner.Params.System = []anthropic.BetaTextBlockParam{...}` — the non-beta type is `anthropic.TextBlockParam`).
- If `Request.SystemPrompt == ""` AND `Request.Messages[0].Role == "system"`, fall back: lift the message to `System` and drop it from `Messages`.
- Conformance fixture `generate_happy_claude-3-5-haiku.json` includes a `body_assertions` entry checking `system` is at top level, NOT in `messages`.

**Warning signs:** Anthropic 400 `invalid_request_error` with body mentioning `messages.0.role: Input tag '"system"' invalid`.

### Pitfall D: Ollama testcontainers cold-pull is 3–5 min — flaky CI

**What goes wrong:** Nightly job times out at 10 min default GitHub Actions step timeout; every run starts cold-pull of `llama3.1:8b-instruct-q4_K_M` (~4.7GB at registry CDN); workflow goes RED daily.

**Why it happens:** Default `actions/cache` is not configured for testcontainers' Docker volume.

**How to avoid:**
- Cache `~/.cache/ollama` (or wherever testcontainers mounts the model volume — verify at plan-time) using `actions/cache@v4` keyed on the pinned model + image versions.
- Set explicit step timeout: `timeout-minutes: 30` on the test step.
- Pin `ollama/ollama:0.5.7` (or current stable; verify at plan-time) — don't use `:latest`.
- For first cold run, accept slow start; on warm runs, verify cache hit in the workflow logs.

**Warning signs:** Nightly workflow consistently >15 min; cache miss on every run.

### Pitfall E: goleak fires false-positives from `httptest.Server` connection reuse

**What goes wrong:** `httptest.Server` keep-alives leave connection-reading goroutines around after `server.Close()`. `goleak.VerifyTestMain` reports them as leaks; PR CI goes red despite no real bug.

**Why it happens:** Go's net/http transport reuses connections; idle reader goroutines linger briefly.

**How to avoid:**
- Use `goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop"))` if false-positives occur.
- Alternatively, set `http.Client.Transport.DisableKeepAlives = true` on the test client (acceptable in tests; production users get keep-alives by default).
- Verify at plan-time which approach the goleak community currently recommends — see [goleak README](https://github.com/uber-go/goleak/blob/master/README.md).

**Warning signs:** goleak fails with stack mentioning `persistConn.readLoop` after passing tests.

### Pitfall F: Capture script committed with real API key

**What goes wrong:** Contributor runs `bash scripts/capture-fixtures-openai.sh` with `OPENAI_API_KEY=sk-...` in env; bash leaks the key into a saved log or `.env` file accidentally `git add`-ed.

**Why it happens:** Bash scripts handling secrets are fragile.

**How to avoid:**
- Capture script `set -u` (fails on unset var), reads `$OPENAI_API_KEY` from env (no fallback to a file), uses `curl` with `-H "Authorization: Bearer $OPENAI_API_KEY"` and pipes response to `jq` then writes to `testdata/openai/<scenario>.json`.
- The output JSON contains ONLY the response body and request shape — never API keys, never tokens.
- `.gitignore` blocks `testdata/**/*.local.json` and `**/.env`.
- README of capture script: "Run locally only; never commit your API key. The captured JSON is safe to commit."

---

## Code Examples

Verified patterns from official sources (Context7 IDs `/openai/openai-go`, `/anthropics/anthropic-sdk-go`, `/ollama/ollama`).

### OpenAI Generate (sync, Chat Completions per D-01)

```go
// Source: Context7 /openai/openai-go (verified 2026-05-10) — https://github.com/openai/openai-go
// Per-D-01: Chat Completions only; no Responses API in Phase 1.
package openai

import (
    "context"

    "github.com/costa92/llm-agent/llm"
    openai "github.com/openai/openai-go/v3"
)

func (o *OpenAI) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
    sdkReq := o.toSDKRequest(req)
    completion, err := o.client.Chat.Completions.New(ctx, sdkReq)
    if err != nil {
        return llm.Response{}, wrapErr(err)
    }
    return o.fromSDKResponse(completion), nil
}

func (o *OpenAI) toSDKRequest(req llm.Request) openai.ChatCompletionNewParams {
    msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages)+1)
    if req.SystemPrompt != "" {
        msgs = append(msgs, openai.SystemMessage(req.SystemPrompt))
    }
    for _, m := range req.Messages {
        switch m.Role {
        case "user":
            msgs = append(msgs, openai.UserMessage(m.Content))
        case "assistant":
            msgs = append(msgs, openai.AssistantMessage(m.Content))
        case "system":
            // Allow system in Messages as backstop; lifted to SystemMessage above takes priority.
            msgs = append(msgs, openai.SystemMessage(m.Content))
        }
    }
    p := openai.ChatCompletionNewParams{
        Model:    o.info.Model, // bound at construction
        Messages: msgs,
    }
    if req.MaxOutputTokens > 0 {
        p.MaxCompletionTokens = openai.Int(int64(req.MaxOutputTokens))
    }
    if req.Temperature != nil {
        p.Temperature = openai.Float(float64(*req.Temperature))
    }
    // NOTE: req.Tools intentionally NOT mapped — Phase 1 ignores tools (Pitfall A).
    return p
}

func (o *OpenAI) fromSDKResponse(c *openai.ChatCompletion) llm.Response {
    var text string
    if len(c.Choices) > 0 {
        text = c.Choices[0].Message.Content
    }
    finish := llm.FinishReasonUnknown
    if len(c.Choices) > 0 {
        finish = mapFinishReason(string(c.Choices[0].FinishReason))
    }
    return llm.Response{
        Text:         text,
        FinishReason: finish,
        Provider:     "openai",
        Model:        c.Model, // mirror SDK's response.model (may differ from request)
        Usage: llm.Usage{
            InputTokens:  int(c.Usage.PromptTokens),
            OutputTokens: int(c.Usage.CompletionTokens),
            TotalTokens:  int(c.Usage.TotalTokens),
            Source:       llm.UsageReported,
        },
    }
}

func mapFinishReason(s string) llm.FinishReason {
    // OpenAI Chat Completions: "stop", "length", "content_filter", "tool_calls", "function_call"
    switch s {
    case "stop":
        return llm.FinishReasonStop
    case "length":
        return llm.FinishReasonLength
    case "content_filter":
        return llm.FinishReasonContentFilter
    case "tool_calls":
        return llm.FinishReasonToolCalls
    case "function_call":
        return llm.FinishReasonFunctionCall
    default:
        return llm.FinishReasonUnknown
    }
}
```

### Anthropic Generate (sync, non-beta `client.Messages.New` per D-01-style stability rationale)

```go
// Source: Context7 /anthropics/anthropic-sdk-go (verified 2026-05-10) — non-beta MessageNewParams
// Pitfall C: Request.SystemPrompt MUST lift to top-level System []TextBlockParam, NOT messages[0].
package anthropic

import (
    "context"

    "github.com/costa92/llm-agent/llm"
    anthropic "github.com/anthropics/anthropic-sdk-go"
)

func (a *Anthropic) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
    sdkReq := a.toSDKRequest(req)
    msg, err := a.client.Messages.New(ctx, sdkReq)
    if err != nil {
        return llm.Response{}, wrapErr(err)
    }
    return a.fromSDKResponse(msg), nil
}

func (a *Anthropic) toSDKRequest(req llm.Request) anthropic.MessageNewParams {
    msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
    sysPrompt := req.SystemPrompt
    for _, m := range req.Messages {
        switch m.Role {
        case "user":
            msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
        case "assistant":
            msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
        case "system":
            // Pitfall C: system goes to the top-level System field, NOT messages.
            // If both Request.SystemPrompt and a system-role message exist, concatenate.
            if sysPrompt == "" {
                sysPrompt = m.Content
            } else {
                sysPrompt = sysPrompt + "\n\n" + m.Content
            }
        }
    }
    p := anthropic.MessageNewParams{
        Model:     anthropic.Model(a.info.Model), // string-aliased typed model name
        MaxTokens: 1024, // Anthropic requires MaxTokens; default if Request.MaxOutputTokens unset
        Messages:  msgs,
    }
    if sysPrompt != "" {
        p.System = []anthropic.TextBlockParam{{Text: sysPrompt}}
    }
    if req.MaxOutputTokens > 0 {
        p.MaxTokens = int64(req.MaxOutputTokens)
    }
    if req.Temperature != nil {
        // Anthropic uses anthropic.Float for optional float params — verify at plan-time
        // (Context7 patterns suggest direct float64 with omitempty).
    }
    return p
}

func (a *Anthropic) fromSDKResponse(m *anthropic.Message) llm.Response {
    var text string
    for _, block := range m.Content {
        // Phase 1: extract only text blocks; tool_use blocks ignored (Pitfall A).
        if block.Type == "text" {
            text += block.Text
        }
    }
    return llm.Response{
        Text:         text,
        FinishReason: mapAnthropicStopReason(string(m.StopReason)),
        Provider:     "anthropic",
        Model:        string(m.Model),
        Usage: llm.Usage{
            InputTokens:  int(m.Usage.InputTokens),
            OutputTokens: int(m.Usage.OutputTokens),
            TotalTokens:  int(m.Usage.InputTokens + m.Usage.OutputTokens),
            Source:       llm.UsageReported,
        },
    }
}

func mapAnthropicStopReason(s string) llm.FinishReason {
    // Anthropic: "end_turn", "max_tokens", "stop_sequence", "tool_use"
    switch s {
    case "end_turn", "stop_sequence":
        return llm.FinishReasonStop
    case "max_tokens":
        return llm.FinishReasonLength
    case "tool_use":
        return llm.FinishReasonToolCalls
    default:
        return llm.FinishReasonUnknown
    }
}
```

### Ollama Generate (sync `/api/chat` with `Stream: new(bool)` per Context7-verified pattern)

```go
// Source: Context7 /ollama/ollama (verified 2026-05-10) +
//         WebSearch (verified 2026-05-10): api.NewClient(base *url.URL, http *http.Client) *Client
//         per https://github.com/ollama/ollama/blob/main/api/client.go (issue #2948)
package ollama

import (
    "context"
    "net/http"
    "net/url"

    "github.com/costa92/llm-agent/llm"
    api "github.com/ollama/ollama/api"
)

func (o *Ollama) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
    sdkReq := o.toSDKRequest(req)
    var captured api.ChatResponse
    // Non-streaming: invoke the callback exactly once with the final non-stream response.
    err := o.client.Chat(ctx, sdkReq, func(resp api.ChatResponse) error {
        captured = resp
        return nil
    })
    if err != nil {
        return llm.Response{}, o.wrapErr(err)
    }
    return o.fromSDKResponse(captured), nil
}

func (o *Ollama) toSDKRequest(req llm.Request) *api.ChatRequest {
    msgs := make([]api.Message, 0, len(req.Messages)+1)
    if req.SystemPrompt != "" {
        msgs = append(msgs, api.Message{Role: "system", Content: req.SystemPrompt})
    }
    for _, m := range req.Messages {
        msgs = append(msgs, api.Message{Role: m.Role, Content: m.Content})
    }
    streamOff := false
    return &api.ChatRequest{
        Model:    o.info.Model,
        Messages: msgs,
        Stream:   &streamOff, // Pointer-to-false disables streaming; Context7-verified pattern
        // Phase 1 ignores req.Tools — Pitfall A.
    }
}

func (o *Ollama) fromSDKResponse(r api.ChatResponse) llm.Response {
    return llm.Response{
        Text:         r.Message.Content,
        FinishReason: mapOllamaDoneReason(r.DoneReason),
        Provider:     "ollama",
        Model:        r.Model,
        Usage: llm.Usage{
            InputTokens:  r.PromptEvalCount,
            OutputTokens: r.EvalCount,
            TotalTokens:  r.PromptEvalCount + r.EvalCount,
            Source:       llm.UsageReported,
        },
    }
}

func mapOllamaDoneReason(s string) llm.FinishReason {
    // Ollama done_reason: "stop", "length", "load"
    switch s {
    case "stop":
        return llm.FinishReasonStop
    case "length":
        return llm.FinishReasonLength
    default:
        return llm.FinishReasonUnknown
    }
}

// In options.go: build the api.Client with custom HTTP client + base URL for httptest.
func newOllamaClient(baseURL string, httpClient *http.Client) (*api.Client, error) {
    if baseURL == "" {
        baseURL = "http://localhost:11434"
    }
    u, err := url.Parse(baseURL)
    if err != nil {
        return nil, err
    }
    if httpClient == nil {
        httpClient = http.DefaultClient
    }
    return api.NewClient(u, httpClient), nil
}
```

---

## Conformance harness shape — `internal/contract/`

### Layout (per D-04 locked)

```
llm-agent-providers/internal/contract/
├── contract.go         # Shared helpers: LoadFixture, NewMockServer, AssertGenerate, ChatModelFactory
├── generate_test.go    # Table-driven test iterating (factory × fixture)
├── main_test.go        # TestMain → goleak.VerifyTestMain
└── testdata/
    ├── openai/
    │   ├── generate_happy_gpt-4o-mini.json
    │   ├── generate_401_invalid_api_key.json
    │   ├── generate_429_rate_limit.json
    │   ├── generate_429_quota_exhausted.json
    │   └── generate_500_server_error.json
    ├── anthropic/
    │   ├── generate_happy_claude-3-5-haiku.json
    │   ├── generate_400_invalid_request.json
    │   ├── generate_401_invalid_api_key.json
    │   ├── generate_429_rate_limit.json
    │   └── generate_529_overloaded.json
    └── ollama/
        ├── generate_happy_llama3.1-8b.json
        ├── generate_404_model_not_pulled.json
        └── generate_500_oom.json
```

### Fixture JSON schema (per D-04)

```json
{
  "scenario": "generate_happy_gpt-4o-mini",
  "request": {
    "method": "POST",
    "path": "/v1/chat/completions",
    "body_assertions": [
      "model=gpt-4o-mini",
      "messages contains 'hello'"
    ]
  },
  "response": {
    "status": 200,
    "headers": {"Content-Type": "application/json"},
    "body": "{\"id\":\"chatcmpl-AbC123\",\"object\":\"chat.completion\",\"created\":1715300000,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"Hello! How can I help?\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":8,\"total_tokens\":18}}"
  },
  "expect": {
    "error_type": null,
    "response_text": "Hello! How can I help?",
    "finish_reason": "stop",
    "usage_input_tokens": 10,
    "usage_output_tokens": 8,
    "usage_source": "reported",
    "provider": "openai"
  }
}
```

For **error scenarios** (401/429/500/etc.), the fixture's `response.body` is the provider's actual error JSON, and `expect.error_type` is one of `"AuthError"`, `"RateLimitError"`, `"InvalidRequestError"`, `"TransientError"`. The conformance test asserts `errors.As(err, &<expected-type>)` succeeds AND `err.Error()` contains the expected substring.

**Per-provider response body shape (verified):**

- **OpenAI Chat Completions happy path:** `{id, object, created, model, choices[0].{index, message:{role, content}, finish_reason}, usage:{prompt_tokens, completion_tokens, total_tokens}}` — ~500 bytes.
- **Anthropic Messages happy path:** `{id, type:"message", role:"assistant", content:[{type:"text", text:"..."}], model, stop_reason:"end_turn", stop_sequence:null, usage:{input_tokens, output_tokens}}` — ~400 bytes.
- **Ollama /api/chat happy path:** `{model, created_at, message:{role:"assistant", content:"..."}, done:true, done_reason:"stop", total_duration, load_duration, prompt_eval_count, prompt_eval_duration, eval_count, eval_duration}` — ~600 bytes.

### Shared helpers (`contract.go` sketch)

```go
// llm-agent-providers/internal/contract/contract.go
package contract

import (
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/costa92/llm-agent/llm"
)

// Fixture is the parsed shape of testdata/<provider>/<scenario>.json.
type Fixture struct {
    Scenario string `json:"scenario"`
    Request  struct {
        Method          string   `json:"method"`
        Path            string   `json:"path"`
        BodyAssertions  []string `json:"body_assertions"`
    } `json:"request"`
    Response struct {
        Status  int               `json:"status"`
        Headers map[string]string `json:"headers"`
        Body    string            `json:"body"`
    } `json:"response"`
    Expect struct {
        ErrorType         string `json:"error_type,omitempty"`         // "" | "AuthError" | "RateLimitError" | ...
        ResponseText      string `json:"response_text,omitempty"`
        FinishReason      string `json:"finish_reason,omitempty"`
        UsageInputTokens  int    `json:"usage_input_tokens,omitempty"`
        UsageOutputTokens int    `json:"usage_output_tokens,omitempty"`
        UsageSource       string `json:"usage_source,omitempty"`
        Provider          string `json:"provider,omitempty"`
    } `json:"expect"`
}

// LoadFixture reads testdata/<provider>/<scenario>.json relative to the test file.
func LoadFixture(t *testing.T, provider, scenario string) Fixture {
    t.Helper()
    p := filepath.Join("testdata", provider, scenario+".json")
    data, err := os.ReadFile(p)
    if err != nil {
        t.Fatalf("LoadFixture: %v", err)
    }
    var f Fixture
    if err := json.Unmarshal(data, &f); err != nil {
        t.Fatalf("LoadFixture: %v", err)
    }
    return f
}

// NewMockServer returns an httptest.Server that:
//   - asserts request method/path matches Fixture.Request
//   - asserts request body contains all Fixture.Request.BodyAssertions substrings
//   - replies with Fixture.Response.Status / Headers / Body
func NewMockServer(t *testing.T, f Fixture) *httptest.Server {
    t.Helper()
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if f.Request.Method != "" && r.Method != f.Request.Method {
            t.Errorf("method: got %q want %q", r.Method, f.Request.Method)
        }
        if f.Request.Path != "" && !strings.HasPrefix(r.URL.Path, f.Request.Path) {
            t.Errorf("path: got %q want prefix %q", r.URL.Path, f.Request.Path)
        }
        body, _ := io.ReadAll(r.Body)
        for _, want := range f.Request.BodyAssertions {
            if !assertBody(string(body), want) {
                t.Errorf("body assertion failed: %q not satisfied by %q", want, string(body))
            }
        }
        for k, v := range f.Response.Headers {
            w.Header().Set(k, v)
        }
        w.WriteHeader(f.Response.Status)
        _, _ = w.Write([]byte(f.Response.Body))
    }))
}

// assertBody is a tiny DSL for body_assertions:
//   "key=value" → JSON contains "key":"value"
//   "key contains 'X'" → JSON contains the substring 'X' under key
// Phase 1 implementation can be a simple substring check; richer assertions are P2.
func assertBody(body, assertion string) bool {
    return strings.Contains(body, strings.TrimSpace(strings.TrimPrefix(assertion, "model=")))
}

// ChatModelFactory builds an llm.ChatModel pointed at the given baseURL.
// Each adapter package exports a contract.NewFactory("openai", openai.New) helper that the
// conformance test registers; the test then iterates (factory × fixture) pairs.
type ChatModelFactory func(baseURL string) (llm.ChatModel, error)

// AssertGenerate runs Generate against the adapter, then asserts against Fixture.Expect.
func AssertGenerate(t *testing.T, model llm.ChatModel, f Fixture) {
    t.Helper()
    req := llm.Request{
        Messages: []llm.Message{{Role: "user", Content: "hello"}},
    }
    resp, err := model.Generate(t.Context(), req)
    switch f.Expect.ErrorType {
    case "":
        if err != nil {
            t.Fatalf("Generate: unexpected error: %v", err)
        }
        if f.Expect.ResponseText != "" && resp.Text != f.Expect.ResponseText {
            t.Errorf("Text: got %q want %q", resp.Text, f.Expect.ResponseText)
        }
        if f.Expect.FinishReason != "" && string(resp.FinishReason) != f.Expect.FinishReason {
            t.Errorf("FinishReason: got %q want %q", resp.FinishReason, f.Expect.FinishReason)
        }
        if f.Expect.UsageInputTokens != 0 && resp.Usage.InputTokens != f.Expect.UsageInputTokens {
            t.Errorf("Usage.InputTokens: got %d want %d", resp.Usage.InputTokens, f.Expect.UsageInputTokens)
        }
        // ... and so on for OutputTokens, Source, Provider, Model
    case "AuthError":
        var e *llm.AuthError
        if !errorsAs(err, &e) {
            t.Errorf("expected *llm.AuthError, got %T: %v", err, err)
        }
    case "RateLimitError":
        var e *llm.RateLimitError
        if !errorsAs(err, &e) {
            t.Errorf("expected *llm.RateLimitError, got %T: %v", err, err)
        }
    case "InvalidRequestError":
        var e *llm.InvalidRequestError
        if !errorsAs(err, &e) {
            t.Errorf("expected *llm.InvalidRequestError, got %T: %v", err, err)
        }
    case "TransientError":
        var e *llm.TransientError
        if !errorsAs(err, &e) {
            t.Errorf("expected *llm.TransientError, got %T: %v", err, err)
        }
    }
}

// errorsAs is the package-private alias for errors.As (so tests compile without re-importing).
func errorsAs(err error, target any) bool {
    return errors.As(err, target)  // imports "errors" at file top
}
```

### Test driver (`generate_test.go` sketch)

```go
// llm-agent-providers/internal/contract/generate_test.go
package contract

import (
    "testing"

    "github.com/costa92/llm-agent/llm"
    "github.com/costa92/llm-agent-providers/anthropic"
    "github.com/costa92/llm-agent-providers/ollama"
    "github.com/costa92/llm-agent-providers/openai"
)

// AdapterFactories registers the three adapter packages. Adding a NEW provider in the
// future = drop a new line here; the conformance suite picks it up automatically.
var AdapterFactories = map[string]ChatModelFactory{
    "openai": func(baseURL string) (llm.ChatModel, error) {
        return openai.New(openai.WithModel("gpt-4o-mini"), openai.WithAPIKey("test"), openai.WithBaseURL(baseURL))
    },
    "anthropic": func(baseURL string) (llm.ChatModel, error) {
        return anthropic.New(anthropic.WithModel("claude-3-5-haiku-20241022"), anthropic.WithAPIKey("test"), anthropic.WithBaseURL(baseURL))
    },
    "ollama": func(baseURL string) (llm.ChatModel, error) {
        return ollama.New(ollama.WithModel("llama3.1:8b"), ollama.WithBaseURL(baseURL))
    },
}

// TestGenerate_Conformance is the single table-driven test that exercises every
// (provider, scenario) pair. New scenarios = drop a JSON file in testdata/<provider>/.
func TestGenerate_Conformance(t *testing.T) {
    cases := []struct {
        provider string
        scenario string
    }{
        {"openai", "generate_happy_gpt-4o-mini"},
        {"openai", "generate_401_invalid_api_key"},
        {"openai", "generate_429_rate_limit"},
        {"openai", "generate_429_quota_exhausted"},
        {"openai", "generate_500_server_error"},
        {"anthropic", "generate_happy_claude-3-5-haiku"},
        {"anthropic", "generate_400_invalid_request"},
        {"anthropic", "generate_401_invalid_api_key"},
        {"anthropic", "generate_429_rate_limit"},
        {"anthropic", "generate_529_overloaded"},
        {"ollama", "generate_happy_llama3.1-8b"},
        {"ollama", "generate_404_model_not_pulled"},
        {"ollama", "generate_500_oom"},
    }
    for _, c := range cases {
        t.Run(c.provider+"/"+c.scenario, func(t *testing.T) {
            t.Parallel()
            f := LoadFixture(t, c.provider, c.scenario)
            srv := NewMockServer(t, f)
            defer srv.Close()
            factory := AdapterFactories[c.provider]
            model, err := factory(srv.URL)
            if err != nil {
                t.Fatalf("factory: %v", err)
            }
            AssertGenerate(t, model, f)
        })
    }
}
```

### goleak integration (`main_test.go`)

```go
// llm-agent-providers/internal/contract/main_test.go
//
// Phase 1 has no goroutines (sync Generate), but the harness lands here so Phase 2's
// streaming work inherits it. Pitfall 3 prevention.
package contract

import (
    "testing"

    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m,
        // Ignore httptest.Server's persistConn readLoop false-positives (Pitfall E).
        // Verify at plan-time whether this ignore is still needed under Go 1.26.
        goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop"),
    )
}
```

### Capture scripts (`scripts/capture-fixtures-<provider>.sh` sketch)

Each script:
1. Reads `$<PROVIDER>_API_KEY` from env (fails if unset).
2. Calls the real API once via `curl` with the canonical happy-path request.
3. Writes the response body verbatim into `internal/contract/testdata/<provider>/generate_happy_<model>.json` using a `jq`-shaped wrapper that matches the Fixture schema above.
4. Optionally accepts a `--scenario <name>` flag to capture an error-path fixture (e.g., `--scenario 401_invalid_api_key` sends a known-bad key).

```bash
#!/usr/bin/env bash
# scripts/capture-fixtures-openai.sh
set -euo pipefail
: "${OPENAI_API_KEY:?must be set; never commit this key}"

SCENARIO="${1:-generate_happy_gpt-4o-mini}"
MODEL="gpt-4o-mini"
OUT="internal/contract/testdata/openai/${SCENARIO}.json"

# Capture the happy-path response. For error scenarios, swap the curl args.
RESPONSE_BODY=$(curl -sS -X POST https://api.openai.com/v1/chat/completions \
    -H "Authorization: Bearer ${OPENAI_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"${MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}")

# Wrap into our Fixture schema:
jq -n \
   --arg scenario "${SCENARIO}" \
   --arg model "${MODEL}" \
   --argjson body "${RESPONSE_BODY}" \
   '{
      scenario: $scenario,
      request: {
        method: "POST",
        path: "/v1/chat/completions",
        body_assertions: ["model=" + $model, "messages contains 'hello'"]
      },
      response: {
        status: 200,
        headers: {"Content-Type": "application/json"},
        body: ($body | tostring)
      },
      expect: {
        error_type: null,
        response_text: $body.choices[0].message.content,
        finish_reason: $body.choices[0].finish_reason,
        usage_input_tokens: $body.usage.prompt_tokens,
        usage_output_tokens: $body.usage.completion_tokens,
        usage_source: "reported",
        provider: "openai"
      }
    }' > "${OUT}"

echo "Captured: ${OUT}"
echo "Inspect with: jq . ${OUT}"
echo "Commit when ready."
```

Anthropic and Ollama scripts follow the same shape; key differences:
- Anthropic: `Authorization: x-api-key: $ANTHROPIC_API_KEY` + `anthropic-version: 2023-06-01` header; path `/v1/messages`.
- Ollama: no API key; runs against `http://localhost:11434` (capture from a local Ollama daemon, NOT a hosted API).

---

## CI YAML sketches

### Existing (Phase 0): `llm-agent-providers/.github/workflows/test.yml`

Phase 0 already shipped this — runs `go vet/build/test` on PR with `GOWORK=off`. Phase 1 does NOT modify it (the conformance suite under `internal/contract/` runs as part of `go test ./...`, mock-only via httptest).

### NEW (Phase 1): `llm-agent-providers/.github/workflows/nightly-ollama-live.yml`

```yaml
# llm-agent-providers/.github/workflows/nightly-ollama-live.yml
# OLL-08: nightly testcontainers-go conformance against a real Ollama container.
name: nightly-ollama-live

on:
  schedule:
    - cron: '0 3 * * *'   # 03:00 UTC daily
  workflow_dispatch:       # manual trigger for ad-hoc runs

env:
  GOWORK: off
  OLLAMA_TC_IMAGE: ollama/ollama:0.5.7
  OLLAMA_TC_MODEL: llama3.1:8b-instruct-q4_K_M

jobs:
  ollama-live-conformance:
    runs-on: ubuntu-latest
    timeout-minutes: 45    # cold pull is ~3-5 min; warm runs <2 min; budget 45m for first runs
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      # Cache the Ollama model volume across runs. Key on image + model so a
      # version bump invalidates cleanly. (Pitfall D mitigation.)
      - name: Cache Ollama model volume
        uses: actions/cache@v4
        with:
          path: ~/.cache/ollama
          key: ollama-${{ env.OLLAMA_TC_IMAGE }}-${{ env.OLLAMA_TC_MODEL }}
          restore-keys: |
            ollama-${{ env.OLLAMA_TC_IMAGE }}-

      - name: Verify Docker available
        run: docker info | head -3

      - name: Run conformance suite against testcontainer Ollama
        run: |
          go test -v -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live -tags ollama_live
        env:
          OLLAMA_TC_IMAGE: ${{ env.OLLAMA_TC_IMAGE }}
          OLLAMA_TC_MODEL: ${{ env.OLLAMA_TC_MODEL }}
```

The matching test (build-tagged so PR CI never runs it):

```go
//go:build ollama_live

package contract

import (
    "context"
    "os"
    "testing"

    tcollama "github.com/testcontainers/testcontainers-go/modules/ollama"
    "github.com/costa92/llm-agent-providers/ollama"
)

// TestGenerate_Ollama_Live spins up a real Ollama container, pre-pulls the pinned
// model, and runs the conformance suite's Ollama factory against it.
func TestGenerate_Ollama_Live(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping ollama-live test in -short mode")
    }
    ctx := context.Background()
    image := getenv("OLLAMA_TC_IMAGE", "ollama/ollama:0.5.7")
    model := getenv("OLLAMA_TC_MODEL", "llama3.1:8b-instruct-q4_K_M")

    container, err := tcollama.Run(ctx, image)
    if err != nil {
        t.Fatalf("tcollama.Run: %v", err)
    }
    t.Cleanup(func() { _ = container.Terminate(ctx) })

    // Pre-pull the model. (testcontainers-go does NOT have a startup-time WithModel
    // option — Exec is the official documented path.)
    if _, _, err := container.Exec(ctx, []string{"ollama", "pull", model}); err != nil {
        t.Fatalf("ollama pull: %v", err)
    }

    baseURL, err := container.ConnectionString(ctx)
    if err != nil {
        t.Fatalf("ConnectionString: %v", err)
    }

    adapter, err := ollama.New(ollama.WithModel(model), ollama.WithBaseURL(baseURL))
    if err != nil {
        t.Fatalf("ollama.New: %v", err)
    }

    // Run the same fixture-driven assertions, but happy-path only against a real model.
    f := LoadFixture(t, "ollama", "generate_happy_llama3.1-8b")
    AssertGenerate(t, adapter, f)
}

func getenv(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}
```

---

## PROVIDER_AUTHORING.md v0.1 outline (in `llm-agent` core repo)

**Location:** `llm-agent/PROVIDER_AUTHORING.md`
**Length goal:** ~150–250 lines. Phase 1 v0.1 is the **Generate-only** contract; v0.2 (Phase 2) adds Stream; v0.3 (Phase 4) adds Tools/Embeddings/StructuredOutputs.

### Sections

1. **Audience and Scope.**
   "If you're writing a Go provider adapter that satisfies `llm.ChatModel.Generate`, this guide is for you. v0.1 covers Generate (sync) only; streaming, tool calling, and embeddings are documented in subsequent versions of this guide."

2. **The Contract** (1 page).
   Cross-link to `llm/chatmodel.go`, `llm/types.go`, `llm/info.go`, `llm/errors.go`. State the 3-method `ChatModel` interface; show the `Request` / `Response` / `ProviderInfo` / `Capabilities` shapes the adapter must produce/consume. Link to `llm/scripted.go` and `llm/chat_only_mock.go` as reference shapes.

3. **The Generate Contract** (1 page).
   Required behavior:
   - Honor `Request.Messages` and `Request.SystemPrompt` (lift system content to provider's home — top-level for Anthropic, role-tagged message for OpenAI/Ollama).
   - Pass through `Request.MaxOutputTokens` / `Request.Temperature` if non-zero / non-nil.
   - Return `Response.Provider == <provider name>` and `Response.Model == <bound model name>`.
   - Populate `Response.Usage` with `Source: llm.UsageReported` if the wire format provides counts; `UsageUnknown` otherwise.
   - Normalize `FinishReason` to the constants in `llm/legacy.go` (`FinishReasonStop`/`Length`/`ContentFilter`/`ToolCalls`/`FunctionCall`/`Unknown`).
   - **Phase 1 limitation:** ignore `Request.Tools`. `Capabilities.Tools = false`. Document plainly: "Tools are honored beginning in v0.2 of this guide (Phase 3 of v0.3 milestone)."

4. **Constructor Pattern (D-02 canonical)** (1 page).
   Show the exact functional-options shape. Required options for every provider: `WithModel` (REQUIRED), `WithAPIKey`, `WithBaseURL`, `WithHTTPClient`, `WithTimeout`. Per-provider extras allowed but should be prefixed for clarity (`WithOrganization`, `WithBetaHeader`, `WithHost`).

   - Show the canonical `New(opts ...Option) (*X, error)` snippet.
   - Document that `New` MUST return error if `WithModel` is empty.
   - Document env-var fallbacks (`OPENAI_API_KEY` / `ANTHROPIC_API_KEY`; Ollama: `OLLAMA_HOST`).

5. **Error Taxonomy (D-03 canonical mapping table)** (1 page).
   Reproduce the table verbatim from D-03. State the rule: "Every adapter wraps SDK errors into one of `llm.AuthError` / `llm.RateLimitError` / `llm.InvalidRequestError` / `llm.TransientError`, preserving the SDK error in the `Wrapped` field for `errors.Unwrap` chain traversal." Provide the OpenAI `wrapErr` snippet from §"Pattern 2" above as the canonical example.

   Document provider-specific override pattern: "If your provider has a quirk (Anthropic 529 = overloaded; OpenAI insufficient_quota = 429 with quota reason; Ollama 404 = model-not-pulled), implement it in your adapter's `errors.go`. **Do NOT push provider-specific logic into core.**"

6. **Conformance Test Pattern (D-04 canonical)** (1 page).
   "Your adapter passes if it makes `internal/contract/generate_test.go` green when iterated against your factory."

   - How to register a factory: drop a line in `AdapterFactories` map.
   - How to add fixtures: `testdata/<your-provider>/generate_<scenario>.json`, schema documented in §"Fixture JSON schema".
   - How to capture real fixtures: `scripts/capture-fixtures-<provider>.sh` (local-only).
   - goleak: any new adapter must pass `goleak.VerifyTestMain` from day one.

7. **Phase 1 Boundary** (½ page).
   List explicitly what Phase 1 adapters do NOT do, with forward references to Phase 2/3/4 / future versions of this guide:
   - No streaming (Phase 2).
   - No native tool calling (Phase 3).
   - No embeddings (Phase 4).
   - No three-state cost record (Phase 2 — `Source` is always `Reported` or unset in Phase 1).
   - No retry state machine (Phase 2).
   - No OTel instrumentation (Phase 5; users compose with `WithHTTPClient` themselves).

8. **Cross-references** (½ page).
   - `llm/scripted.go` — full-capability mock; reference for "what does a fully-featured adapter look like."
   - `llm/chat_only_mock.go` — minimal `ChatModel`-only adapter; reference for "what's the smallest viable adapter."
   - `.planning/research/STACK.md` — versions to use, alternatives considered.
   - `.planning/research/PITFALLS.md` Pitfalls 1, 2, 3, 4, 5, 6, 19, 20, 21, 22 — what NOT to do.

**Lint check:** Phase 1 v0.1 is markdown only. No doctest. Future versions of this guide may extract Go snippets and `go vet`-check them; that's a v0.2+ concern.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26) + `go.uber.org/goleak` v1.3.0+ for goroutine assertions |
| Config file | None (Go's `go test` is config-free); `goleak` invoked from `internal/contract/main_test.go` |
| Quick run command | `cd llm-agent-providers && go test -short ./...` (≤ 30s; runs unit tests + conformance mock-only) |
| Full suite command | `cd llm-agent-providers && go test ./...` (~ 1 min; same as quick — Phase 1 has no slow tests in PR CI) |
| Nightly Ollama-live | `cd llm-agent-providers && go test -tags ollama_live -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live` (run only by nightly workflow) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OAI-01 | OpenAI Generate happy path returns Response.FinishReason=stop | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_Happy` | ❌ Wave 0 (build) |
| OAI-01 | OpenAI Generate with system message lifts to messages[0] role=system | unit (httptest body assertion) | `go test ./openai/... -run TestGenerate_OpenAI_SystemPrompt` | ❌ Wave 0 |
| OAI-05 | OpenAI 401 returns *llm.AuthError | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_401` | ❌ Wave 0 |
| OAI-05 | OpenAI 429 returns *llm.RateLimitError; insufficient_quota sets Reason | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_429` | ❌ Wave 0 |
| OAI-05 | OpenAI 500/502/503/504 returns *llm.TransientError | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_5xx` | ❌ Wave 0 |
| OAI-05 | OpenAI 400/404/422 returns *llm.InvalidRequestError | unit (httptest) | `go test ./openai/... -run TestGenerate_OpenAI_4xxOther` | ❌ Wave 0 |
| ANT-01 | Anthropic Generate happy path with claude-3-5-haiku | unit (httptest) | `go test ./anthropic/... -run TestGenerate_Anthropic_Happy` | ❌ Wave 0 |
| ANT-01 | Anthropic SystemPrompt lifts to top-level System []TextBlockParam (Pitfall C) | unit (httptest body assertion) | `go test ./anthropic/... -run TestGenerate_Anthropic_SystemTopLevel` | ❌ Wave 0 |
| ANT-05 | Anthropic 529 overloaded → *llm.RateLimitError | unit (httptest) | `go test ./anthropic/... -run TestGenerate_Anthropic_529` | ❌ Wave 0 |
| ANT-05 | Anthropic 400 invalid_request_error → *llm.InvalidRequestError | unit (httptest) | `go test ./anthropic/... -run TestGenerate_Anthropic_400` | ❌ Wave 0 |
| OLL-01 | Ollama Generate happy path against llama3.1:8b | unit (httptest) | `go test ./ollama/... -run TestGenerate_Ollama_Happy` | ❌ Wave 0 |
| OLL-01 | Ollama with bound model returns Response.Model matching | unit (httptest) | `go test ./ollama/... -run TestGenerate_Ollama_ModelEcho` | ❌ Wave 0 |
| OLL-05 | Ollama 404 model-not-pulled → *llm.InvalidRequestError | unit (httptest) | `go test ./ollama/... -run TestGenerate_Ollama_404ModelNotPulled` | ❌ Wave 0 |
| OLL-05 | Ollama no daemon reachable → *llm.TransientError | unit (httptest with closed listener) | `go test ./ollama/... -run TestGenerate_Ollama_NoDaemon` | ❌ Wave 0 |
| OLL-08 | Nightly Ollama-live container runs Generate successfully | integration (testcontainers, build-tagged) | `go test -tags ollama_live -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live` | ❌ Wave 0 |
| CONF-01 | Shared httptest harness loads fixtures and starts server | unit (LoadFixture / NewMockServer round-trip) | `go test ./internal/contract/... -run TestContractHelpers` | ❌ Wave 0 |
| CONF-02 | Same fixture matrix runs against all 3 adapters; identical normalized output for happy path | conformance (table-driven) | `go test ./internal/contract/... -run TestGenerate_Conformance` | ❌ Wave 0 |
| CONF-07 | Capture script per provider produces a Fixture JSON | manual-only (smoke run with real key) | n/a — manual; documented in script docstring | ❌ Wave 0 (script files) |
| CONF-08 | goleak.VerifyTestMain reports zero leaks | TestMain | `go test ./internal/contract/...` (goleak is automatic) | ❌ Wave 0 |
| CORE-11 | Provider Author Guide v0.1 exists in core repo | manual-only (markdown-existence + lint) | `test -f PROVIDER_AUTHORING.md && wc -l PROVIDER_AUTHORING.md` | ❌ Wave 0 (file) |

### Sampling Rate (per `nyquist_validation: true`)

- **Per task commit:** `cd llm-agent-providers && go test -short ./<package>/...` for the package the task touched (≤ 5s typical).
- **Per wave merge:** `cd llm-agent-providers && go test ./...` (full mock suite; ~ 1 min; goleak runs at end).
- **Phase gate:** Full suite green + nightly-ollama-live green at least once before `/gsd-verify-work`. If the nightly hasn't run yet because the workflow file is brand-new, manually trigger it (`workflow_dispatch`).

### Wave 0 Gaps

These Wave-0 setup items must land before any per-requirement test can run:

- [ ] `llm-agent-providers/internal/contract/contract.go` — shared helpers (LoadFixture, NewMockServer, AssertGenerate, ChatModelFactory)
- [ ] `llm-agent-providers/internal/contract/main_test.go` — `TestMain` with `goleak.VerifyTestMain`
- [ ] `llm-agent-providers/internal/contract/generate_test.go` — table-driven driver iterating `AdapterFactories × testdata/`
- [ ] `llm-agent-providers/internal/contract/testdata/{openai,anthropic,ollama}/*.json` — capture happy-path fixtures via the per-provider scripts; hand-craft error-path fixtures (since real-API errors require real bad keys / quota states)
- [ ] `llm-agent-providers/scripts/capture-fixtures-{openai,anthropic,ollama}.sh` — bash one-shots (3 files)
- [ ] `llm-agent-providers/.github/workflows/nightly-ollama-live.yml` — CI workflow file
- [ ] Framework install (already in providers' go.mod after Phase 0): add the 3 SDK requires + testcontainers-go + goleak via `go get`

**Core-repo Wave-0 deliverables (before any sister-repo adapter compiles):**

- [ ] **Extend `llm/errors.go`** to add the 4 typed error structs (`AuthError`, `RateLimitError`, `InvalidRequestError`, `TransientError`) — see [§"Open Questions"](#open-questions-resolved) Q1 RESOLVED. **This is the planner's first decision point**: do this in Phase 1 (recommended, since it's small and adapters block on it) or insert as Phase 0.5.
- [ ] `llm-agent/PROVIDER_AUTHORING.md` — v0.1 markdown.

---

## Migration / integration notes

### Where each adapter package sits

```
llm-agent-providers/                          (sister repo; require github.com/costa92/llm-agent v0.3.0-pre.1 — already pinned by Phase 0)
├── go.mod
├── go.sum
├── openai/                                   NEW — Phase 1 plan 01-01
│   ├── openai.go
│   ├── options.go
│   ├── map.go
│   ├── errors.go
│   ├── doc.go
│   ├── openai_test.go
│   └── README.md
├── anthropic/                                NEW — Phase 1 plan 01-02
│   └── (same shape as openai/)
├── ollama/                                   NEW — Phase 1 plan 01-03
│   └── (same shape; map.go has the api.NewClient(*url.URL, *http.Client) wiring)
├── internal/
│   └── contract/                             NEW — Phase 1 plan 01-04
│       ├── contract.go
│       ├── generate_test.go
│       ├── main_test.go
│       └── testdata/
│           ├── openai/
│           ├── anthropic/
│           └── ollama/
├── scripts/
│   ├── workspace.sh                          (existing — Phase 0)
│   ├── capture-fixtures-openai.sh            NEW — Phase 1 plan 01-04
│   ├── capture-fixtures-anthropic.sh         NEW — Phase 1 plan 01-04
│   └── capture-fixtures-ollama.sh            NEW — Phase 1 plan 01-04
└── .github/workflows/
    ├── test.yml                              (existing — Phase 0)
    ├── release-precheck.yml                  (existing — Phase 0)
    └── nightly-ollama-live.yml               NEW — Phase 1 plan 01-04
```

### Sister-repo dependency direction (acyclic, K6 honored)

- `llm-agent-providers/openai`, `anthropic`, `ollama` — each imports `github.com/costa92/llm-agent/llm` (one direction). No cross-imports between adapter packages.
- `llm-agent-providers/internal/contract` — imports `github.com/costa92/llm-agent/llm` AND each adapter package (so it can register factories). Internal package — third parties cannot import it.
- `llm-agent` (core) — touched only for `PROVIDER_AUTHORING.md` + the typed-error extension to `llm/errors.go` (Wave 0).

### How each adapter imports from llm core

Every adapter's `openai.go` (and `anthropic.go`, `ollama.go`) imports:

```go
import (
    "context"
    // stdlib

    "github.com/costa92/llm-agent/llm"  // ChatModel, ProviderInfo, Capabilities, Request, Response, AuthError, RateLimitError, InvalidRequestError, TransientError, FinishReason*, UsageReported

    openai "github.com/openai/openai-go/v3"      // (or anthropic-sdk-go / ollama/api per provider)
    "github.com/openai/openai-go/v3/option"
)
```

The adapter's package-level `var _ llm.ChatModel = (*OpenAI)(nil)` compile-time assertion is the canonical marker that the contract is satisfied.

### Tag/Release sequencing

Phase 1 produces no release tags on its own. The Phase-1-complete state of `llm-agent-providers` is what gets tagged at Phase 4 close (when the full conformance suite is complete) — current plan is sister-repo `v0.1.0` at end of Phase 4 per ROADMAP.md. Phase 1 may push intermediate tags like `v0.1.0-pre.1` for test purposes; these are NOT release tags and are fine.

The core repo's `PROVIDER_AUTHORING.md` lands as part of the Phase 1 commit stream into `main`; no tag bump needed (it's a documentation addition, no code change).

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26 | All adapter compilation | (planner: verify) | (planner: verify with `go version`) | None — required |
| Docker | Nightly Ollama-live workflow only | (planner: verify in CI; local dev optional) | (CI runner: pre-installed) | None for nightly; PR CI doesn't need Docker |
| `gh` CLI | Capture scripts (optional, for `gh secret`-style auth flows) | (planner: verify locally) | latest | curl + env-var pattern works without `gh` |
| `jq` | Capture scripts (response post-processing) | likely yes (`apt install jq`) | 1.6+ | hand-write JSON wrapping if missing — slower but works |
| `curl` | Capture scripts | yes (stdlib of all Linux distros) | any | None — required |
| `OPENAI_API_KEY` env (LOCAL only) | `capture-fixtures-openai.sh` | requires user provision | n/a | Capture script fails fast with clear message; PR CI never runs this |
| `ANTHROPIC_API_KEY` env (LOCAL only) | `capture-fixtures-anthropic.sh` | requires user provision | n/a | Same |
| Local Ollama daemon (`http://localhost:11434`) | `capture-fixtures-ollama.sh` only | requires user-installed Ollama with model pulled | latest | Capture script fails with "is Ollama running?" message |
| GitHub Actions secrets for nightly | None (nightly uses testcontainers + container; no API keys) | n/a | n/a | n/a |

**Missing dependencies with no fallback:** None at the CI / development-environment level. PR CI is fully offline (httptest + mock fixtures).

**Missing dependencies with fallback:** `jq` (workaround documented above); `gh` (workaround: env vars).

---

## State of the Art

| Old Approach | Current Approach (2026-05-10) | When Changed | Impact |
|--------------|-------------------------------|--------------|--------|
| `sashabaranov/go-openai` (unofficial) | `openai-go/v3` (official) | OpenAI shipped v3 in mid-2025 | Use the official SDK for new code per STACK.md. The unofficial one tracks behind on Responses API + tool semantics |
| Anthropic Beta Messages (`client.Beta.Messages.New`) | Non-beta `client.Messages.New` for stable features | anthropic-sdk-go v1+ promoted Messages out of beta | Phase 1 uses non-beta. Beta path remains for genuinely beta features (prompt caching, structured outputs in 2025-11-13 beta) |
| `tcollama.RunContainer(ctx, req)` | `tcollama.Run(ctx, image, opts...)` | testcontainers-go v0.30+ | RunContainer is **deprecated** and will be removed in next major. Use `Run` from day one |
| `goleak.VerifyNone(t)` per-test | `goleak.VerifyTestMain(m)` in TestMain | Industry standard since early 2024 | Per-test version doesn't compose with `t.Parallel`; TestMain version verifies after all tests complete (correct for parallel suites) |
| `dnaeon/go-vcr` cassettes | `httptest.Server` + `testdata/*.json` | n/a — D-04 chose this for Phase 1 | Cassettes were a viable alternative; rejected for the dep-ceremony cost |

**Deprecated / outdated to avoid:**

- `client.Chat.Completions.NewStreaming` in Phase 1 — that's Phase 2's tool. Phase 1 uses non-streaming `client.Chat.Completions.New` only.
- `client.Beta.Messages.NewStreaming` in Phase 1 — Phase 2's tool.
- `api.ClientFromEnvironment()` for adapter construction when `WithBaseURL` is provided — fall back to it ONLY when neither `WithBaseURL` nor `WithHTTPClient` is set (env-fallback default).
- `RunContainer`, `ollama.New(ollama.WithContainer(...))` — both deprecated in current testcontainers-go.

---

## Assumptions Log

> Claims tagged `[ASSUMED]` in this research. The planner and discuss-phase use this section to identify decisions that need user confirmation before execution.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 4 typed error structs (`*llm.AuthError`, `*llm.RateLimitError`, `*llm.InvalidRequestError`, `*llm.TransientError`) need to be added to `llm/errors.go` in core. Current `llm/errors.go` only has the 2 sentinels (`ErrCapabilityNotSupported`, `ErrScriptExhausted`). REQUIREMENTS.md and CONTEXT.md and CLAUDE.md all reference these typed errors as if they exist. **`[ASSUMED]`** that this is a Wave-0 deliverable for Phase 1 (not an oversight in Phase 0 verification). | Migration / integration notes | If wrong (i.e., the planner thinks Phase 0 should be re-opened), Phase 1 blocks until errors are added. **Recommended resolution:** add to Phase 1 as Wave-0; small (~50 LOC). See Q1 RESOLVED below. |
| A2 | Anthropic SDK exports `*apierror.Error` from a non-internal path OR provides an alternative public type (e.g., `*anthropic.APIError`) that adapters can `errors.As` against. Context7 docs show `internal/apierror` import path which would NOT be importable from outside the SDK. **`[ASSUMED]`** that v1.41.0 either (a) re-exports the type publicly, (b) has a public alternative, or (c) we use a non-typed status-code extraction. | Pitfall B | If wrong, Anthropic's typed-error mapping needs an alternative strategy (e.g., parse `err.Error()` for known prefixes; or use SDK helper `anthropic.IsRateLimitError(err)` if it exists). Verify at plan-time. See Q2 RESOLVED. |
| A3 | Ollama Go SDK does NOT expose HTTP status code on its `error` return values from `client.Chat`/`client.Generate`. **`[ASSUMED]`** that recovering status code requires a custom `*http.Client` with a `RoundTripper` that captures the last response's status code in a struct field. WebSearch verified `api.NewClient(*url.URL, *http.Client) *Client` allows custom transport injection. | Pitfall B, Ollama Generate code example | If wrong (i.e., SDK exposes status code somehow), the adapter's `errors.go` is simpler. Verify at plan-time by reading `github.com/ollama/ollama/api` source. See Q3 RESOLVED. |
| A4 | `goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop")` is the canonical workaround for `httptest.Server` keep-alive false-positives in Go 1.26 + goleak v1.3.0. **`[ASSUMED]`** based on the goleak README pattern; not directly verified for this Go version. | Pitfall E | False-positives on PR CI block all Phase 1 work. Mitigation: verify at plan-time; if the ignore doesn't work, use `Transport.DisableKeepAlives = true` on the test client. |
| A5 | Anthropic non-beta `MessageNewParams.System` field is `[]anthropic.TextBlockParam` (lift verified for the Beta variant via Context7; non-beta is the stable analog). **`[ASSUMED]`** that the field name and type are identical between beta and non-beta. | Pitfall C, Anthropic Generate code example | If the type differs, the adapter compiles fine — just adjust the type assertion. Low risk; verify via `go doc` at plan-time. |
| A6 | Capture-script real-API capture works via simple `curl` for OpenAI (`POST /v1/chat/completions`), Anthropic (`POST /v1/messages` with `anthropic-version: 2023-06-01` header), Ollama (`POST http://localhost:11434/api/chat`). **`[ASSUMED]`** that all three providers' wire formats are capturable in <50 lines of bash + jq. | Capture scripts section | Low risk; if a provider has weirder auth (mTLS, OAuth-style flow), the script grows. None of the 3 providers use such auth as of 2026-05-10. |
| A7 | The Ollama testcontainers `Exec` command for pre-pulling a model returns within the workflow's `timeout-minutes: 45` budget on a clean cache. First-pull ~3-5 min for a 4.7GB quantized model. **`[ASSUMED]`** based on FlexPrice / community reports; not directly measured for `llama3.1:8b-instruct-q4_K_M`. | Pitfall D, CI YAML sketches | If pulls take longer (10+ min), bump `timeout-minutes`. The cache should make warm runs fast. |

**If this table is empty:** N/A — 7 assumptions logged; A1–A3 are the most consequential (require planner confirmation before Wave-0 work starts).

---

## Open Questions (RESOLVED)

These are decisions the research made on the planner's behalf. The planner should accept these unless they conflict with milestone constraints.

### Q1 RESOLVED: Where do the 4 typed error structs live?

**The question:** D-03 references `*llm.AuthError`, `*llm.RateLimitError`, `*llm.InvalidRequestError`, `*llm.TransientError` as types in the core `llm/` package. The current `llm/errors.go` (after Phase 0) has only the 2 sentinels. Where do these types come from?

**RESOLVED:** Add them to `llm/errors.go` as a Phase 1 Wave-0 deliverable in the **core** repo. The shape is:

```go
// llm/errors.go — additions (Phase 1 Wave-0)

// AuthError is returned by adapters when the provider rejects credentials.
// Wraps the SDK error in the Unwrap chain.
type AuthError struct {
    Provider string // "openai" | "anthropic" | "ollama"
    Wrapped  error
}

func (e *AuthError) Error() string {
    return fmt.Sprintf("%s: authentication failed: %v", e.Provider, e.Wrapped)
}
func (e *AuthError) Unwrap() error { return e.Wrapped }

// RateLimitError indicates the provider is rate-limiting the caller.
// RetryAfter is the value of any Retry-After header (provider-specific format).
// Reason is an optional discriminator (e.g., "quota_exhausted" for OpenAI insufficient_quota).
type RateLimitError struct {
    Provider   string
    RetryAfter string // raw header value; consumer parses
    Reason     string // optional: "quota_exhausted", "tier_limit", ""
    Wrapped    error
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("%s: rate limited (reason=%q, retry_after=%q): %v",
        e.Provider, e.Reason, e.RetryAfter, e.Wrapped)
}
func (e *RateLimitError) Unwrap() error { return e.Wrapped }

// InvalidRequestError indicates the request was malformed, the model name was wrong,
// the model wasn't pulled (Ollama), or any other 4xx-other condition.
type InvalidRequestError struct {
    Provider string
    Wrapped  error
}

func (e *InvalidRequestError) Error() string {
    return fmt.Sprintf("%s: invalid request: %v", e.Provider, e.Wrapped)
}
func (e *InvalidRequestError) Unwrap() error { return e.Wrapped }

// TransientError indicates a 5xx, network failure, or context.DeadlineExceeded —
// the caller MAY retry per the K4 retry state machine (Phase 2).
type TransientError struct {
    Provider string
    Wrapped  error
}

func (e *TransientError) Error() string {
    return fmt.Sprintf("%s: transient error: %v", e.Provider, e.Wrapped)
}
func (e *TransientError) Unwrap() error { return e.Wrapped }
```

**Why core, not sister repo:** D-03 explicitly says "Each adapter wraps SDK errors into one of `llm.*Error` ..." — these are SHARED types so consumers (agent layer, tests, OTel adapter) can `errors.As` against a common type regardless of which provider produced the error. They cannot live per-adapter.

**Planner action:** Include this as a Wave-0 task in Phase 1, ~50 LOC, lives in `llm/errors.go`. The Phase-0 verification report (00-VERIFICATION.md) confirmed Phase 0 was scoped to interface-level work; these typed errors are Phase 1's typed-error-taxonomy work (per requirement OAI-05/ANT-05/OLL-05). Cross-reference: `_test.go` for round-trip / Unwrap chain.

### Q2 RESOLVED: How do we type-assert against Anthropic SDK errors?

**The question:** Context7 examples show `var apiErr *apierror.Error` from `github.com/anthropics/anthropic-sdk-go/internal/apierror`. The `internal/` path makes that type unimportable.

**RESOLVED:** Two-tier strategy, evaluated at plan-time:

1. **Preferred:** check if v1.41.0 re-exports the type at a public path. Run `go doc github.com/anthropics/anthropic-sdk-go.Error` — if it shows a type, use that.
2. **Fallback:** the Anthropic Go SDK conventionally exposes a public `anthropic.NewClient`-level mechanism. As of v1.41.0, the recommended pattern for SDK consumers is to use the SDK's middleware hook or to inspect raw HTTP response. The cleanest fallback for our adapter:
   - Inject a custom `*http.Client` via `WithHTTPClient`/`option.WithHTTPClient`; the transport records the last response's `StatusCode` in an adapter-local field.
   - On error from `client.Messages.New`, the adapter consults the recorded status code and dispatches to the `wrapErr` switch.

**Planner action:** Spend 30 min at plan-time running `go doc -all github.com/anthropics/anthropic-sdk-go | head -100` to enumerate exported types. Document which path is used in the Anthropic adapter's `errors.go`. If neither path works cleanly, fall back to (2) — RoundTripper-based status-code capture.

### Q3 RESOLVED: How do we type-assert / extract HTTP status from Ollama SDK errors?

**The question:** `github.com/ollama/ollama/api` returns plain `error` from `client.Chat`. No typed-error hierarchy.

**RESOLVED:** Use the same RoundTripper-based status-code-capture pattern as Q2's fallback:

```go
// llm-agent-providers/ollama/options.go — sketch
type statusCapturingTransport struct {
    inner http.RoundTripper
    last  *int32 // atomic; *int32 not int because we want lock-free read in Generate
}

func (t *statusCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    resp, err := t.inner.RoundTrip(req)
    if resp != nil {
        atomic.StoreInt32(t.last, int32(resp.StatusCode))
    }
    return resp, err
}

// In ollama.New:
//   sct := &statusCapturingTransport{inner: cfg.httpClient.Transport, last: new(int32)}
//   wrappedClient := &http.Client{Transport: sct, Timeout: cfg.httpClient.Timeout}
//   apiClient := api.NewClient(parsedURL, wrappedClient)
//   ollama := &Ollama{client: apiClient, lastStatus: sct.last, info: ...}
//
// In ollama.wrapErr:
//   status := int(atomic.LoadInt32(o.lastStatus))
//   switch {
//     case status == 401 || status == 403:
//         return &llm.AuthError{Provider: "ollama", Wrapped: err}
//     // ...
//   }
```

**Why this works:** Ollama is a local daemon — it doesn't have OAuth refresh tokens or other reasons the inner client would fire multiple round-trips per `client.Chat` call. The "last status" is unambiguously the status of the call that produced the error.

**Caveat:** If Ollama's SDK ever does retry internally, the captured status will be the LAST attempt's, not the first. Document this in the adapter's `errors.go` comment. Acceptable for Phase 1.

**Planner action:** Include this RoundTripper construction in plan 01-03 (Ollama adapter). The same pattern is the fallback for Q2 if Anthropic's typed-error path is unavailable.

### Q4 RESOLVED: Tool field passthrough — pass through, parse, or ignore?

**The question:** OpenAI and Anthropic SDKs both support `tools` in non-streaming responses; the model can return tool-call content blocks. Should Phase 1 adapters honor `Request.Tools`?

**RESOLVED:** **Ignore `Request.Tools` entirely in Phase 1.** All 3 adapters set `Capabilities.Tools = false` in `Info()`. Document in PROVIDER_AUTHORING.md v0.1: "Phase 1 adapters MUST NOT honor Request.Tools; the field is reserved for Phase 3."

**Why:** The recommendation in CONTEXT.md hedged between "pass through, parse single tool-call response" and "no tool field handling, period." The latter is cleanest because:
1. Phase 3 will introduce per-tool-call indexing (Pitfall 1), parallel tool calls (OpenAI), multi-block tool_use (Anthropic), per-model strategy table (Ollama Pitfall 19), and dedupe by `(message_id, tool_use_id)` (Pitfall 4). NONE of that infrastructure exists in Phase 1.
2. "Pass through and parse single tool-call response" is a false economy — the moment a real model emits parallel tool calls, the Phase-1 parser breaks subtly. Better to fail explicitly (`Capabilities.Tools = false`) than silently degrade.
3. Conformance is easier: assert `Tools: false` everywhere and assert `Request.Tools` is not in the wire body.

**Planner action:** Each adapter's `map.go` does NOT pass `Request.Tools` to the SDK. `Info()` returns `Capabilities{Tools: false, Embeddings: false, StructuredOutputs: false, PromptCaching: false}` for all 3.

### Q5 RESOLVED: testcontainers-go API choice

**The question:** Is `RunContainer` or `Run` the canonical entry point in current testcontainers-go releases? CONTEXT.md mentions both shapes.

**RESOLVED:** Use `tcollama.Run(ctx, "ollama/ollama:0.5.7")` (verified [official docs 2026-05-10](https://golang.testcontainers.org/modules/ollama/)). `RunContainer` is deprecated and slated for removal. Pre-pull the model via `container.Exec(ctx, []string{"ollama", "pull", model})` after `Run` returns — there is no startup-time `WithModel` option for the Ollama module (the `WithModel` option exists for the `dockermodelrunner` module but not the `ollama` module).

**Planner action:** In nightly-ollama-live test, use `tcollama.Run` + `Exec` + `ConnectionString` per the §"CI YAML sketches" sketch.

### Q6 RESOLVED: Where does `PROVIDER_AUTHORING.md` live?

**The question:** CORE-11 says "Provider Author Guide v0.1 in `llm-agent` core." CONTEXT.md confirms. But the guide describes how to write a sister-repo provider — does it live in core or in `llm-agent-providers`?

**RESOLVED:** Lives in the **core repo** at `llm-agent/PROVIDER_AUTHORING.md`. The contract being documented is in core; the guide describing how to satisfy that contract belongs where the contract lives. Sister repos can link to it from their READMEs.

**Planner action:** The Phase 1 plan that ships PROVIDER_AUTHORING.md (CONTEXT.md mentions plan `01-05`) creates a file in **`llm-agent/`** (not in `llm-agent-providers/`). This is the only Phase 1 file that lands in the core repo apart from the Wave-0 typed-error extension.

### Q7 RESOLVED: How does each adapter expose its factory to the conformance harness?

**The question:** `internal/contract/generate_test.go` registers a `map[string]ChatModelFactory`. How do adapters expose their factories without circular imports?

**RESOLVED:** The conformance test imports each adapter package directly and constructs the factory inline:

```go
var AdapterFactories = map[string]ChatModelFactory{
    "openai": func(baseURL string) (llm.ChatModel, error) {
        return openai.New(openai.WithModel("gpt-4o-mini"), openai.WithAPIKey("test"), openai.WithBaseURL(baseURL))
    },
    // ...
}
```

This works because `internal/contract/` is internal — only the conformance test (also in `internal/contract/`) imports both `openai` and `anthropic` and `ollama`. **No circular import** because adapter packages do NOT import `internal/contract`. Contract package imports adapter packages; adapter packages import only core `llm`.

**Planner action:** No special factory-registration mechanism needed. Drop the inline `ChatModelFactory` definitions in `generate_test.go`. Adding a new provider = add a row to the map.

### Q8 RESOLVED: Goroutine-leak ignore list for httptest in Phase 1

**The question:** Pitfall E flags potential `goleak` false-positives from `httptest.Server` keep-alive connections.

**RESOLVED:** Start with **no** `IgnoreTopFunction` workaround. Phase 1's tests are sync (no streaming, no per-test goroutines); httptest's connection-readers should drain by `server.Close()`. If false-positives appear in PR CI, add `goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop")` per the goleak README. Verify at plan-time which is needed.

**Planner action:** Ship `main_test.go` with `goleak.VerifyTestMain(m)` (no ignore list). If the first PR CI run shows a leak from `persistConn.readLoop`, follow up with the ignore line in a small-fix PR.

---

## Sources

### Primary (HIGH confidence — Context7-verified 2026-05-10)

- **`/openai/openai-go`** — Chat Completions API (`client.Chat.Completions.New`), `option.WithAPIKey` / `option.WithBaseURL` / `option.WithHTTPClient` / `option.WithHeader` / `option.WithMaxRetries` / `option.WithRequestTimeout` / `option.WithMiddleware`, `*openai.Error` with `StatusCode int` / `Type string` / `Code string` / `Message string` / `Headers() http.Header` / `DumpRequest(true)`. Verified ✓
- **`/anthropics/anthropic-sdk-go`** — `client.Messages.New(ctx, anthropic.MessageNewParams{...})` non-beta path; `option.WithBaseURL` / `option.WithAPIKey` / `option.WithMaxRetries` / `option.WithMiddleware`; `*apierror.Error` with `StatusCode int` / `RequestID string` / `RawJSON()` / `DumpRequest(true)`; `MessageNewParams.System []TextBlockParam` (lift verified for beta variant; non-beta inferred). Verified ✓ (with caveat on `apierror` import path — see Q2)
- **`/ollama/ollama`** — `api.ClientFromEnvironment()` + `api.NewClient(*url.URL, *http.Client)` (verified via WebSearch on [client.go source](https://github.com/ollama/ollama/blob/main/api/client.go)); `client.Chat(ctx, *api.ChatRequest, func(api.ChatResponse) error)` callback shape; `Stream: new(bool)` (pointer-to-false) for non-streaming. Verified ✓
- **`/testcontainers/testcontainers-go`** modules/ollama — `tcollama.Run(ctx, image, opts...)`, `container.Exec(ctx, []string{"ollama", "pull", model})`, `container.ConnectionString(ctx)`. `RunContainer` deprecated. Verified ✓ via Context7 + [golang.testcontainers.org/modules/ollama](https://golang.testcontainers.org/modules/ollama/)
- **`go.uber.org/goleak`** — `goleak.VerifyTestMain(m, opts...)` in `func TestMain(m *testing.M)`; `goleak.IgnoreTopFunction(funcName)` for false-positive suppression. Verified ✓ via [goleak README](https://github.com/uber-go/goleak/blob/master/README.md) and [pkg.go.dev goleak](https://pkg.go.dev/go.uber.org/goleak)

### Secondary (MEDIUM confidence — verified via Context7 + WebSearch)

- Ollama issue #2948 (`api.NewClient(*url.URL, *http.Client)` signature) — [github.com/ollama/ollama/issues/2948](https://github.com/ollama/ollama/issues/2948)
- testcontainers Ollama module current API confirmation — [WebFetch official docs 2026-05-10](https://golang.testcontainers.org/modules/ollama/)
- Anthropic SDK error type path inference — Context7 examples show `internal/apierror`; canonical resolution requires `go doc -all` at plan-time

### Tertiary (LOW confidence — needs validation at plan-time)

- Q4 (no tool passthrough): user-led recommendation. The "ignore tools entirely" choice was made by this research; the planner may revisit if a user signals they want pass-through-with-Tools-true.
- Q8 (goleak ignore needed?): empirical — first PR CI run will tell us. Defer to "see what fires" rather than pre-ignoring.

### Local context
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/01-walking-skeleton-generate/01-CONTEXT.md` — D-01 to D-04 locked; canonical refs
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/REQUIREMENTS.md` — Phase 1 requirement IDs OAI-01/05, ANT-01/05, OLL-01/05/08, CONF-01/02/07/08, CORE-11
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/STACK.md` — verified versions: openai-go v3.35.0, anthropic-sdk-go v1.41.0, ollama/api v0.23.2
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/PITFALLS.md` — Pitfalls 1, 2, 3, 4, 5, 6, 12, 13, 14, 15, 19, 20, 21, 22 catalogued
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/00-keystone-interfaces/00-VERIFICATION.md` — Phase 0 PASS confirms locked Core repo surface
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/{chatmodel,info,types,errors,scripted,chat_only_mock,capabilities,stream,legacy,doc}.go` — locked Phase 0 surface; verified contents to write Phase 1 against
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/CLAUDE.md` — 8 hard rules verified honored throughout this research

---

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — versions Context7-verified 2026-05-10; STACK.md cross-references; SDK shapes confirmed via direct doc retrieval
- Architecture: **HIGH** — D-01 through D-04 lock the four big design questions; remaining work is mechanical
- Pitfalls: **HIGH** on the 6 Phase-1-relevant pitfalls (1, 2, 3, 4, 5, 19) — all sourced; mitigation strategies documented
- Conformance harness shape: **HIGH** — D-04 locked the format; sketch above is concrete and testable
- Code examples: **HIGH** — all 3 Generate sketches verified against Context7-fetched SDK examples
- Anthropic typed-error path: **MEDIUM** — `internal/apierror` is the documented type; public re-export needs plan-time `go doc` confirmation (Q2)
- Ollama HTTP-status capture: **MEDIUM** — RoundTripper pattern is sound but specific-to-Ollama; no community example found, so this research is the first canonical writeup (Q3)
- goleak ignore list: **LOW** — Q8 deferred to "see what PR CI says" empirically

**Research date:** 2026-05-10
**Valid until:** 2026-06-09 (30 days for stable SDK shapes; bring forward if openai-go v4 or anthropic-sdk-go v2 ships within window)

---

*Phase: 01-walking-skeleton-generate*
*Research completed: 2026-05-10*
*Ready for `/gsd-plan-phase 1`: yes — 7 open questions resolved, 7 assumptions logged for planner confirmation*
