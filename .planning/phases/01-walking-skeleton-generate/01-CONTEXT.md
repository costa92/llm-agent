# Phase 1: Walking Skeleton — Generate (sync) only - Context

**Gathered:** 2026-05-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Lock the `ChatModel.Generate` + `ProviderInfo(model)` contract against ALL THREE real wire formats (OpenAI Chat Completions, Anthropic Messages, Ollama `/api/chat`) before introducing streaming. Phase 1 produces:

- `llm-agent-providers/openai/` — Generate-only adapter against `github.com/openai/openai-go/v3` (Chat Completions API)
- `llm-agent-providers/anthropic/` — Generate-only adapter against `github.com/anthropics/anthropic-sdk-go`
- `llm-agent-providers/ollama/` — Generate-only adapter against `github.com/ollama/ollama/api`
- `llm-agent-providers/internal/contract/` — shared httptest+testdata conformance harness; `goleak` integrated; runs same fixtures against all 3
- `llm-agent-providers/scripts/capture-fixtures-{openai,anthropic,ollama}.sh` — one-shot real-API capture scripts (local only; never run in CI)
- Typed error taxonomy implemented per-adapter — RateLimitError, AuthError, InvalidRequestError, TransientError
- `PROVIDER_AUTHORING.md` v0.1 in `llm-agent` core — documents the Generate contract, functional-options constructor pattern, and HTTP-status → typed-error mapping table
- Per-repo `test.yml` already runs PR mock-only CI; this phase ADDS a nightly Ollama-live workflow using testcontainers-go that runs the conformance suite against a real `llama3.1:8b` container

NO streaming, NO native tool calling, NO embeddings. Those are Phases 2, 3, 4. Tool calling at Phase 1 is allowed only as a request struct field — adapters may pass `Tools` through the wire but are not required to honor parallel-tool-call streaming, dedupe, or capability-degraded fallback.

</domain>

<decisions>
## Implementation Decisions

### OpenAI API Surface (Area 1)

- **D-01 (P1): Phase 1 OpenAI adapter targets the Chat Completions API only.**
  - Concrete API path: `client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{...})` via openai-go v3.
  - **Why:** Chat Completions has the broadest model coverage, the deepest documentation, the most community examples, and is the de-facto baseline for "does Generate work?" The Responses API's Stateful-conversation advantage is irrelevant for a sync `Generate(ctx, Request) Response` call.
  - **Responses API deferred:** add as a follow-up in Phase 2 (streaming) or Phase 3 (tools), where Responses' content-block streaming and built-in tool runner add real value. When it lands, gate via constructor option `WithAPIVersion(openai.APIChat | openai.APIResponses)` — default stays Chat Completions for v0.3.
  - **Cascades into:** OAI-01 implementation detail; PROVIDER_AUTHORING.md v0.1 documents Chat Completions as the Phase-1 target; conformance suite captures Chat Completions wire format.

### Provider Constructor Pattern (Area 2)

- **D-02 (P1): Functional options across all three adapters.** Uniform package-level `New` constructor: `openai.New(opts ...Option) *OpenAI`, same shape for `anthropic.New(...)` and `ollama.New(...)`.
  - Universal options (every adapter):
    - `WithModel(string) Option` — required (no default; agent layer needs deterministic per-(provider × model) capability binding per K2)
    - `WithAPIKey(string) Option` — overrides env-var default
    - `WithHTTPClient(*http.Client) Option` — for retries, custom transports, OTel wrap
    - `WithBaseURL(string) Option` — useful for proxies, Azure OpenAI endpoints, Ollama remote hosts
    - `WithTimeout(time.Duration) Option` — request-level timeout (separate from ctx)
  - Per-provider extras follow the same pattern with provider-prefixed names where ambiguity exists (e.g., `openai.WithOrganization`, `anthropic.WithBetaHeader`, `ollama.WithHost`).
  - **API key sourcing default:** if `WithAPIKey` is not provided, adapter reads `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` env var. Ollama has no key — it reads `OLLAMA_HOST` for base URL, defaults to `http://localhost:11434`.
  - **Why:** Go-idiomatic (stdlib `log/slog.NewJSONHandler` + `slog.NewTextHandler`, gRPC, openai-go v3 itself). Default values clean (omit option = default). Adding new options is BC-additive (no struct field break). Cost is verbose call sites, but providers are constructed once per process — not a hot path.
  - **No default model:** every adapter requires explicit `WithModel(...)`. Constructor returns error from `New` (or panics with clear message) if model is empty. This enforces K2 — `ProviderInfo` reflects the bound model's capabilities; an unbound provider is meaningless.
  - **Cascades into:** all three adapter constructor signatures; PROVIDER_AUTHORING.md documents the pattern as canonical for sister-repo providers.

### Error Classification Mapping (Area 3)

- **D-03 (P1): Per-adapter mapping; PROVIDER_AUTHORING.md documents the recommended HTTP-status → typed-error table; adapters may extend with provider-specific overrides.**
  - **Recommended mapping table (PROVIDER_AUTHORING.md Phase 1 contract):**

    | HTTP / Cause | Typed error |
    |--------------|-------------|
    | 401, 403 | `*llm.AuthError` |
    | 429 | `*llm.RateLimitError` |
    | 4xx other (400, 404, 422, etc.) | `*llm.InvalidRequestError` |
    | 5xx | `*llm.TransientError` |
    | `errors.Is(err, context.DeadlineExceeded)` | `*llm.TransientError` |
    | `errors.Is(err, context.Canceled)` | propagate as-is (NOT a typed llm error) |
    | network error (DNS, TCP reset, etc.) | `*llm.TransientError` |

  - **Provider-specific overrides** (adapters override per the recommendation, NOT a llm-core helper):
    - OpenAI: `insufficient_quota` (429 with specific code) → still `*llm.RateLimitError` but with `Reason: "quota_exhausted"`.
    - Anthropic: `overloaded_error` (529) → `*llm.RateLimitError` (semantically rate-limit-like, NOT transient).
    - Anthropic: `invalid_request_error` (400 with specific JSON code) → `*llm.InvalidRequestError`.
    - Ollama: model-not-pulled (404 with specific message body) → `*llm.InvalidRequestError` (not transient — operator action required).
  - **Per-adapter `errors.go` file** does the wrapping. NO shared `llm-agent-providers/internal/errors` helper — each adapter's mapping is independent (different SDK error types). Avoids brittle cross-provider coupling.
  - **Original SDK error preserved via `errors.Unwrap` chain.** Users can `errors.As(err, &openAIErr)` if they need provider-specific detail.
  - **Conformance assertion:** the conformance suite asserts that fixtures with HTTP 401/429/500/etc. produce the correct typed error (regardless of which adapter is under test). This forces all 3 adapters to converge on the same external semantics.
  - **Cascades into:** OAI-05, ANT-05, OLL-05 (typed error taxonomy per adapter); CONF-02 (conformance suite asserts error mapping); PROVIDER_AUTHORING.md (Phase 1 has the mapping table).

### Conformance Fixture Format (Area 4)

- **D-04 (P1): testdata/*.json files + httptest server loader.**
  - **Layout:**
    ```
    llm-agent-providers/
      internal/contract/
        contract.go              # shared helpers: LoadFixture, NewMockServer, AssertGenerate
        generate_test.go         # the conformance suite (runs against all 3 adapters)
        testdata/
          openai/
            generate_happy_gpt-4o-mini.json
            generate_401.json
            generate_429.json
            generate_500.json
          anthropic/
            generate_happy_claude-3-5-haiku.json
            generate_400_invalid_request.json
            generate_529_overloaded.json
          ollama/
            generate_happy_llama3.1-8b.json
            generate_404_model_not_pulled.json
            generate_500_oom.json
      scripts/
        capture-fixtures-openai.sh
        capture-fixtures-anthropic.sh
        capture-fixtures-ollama.sh
    ```
  - **Fixture format:** each JSON file contains both the **request** (what the SDK should have sent) and the **response** (what the server should reply). Conformance test loads, starts httptest.Server, configures adapter via `WithBaseURL(server.URL)`, calls Generate, asserts response matches.
    ```json
    {
      "request": {
        "method": "POST",
        "path": "/v1/chat/completions",
        "body_assertions": ["model=gpt-4o-mini", "messages contains 'hello'"]
      },
      "response": {
        "status": 200,
        "headers": {"Content-Type": "application/json"},
        "body": "{...verbatim from real API capture...}"
      }
    }
    ```
  - **`scripts/capture-fixtures-<provider>.sh`** — runs the real adapter against the real API (using a real key from env), saves request+response pairs to `testdata/<provider>/<scenario>.json`. Local-only — never invoked from CI. Contributors run it once when adding/refreshing fixtures, then commit.
  - **goleak in conformance suite:** every test in `generate_test.go` wraps with `goleak.VerifyTestMain` — Phase 1 catches any goroutine-leaking adapter early (Pitfall 3 prevention before streaming lands in Phase 2).
  - **Why this format:** version-controlled, diff-friendly, easy to inspect ("what does the OpenAI 429 body actually look like?"), no external recording dep, no API keys in PR CI. Adding a new fixture = drop a JSON file. Refactoring the adapter doesn't break fixture format.
  - **NOT chosen — `dnaeon/go-vcr`:** would force `llm-agent-providers` to take a non-stdlib dep for testing only (acceptable per the sister-repo deps policy, but it's still an extra dep with its own version-pinning ceremony). The testdata-JSON approach has no such cost.
  - **NOT chosen — inline httptest JSON in test code:** a 1KB OpenAI streaming chunk in a Go string literal is illegible; refactoring requires Go recompile.
  - **Cascades into:** CONF-01 (harness shape), CONF-02 (Generate-only conformance), CONF-07 (capture script per provider), CONF-08 (goleak integration). Format is reused as-is for streaming (Phase 2) and tools (Phase 3) — JSON files just gain SSE/streaming-content fields.

### Claude's Discretion

- **Package layout in `llm-agent-providers`:** subpackages `openai/`, `anthropic/`, `ollama/` at top level. `internal/contract/` for the shared harness. No `pkg/` or `cmd/` subdirs (this is a library module, not a binary). Match the existing core repo style: flat top-level subpackages.
- **Default option ordering in constructor invocations:** users typically write `openai.New(openai.WithModel("..."), openai.WithAPIKey("..."))`. The order doesn't matter for application but readability suggests `Model` first.
- **Test naming convention:** `TestGenerate_OpenAI_Happy`, `TestGenerate_Anthropic_429`, etc. — provider name in the middle, scenario at the end. Matches Go convention of `TestSubject_Variant`.
- **Nightly Ollama-live workflow location:** `llm-agent-providers/.github/workflows/nightly-ollama-live.yml` (separate file from `test.yml`). Schedules at 03:00 UTC. Pulls `llama3.1:8b-instruct-q4_K_M` (~4.7GB) once, caches the testcontainers image.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (always)
- `.planning/PROJECT.md` — milestone scope, Core Value (stdlib-only core), Validated/Active/Out of Scope
- `.planning/REQUIREMENTS.md` — Phase 1 covers OAI-01, OAI-05, ANT-01, ANT-05, OLL-01, OLL-05, OLL-08, CONF-01, CONF-02, CONF-07, CONF-08, CORE-11
- `.planning/ROADMAP.md` §"Phase 1: Three-provider walking skeleton — Generate (sync) only" — phase scope, success criteria, pitfalls guarded
- `.planning/STATE.md` — current position
- `CLAUDE.md` — 8 hard rules: stdlib-only core (sister repos may take deps); no K8s; capabilities per-(provider × model); typed StreamEvent union
- `DEPRECATIONS.md` — `llm.LegacyClient` removal at v0.4.0; Phase 1 doesn't touch this

### Phase 0 outputs (this phase consumes)
- `llm-agent/llm/chatmodel.go` — `ChatModel` interface signature; what each adapter must implement
- `llm-agent/llm/info.go` — `ProviderInfo` + `Capabilities` shape; each adapter's `Info()` returns this
- `llm-agent/llm/types.go` — `Request`, `Response`, `Message`, `Tool`, `ToolCall`, `Vector`, `Usage`, `UsageSource`, `FinishReason`
- `llm-agent/llm/errors.go` — `RateLimitError`, `AuthError`, `InvalidRequestError`, `TransientError` types; adapters wrap SDK errors into these
- `llm-agent/llm/scripted.go` — ScriptedLLM v2; conformance suite uses this as the baseline (every conformance scenario passes for ScriptedLLM by construction)
- `llm-agent/llm/chat_only_mock.go` — ChatOnlyMock; reference for "minimum viable adapter that satisfies just ChatModel"
- `llm-agent/.github/workflows/test.yml` — pattern for per-repo PR CI
- `llm-agent-providers/.github/workflows/test.yml` — sister-repo CI shape (already created Phase 0)
- `llm-agent-providers/scripts/workspace.sh` — go.work writer for cross-repo dev

### Research bundle (Phase 1 directly consumes)
- `.planning/research/SUMMARY.md` §"The 5–7 Keystone Decisions" — K2 (per-(provider × model) ProviderInfo) is exercised first time in Phase 1
- `.planning/research/STACK.md` §"Provider SDKs" — versions to use:
  - `github.com/openai/openai-go/v3` v3.35.0
  - `github.com/anthropics/anthropic-sdk-go` v1.41.0
  - `github.com/ollama/ollama/api` v0.23.2
- `.planning/research/STACK.md` §"Testing — testcontainers-go" — Ollama-live nightly setup
- `.planning/research/PITFALLS.md` Pitfalls 3 (goroutine leak harness lands here for use in Phase 2), 19 (Ollama per-model — note for record, not exercised until tools land in Phase 3), 20 (perfectionism — Phase 1 forces breadth)
- `.planning/research/FEATURES.md` §"Provider adapters — table stakes" — what's MUST per provider

### External SDK references (consult during implementation)
- [openai/openai-go v3 — Chat Completions](https://github.com/openai/openai-go) — verify Chat Completions params struct; per Q3 (Responses API deferred), do NOT use `client.Responses.*` in Phase 1
- [anthropics/anthropic-sdk-go — Messages.New](https://github.com/anthropics/anthropic-sdk-go) — sync Generate maps to `client.Messages.New(ctx, ...)`
- [Ollama API — /api/chat](https://github.com/ollama/ollama/blob/main/docs/api.md) — sync Generate uses chat endpoint with `stream: false`
- [testcontainers-go/modules/ollama](https://golang.testcontainers.org/modules/ollama/) — nightly Ollama-live container

### Codebase artifacts to reference at planning time
- `llm-agent/llm/scripted.go` — ScriptedLLM v2 satisfies all capabilities; conformance suite uses this as the "ground truth" baseline
- `llm-agent/llm/chat_only_mock.go` — ChatOnlyMock; reference shape for adapters that ONLY satisfy ChatModel (Phase 1's exact target)
- `llm-agent/example_simple_test.go` — Simple paradigm example; conformance suite can mirror this shape for "happy path" assertions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`llm/scripted.go` ScriptedLLM v2** — the conformance suite's baseline. Every fixture scenario (happy / 401 / 429 / 500) is asserted to pass for ScriptedLLM (configured via Script). Real adapters must converge on the same observable behavior.
- **`llm/chat_only_mock.go` ChatOnlyMock** — the minimum-viable shape. Each new adapter (OpenAI/Anthropic/Ollama) is structurally similar to ChatOnlyMock + real HTTP transport.
- **`llm/errors.go`** — typed error sentinels live here. Adapters import these and wrap SDK errors. NO new error types in Phase 1 (Phase 0 created the universe).
- **`.github/workflows/test.yml` (per-repo, in providers sister repo)** — pattern: matrix Go version, lint+vet+test, GOWORK=off env. Phase 1 ADDS `nightly-ollama-live.yml` alongside (separate workflow file).
- **`scripts/workspace.sh`** — already in providers sister repo from Phase 0. Phase 1 doesn't touch it.

### Established Patterns

- **Functional options pattern (D-02 above)** — Go-idiomatic; matches stdlib slog, gRPC, openai-go v3. Adapters surface options at package level (`openai.WithModel`, not `openai.OpenAI{}.WithModel`).
- **Error wrapping with `errors.As/Is`-friendly types** — sentinel errors at the bottom (in `llm/errors.go`), wrapped by adapters preserving the SDK error in the Unwrap chain.
- **Test files colocated with code** — `openai/openai_test.go` next to `openai/openai.go`; conformance suite lives separately at `internal/contract/generate_test.go` because it iterates over all providers.
- **`json` tag everywhere on public types** — already established in core; conformance fixtures' JSON parses these tags.
- **No init()** — adapters constructed explicitly via `New(...)`. No package-level state, no `init()` env-var reads. Read env vars only at construction time when an option isn't supplied.
- **Sentinel-error block** — single `var (...)` block with `<package>: <description>` prefix; companion test using `errors.Is`. From `llm/errors.go` and pre-Phase-0 `agent.go:127-136`.

### Integration Points

- **Where new code connects to existing system:**
  - **`llm-agent-providers/openai`, `anthropic`, `ollama`** — three new packages in the sister repo. Each implements `llm.ChatModel` from the core repo. Wire format-specific.
  - **`llm-agent-providers/internal/contract`** — internal package (cannot be imported externally) — shared conformance harness. Tests run as `go test ./internal/contract/...`.
  - **`llm-agent-providers/scripts/capture-fixtures-*.sh`** — bash scripts; depend on `gh`, `curl`, real API keys (from env or `~/.netrc`). Local-only.
  - **`llm-agent-providers/.github/workflows/nightly-ollama-live.yml`** — new GHA workflow alongside the existing `test.yml`. `cron: '0 3 * * *'`. Uses `testcontainers-go/modules/ollama`. Pulls `llama3.1:8b-instruct-q4_K_M`.
  - **`llm-agent/PROVIDER_AUTHORING.md`** — new file in core repo (NOT in providers sister repo — the guide describes how to write a provider, so it lives where the contract lives). Documents D-01..D-04 from Phase 1 + the Phase 0 keystones it consumes.
  - **`require github.com/costa92/llm-agent v0.3.0-pre.1`** — already in providers sister repo's `go.mod`. Tag is now pushed (verified live). First Phase-1 PR's CI will run with the dependency resolvable.

- **Sister repo dependency direction (still acyclic):**
  - `llm-agent-providers` depends on `llm-agent` (`require ... v0.3.0-pre.1`). One direction. K6 honored.
  - Each adapter package within `llm-agent-providers` is INDEPENDENT — `openai/` does NOT import from `anthropic/` or `ollama/`. They're parallel.

</code_context>

<specifics>
## Specific Ideas

- **Constructor signature exact form:**
  ```go
  // llm-agent-providers/openai/openai.go
  func New(opts ...Option) (*OpenAI, error) {
      cfg := defaults()
      for _, opt := range opts {
          opt(&cfg)
      }
      if cfg.Model == "" {
          return nil, errors.New("openai: WithModel is required")
      }
      // ... build client
      return &OpenAI{...}, nil
  }
  ```
- **Error mapping example (OpenAI):**
  ```go
  func wrapErr(err error) error {
      var apiErr *openai.Error
      if !errors.As(err, &apiErr) { return err }
      switch apiErr.StatusCode {
      case 401, 403:
          return &llm.AuthError{Provider: "openai", Wrapped: err}
      case 429:
          return &llm.RateLimitError{Provider: "openai", RetryAfter: apiErr.Headers.Get("Retry-After"), Wrapped: err}
      case 500, 502, 503, 504:
          return &llm.TransientError{Provider: "openai", Wrapped: err}
      default:
          if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
              return &llm.InvalidRequestError{Provider: "openai", Wrapped: err}
          }
      }
      return err
  }
  ```
- **Fixture file naming convention:** `<scenario>_<modifier>.json`. e.g., `generate_happy_gpt-4o-mini.json`, `generate_429_with_retry_after.json`, `generate_400_invalid_model.json`. Provider directory provides the wire-format namespace.
- **Nightly Ollama-live model pin:** `llama3.1:8b-instruct-q4_K_M` (~4.7GB). Quantized for fast container start (~3-5min cold pull, near-instant warm). Conformance suite asserts Generate works against this specific model. If Ollama upstream changes the model's behavior, the nightly workflow goes RED — surface as actionable signal, NOT a blocker for PR CI.

</specifics>

<deferred>
## Deferred Ideas

- **Streaming on all 3 providers** — Phase 2 scope (CONF-03; OAI-02, ANT-02, OLL-02). Phase 1 explicitly stays sync-only.
- **Native tool calling** — Phase 3 scope. Tools may pass through Request struct in Phase 1 but adapters NOT required to honor them.
- **Embeddings** — Phase 4 scope.
- **Responses API for OpenAI** — Phase 2 (with streaming) or Phase 3 (with tools). `WithAPIVersion(...)` constructor option NOT added in Phase 1.
- **Anthropic prompt caching (`Message.CacheControl`)** — P2 / v0.4 (per FEATURES.md). NOT in Phase 1.
- **Cost-table for token-cost estimation** — DIFF-04 in v2 requirements. Phase 1's three-state Reported/Estimated/Unknown is a Phase 2 concern (lands with streaming + retry SM per K4); Phase 1 just exposes the Usage field as-Reported by the provider.
- **OpenTelemetry instrumentation** — Phase 5 scope. Phase 1 adapters expose `WithHTTPClient` so users CAN wrap the transport with OTel later, but adapters do NOT add OTel themselves.
- **Per-model strategy table (Ollama)** — Phase 3 scope (OLL-03 tool calling). Phase 1's Ollama adapter binds a model and calls `/api/chat` regardless of model — model-specific tool-call parsing is Phase 3.

</deferred>

---

*Phase: 1-walking-skeleton-generate*
*Context gathered: 2026-05-10*
