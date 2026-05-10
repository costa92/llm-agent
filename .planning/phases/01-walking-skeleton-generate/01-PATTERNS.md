# Phase 1: Walking Skeleton — Generate (sync) only — Pattern Map

**Mapped:** 2026-05-10
**Files analyzed:** 27 new + 1 modified (across 2 repos)
**Analogs found:** 22 strong / 27 (5 are "fresh" — see §No Analog Found)

This map tells the executor *exactly* which existing-repo file to copy patterns from for each new/modified file in Phase 1. Concrete excerpts with file paths and line numbers — no abstraction. The core insight: every file the executor writes already has a structurally-identical neighbor in the codebase or in RESEARCH.md's verified Context7 sketches.

---

## File Classification

### Sister repo (`llm-agent-providers`) — net-new

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `openai/openai.go` | adapter (ChatModel impl) | request-response (HTTP) | `llm/chat_only_mock.go` (shape) + `llm/scripted.go` (full options pattern) | role-match (mock → real-HTTP) |
| `openai/options.go` | constructor / functional options | config | `llm/scripted.go` lines 176–196 (`ScriptedOption` + `WithProvider/WithModel/...`) | exact (functional-options idiom) |
| `openai/map.go` | mapper (SDK ↔ llm types) | transform | RESEARCH.md §"Code Examples" `toSDKRequest` / `fromSDKResponse` (Context7-verified) | fresh; mirror the sketch |
| `openai/errors.go` | error wrapper | transform (typed-error mapping) | `llm/errors.go` (sentinel block + comment style) + `agent.go:127-136` | role-match (sentinel → struct types, same block style) |
| `openai/doc.go` | package doc | static | `llm/doc.go` lines 1–62 | exact |
| `openai/openai_test.go` | unit test (httptest) | request-response | `llm/llm_test.go` lines 13–243 (table-driven, internal package) | exact (test conventions) |
| `openai/README.md` | adapter README | doc | `examples/README.md` (1-screen install + minimal usage) | role-match |
| `anthropic/*` (6 files) | same shape as `openai/*` | same | same as openai/ analogs above | exact (mirrors openai/) |
| `ollama/*` (6 files) | same shape | same | same as openai/ analogs above + RoundTripper sketch in RESEARCH.md Q3 | exact + fresh (RoundTripper) |
| `internal/contract/contract.go` | conformance helper (LoadFixture / NewMockServer / AssertGenerate) | request-response | RESEARCH.md §"Conformance harness shape" lines 882–1025 (verbatim sketch) | fresh; sketch is concrete |
| `internal/contract/generate_test.go` | conformance test driver (table-driven) | request-response | `llm/llm_test.go` lines 180–196 (sentinel `errors.Is` table) + RESEARCH.md sketch lines 1030–1093 | exact + sketch |
| `internal/contract/main_test.go` | TestMain (goleak) | bootstrap | RESEARCH.md sketch lines 1097–1117 | fresh; 8-line sketch |
| `internal/contract/ollama_live_test.go` | testcontainers integration test (build-tagged) | integration | RESEARCH.md sketch lines 1240–1296 | fresh; build-tagged |
| `internal/contract/testdata/{openai,anthropic,ollama}/*.json` | test fixtures | data | RESEARCH.md fixture-JSON schema lines 843–871 | fresh; capture from real APIs via §scripts |
| `scripts/capture-fixtures-openai.sh` | shell script (one-shot real-API capture) | data | `scripts/workspace.sh` lines 1–41 (style: `#!/usr/bin/env bash`, `set -euo pipefail`, comments) + RESEARCH.md sketch lines 1127–1174 | role-match (different purpose, same bash idiom) |
| `scripts/capture-fixtures-anthropic.sh` | same | same | same | same |
| `scripts/capture-fixtures-ollama.sh` | same | same | same | same |
| `.github/workflows/nightly-ollama-live.yml` | CI workflow (cron) | event-driven | `.github/workflows/test.yml` lines 1–60 + RESEARCH.md sketch lines 1190–1235 | role-match (cron-trigger vs PR-trigger; same overall shape) |
| `go.mod` (additive only) | dependency manifest | config | existing `llm-agent-providers/go.mod` (already requires `costa92/llm-agent v0.3.0-pre.1`) | exact-extension (3 SDK requires + goleak + testcontainers-go added) |

### Core repo (`llm-agent`) — modify/new

| File | Action | Role | Data Flow | Closest Analog | Match Quality |
|------|--------|------|-----------|----------------|---------------|
| `llm/errors.go` | **modify** — add 4 typed-error structs (`AuthError`, `RateLimitError`, `InvalidRequestError`, `TransientError`) | error types | typed-error transform | `agent.go:127-136` (sentinel block) + RESEARCH.md Q1-RESOLVED lines 1567–1616 (verbatim shape) | exact (extends existing sentinel-block file) |
| `llm/errors_test.go` | **new** — test `errors.As` chain + `Unwrap` round-trip for the 4 new types | unit test | request-response | `llm/llm_test.go` lines 180–196 (`TestSentinelErrors_ErrorsIs` — same shape, switch from `errors.Is` to `errors.As`) | exact |
| `PROVIDER_AUTHORING.md` | **new** | doc | static | `README.md` (top-level doc style — bilingual headings allowed) + `DEPRECATIONS.md` (table + explanatory prose format) | role-match |

**Total:** 27 net-new files in `llm-agent-providers` + 1 modify + 2 new in `llm-agent` core.

---

## Pattern Assignments

### `openai/openai.go` (adapter, request-response)

**Analog:** `llm/chat_only_mock.go` (full file, 35 lines — minimum-viable ChatModel) + `llm/scripted.go` lines 29–99 (concurrent-safe + Info() + compile-time assertion).

**Imports pattern** (mirror `llm/chat_only_mock.go:1-3` + RESEARCH.md lines 259–268):

```go
package openai

import (
    "context"
    "errors"
    "net/http"
    "os"
    "time"

    "github.com/costa92/llm-agent/llm"
    openai "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
)
```

**Compile-time interface assertion** (copy from `llm/chat_only_mock.go:18`):

```go
// Compile-time: ChatModel ONLY — Phase 1 ignores tools / embeddings / structured outputs.
var _ llm.ChatModel = (*OpenAI)(nil)
```

**Struct shape** (copy from RESEARCH.md lines 270–283; mirrors `ScriptedLLM` shape `llm/scripted.go:29-38` minus the script-cursor fields):

```go
type OpenAI struct {
    client *openai.Client
    info   llm.ProviderInfo
}
```

**Constructor body** (copy from RESEARCH.md lines 293–329 — verified Context7 SDK options).

**Generate method** (copy from RESEARCH.md lines 542–549 — verified Context7 `client.Chat.Completions.New`).

**Stream stub method** (Phase-1 placeholder; pattern from RESEARCH.md §Pattern 4 lines 417–422):

```go
// Stream is a Phase-1 stub. Streaming lands in Phase 2 (CONF-03; OAI-02).
func (o *OpenAI) Stream(_ context.Context, _ llm.Request) (llm.StreamReader, error) {
    return nil, errors.New("openai: streaming not implemented in Phase 1; use Generate")
}
```

**Info method** (copy idiom from `llm/chat_only_mock.go:28-34`):

```go
func (o *OpenAI) Info() llm.ProviderInfo { return o.info }
```

---

### `openai/options.go` (config / functional options)

**Analog:** `llm/scripted.go` lines 176–196 — exact functional-options idiom.

**Lines 176–196 of `llm/scripted.go` (the canonical shape to mirror):**

```go
// ScriptedOption configures a ScriptedLLM at construction time.
type ScriptedOption func(*ScriptedLLM)

// WithProvider sets the Provider field returned by Info().
func WithProvider(p string) ScriptedOption { return func(s *ScriptedLLM) { s.provider = p } }

// WithModel sets the Model field returned by Info().
func WithModel(m string) ScriptedOption { return func(s *ScriptedLLM) { s.model = m } }

// WithCapabilities sets the Capabilities returned by Info().
func WithCapabilities(c Capabilities) ScriptedOption { return func(s *ScriptedLLM) { s.caps = c } }

// WithResponses appends scripted Responses; Generate/Stream consume in order.
func WithResponses(rs ...Response) ScriptedOption {
    return func(s *ScriptedLLM) { s.resps = append(s.resps, rs...) }
}
```

**Translate the idiom for the adapter** (one-liner functions, single-sentence godoc, capital-letter start, period termination — these are the locked conventions). The exact set of options for OpenAI is in CONTEXT.md D-02:

- `WithModel(string) Option` (REQUIRED)
- `WithAPIKey(string) Option`
- `WithBaseURL(string) Option`
- `WithHTTPClient(*http.Client) Option`
- `WithTimeout(time.Duration) Option`
- `WithOrganization(string) Option` (OpenAI-specific)

**Reference exact shape:** RESEARCH.md lines 275–291.

---

### `openai/map.go` (transform — SDK ↔ llm types)

**Analog:** None in the codebase (mapping is provider-specific). Use RESEARCH.md sketch verbatim.

**`toSDKRequest` excerpt** (RESEARCH.md lines 551–579 — Context7-verified):

```go
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
            msgs = append(msgs, openai.SystemMessage(m.Content))
        }
    }
    p := openai.ChatCompletionNewParams{
        Model:    o.info.Model,
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
```

**`fromSDKResponse` + `mapFinishReason`:** RESEARCH.md lines 581–620.

**Critical:** the Pitfall-A comment (`req.Tools intentionally NOT mapped`) is mandatory — it documents the Phase-1 boundary and is asserted by conformance test `TestGenerate_<Provider>_ToolsFieldIgnored`.

---

### `openai/errors.go` (typed-error wrapping)

**Analog (file structure):** `llm/errors.go` lines 1–22 (single `var ()` block, godoc on the type, terminating period; bare canonical-wrap-pattern example).

**Analog (function shape):** RESEARCH.md §Pattern 2 lines 355–399 — verified Context7 `errors.As(err, &apiErr)` against `*openai.Error`.

**Function header style** (mirror `llm/errors.go:5-21` for godoc tone):

```go
// wrapErr converts an openai-go SDK error into one of llm.AuthError /
// llm.RateLimitError / llm.InvalidRequestError / llm.TransientError per the
// HTTP-status mapping documented in PROVIDER_AUTHORING.md (D-03 of Phase 1).
//
// The original SDK error is preserved in the Wrapped field; callers may
// errors.As(err, &apiErr) for provider-specific detail.
func wrapErr(err error) error { ... }
```

**Body:** copy verbatim from RESEARCH.md lines 355–399, including the OpenAI-specific `insufficient_quota` override.

---

### `openai/doc.go` (package doc)

**Analog:** `llm/doc.go` lines 1–62 — exact style.

**Lines 1–28 of `llm/doc.go` (the model — heredoc/list-style overview):**

```go
// Package llm owns the capability-aware LLM-provider contract for the
// agents framework.
//
// The contract is intentionally narrow — only the types an Agent or
// Tool implementation needs to call a model:
//
//   - ChatModel          base interface (Generate + Stream + Info)
//   ...
package llm
```

**Translate for OpenAI adapter:** package overview + capability statement (`Capabilities.Tools = false` in Phase 1) + 1-line example referring to `New(WithModel(...))` constructor.

---

### `openai/openai_test.go` (httptest unit test)

**Analog:** `llm/llm_test.go` lines 13–243 (entire file). Internal-package convention (`package openai`, NOT `package openai_test`), table-driven, `testing.T.Run` for sub-cases.

**Compile-time-assertion pattern** (copy idiom from `llm/llm_test.go:13-18`):

```go
var (
    _ llm.ChatModel = (*OpenAI)(nil)
)
```

**Table-driven test pattern** (lines 180–196 of `llm/llm_test.go`):

```go
func TestSentinelErrors_ErrorsIs(t *testing.T) {
    cases := []struct {
        name string
        s    error
    }{
        {"ErrCapabilityNotSupported", ErrCapabilityNotSupported},
        {"ErrScriptExhausted", ErrScriptExhausted},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            wrapped := fmt.Errorf("wrap: %w", c.s)
            if !errors.Is(wrapped, c.s) {
                t.Errorf("errors.Is(wrapped, %s) = false, want true", c.name)
            }
        })
    }
}
```

**Test naming** (per CONTEXT.md `Claude's Discretion`):
- `TestGenerate_OpenAI_Happy`
- `TestGenerate_OpenAI_401`
- `TestGenerate_OpenAI_429`
- `TestGenerate_OpenAI_429_QuotaExhausted`
- `TestGenerate_OpenAI_5xx`
- `TestGenerate_OpenAI_4xxOther`
- `TestGenerate_OpenAI_SystemPrompt`
- `TestGenerate_OpenAI_ToolsFieldIgnored`
- `TestStream_Phase1NotImplemented`

**httptest server setup pattern:** RESEARCH.md lines 941–962 (`NewMockServer` helper). Per-package tests can call `httptest.NewServer` directly without going through `internal/contract` — that's the conformance suite's job.

---

### `anthropic/*` files

**Pattern source:** mirror `openai/*` exactly. The only divergences:

| File | Anthropic divergence from OpenAI pattern |
|------|------------------------------------------|
| `anthropic.go` | RESEARCH.md lines 637–717 (Generate body); `client.Messages.New(ctx, sdkReq)` instead of `client.Chat.Completions.New` |
| `options.go` | drop `WithOrganization`; add `WithBetaHeader(string) Option` (per CONTEXT.md D-02 per-provider extras) |
| `map.go` | **CRITICAL Pitfall C:** `Request.SystemPrompt` MUST lift to top-level `MessageNewParams.System []TextBlockParam`, NOT into messages. RESEARCH.md lines 646–681 has the exact lift logic |
| `errors.go` | Q2 RESOLVED: type-assert `*apierror.Error` if v1.41.0 re-exports it; fallback is RoundTripper-based status capture per Q3 (same pattern as Ollama). Additional override: 529 `overloaded_error` → `RateLimitError` (NOT TransientError); 400 `invalid_request_error` → `InvalidRequestError` |
| `anthropic_test.go` | mirror openai_test.go scenarios; add `TestGenerate_Anthropic_SystemTopLevel` (asserts wire body has `system` at top level, NOT in `messages`) |

---

### `ollama/*` files

**Pattern source:** mirror `openai/*` with these divergences:

| File | Ollama divergence |
|------|-------------------|
| `ollama.go` | RESEARCH.md lines 737–793 (Generate body uses `client.Chat(ctx, req, callback)` invoked once with `Stream: &false`) |
| `options.go` | drop `WithAPIKey` (Ollama is keyless) and `WithOrganization`; add `WithHost(string) Option` (alias for `WithBaseURL`); env-var fallback reads `OLLAMA_HOST`. Plus the **RoundTripper status-capturing transport** from RESEARCH.md Q3 lines 1642–1669 — this is required to recover HTTP status codes for typed-error mapping |
| `map.go` | RESEARCH.md lines 751–793; Ollama accepts `role: "system"` in messages (unlike Anthropic) |
| `errors.go` | Reads HTTP status from the captured-status `*int32` (atomic load) — not from `errors.As`. Override: 404 with `model not found` body → `*llm.InvalidRequestError` (not transient — operator must `ollama pull <model>`). RESEARCH.md Q3 lines 1660–1673 |
| `ollama_test.go` | scenarios: `TestGenerate_Ollama_Happy`, `TestGenerate_Ollama_404ModelNotPulled`, `TestGenerate_Ollama_NoDaemon` (assert `*llm.TransientError` when the test server is closed before the call) |

**Ollama-specific helper** (newOllamaClient — RESEARCH.md lines 796–808):

```go
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

### `internal/contract/contract.go` (conformance helpers)

**Analog:** None in repo (this is a fresh pkg). RESEARCH.md lines 882–1025 has the verbatim sketch.

**Imports** (RESEARCH.md lines 885–895):

```go
package contract

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/costa92/llm-agent/llm"
)
```

**Fixture struct** (RESEARCH.md lines 898–920) — JSON-tagged like the locked `llm/types.go` convention.

**LoadFixture** (RESEARCH.md lines 922–935): `t.Helper()`; `os.ReadFile(filepath.Join("testdata", provider, scenario+".json"))`; fail with `t.Fatalf`.

**NewMockServer** (RESEARCH.md lines 937–962): `httptest.NewServer(http.HandlerFunc(...))`; method/path assertion + body substring assertion + reply with fixture status/headers/body.

**AssertGenerate** (RESEARCH.md lines 977–1020): switches on `f.Expect.ErrorType` against `errors.As` for the 4 typed errors. **CRITICAL:** uses `t.Context()` (Go 1.26 feature; verified available because core repo is `go 1.26.0` — see `llm-agent/go.mod:3`).

**ChatModelFactory type** (RESEARCH.md lines 972–975): `type ChatModelFactory func(baseURL string) (llm.ChatModel, error)`.

---

### `internal/contract/generate_test.go` (conformance driver)

**Analog (table-driven shape):** `llm/llm_test.go:180-196` (the `TestSentinelErrors_ErrorsIs` table); RESEARCH.md lines 1030–1093 has the verbatim sketch.

**Adapter-factory map** (RESEARCH.md lines 1045–1055):

```go
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
```

**Test driver** (RESEARCH.md lines 1059–1092): table of `{provider, scenario}` pairs, `t.Run(provider+"/"+scenario, ...)`, `t.Parallel()`, `defer srv.Close()`.

**Per-requirement test count:** 13 cases listed in RESEARCH.md lines 1060–1077 covering OAI-01/05, ANT-01/05, OLL-01/05.

---

### `internal/contract/main_test.go` (TestMain — goleak)

**Analog:** None in core repo (the core uses stdlib-only `testing` without goleak). RESEARCH.md lines 1097–1117 has the verbatim sketch.

**Per Q8 RESOLVED (RESEARCH.md lines 1725–1732):** ship initially WITHOUT `IgnoreTopFunction`. Add `goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop")` only if PR CI fires false-positives.

```go
package contract

import (
    "testing"

    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

---

### `internal/contract/ollama_live_test.go` (build-tagged integration test)

**Analog:** None in repo. RESEARCH.md lines 1240–1296 has the verbatim sketch.

**Build tag** (line 1 — required to keep this test out of PR CI):

```go
//go:build ollama_live
```

**Test body uses Q5-RESOLVED API:** `tcollama.Run(ctx, image)` + `container.Exec(ctx, []string{"ollama", "pull", model})` + `container.ConnectionString(ctx)`.

---

### `internal/contract/testdata/<provider>/<scenario>.json` (test fixtures)

**Analog:** None — fresh data files. RESEARCH.md fixture-JSON schema lines 843–871 + per-provider response shapes lines 875–877.

**Schema (per fixture):** `scenario`, `request{method, path, body_assertions[]}`, `response{status, headers, body}`, `expect{error_type, response_text, finish_reason, usage_input_tokens, usage_output_tokens, usage_source, provider}`.

**Per-provider real-API body shapes** (RESEARCH.md lines 875–877) — use these as the verbatim response body when capturing:
- OpenAI happy: `{id, object, created, model, choices[0].{index, message:{role, content}, finish_reason}, usage:{prompt_tokens, completion_tokens, total_tokens}}` ~500 bytes
- Anthropic happy: `{id, type:"message", role:"assistant", content:[{type:"text", text:"..."}], model, stop_reason:"end_turn", stop_sequence:null, usage:{input_tokens, output_tokens}}` ~400 bytes
- Ollama happy: `{model, created_at, message:{role:"assistant", content:"..."}, done:true, done_reason:"stop", total_duration, ..., prompt_eval_count, eval_count}` ~600 bytes

**Capture mechanism:** the per-provider scripts (next §).

---

### `scripts/capture-fixtures-{openai,anthropic,ollama}.sh` (real-API capture)

**Analog (bash style):** `scripts/workspace.sh` lines 1–41.

**Excerpt from `scripts/workspace.sh:1-12` (the canonical bash header style for this repo family):**

```bash
#!/usr/bin/env bash
# scripts/workspace.sh — write a sibling-aware go.work above this repo.
#
# Usage: run from any of the 4 sibling clones:
#   <parent>/llm-agent
#   <parent>/llm-agent-providers
#   <parent>/llm-agent-otel
#   <parent>/llm-agent-customer-support
#
# Result: <parent>/go.work points at all 4 modules. The file is gitignored
# in every repo (Pitfall 13). Idempotent — safe to re-run.
set -euo pipefail
```

**Mandatory bash conventions** (from `scripts/workspace.sh`):
1. Shebang `#!/usr/bin/env bash` (NOT `/bin/bash`).
2. Header comment block — purpose, usage, result.
3. `set -euo pipefail` first executable line.
4. Idempotency note in header (these capture scripts are idempotent — re-running overwrites the fixture).

**Capture-script body:** RESEARCH.md lines 1127–1174 has the verbatim OpenAI sketch. Anthropic/Ollama variants per RESEARCH.md line 1176–1178:
- Anthropic: `Authorization: x-api-key: $ANTHROPIC_API_KEY` + `anthropic-version: 2023-06-01` header; path `/v1/messages`.
- Ollama: no API key; runs against `http://localhost:11434`.

**Pitfall F mitigation** (RESEARCH.md lines 511–520):
- `: "${OPENAI_API_KEY:?must be set; never commit this key}"` — fail fast on unset.
- Output JSON contains response body + request shape — never API keys, never tokens.
- `.gitignore` blocks `testdata/**/*.local.json` and `**/.env`.

---

### `.github/workflows/nightly-ollama-live.yml` (cron-trigger CI)

**Analog:** `.github/workflows/test.yml` (entire file, 60 lines) — exact YAML structure to mirror.

**Excerpt from `.github/workflows/test.yml:1-25` (header conventions to copy):**

```yaml
name: test

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

concurrency:
  group: test-${{ github.ref }}
  cancel-in-progress: true

env:
  GOWORK: off  # INFRA-02: CI never picks up a workspace file silently

jobs:
  go:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
```

**Conventions to mirror:**
- `GOWORK: off` env block (INFRA-02 hard rule per CLAUDE.md #4).
- `actions/checkout@v4` + `actions/setup-go@v5` with `go-version-file: go.mod`.
- `concurrency` group for cancel-in-progress on re-pushes.
- `timeout-minutes` explicit per job.

**Diverge from test.yml** for:
- Trigger: `schedule: - cron: '0 3 * * *'` + `workflow_dispatch` instead of `push`/`pull_request`.
- `timeout-minutes: 45` instead of 10 (Pitfall D — cold pull is 3–5 min).
- Add `actions/cache@v4` step keyed on Ollama image+model (Pitfall D mitigation; RESEARCH.md lines 1218–1225).
- Add Docker availability check before testcontainers run.
- Run command: `go test -v -timeout 30m ./internal/contract/... -run TestGenerate_Ollama_Live -tags ollama_live`.

**Verbatim sketch:** RESEARCH.md lines 1190–1235.

---

### `go.mod` extensions (sister repo `llm-agent-providers/go.mod`)

**Analog (minimal-extension pattern):** existing `llm-agent-providers/go.mod` already has `require github.com/costa92/llm-agent v0.3.0-pre.1` (per CONTEXT.md "Phase 0 outputs" line 154; RESEARCH.md migration notes line 1428).

**Action (Wave-0):** run `go get` for these adds (RESEARCH.md lines 145–151):

```bash
go get github.com/openai/openai-go/v3@v3.35.0
go get github.com/anthropics/anthropic-sdk-go@v1.41.0
go get github.com/ollama/ollama/api@v0.23.2
go get -t github.com/testcontainers/testcontainers-go
go get -t github.com/testcontainers/testcontainers-go/modules/ollama
go get -t go.uber.org/goleak
```

**`-t` flag for test-only deps:** testcontainers-go and goleak only used in `_test.go` files. The flag is significant — it places them in `// indirect` lines or test-block requires depending on go-mod-tidy outcome.

**`go.sum` will appear:** the sister repo will gain a `go.sum` after first `go mod tidy`. This is fine — sister repos may take deps (CLAUDE.md hard rule #1 applies to **core** only).

---

### `llm/errors.go` (core repo modify — Wave 0)

**Analog:** `llm/errors.go` lines 1–22 (entire current file) — sentinel-block style. The file is being EXTENDED, not rewritten.

**Existing file structure** (ALL OF IT — 22 lines):

```go
package llm

import "errors"

// Sentinel errors for the llm package. Callers detect via errors.Is.
// Both sentinels MUST survive `fmt.Errorf("...: %w", sentinel)` wrapping.
var (
    // ErrCapabilityNotSupported is returned by methods on capability
    // interfaces when the bound model does not actually support the
    // capability — even though the Go type implements the interface.
    //
    // Canonical wrap pattern:
    //   return nil, fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported)
    //
    // Callers detect with errors.Is(err, llm.ErrCapabilityNotSupported).
    ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")

    // ErrScriptExhausted is returned by ScriptedLLM when the script runs
    // out of pre-recorded responses. Test code matches with errors.Is.
    ErrScriptExhausted = errors.New("llm: scripted llm: script exhausted")
)
```

**Append** (RESEARCH.md Q1-RESOLVED lines 1567–1616 — verbatim 4 typed-error structs):

- `AuthError{Provider, Wrapped}` + `Error()` + `Unwrap()`
- `RateLimitError{Provider, RetryAfter, Reason, Wrapped}` + `Error()` + `Unwrap()`
- `InvalidRequestError{Provider, Wrapped}` + `Error()` + `Unwrap()`
- `TransientError{Provider, Wrapped}` + `Error()` + `Unwrap()`

**Add import:** `"fmt"` (for `fmt.Sprintf` in the four `Error()` methods).

**Godoc style:** mirror the existing `ErrCapabilityNotSupported` block — capital-letter start, period termination, optional canonical-usage example, callers-detect-with note.

**Critical:** the new types MUST satisfy the Go errors-chain interface — `Unwrap()` returns the wrapped SDK error so consumers can `errors.As(err, &openaiErr)` for provider detail.

---

### `llm/errors_test.go` (core repo new — Wave 0)

**Analog:** `llm/llm_test.go` lines 180–196 (`TestSentinelErrors_ErrorsIs`).

**Translate idiom from `errors.Is` → `errors.As`** for the 4 new struct types. Plus a round-trip test that confirms `Unwrap()` traversal works (`errors.As` on a doubly-wrapped error — `fmt.Errorf("outer: %w", &llm.AuthError{Wrapped: someInner})`).

**Test naming convention** (mirroring `TestSentinelErrors_ErrorsIs`):
- `TestTypedErrors_ErrorsAs` (the table-driven version)
- `TestTypedErrors_UnwrapChain` (asserts `errors.Unwrap(authErr) == innerErr`)

---

### `PROVIDER_AUTHORING.md` (core repo new — Phase 1 doc)

**Analog (overall doc tone):** `README.md` lines 1–80 (bilingual heading style is OK; bullet-list package overview).

**Analog (table format):** `DEPRECATIONS.md` lines 14–22 (markdown table; first column Symbol; subsequent columns Action / Migration etc.).

**Excerpt from `DEPRECATIONS.md:14-22` (the canonical table style to copy):**

```markdown
## Active deprecations

| Symbol | Deprecated In | Removed In | Migration |
|---|---|---|---|
| `llm.Client` (interface) | v0.3.0 | v0.4.0 | Use `llm.ChatModel`. See [migration guide](docs/migration-v0.2-to-v0.3.md). The `type Client = LegacyClient` alias preserves source compatibility through v0.3.x. |
```

**Section structure** (RESEARCH.md lines 1305–1357, 8 sections):
1. Audience and Scope
2. The Contract (cross-link to `llm/chatmodel.go`, `llm/types.go`, `llm/info.go`, `llm/errors.go`)
3. The Generate Contract
4. Constructor Pattern (D-02 canonical — exact functional-options shape)
5. Error Taxonomy (D-03 canonical mapping table — VERBATIM the table from CONTEXT.md lines 53–63)
6. Conformance Test Pattern (D-04 canonical)
7. Phase 1 Boundary (what we DON'T do)
8. Cross-references

**Length goal:** 150–250 lines (RESEARCH.md line 1303).

**Critical sections to include verbatim:**
- The HTTP-status → typed-error mapping table from CONTEXT.md lines 53–63.
- The OpenAI `wrapErr` snippet from RESEARCH.md §Pattern 2 lines 355–399 as the canonical example.
- Phase 1 Boundary explicit list: "no streaming, no tools, no embeddings, no three-state cost record, no retry SM, no OTel."

---

## Shared Patterns

### 1. Sentinel-error / typed-error block style

**Source:** `llm/errors.go:1-22` + `agent.go:127-136`.

**Apply to:** `llm/errors.go` (new typed-error structs), `openai/errors.go`, `anthropic/errors.go`, `ollama/errors.go`.

**Concrete pattern excerpt** (`agent.go:127-136`):

```go
// Sentinel errors. Subpackage stays portable — does not import pkg/errors.
// Callers in internal/* translate via errors.Is at the boundary.
var (
    ErrMaxStepsExceeded      = errors.New("agents: max steps exceeded")
    ErrToolNotFound          = errors.New("agents: tool not found")
    ErrToolAlreadyRegistered = errors.New("agents: tool already registered")
    ErrPlanningFailed        = errors.New("agents: planning failed")
    ErrParseToolCall         = errors.New("agents: failed to parse tool call")
    ErrEmptyInput            = errors.New("agents: empty input")
)
```

**Conventions:**
- Single `var (...)` block when multiple sentinels share the same package.
- Error string format: `<package>: <description>` — lowercase prefix, colon, lowercase description.
- Each sentinel gets a godoc with `// ErrXxx is returned by ...` shape.
- Companion test `TestSentinelErrors_ErrorsIs` uses `fmt.Errorf("wrap: %w", sentinel)` round-trip — pattern in `llm/llm_test.go:180-196` and `agent_test.go:18-33`.

### 2. Functional-options constructor pattern (D-02)

**Source:** `llm/scripted.go:53-64` (NewScriptedLLM) + `llm/scripted.go:176-196` (option functions).

**Apply to:** `openai/options.go`, `anthropic/options.go`, `ollama/options.go`.

**Pattern excerpt** (`llm/scripted.go:53-64`):

```go
func NewScriptedLLM(opts ...ScriptedOption) *ScriptedLLM {
    s := &ScriptedLLM{
        provider: "scripted",
        model:    "test",
        caps:     Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true, PromptCaching: false},
        embedDim: 4,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

**Phase 1 adapter divergence:** `New(opts ...Option) (*X, error)` instead of `(*X)` — adapters MUST validate `WithModel` is set and return error if not (CONTEXT.md D-02 line 47).

### 3. Compile-time interface assertion at file scope

**Source:** `llm/chat_only_mock.go:18`, `llm/scripted.go:43-48`, `agent_test.go:9-16`.

**Apply to:** `openai/openai.go`, `anthropic/anthropic.go`, `ollama/ollama.go` (each adapter as `var _ llm.ChatModel = (*X)(nil)`).

**Pattern excerpt** (`llm/scripted.go:43-48`):

```go
var (
    _ ChatModel         = (*ScriptedLLM)(nil)
    _ ToolCaller        = (*ScriptedLLM)(nil)
    _ Embedder          = (*ScriptedLLM)(nil)
    _ StructuredOutputs = (*ScriptedLLM)(nil)
)
```

**Phase 1:** adapters assert `_ llm.ChatModel = (*OpenAI)(nil)` ONLY — they do NOT assert `ToolCaller`/`Embedder`/`StructuredOutputs` (those interfaces are honored from Phase 3+).

### 4. Internal-package test convention (`package <name>`, NOT `<name>_test`)

**Source:** `llm/llm_test.go:1` (`package llm`), `agent_test.go:1` (`package agents`).

**Apply to:** ALL new `_test.go` files in this phase — `openai/openai_test.go`, `anthropic/anthropic_test.go`, `ollama/ollama_test.go`, `internal/contract/generate_test.go`, `internal/contract/main_test.go`, `llm/errors_test.go`.

**Why:** the codebase deliberately uses internal-package tests so that test code can read unexported symbols (e.g., `wrapErr`, `toSDKRequest`) without exporting them.

### 5. JSON tags on every public struct field

**Source:** `llm/types.go:13-19` (Request), `llm/types.go:23-30` (Response), `llm/info.go:8-12` (ProviderInfo).

**Apply to:**
- `internal/contract/contract.go` Fixture struct (RESEARCH.md lines 898–920).
- `llm/errors.go` new typed-error structs (no JSON tags needed since errors aren't typically JSON-serialized — but the PATTERN is "if it's a struct exposed in the API, add tags" — exempt errors).

**Pattern excerpt** (`llm/types.go:13-19`):

```go
type Request struct {
    Messages        []Message      `json:"messages"`
    SystemPrompt    string         `json:"system_prompt,omitempty"`
    MaxOutputTokens int            `json:"max_output_tokens,omitempty"`
    Temperature     *float32       `json:"temperature,omitempty"`
    Metadata        map[string]any `json:"metadata,omitempty"`
}
```

**Convention:** snake_case for JSON tag values; `omitempty` for fields that legitimately can be empty.

### 6. Godoc style — capital first letter, period terminator, `// Deprecated:` keyword

**Source:** `llm/errors.go:5-21` + `llm/legacy.go:1-7` + `llm/scripted.go:10-48`.

**Apply to:** ALL exported symbols in new code.

**Conventions** (extracted from `llm/legacy.go:5-7`):

```go
// LegacyClient is the v0.2 LLM contract — superseded by ChatModel.
//
// Deprecated: Use llm.ChatModel instead. LegacyClient will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
```

- First sentence is a complete sentence starting with the symbol name.
- Em-dash `—` (U+2014) is the in-house punctuation for parenthetical clauses.
- `// Deprecated:` keyword on its own paragraph (gopls/staticcheck hook).
- Multiple paragraphs separated by `//` (blank-line comment).

### 7. CI workflow conventions (sister repos)

**Source:** `.github/workflows/test.yml:1-25` (header) + `.github/workflows/test.yml:13-15` (env) + `.github/workflows/test.yml:21-25` (setup-go).

**Apply to:** `llm-agent-providers/.github/workflows/nightly-ollama-live.yml` (new).

**Conventions:**
- `env: GOWORK: off` block (CLAUDE.md hard rule #4 enforcement).
- `concurrency` group for ref-keyed cancellation.
- `actions/checkout@v4` (NOT v3 or older).
- `actions/setup-go@v5` with `go-version-file: go.mod` (NOT hardcoded version).
- `cache: true` on setup-go.
- Explicit `timeout-minutes` per job.

### 8. Stdlib-only constraint (CORE only)

**Source:** CLAUDE.md hard rule #1; `go.mod:1-3` (core has no requires beyond Go version).

**Apply to:** `llm/errors.go` (core repo) — must NOT add any non-stdlib import.

**EXEMPT:** all sister-repo files (`llm-agent-providers/...`) — sister repos may take deps. Adding `openai-go/v3` / `anthropic-sdk-go` / `ollama/api` / `goleak` / `testcontainers-go` is correct because they live in `llm-agent-providers/go.mod`, not core.

### 9. "Phase 1 boundary" comment marker

**Apply to:** every place in the adapter that intentionally drops Phase 2/3/4 functionality.

**Excerpt to include verbatim** (from RESEARCH.md line 577):

```go
// NOTE: req.Tools intentionally NOT mapped — Phase 1 ignores tools (Pitfall A).
```

**Variations:**
- `// Stream is a Phase-1 stub. Streaming lands in Phase 2 (CONF-03; OAI-02).`
- `// NOTE: req.Tools intentionally NOT mapped — Phase 1 ignores tools (Pitfall A).`
- `// Phase 1: extract only text blocks; tool_use blocks ignored (Pitfall A).`

These markers are search-targets when Phase 2/3 lands — `git grep "Phase 1"` finds every place that needs revisiting.

---

## No Analog Found

Files with no close match in the existing codebase (executor uses RESEARCH.md sketches and Context7-verified examples instead):

| File | Role | Data Flow | Reason | RESEARCH.md Reference |
|------|------|-----------|--------|------------------------|
| `internal/contract/contract.go` | conformance helper | request-response | First test-harness package in the umbrella; httptest+testdata pattern is fresh | Lines 882–1025 |
| `internal/contract/main_test.go` | TestMain (goleak) | bootstrap | `goleak` not used in core repo (stdlib-only); first integration here | Lines 1097–1117 |
| `internal/contract/ollama_live_test.go` | testcontainers integration | integration | `testcontainers-go` not used in core; first integration here | Lines 1240–1296 |
| `scripts/capture-fixtures-*.sh` | bash one-shot | data | Closest existing script is `scripts/workspace.sh` but it does completely different work; bash *style* matches, content does not | Lines 1127–1178 |
| `internal/contract/testdata/*/*.json` | test fixtures | data | No existing testdata in repo; format is fresh per D-04 | Lines 843–877 |
| RoundTripper status-capture pattern (within `ollama/options.go`) | infrastructure | request-response | First use of a custom transport in the umbrella | Lines 1642–1669 (Q3 RESOLVED) |

For each, RESEARCH.md provides a complete, Context7-verified or web-search-verified sketch — the executor copies the sketch as-is, then adapts to the locked patterns (sentinel-block error style, functional-options idiom, internal-package tests, godoc conventions).

---

## Critical Pitfalls Summary (Must-Reads for Executor)

The executor MUST keep these pitfalls visible while writing each file:

| Pitfall | Affects Files | Mitigation Reference |
|---------|---------------|----------------------|
| A — Honoring `Request.Tools` in Phase 1 | All `map.go` + all `Info()` returns | RESEARCH.md lines 443–456; `Capabilities.Tools = false`; do NOT pass Tools to SDK |
| B — Wrong SDK error type for `errors.As` | All `errors.go` | RESEARCH.md lines 458–468; verify Anthropic via `go doc -all` (Q2); use RoundTripper for Ollama (Q3) |
| C — Anthropic system message in wrong place | `anthropic/map.go` | RESEARCH.md lines 471–481; lift `SystemPrompt` to top-level `System []TextBlockParam`, NEVER to `messages[0]` |
| D — testcontainers cold-pull timeout | `nightly-ollama-live.yml` | RESEARCH.md lines 484–495; `actions/cache@v4` + `timeout-minutes: 45` |
| E — goleak false-positives from httptest keep-alives | `main_test.go` | RESEARCH.md lines 498–508; ship without ignore initially; add `IgnoreTopFunction("net/http.(*persistConn).readLoop")` only if PR CI fires |
| F — Capture script committing real API key | `scripts/capture-fixtures-*.sh` | RESEARCH.md lines 511–520; `set -u`, no env-file fallback, `.gitignore` patterns |

---

## Metadata

**Analog search scope:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/` (core repo: `llm/`, `agents.go`/`agent.go`, top-level files, `.github/workflows/`, `scripts/`, top-level Markdown).
**Files scanned:** 14 Go files in `llm/`, 1 workflow YAML, 1 bash script, 4 top-level markdown.
**Pattern extraction date:** 2026-05-10
**Sister-repo files:** none read directly (the sister repo lives outside the search scope at `/tmp/llm-agent-providers/`); patterns sourced from CONTEXT.md / RESEARCH.md citations of those files.
**Quality:** every Phase 1 file is mapped to either (a) an existing-repo analog with line numbers, or (b) a Context7-verified sketch in RESEARCH.md with line numbers. Zero abstraction — executor receives concrete bytes to mirror.
