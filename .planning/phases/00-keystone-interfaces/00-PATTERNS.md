# Phase 0: Multi-repo infra + keystone interfaces - Pattern Map

**Mapped:** 2026-05-10
**Files analyzed:** 24 (15 new, 9 modified, +3 sister-repo skeletons)
**Analogs found:** 22 / 24 (2 fresh — sister repos with no in-repo precedent)

> Reading order: this PATTERNS.md is consumed by `gsd-planner`. Each plan in Phase 0 (00-01..00-05 per RESEARCH.md "Primary recommendation") references the relevant sub-section here for concrete excerpts. **All path/line citations are absolute paths**; planner / executor copy patterns verbatim.

---

## File Classification

### Plan 00-01 — Core `llm/` reboot (interfaces + types + ScriptedLLM v2 + ChatOnlyMock + LegacyClient)

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `llm/chatmodel.go` | interface (contract) | request-response | `llm/client.go` (lines 14-17) | exact (same role, same flow — interface in same package) |
| `llm/capabilities.go` | interface (contract) | request-response + transform | `llm/client.go` (`Client` interface) | role-match (interface set vs single interface) |
| `llm/stream.go` | interface + types | streaming | `llm/client.go` (lines 65-79: `StreamChunk`, `StreamUsage`) | role-match (channel-based vs iterator) |
| `llm/info.go` | type (data) | data | `llm/client.go` (lines 39-50: `GenerateResponse`) | role-match (struct with `json:` tags) |
| `llm/types.go` | type (data) | data | `llm/client.go` (entire file) | exact (existing patterns ported) |
| `llm/errors.go` | sentinel errors | data | `agent.go` (lines 129-136: `ErrMaxStepsExceeded` block) | exact (sentinel-error idiom) |
| `llm/scripted.go` | mock implementation | request-response | `scriptedllm_test.go` + `examples/scriptedllm/scriptedllm.go` | exact (promoted from `_test.go`) |
| `llm/chat_only_mock.go` | mock implementation | request-response | `scriptedllm_test.go` (sync.Mutex + cursor) | role-match (subset) |
| `llm/doc.go` | package doc | docs | `llm/doc.go` (existing) | exact (same file, replaced) |
| `llm/llm_test.go` | test | type-level + unit | `agent_test.go` (lines 9-16: `var _ Agent = (*…)(nil)` block) | exact (interface-satisfaction test pattern) |
| `llm/legacy.go` | interface + types (deprecated) | request-response + streaming | `llm/client.go` (verbatim move) | exact (file rename) |

### Plan 00-02 — Migration guide + DEPRECATIONS + CHANGELOG

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `docs/migration-v0.2-to-v0.3.md` | docs (markdown) | n/a | `CHANGELOG.md` (lines 71-83: v0.2.0 Migration block) + `README.md` (table-of-packages) | role-match (markdown migration prose) |
| `DEPRECATIONS.md` | docs (markdown) | n/a | `CHANGELOG.md` Deprecated-section pattern | role-match (no in-repo analog; new file type) |
| `CHANGELOG.md` (modified) | docs (markdown) | n/a | itself (lines 12-83 = v0.1.0 + v0.2.0 entries) | exact (extend existing format) |

### Plan 00-03 — Sister repo skeletons (3 fresh repos)

> **NOTE: Sister repos have no in-repo analog as repos.** What they DO have are file-by-file analogs *inside this repo* (LICENSE, OWNERS, .gitignore, .github/workflows/test.yml, README.md). The planner clones these per-file shapes into each sister-repo's initial commit.

| Sister Repo | New File | Role | Closest Analog | Match Quality |
|-------------|----------|------|----------------|---------------|
| (3 repos) × | `LICENSE` | legal | `LICENSE` (this repo, verbatim copy) | exact |
| (3 repos) × | `OWNERS` | meta | `OWNERS` (this repo, line 14 `labels:` substituted per-repo) | exact |
| (3 repos) × | `.gitignore` | config | `.gitignore` (this repo, +`go.work`/`go.work.sum` added) | exact |
| (3 repos) × | `README.md` | docs | `README.md` (this repo) — doc-style template | role-match (sister repos focus on cross-repo iteration; this repo focuses on package taxonomy) |
| (3 repos) × | `go.mod` | config | `go.mod` (this repo, lines 1-3) | exact (same module declaration shape; sister repos add `require` line) |
| (3 repos) × | `.github/workflows/test.yml` | CI | `.github/workflows/test.yml` (this repo) | exact (same shape; sister repos add `env: GOWORK: off`) |
| (3 repos) × | `.github/workflows/release-precheck.yml` | CI | none in repo (NEW) — copied verbatim from RESEARCH.md §"CI YAML Sketches" | fresh |
| (3 repos) × | `scripts/workspace.sh` | shell script | none in repo (NEW) — copied verbatim from STACK.md per RESEARCH.md | fresh |

### Plan 00-04 — Core `.gitignore` + workspace script + per-repo CI hardening

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.gitignore` (modified) | config | n/a | itself (lines 1-19) | exact (append section) |
| `scripts/workspace.sh` | shell | n/a | none in repo (NEW); STACK.md spec | fresh |
| `.github/workflows/test.yml` (modified) | CI | n/a | itself | exact (add `env: GOWORK: off`) |

### Plan 00-05 — Umbrella CI + release-precheck (core repo)

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `.github/workflows/umbrella.yml` | CI | event-driven (PR) | `.github/workflows/test.yml` | role-match (multi-repo checkout/build vs single-repo) |
| `.github/workflows/release-precheck.yml` | CI | event-driven (release-branch push) | `.github/workflows/test.yml` | role-match (different `on:` trigger + script body) |

---

## Pattern Assignments

### `llm/chatmodel.go` (interface, request-response)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go`

**Package + import pattern** (lines 1-10):
```go
// Package llm owns the LLM-provider contract for the agents framework.
// It is intentionally narrow: only the types an Agent needs to call a
// model. ...
package llm

import (
	"context"
	"encoding/json"
)
```
- Package comment is in `doc.go`, not `client.go` — so `chatmodel.go` opens straight with `package llm` (no leading comment block).
- Stdlib-only imports. `context` is the only import for `chatmodel.go`.

**Interface declaration pattern** (lines 12-17):
```go
// Client is the portable seam between business code and LLM providers.
// Generate is one-shot; GenerateStream streams tokens over <-chan StreamChunk.
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}
```
**Conventions to copy:**
- 2-line godoc above the interface — first line "X is …" (Go style), second line one-sentence purpose.
- Method signature: `(ctx context.Context, req X)` first; return `(Y, error)` last.
- Single interface per file when it's the file's headline; supporting types in same file or sibling files.

For new `ChatModel`:
```go
// ChatModel is the base contract every provider implements.
// Smallest possible interface: Generate, Stream, Info.
//
// Capabilities beyond text generation are expressed as separate
// interfaces (ToolCaller, Embedder, StructuredOutputs); callers
// type-assert to detect. ProviderInfo.Capabilities is the runtime
// signal for per-(provider × model) variation that type assertion
// cannot see — see doc.go for the canonical negotiation pattern.
type ChatModel interface {
    Generate(ctx context.Context, req Request) (Response, error)
    Stream(ctx context.Context, req Request) (StreamReader, error)
    Info() ProviderInfo
}
```

---

### `llm/capabilities.go` (interface set)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (interface declaration shape) + `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/tool.go` (lines 16-21 — multi-method interface with godoc)

**Interface-with-doc pattern** (`tool.go` lines 10-21):
```go
// Tool is a capability unit an Agent may invoke.
//
// Description is shown to the LLM (it decides whether to call); Schema describes
// the parameters as raw JSON Schema (we don't validate it — upstream provider does);
// Execute does the work and returns a string suitable for either prompt-injection
// (ReActAgent's Observation) or aggregation (FunctionCallAgent's answer).
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}
```
**Conventions to copy:**
- Multi-paragraph godoc separated by blank-comment-line (`//\n//`) for interfaces with multiple methods that warrant per-method explanation.
- Use `json.RawMessage` for raw-JSON pass-through (existing pattern, line 19 + line 20).

For `ToolCaller`, `Embedder`, `StructuredOutputs`: see RESEARCH.md §"`llm/capabilities.go`" for exact shape. The pattern to copy from `tool.go` is the multi-paragraph godoc + the `(ctx context.Context, …)` signature ordering.

---

### `llm/stream.go` (interface + struct + enum)

**Analogs:**
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (lines 52-79: `FinishReason` const-block + `StreamChunk` + `StreamUsage`)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent.go` (lines 67-76: `StepKind` enum)

**String-typed enum pattern** (`client.go` lines 52-63):
```go
// FinishReason mirrors the OpenAI /v1/chat/completions stop_reason field so
// that providers that surface this can pass it through without conversion.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonUnknown       FinishReason = "unknown"
)
```

**Uint-typed enum pattern with `iota`** (`agent.go` lines 67-76):
```go
// StepKind enumerates trace step types.
type StepKind string

const (
	StepThought     StepKind = "thought"
	StepAction      StepKind = "action"
	...
)
```
**Convention:** Existing repo prefers **string-typed** enums (better godoc + JSON readability). RESEARCH.md proposes `StreamEventKind uint8` with `iota`. Either is acceptable; **prefer string-typed** to match existing repo conventions unless the planner has a specific size/performance argument. Document the choice.

**Struct with `json:` tags** (`client.go` lines 65-79):
```go
type StreamChunk struct {
	Text  string       `json:"text"`
	Done  bool         `json:"done"`
	Usage *StreamUsage `json:"usage,omitempty"`
	// ToolCall is set on stream chunks when the model emits a function
	// call delta. Done==true with ToolCall set marks the end of a tool
	// invocation; subsequent calls are a new chunk sequence.
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}
```
**Conventions to copy:**
- Every public type field carries `json:"…"`. Pointer-typed fields use `,omitempty`.
- snake_case for JSON keys (NOT camelCase).
- Field-level godoc above the field, NOT inline.

For `StreamReader` (interface):
- `Next() (StreamEvent, error)` returning `io.EOF` is idiomatic Go (cite `bufio.Scanner`); use stdlib `io` import.
- `Close() error` — match standard `io.Closer` shape.

---

### `llm/info.go` (type — `ProviderInfo`, `Capabilities`)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (lines 39-50: `GenerateResponse` — struct with all-tagged fields)

**Struct-with-tags pattern** (`client.go` lines 39-50):
```go
type GenerateResponse struct {
	Text         string         `json:"text"`
	FinishReason FinishReason   `json:"finish_reason,omitempty"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model,omitempty"`
	UsageToken   int            `json:"usage_token,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	// ToolCalls are populated when the model decides to invoke one or
	// more registered Tools. Callers route them to executors and feed
	// results back via History on the next turn.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```
**Conventions to copy:**
- All-fields-tagged.
- Required fields drop `,omitempty`; optional/empty-zero fields add it.
- Inline single-line godoc OK for self-evident fields; multi-line godoc above the field for fields needing explanation.

For `ProviderInfo`:
```go
// ProviderInfo describes a bound provider+model combination.
// Returned by ChatModel.Info(). Capabilities reflect THIS bound model,
// not the provider type generically (Pitfall 6).
type ProviderInfo struct {
    Provider     string       `json:"provider"`
    Model        string       `json:"model"`
    Capabilities Capabilities `json:"capabilities"`
}
```
For `Capabilities`: Each bool field gets a trailing `// comment` per RESEARCH.md (matches `client.go` comment style for `ToolCalls`).

---

### `llm/types.go` (`Request`, `Response`, `Message`, `Tool`, `ToolCall`, `Vector`, `Usage`)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (entire file — every type to be copied with renames)

**Tool struct pattern** (`client.go` lines 81-89):
```go
// Tool declares a function the model may call. Parameters is a raw
// JSON Schema document — this package doesn't validate it (the
// upstream provider does) so callers can use whatever schema dialect
// their provider expects.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}
```
**Conventions to copy:**
- `json.RawMessage` for opaque-pass-through JSON (NOT `string`, NOT `[]byte`, NOT `any`).
- Multi-paragraph godoc explaining WHY (architectural choice), not just WHAT.

**Message struct pattern** (`client.go` lines 32-37):
```go
// Message represents a single turn in a conversation. Role is one of
// "user" / "assistant" (provider-specific extras like "system" / "tool"
// land in Metadata-shaped extensions if a provider ever needs them).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
```
**Convention:** `Message` ports verbatim — same Role/Content fields, same tags. Only doc updates (Role values now include "tool").

---

### `llm/errors.go` (sentinel errors)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent.go` (lines 127-136)

**Sentinel-error block pattern** (`agent.go` lines 127-136):
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
**Conventions to copy:**
- Single `var (…)` block grouping all sentinels.
- Error string prefix: `"<package>: <description>"` — for `llm/errors.go`, prefix is `"llm: "`.
- Block-level godoc above the `var (…)` opener, not per-var.
- Existing test pattern (`agent_test.go` lines 18-33) iterates sentinels through `errors.Is(wrapped, sentinel)` — apply same test to `llm.ErrCapabilityNotSupported` and `llm.ErrScriptExhausted`.

For new file:
```go
package llm

import "errors"

// Sentinel errors for the llm package. Callers detect via errors.Is.
var (
    ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")
    ErrScriptExhausted        = errors.New("llm: scripted llm: script exhausted")
)
```

---

### `llm/scripted.go` (full-capability mock — promoted from `_test.go`)

**Analogs:**
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/scriptedllm_test.go` (the entire file — design template)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/examples/scriptedllm/scriptedllm.go` (cursor-based variant + functional helpers `Text`, `ToolCall`)

**Mock struct pattern** (`scriptedllm_test.go` lines 14-18):
```go
// scriptedLLM is a test helper that returns pre-set GenerateResponse values
// in order on each call. After the script is exhausted it returns
// errScriptExhausted. Concurrent-safe via mu.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.GenerateResponse
}
```
**Conventions to copy:**
- `sync.Mutex` (not `sync.RWMutex`) for cursor protection — script advancement is always write.
- Cursor field name: existing repo uses both `calls` (test helper) and `cursor` (examples helper). **Prefer `cursor`** to match the published `examples/scriptedllm/scriptedllm.go` shape (the public-facing template).
- Slice of responses + integer counter — no goroutines, no channels.

**Generate method pattern** (`scriptedllm_test.go` lines 26-36):
```go
func (s *scriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.GenerateResponse{}, errScriptExhausted
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}
```
**Convention:** `_ context.Context, _ llm.GenerateRequest` (underscored — args ignored by deterministic mock). Lock-defer-unlock at function entry.

**Functional-options + helpers pattern** (`examples/scriptedllm/scriptedllm.go` lines 18-43):
```go
// New returns an llm.Client that yields the given responses in order. When
// the script runs out, subsequent calls return ErrScriptExhausted.
func New(responses ...llm.GenerateResponse) llm.Client {
	return &client{responses: responses}
}

// Text is a convenience constructor for plain-text responses ending in
// FinishReasonStop.
func Text(s string) llm.GenerateResponse {
	return llm.GenerateResponse{Text: s, FinishReason: llm.FinishReasonStop}
}

// ToolCall builds a tool-call response (FinishReasonToolCalls) for the given
// tool name and JSON arguments string.
func ToolCall(name, argsJSON string) llm.GenerateResponse { … }
```
**Convention:** Public mock provides 2-3 ergonomic builders alongside the constructor — `Text`, `ToolCall`. New ScriptedLLM v2 should expose `TextResponse`, `ToolCallResponse` (per RESEARCH.md naming).

**Compile-time interface satisfaction** (`agent_test.go` lines 9-16):
```go
var (
	_ Agent = (*SimpleAgent)(nil)
	_ Agent = (*ReActAgent)(nil)
	_ Agent = (*ReflectionAgent)(nil)
	_ Agent = (*PlanAndSolveAgent)(nil)
	_ Agent = (*FunctionCallAgent)(nil)
)
```
**Convention:** Use this exact shape inside `llm/scripted.go` for all 4 capability interfaces:
```go
var (
    _ ChatModel         = (*ScriptedLLM)(nil)
    _ ToolCaller        = (*ScriptedLLM)(nil)
    _ Embedder          = (*ScriptedLLM)(nil)
    _ StructuredOutputs = (*ScriptedLLM)(nil)
)
```

---

### `llm/chat_only_mock.go` (capability-degrade mock)

**Analog:** Same as `llm/scripted.go` — but with **ONE** `var _` assertion (only `ChatModel`). Negative assertions belong in `llm/llm_test.go`.

**Convention from `agent_test.go` lines 9-16:** Use a single `var _` for ChatModel; deliberately omit ToolCaller/Embedder/StructuredOutputs. Pair with a runtime test (`TestChatOnlyMockExcludesCapabilities`, RESEARCH.md lines 760-771) that asserts NEGATIVE: `if _, ok := m.(ToolCaller); ok { t.Fatal(...) }`.

---

### `llm/doc.go` (package overview)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/doc.go` (existing — full file is 18 lines)

**Existing doc.go shape** (the entire file):
```go
// Package llm holds the LLM-provider contract used by the agents
// framework. It is the only package outside agents/* that an Agent or
// Tool implementation depends on at the type level.
//
// The contract is intentionally narrow:
//
//   - Client    one-shot Generate + token-streaming GenerateStream
//   - Tool      function-call schema (JSON Schema parameters)
//   - ToolCall  function-call invocation returned by the model
//   - Message   single conversation turn
//   - StreamChunk / StreamUsage  streaming primitives
//   - FinishReason + 6 const  OpenAI-compatible stop reasons
//
// Provider implementations (HTTP, Ollama, etc.) live in the parent
// AICS repo's pkg/llm package, which type-aliases everything here.
// External users plug their own provider by implementing Client and
// passing it to any Agent constructor.
package llm
```
**Conventions to copy for the rebooted version:**
- Comment block precedes `package llm` line directly (no blank line between final comment and `package`).
- Bullet list with 4-space indent + 2-space spacing between symbol and description.
- `package llm` is the FINAL line of the file — no imports, no code.
- Replace bullet list with new types: `ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamEvent`, `StreamReader`, `ProviderInfo`, `Capabilities`, `LegacyClient` (with deprecation note).

The new `doc.go` should add the **canonical capability-negotiation idiom** (RESEARCH.md §"Pattern 1") as a runnable godoc example block — but be careful: godoc examples in `doc.go` are unusual; prefer to put runnable examples in `llm_test.go` (`Example_capabilityNegotiation` per Go convention) and reference from `doc.go`.

---

### `llm/llm_test.go` (interface satisfaction + behavior tests)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent_test.go` (entire file)

**Test file structure** (`agent_test.go`):
```go
package agents

import (
	"errors"
	"fmt"
	"testing"
)

// Compile-time interface conformance check for all 5 Agents.
var (
	_ Agent = (*SimpleAgent)(nil)
	...
)

func TestSentinelErrors_ErrorsIs(t *testing.T) { ... }
func TestStepKind_Constants(t *testing.T) { ... }
```
**Conventions to copy:**
- Test package name: same as the package under test (NOT `<pkg>_test`). Existing repo uses internal tests universally.
- Imports: stdlib only (`errors`, `fmt`, `testing`) — verified across all `*_test.go` files in repo root.
- Test names: `TestThing_Behavior` (snake_case after underscore) — existing convention from `simple_test.go` (`TestSimpleAgent_Run_TransparentlyForwards`).
- Table tests: `cases := []…` slice + `for _, c := range cases` — see `agent_test.go` lines 19-33.
- Error assertions: `errors.Is(wrapped, sentinel)` (idiomatic Go 1.13+).
- Logging: `t.Errorf("X = %v", x)` for soft assertions; `t.Fatalf` only when subsequent assertions cannot run.

Apply to `llm/llm_test.go` per RESEARCH.md test inventory (`TestChatOnlyMockExcludesCapabilities`, `TestScriptedLLM_Capabilities`, `TestStreamReaderClosesIdempotent`, `TestLegacyClientAlias`).

---

### `llm/legacy.go` (renamed from `client.go`)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (entire file — verbatim move with rename + deprecation comments)

**Rename + alias + deprecation pattern** (NEW — no in-repo precedent for a Deprecated comment yet):
```go
// LegacyClient is the v0.2 LLM contract.
//
// Deprecated: Use llm.ChatModel instead. LegacyClient will be removed
// in v0.4.0. See docs/migration-v0.2-to-v0.3.md for a migration guide
// with worked examples.
type LegacyClient interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}

// Client is an alias for LegacyClient retained for v0.2 source compatibility.
//
// Deprecated: Use llm.ChatModel instead. Client will be removed in v0.4.0.
type Client = LegacyClient
```
**Conventions to copy from `llm/client.go` lines 14-17 (the existing interface):**
- Method bodies preserved verbatim (signatures + return types unchanged).
- Same `json:` tags on companion types `GenerateRequest`, `GenerateResponse`, `StreamChunk`, `StreamUsage` — preserved verbatim.

**Deprecation comment format** (per CONTEXT.md §"Specifics" line 167):
```
// Deprecated: Use llm.ChatModel instead. LegacyClient will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
```
- The keyword `Deprecated:` (with capital D and trailing colon) is REQUIRED — `gopls` and `staticcheck` recognize this exact spelling for IDE warnings.
- Apply to: `LegacyClient`, `Client` alias, `GenerateRequest`, `GenerateResponse`, `StreamChunk`, `StreamUsage`.

---

### `docs/migration-v0.2-to-v0.3.md` (migration guide)

**Analogs:**
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/CHANGELOG.md` lines 71-83 (v0.2.0 Migration block — diff style)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/README.md` lines 60-77 (Packages table — markdown table style)

**Diff-style migration block** (`CHANGELOG.md` lines 71-83):
```markdown
### Migration

\`\`\`diff
- import "github.com/costa92/aics-core/pkg/llm/agents"
+ import "github.com/costa92/llm-agent"

- import "github.com/costa92/aics-core/pkg/llm/agents/llm"
+ import "github.com/costa92/llm-agent/llm"
\`\`\`

\`\`\`bash
go get github.com/costa92/llm-agent@v0.2.0
\`\`\`
```
**Conventions to copy:**
- ` ```diff ` fenced blocks for line-level changes.
- ` ```go ` (or ` ```bash `) for runnable snippets.
- One blank line between heading and code block.
- Migration sections are SHORT (≤ 30 lines) — the Migration block in v0.2.0 is intentionally minimal; a longer file like `docs/migration-v0.2-to-v0.3.md` extends this style with multiple worked examples but keeps each example small.

**Markdown table style** (`README.md` lines 62-77):
```markdown
| Package | Purpose |
|---|---|
| `agents` | Agent / Tool interface + 5 paradigm constructors ... |
```
**Convention:** Three-column-or-wider tables in Migration guide use `|---|` (NOT `| --- |` / `|-------|`) — match repo style. Backticks around symbol names: ``` `llm.Client` ```.

---

### `DEPRECATIONS.md` (new file)

**Analog:** None in repo (NEW file type). Closest shape: `CHANGELOG.md` Deprecated-section format (RESEARCH.md prescribes a table).

**Recommended shape (lifted from RESEARCH.md):**
```markdown
# Deprecations

| Symbol | Deprecated In | Removed In | Migration |
|---|---|---|---|
| `llm.Client` | v0.3.0 | v0.4.0 | Use `llm.ChatModel`; see [migration guide](docs/migration-v0.2-to-v0.3.md) |
| `llm.LegacyClient` | v0.3.0 | v0.4.0 | Same |
| `llm.GenerateRequest` | v0.3.0 | v0.4.0 | Use `llm.Request` |
| `llm.GenerateResponse` | v0.3.0 | v0.4.0 | Use `llm.Response` |
| `llm.StreamChunk` | v0.3.0 | v0.4.0 | Use `llm.StreamEvent` |
```
**Convention:** Match `README.md` table style (line-noise minimum). Keep entries SORTED by deprecation version, then by symbol name.

---

### `CHANGELOG.md` (modified — add Deprecated section)

**Analog:** itself (`/home/hellotalk/code/go/src/github.com/costa92/llm-agent/CHANGELOG.md` — lines 12-83)

**Existing version-block pattern** (lines 12-42, v0.1.0 entry):
```markdown
## [v0.1.0] — 2026-04-28

Initial module release. ...

### Added

- Standalone Go module: `github.com/costa92/llm-agent`
- New `agents/llm` subpackage owning the LLM contract: ...
```
**Conventions to copy:**
- `## [vX.Y.Z] — YYYY-MM-DD` heading (ISO-8601 date, em-dash separator).
- One-paragraph summary BELOW the heading, BEFORE first `###` subsection.
- `### Added | Changed | Deprecated | Removed | Fixed | Security | Breaking` — keep-a-changelog conventions (declared at lines 6-8 of file).
- Bullet list under each `###`. Backticks around symbols.

For Phase 0 entry:
```markdown
## [Unreleased]

### Added

- New capability-aware interfaces in `llm/`: `ChatModel`, `ToolCaller`, ...

### Deprecated

- `llm.Client` → use `llm.ChatModel`. Removed in v0.4.0.
- ...
```

---

### Sister repo `LICENSE` (3 copies)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/LICENSE` (entire file, verbatim copy)

**Convention:** MIT, `Copyright (c) 2026 costa92` — same year, same owner, byte-identical.

---

### Sister repo `OWNERS` (3 copies)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/OWNERS` (entire file)

**Existing OWNERS** (this repo):
```
# OWNERS — github.com/costa92/llm-agent
#
# Code review and approval for the standalone Go LLM agents framework module.
#
# Format: https://www.kubernetes.io/docs/contribute/participate/roles-and-responsibilities/

approvers:
  - costa92

reviewers:
  - costa92

labels:
  - area/agents
```
**Conventions to copy:**
- Header comment: `# OWNERS — <module path>` + 1-line purpose + format reference.
- `approvers:` / `reviewers:` / `labels:` blocks separated by blank lines.
- Single `costa92` per CONTEXT.md §"Specifics" line 165.

**Per-repo substitutions** (per RESEARCH.md §"Sister Repo Skeleton Breakdown"):
- `llm-agent-providers` → `labels: - area/providers`
- `llm-agent-otel` → `labels: - area/otel`
- `llm-agent-customer-support` → `labels: - area/refsvc`

---

### Sister repo `.gitignore` (3 copies + core update)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.gitignore` (existing — 19 lines)

**Existing .gitignore** (lines 1-19):
```
# Build / test artifacts
*.test
*.out
*.prof
coverage.txt
coverage.html

# Editor / OS
.DS_Store
.idea/
.vscode/
*.swp
*.swo

# Local env
.env
.env.*
!.env.example
```
**Conventions to copy:**
- Section comments (`# Build / test artifacts`) separating thematic groups.
- Blank lines between sections.
- Negative pattern at the end (`!.env.example`).

**Phase 0 additions** (per INFRA-02 + Pitfall 13):
```
# Multi-repo workspace (Pitfall 13)
go.work
go.work.sum
```
Append as a NEW section at the end of each repo's .gitignore (this repo + 3 sister repos).

---

### Sister repo `go.mod` (3 copies)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/go.mod` (existing — 3 lines)

**Existing go.mod** (entire file):
```
module github.com/costa92/llm-agent

go 1.26.0
```
**Conventions to copy:**
- Module path on line 1.
- Blank line.
- `go <version>` on line 3.
- NO `require` block in core (stdlib-only).

**Sister repo extension** (RESEARCH.md §"Sister Repo Skeleton Breakdown"):
```
module github.com/costa92/llm-agent-providers

go 1.26.0

require github.com/costa92/llm-agent v0.3.0-pre.1
```
- Add `require` block separated by blank line from `go` directive.
- Sister repos may take other deps in subsequent phases; their `go.sum` files exist (vs. core which has none).

---

### Sister repo `README.md` (3 copies)

**Analogs:**
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/README.md` (style template — bilingual EN/CN, install snippet, package table, version banner)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/examples/README.md` (sub-module README style — table, `## Run`, single-purpose focus)

**Style elements to copy from `README.md`:**
- H1 with project name + tagline (line 1: `# agents — Go LLM agents framework`).
- Version-status banner (`> **v0.2.0 — 学習 / 原型 stage.** API may break ...` — lines 8-12). Sister repos: substitute `v0.1.0-pre / Phase 0 skeleton` etc.
- Install command in fenced ` ```bash ` block (line 16-18).
- Markdown package/feature table.

**Style elements to copy from `examples/README.md`:**
- Concise (~40 lines) for narrow-purpose READMEs (sister repos are narrow-purpose: one is providers, one is OTel, one is refsvc).
- `## Run` section with cd + go run snippets.
- Cross-reference link back to parent (`./scriptedllm/`) — sister repos link back to `https://github.com/costa92/llm-agent`.

**Per-repo content** (purpose blocks per RESEARCH.md §"Sister Repo Skeleton Breakdown"):
- `llm-agent-providers`: "Provider adapters for `github.com/costa92/llm-agent`. ..."
- `llm-agent-otel`: "OpenTelemetry decorator wrappers for ..."
- `llm-agent-customer-support`: "Reference customer-support service ... **Demo only** ..."

---

### Sister repo `.github/workflows/test.yml` (3 copies + core update)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml` (existing — 58 lines)

**Existing test.yml structure** (lines 1-58):
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
      - name: go mod tidy (drift check)
        run: |
          go mod tidy
          # llm-agent is stdlib-only, so go.sum may not exist. Use
          # `git status --porcelain` (handles missing files cleanly)
          # instead of `git diff --exit-code` (fatals on absent paths).
          drift=$(git status --porcelain go.mod go.sum 2>/dev/null)
          if [ -n "$drift" ]; then
            echo "go mod tidy changed files — commit tidy changes first"
            ...
            exit 1
          fi
      - name: examples go mod tidy (drift check)
        run: |
          cd examples
          ...
      - name: go vet
        run: go vet ./...
      - name: go build
        run: go build ./...
      - name: go test
        run: go test ./...
      - name: examples — vet + build
        run: |
          cd examples
          go vet ./...
          go build ./...
```
**Conventions to copy:**
- `name: test` lowercase top-level workflow name.
- `on:` triggers: push to `main` + PR to `main`.
- `concurrency.group: test-${{ github.ref }}` + `cancel-in-progress: true` — cancel superseded runs.
- `runs-on: ubuntu-latest`, `timeout-minutes: 10`.
- `actions/checkout@v4`, `actions/setup-go@v5` — exact versions; `go-version-file: go.mod` for single source of truth.
- `cache: true` (works regardless of go.sum presence per existing comment).
- `go mod tidy` drift check using `git status --porcelain` (handles stdlib-only no-go.sum cleanly — see comment lines 27-29).
- Steps named with `name:` in lowercase, e.g., `go vet`, `go build`, `go test`.
- Multi-line shell with `run: |`.

**Phase 0 additions** (per INFRA-02 / RESEARCH.md):
```yaml
env:
  GOWORK: off  # INFRA-02: CI never picks up a workspace file silently
```
Insert at JOB level (sister repos) or WORKFLOW level (core repo). RESEARCH.md uses job-level for sister repos and modifies core's existing test.yml to add at workflow or job level.

**Sister-repo-specific tweaks:**
- Sister repos do not have an `examples/` subdirectory yet; drop the two `examples` steps.
- Sister repos have `go.sum` (they take third-party deps eventually); the same `git status --porcelain` drift check still works.
- Empty modules (Phase 0 sister repo skeletons have NO Go source) succeed with `go vet ./... && go build ./...` returning exit 0 (RESEARCH.md Assumption A1).

---

### `.github/workflows/umbrella.yml` (NEW — core repo)

**Analog:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml` (template) + RESEARCH.md §"CI YAML Sketches" (the umbrella sketch is the planner's source).

**Pattern from RESEARCH.md lines 996-1075:**
- Same top-level shape (`name:`, `on:`, `concurrency:`, `jobs:`).
- DIFFERENT trigger: PR-only + `workflow_dispatch:` (NOT push-to-main).
- DIFFERENT timeout: 15 minutes (vs 10 for per-repo — accounts for 4-repo checkout + 4× build).
- 4 sequential `actions/checkout@v4` steps with `repository:` + `path:` parameters for cross-repo.
- One `actions/setup-go@v5` (uses `llm-agent/go.mod` as version source).
- `go work init ./llm-agent ./llm-agent-providers ...` step.
- Per-repo `cd <repo> && go vet/build/test ./...` step.

**Convention from existing `test.yml`:** keep step names lowercase + descriptive (`Build llm-agent-providers against this PR's llm-agent` from RESEARCH.md is fine). Stick to `actions/*@v4|v5` exact versions for consistency.

---

### `.github/workflows/release-precheck.yml` (NEW — 4 copies)

**Analog:** `.github/workflows/test.yml` (template structure) + RESEARCH.md §"Release-precheck" lines 1085-1122 (verbatim spec).

**Pattern from RESEARCH.md:**
- Same top-level shape.
- DIFFERENT trigger: `branches: ['release/**']` push + PR.
- Single job `no-replace` with one step that parses `go mod edit -json`.
- RESEARCH.md uses Python (preinstalled on `ubuntu-latest`) instead of `jq`. Either is fine; **prefer Python** to match RESEARCH.md.

**Decision per RESEARCH.md Q6:** Copy-paste 4 times (NOT reusable workflow). Each repo owns its own release gate.

---

### `scripts/workspace.sh` (NEW — 4 copies)

**Analog:** None in repo (NEW). Source: STACK.md §"Local development pattern" lines 148-161 (per RESEARCH.md citation).

**Convention:**
- Bash 4+ shebang (`#!/usr/bin/env bash`).
- `set -euo pipefail` at top.
- Discovers sibling repos relative to script location.
- Writes `go.work` ABOVE the repo (in the parent of all 4 sibling clones).
- Idempotent — safe to re-run.

The script body comes from STACK.md verbatim (RESEARCH.md says "lifted unchanged"). Planner copies the exact text.

---

## Shared Patterns

### Stdlib-Only Imports (CORE repo)

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/go.mod` (no `require` block) + every existing `*.go` file's import block.

**Apply to:** ALL new files in `llm/` package.

**Verified import inventory across `llm/` package additions** (per RESEARCH.md §"Standard Stack"):
- `context` — stdlib (used in `ChatModel.Generate`, `ToolCaller.WithTools`, etc.)
- `encoding/json` — stdlib (used for `json.RawMessage` fields on `Tool`, `ToolCall`)
- `errors` — stdlib (sentinel errors in `errors.go`)
- `io` — stdlib (`io.EOF` sentinel from `StreamReader.Next`)
- `sync` — stdlib (`sync.Mutex` in `ScriptedLLM`)
- `fmt` — stdlib (only if errors need wrapping)

**ANTI-PATTERN:** Adding ANY non-stdlib import to a `llm/*.go` file violates CLAUDE.md Hard Rule 1. Phase 0 requires `go.mod` stay at:
```
module github.com/costa92/llm-agent

go 1.26.0
```
No `require` block. Verified by `go mod tidy` drift-check in CI (existing test.yml lines 23-35).

---

### `json:` Tag Convention

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (every public-type field carries `json:"…"`).

**Apply to:** ALL fields on ALL public types in `llm/info.go`, `llm/types.go`, `llm/stream.go`.

**Rules** (extracted from `client.go`):
- snake_case JSON keys: `finish_reason`, `tool_calls`, `total_tokens`, `usage_token`. NEVER camelCase.
- Required fields: drop `,omitempty` (e.g., `Text string \`json:"text"\``).
- Optional fields: add `,omitempty` (e.g., `Model string \`json:"model,omitempty"\``).
- Pointer-typed optional fields: pointer + `,omitempty` (e.g., `Usage *StreamUsage \`json:"usage,omitempty"\``).
- `[]ToolCall` optional → `,omitempty` (lines 49 in client.go: `ToolCalls []ToolCall \`json:"tool_calls,omitempty"\``).

**Why this matters for Phase 0:** `Capabilities` struct (D-02) explicitly requires JSON-serializability for OTel attribute emission (Phase 5). Tag convention is non-negotiable.

---

### Godoc Style (Type and Interface Comments)

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` (every public type) + `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent.go` (every public interface).

**Apply to:** ALL public symbols in new `llm/` files.

**Rules:**
- First line of godoc starts with the symbol name: `// X is …` / `// X does …` / `// X represents …`. Required by `golint`.
- Multi-paragraph godoc separated by blank-comment-line (`//\n//`).
- Multi-line explanations precede the symbol (NOT inline trailing comment).
- Architecture-level WHYs (not just WHATs): see `client.go` lines 19-29 — `GenerateRequest`'s `History` field doc explains WHY (multi-turn dialog without forcing mega-prompt construction).
- For interfaces: list each method's purpose in the godoc when the interface has 2+ methods (see `tool.go` lines 10-21).

**Deprecated comment subtype** (NEW for Phase 0):
- Format: `// Deprecated: <use what instead>. <when removed>. <see migration link>.`
- Place AFTER the regular godoc, separated by blank-comment-line.
- Example (from CONTEXT.md §"Specifics" line 167):
  ```go
  // LegacyClient is the v0.2 LLM contract.
  //
  // Deprecated: Use llm.ChatModel instead. LegacyClient will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
  type LegacyClient interface { ... }
  ```

---

### Test File Convention

**Source:** All `*_test.go` files in repo root (`agent_test.go`, `simple_test.go`, `react_test.go`, etc.) — verified pattern.

**Apply to:** `llm/llm_test.go`.

**Rules:**
- Internal test package (same name as production, NOT `<pkg>_test`). All existing tests use this style.
- Imports: stdlib only (`context`, `errors`, `fmt`, `testing`).
- File starts with compile-time `var _ Interface = (*Concrete)(nil)` block when applicable (`agent_test.go` lines 9-16).
- Test names: `TestSubject_Behavior` (PascalCase subject, snake_case behavior).
- Use `t.Errorf` for soft failures, `t.Fatalf` only for hard-stop errors.
- Co-locate test files (NOT in `_test/` subdirectory) — verified across all 12 packages.
- Examples (godoc executable): `ExampleX` named functions with `// Output:` comment block — see `example_simple_test.go` lines 13-33.

---

### Sentinel Error Convention

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent.go` lines 127-136 + `agent_test.go` lines 18-33.

**Apply to:** `llm/errors.go`.

**Rules:**
- `var (…)` block, NOT individual `var Err…`.
- Block-level godoc above `var (`, NOT per-symbol.
- Error message format: `"<package>: <description>"` (e.g., `"llm: capability not supported by bound model"`).
- `errors.New` (NOT `fmt.Errorf`) for plain sentinels.
- Companion test: `TestSentinelErrors_ErrorsIs` asserts each sentinel survives `errors.Is(fmt.Errorf("wrap: %w", sentinel), sentinel)`.

---

### Compile-Time Interface Satisfaction

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/agent_test.go` lines 9-16.

**Apply to:** `llm/scripted.go`, `llm/chat_only_mock.go` (positive assertions in production code; negative assertions in test).

**Rule:**
```go
var (
    _ ChatModel         = (*ScriptedLLM)(nil)
    _ ToolCaller        = (*ScriptedLLM)(nil)
    _ Embedder          = (*ScriptedLLM)(nil)
    _ StructuredOutputs = (*ScriptedLLM)(nil)
)
```
Place in production file (NOT just test) when the type's whole purpose is to satisfy multiple interfaces. This matches `agent_test.go` (existing convention puts it in test) but RESEARCH.md prescribes placing it in `llm/scripted.go` for visibility — **planner choice**: prefer production-file placement for `scripted.go` to make capability claims part of the published API surface.

---

### CI Workflow YAML Style

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml`.

**Apply to:** `umbrella.yml`, `release-precheck.yml`, sister-repo `test.yml`.

**Rules:**
- Lowercase top-level workflow `name:` (e.g., `name: test`).
- `on:` block uses bracket-list syntax for branches: `branches: [main]`.
- `concurrency.group:` includes ref: `test-${{ github.ref }}`.
- `cancel-in-progress: true`.
- `runs-on: ubuntu-latest`, explicit `timeout-minutes:`.
- Action versions pinned to major: `@v4`, `@v5`.
- `actions/setup-go@v5` always uses `go-version-file: go.mod`.
- Step `name:` lowercase, descriptive verb-prefix style: `go vet`, `go mod tidy (drift check)`.
- Multi-line shell with `run: |`.
- Comments embedded in `run: |` blocks for non-obvious choices (see test.yml lines 27-29).

---

### Markdown Doc Style

**Source:** `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/README.md` + `examples/README.md` + `CHANGELOG.md`.

**Apply to:** `docs/migration-v0.2-to-v0.3.md`, `DEPRECATIONS.md`, sister-repo `README.md`s.

**Rules:**
- ATX-style headings (`# H1`, `## H2`, `### H3`).
- Fenced code blocks with language hint: ` ```go `, ` ```bash `, ` ```diff `, ` ```yaml `.
- Tables: `|---|---|` (three-dash separator, no inner padding) — see `README.md` lines 62-64.
- Backticks around symbols, file paths, commands.
- Bilingual EN/CN paragraphs in core repo `README.md` are OPTIONAL for sister-repo READMEs (RESEARCH.md doesn't require them — Claude's discretion). Recommend EN-only for sister repos to keep them lean.
- Banner-style notice: `> **<status>:** <message>` (see `README.md` lines 8-12).
- Diff blocks for migration code: ` ```diff ` with `-` / `+` line prefixes (see `CHANGELOG.md` lines 73-79).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `scripts/workspace.sh` | shell | n/a | No shell scripts in repo today; pattern lifted verbatim from STACK.md |
| `.github/workflows/umbrella.yml` | CI | event-driven | Multi-repo checkout + `go work init` is novel for this repo; pattern from RESEARCH.md sketch |
| `.github/workflows/release-precheck.yml` | CI | event-driven | `go mod edit -json` parsing is novel; pattern from RESEARCH.md sketch (which itself cites PITFALLS Pitfall 12) |
| `DEPRECATIONS.md` | docs | n/a | No equivalent file exists; format prescribed by RESEARCH.md (Pitfall 15) |
| Sister-repo skeletons (3 fresh GitHub repos) | infra | n/a | No in-repo precedent for repo creation; per-file analogs cover the contents — but the act of `gh repo create` is a one-time operation outside any "pattern" |

For these "no analog" items, planner should:
1. Lift the pattern verbatim from RESEARCH.md / STACK.md / PITFALLS.md as cited.
2. Defer to existing in-repo style for ANCILLARY conventions (YAML quoting, comment style, file naming).
3. Note in the plan that the file has no in-repo precedent so reviewers know not to look for one.

---

## Cross-Plan Pattern Map (planner reference)

| Plan | Files | Primary Patterns | Shared Patterns Applied |
|------|-------|------------------|--------------------------|
| 00-01 (`llm/` reboot) | `chatmodel.go`, `capabilities.go`, `stream.go`, `info.go`, `types.go`, `errors.go`, `scripted.go`, `chat_only_mock.go`, `doc.go`, `legacy.go`, `llm_test.go` | Interface-with-doc, struct-with-tags, sentinel errors, mock with `sync.Mutex` + cursor, compile-time `var _` | Stdlib-only, json: tag, godoc style, test file convention |
| 00-02 (migration docs) | `docs/migration-v0.2-to-v0.3.md`, `DEPRECATIONS.md`, `CHANGELOG.md` | Diff blocks, markdown tables, deprecation comment format | Markdown doc style |
| 00-03 (sister repos) | `LICENSE`, `OWNERS`, `.gitignore`, `go.mod`, `README.md`, `.github/workflows/test.yml`, `.github/workflows/release-precheck.yml`, `scripts/workspace.sh` (×3 sister repos) | Verbatim file copies (LICENSE, OWNERS, .gitignore) + per-repo substitutions (OWNERS labels, README purpose, go.mod module path) | Markdown doc style, CI YAML style |
| 00-04 (core CI hardening) | `.gitignore` (modified), `scripts/workspace.sh`, `.github/workflows/test.yml` (modified) | `.gitignore` append-section, `GOWORK: off` injection | CI YAML style |
| 00-05 (umbrella + release-precheck) | `.github/workflows/umbrella.yml`, `.github/workflows/release-precheck.yml` | Multi-repo checkout, `go work init`, `go mod edit -json` parsing | CI YAML style |

---

## Metadata

**Analog search scope:**
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/` (existing package being rebooted)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/*.go` (root agents package — interface + sentinel-error patterns)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/scriptedllm_test.go` + `examples/scriptedllm/scriptedllm.go` (mock LLM templates)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml` (CI template)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/{LICENSE, OWNERS, .gitignore, go.mod, CHANGELOG.md, README.md}` (top-level meta files)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/examples/README.md` (sub-module README style)

**Files scanned:** 24 (7 in llm/, 12 in repo root including `*_test.go`, 5 meta files, 1 CI workflow)

**Pattern extraction date:** 2026-05-10

**Confidence:** HIGH for all in-repo analogs (verified by direct Read of every cited file/range); MEDIUM for "fresh" patterns (umbrella.yml, release-precheck.yml, scripts/workspace.sh, DEPRECATIONS.md) — these have no in-repo precedent and depend on RESEARCH.md / STACK.md / PITFALLS.md fidelity, which RESEARCH.md cites verbatim.
