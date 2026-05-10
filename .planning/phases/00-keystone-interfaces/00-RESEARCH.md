# Phase 0: Multi-repo infra + `llm/v2` keystone interfaces - Research

**Researched:** 2026-05-10
**Domain:** Go module/interface design + multi-repo Go module mechanics + GitHub Actions CI design
**Confidence:** HIGH (codebase fully inspected; decisions D-01..D-04 lock direction; remaining variability is exact field naming and CI YAML phrasing)

## Summary

Phase 0 has two intertwined deliverables: (a) a rebooted `llm/` package containing the K1/K2/K3 keystone interfaces (`ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamEvent`, `StreamReader`, `ProviderInfo`, `Capabilities`), and (b) the multi-repo umbrella infra (3 sister repos + `go.work` writer + `GOWORK=off` per-repo CI + umbrella build + release-precheck `replace` ban). Both must land together because the umbrella CI in (b) has nothing to build against unless (a) compiles, and (a) has nowhere to be exercised cross-repo unless (b)'s skeletons exist. Within Phase 0 these are executed by parallel plans on separate repos, sequenced through a shared interface-locking checkpoint.

The locked decisions (D-01..D-04 in CONTEXT.md) eliminate the largest design uncertainties. What remains is mechanical and well-bounded: precise Go field names and method signatures (constrained directionally by D-01/D-02 and SDK ergonomics), exact GHA YAML for the 3 CI surfaces (per-repo, umbrella, release-precheck), and the migration playbook for ~13 internal `llm.Client` callers in this repo. Nothing in Phase 0 needs new library research — Context7 is not consulted because no third-party library is added; the only "library" being designed is `llm/` itself.

**Primary recommendation:** Plan 5 plans — `00-01` core `llm/` reboot (interfaces + types + ScriptedLLM v2 + ChatOnlyMock + LegacyClient rename), `00-02` migration guide + DEPRECATIONS.md + CHANGELOG, `00-03` sister repo skeletons (3 repos pushed to GitHub with go.mod/LICENSE/OWNERS/README/CI/.gitignore), `00-04` core repo `.gitignore` + `go.work` writer script + per-repo CI hardening, `00-05` umbrella CI + release-precheck CI in core. Plans `00-01` ↔ `00-03` run in parallel (different repos); `00-02`/`00-04`/`00-05` sequence after the core interfaces are locked.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Capability interface contract (ChatModel etc.) | Core repo `llm/` package | — | Stdlib-only. The seam every sister repo binds to. Lives nowhere else. |
| Streaming event union shape | Core repo `llm/` package | — | Same as above; defining the cross-provider language. |
| ScriptedLLM v2 (full-capability mock) | Core repo `llm/` package (non-test code) | Sister-repo conformance (Phase 1) consumes it | Promoted out of `_test.go` per D-03 because conformance suite + agent tests both need it. |
| ChatOnlyMock (capability-degrade mock) | Core repo `llm/` package (non-test code) | Phase 3 agent fallback tests | Same rationale as ScriptedLLM but specifically for missing-capability paths. |
| `LegacyClient` deprecation alias | Core repo `llm/` package | — | `// Deprecated:` godoc lives next to the type; no separate file needed. |
| Migration guide | Core repo `docs/migration-v0.2-to-v0.3.md` | CHANGELOG.md links to it | Documentation is ergonomically near the deprecated symbols. |
| `.gitignore` of `go.work` | Each of 4 repos individually | — | Per Pitfall 13, every repo gitignores its own `go.work`. Cannot be centralized. |
| `go.work` writer script | Each sister repo's `scripts/workspace.sh` | Symlinked or copied in core | Keeps the script next to the user who runs it (sister-repo dev does the cross-repo iteration). |
| Per-repo CI (`test.yml`) | Each of 4 repos `.github/workflows/test.yml` | — | GitHub Actions is repo-scoped; no shared workflow registry exists for org-public reusable workflows in this scenario. |
| Umbrella CI (4-repo build) | `llm-agent/.github/workflows/umbrella.yml` | — | One repo owns the umbrella; per ROADMAP §"Phase 0" success criterion 4. |
| Release-precheck (`replace` ban on tagged-release branches) | Each of 4 repos `.github/workflows/release-precheck.yml` | — | Per-repo because tags are per-repo; INFRA-04. |
| Sister-repo creation + push | One-time GitHub-side operation | gh CLI | Creates the GitHub repos + pushes initial commit. |

## Standard Stack

### Core (no new dependencies — Phase 0 is stdlib-only on the core repo by hard rule)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `context` | 1.26 | `Generate(ctx, ...)`, cancellation | Established existing pattern; `client.go` already uses it. [VERIFIED: codebase] |
| Go stdlib `encoding/json` | 1.26 | `Tool.Parameters`, `ToolCall.Arguments`, `Capabilities` JSON tags | Established existing pattern; `client.go:9`. [VERIFIED: codebase] |
| Go stdlib `errors` | 1.26 | Sentinel errors (`ErrCapabilityNotSupported`, `ErrScriptExhausted`) | `errors.Is` is the public protocol for capability-gap detection. [VERIFIED: codebase] |
| Go stdlib `io` | 1.26 | `io.EOF` returned by `StreamReader.Next` | Idiomatic Go iterator-style sentinel; matches anthropic-sdk-go's `Stream.Err()` ergonomics directionally. [CITED: https://go.dev/ref/spec; codebase doc.go] |
| Go stdlib `sync` | 1.26 | ScriptedLLM v2 concurrency (mutex on cursor) | Existing `scriptedLLM` already uses `sync.Mutex`. [VERIFIED: scriptedllm_test.go:5] |

### Supporting

None added in Phase 0. Sister-repo `go.mod` files declare `require github.com/costa92/llm-agent v0.3.0-pre.1` only; no third-party deps yet (those land per-phase: providers in Phase 1, OTel in Phase 5, refsvc in Phase 6).

### Alternatives Considered

| Instead of | Could Use | Why Rejected |
|------------|-----------|--------------|
| Single `Capabilities` struct field on `ProviderInfo` (D-02) | Bitmask `Caps uint32` | D-02 ratifies the struct shape; bitmask is non-self-documenting and breaks JSON serializability for OTel attrs. |
| Same | Methods on `ProviderInfo` | Forces method dispatch where data + JSON suffices; harder to extend with non-bool fields like `MaxToolsPerCall int`. |
| `<-chan StreamEvent` like current `GenerateStream` | Iterator `Next() (StreamEvent, error)` | See `StreamReader Shape` decision below — iterator wins. |
| `iter.Seq2[StreamEvent, error]` (Go 1.23+ range-over-func) | Iterator with explicit Next/Close | Range-over-func leaks the close-on-cancel discipline; goroutine leak (Pitfall 3) gets harder to enforce. Iterator with explicit `Close()` is what anthropic-sdk-go and openai-go both ship. |
| New package `llm/v2/` | Reboot existing `llm/` (D-01) | D-01 ratifies the reboot. The `/v2` subpackage convention is for Go module major bumps, not for in-repo subpackage versioning. |
| ScriptedLLM remains in `_test.go` | Promote to `llm/scripted.go` (D-03) | D-03 ratifies promotion; sister-repo conformance suites must import it. |

**Installation:** No `go install` step in Phase 0. Verification command for the locked stack:

```bash
go vet ./... && go build ./... && go test ./...
# All four repos must pass. Core repo emits no go.sum (stdlib-only).
```

**Version verification:** No third-party packages in Phase 0 — nothing to verify against `npm view` / `pkg.go.dev`. Sister repos' `require` line points at `github.com/costa92/llm-agent v0.3.0-pre.1`, which gets tagged at the END of Phase 0 (see Open Question Q3 below).

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: Reboot the `llm/` package.** New types (`ChatModel`, `ToolCaller`, `Embedder`, `StructuredOutputs`, `StreamEvent`, `StreamReader`, `ProviderInfo`, `Capabilities`) live IN `llm/`. The current `llm.Client` is renamed to `llm.LegacyClient` with a `// Deprecated:` godoc comment naming `v0.4.0` as the target removal version.
  - Import path stays: `github.com/costa92/llm-agent/llm`.
  - Old types coexist in the same package, side-by-side with new types, until v0.4 removal.
  - Cascades into: CORE-01..09 file layout, migration guide diff examples, deprecation notices in CHANGELOG.

- **D-02: `Capabilities` is an embedded struct field on `ProviderInfo`** (NOT methods, NOT bitmask). Concrete shape:
  ```go
  type ProviderInfo struct {
      Provider     string       // "openai", "anthropic", "ollama"
      Model        string       // "gpt-4o-mini", "claude-3-5-haiku", "llama3.1:8b"
      Capabilities Capabilities
  }

  type Capabilities struct {
      Tools             bool
      Embeddings        bool
      StructuredOutputs bool
      PromptCaching     bool
  }
  ```
  Type assertion stays as PRIMARY signal at compile-time; `Capabilities` is the runtime signal for per-(provider × model) variation.

- **D-03: One full-capability `ScriptedLLM` v2 + small `ChatOnlyMock` for fallback testing.** ScriptedLLM is non-test code (lives in `llm/`); implements `ChatModel + ToolCaller + Embedder + StructuredOutputs`. ChatOnlyMock implements only `ChatModel`.

- **D-04: Phase 0 creates all 3 sister GitHub repos AND pushes the skeleton.** Repos: `github.com/costa92/llm-agent-providers`, `github.com/costa92/llm-agent-otel`, `github.com/costa92/llm-agent-customer-support`. Each gets `go.mod` + `LICENSE` + `OWNERS` + `README.md` + `.github/workflows/test.yml` + `.gitignore`.

### Claude's Discretion

- **Migration guide depth (CORE-09):** target 1 worked example for the Simple paradigm + a generic mapping table covering all type renames and method-signature shifts.
- **`StreamEvent` exact field names:** locked directionally by D-01/D-02 (typed Kind enum + per-tool-call Index + delta fields), but precise field naming is researcher/planner discretion — match Anthropic SDK's `partial_json` and OpenAI SDK's `function.arguments` ergonomics where possible.
- **Sister repo READMEs (INFRA-06):** Claude drafts; user reviews at PR time. Stick to: purpose, install command, cross-repo iteration pattern, link back to the core repo's CLAUDE.md.

### Deferred Ideas (OUT OF SCOPE)

- **Provider Author Guide v0.1** — explicitly Phase 1 (CORE-11). Phase 0 provides the *types* the guide will document.
- **Agent paradigm refactor (CORE-10)** — Phase 3. Phase 0 does NOT touch `simple.go`, `react.go`, etc.
- **OTel decorator implementation** — Phase 5. Phase 0 ensures interfaces COMPOSE under wrapping but does NOT write the decorator.
- **Conformance test harness** — Phase 1 (`internal/contract/`). Phase 0 promotes ScriptedLLM v2 so Phase 1 can import it.
- **Anti-features:** vision, RL training, K8s, cross-framework bridges remain Out of Scope per PROJECT.md and REQUIREMENTS.md.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | 4 sibling Go modules exist with their own `go.mod` | §"Sister Repo Skeleton Breakdown" — each sister repo's go.mod content + module path |
| INFRA-02 | `go.work` is `.gitignore`d in every repo; CI runs with `GOWORK=off` | §"File-by-file breakdown" core .gitignore additions; §"CI YAML Sketches" GOWORK=off env |
| INFRA-03 | A Makefile or shell script in each sister repo writes a sibling-aware `go.work` | §"Sister Repo Skeleton Breakdown" — `scripts/workspace.sh` content lifted from STACK.md |
| INFRA-04 | CI gate rejects `replace` directives on tagged-release branches | §"CI YAML Sketches" — release-precheck workflow with `go mod edit -json | jq` check |
| INFRA-05 | Umbrella CI in `llm-agent` builds all 4 repos against `llm-agent` HEAD on every PR | §"CI YAML Sketches" — umbrella.yml with checkout 4 repos + go.work init + go build/test |
| INFRA-06 | README in each sister repo documents the cross-repo iteration pattern | §"Sister Repo Skeleton Breakdown" — README.md content with `go.work` recommendation + replace escape hatch |
| INFRA-07 | Versioning policy documented across all 4 repos | §"File-by-file breakdown" — VERSIONING.md or section in each README; CHANGELOG `### Breaking` policy |
| CORE-01 | New `llm/` package defines `ChatModel` base interface (Generate + Stream) | §"Concrete Go Type Definitions" — `ChatModel` interface |
| CORE-02 | `ToolCaller` capability interface defines `WithTools(tools) ToolCaller` (immutable) | §"Concrete Go Type Definitions" — `ToolCaller` interface; explicit immutable doc comment |
| CORE-03 | `Embedder` capability interface defines `Embed(ctx, []string) ([]Vector, Usage, error)` | §"Concrete Go Type Definitions" — `Embedder` interface |
| CORE-04 | `StructuredOutputs` capability interface defines `WithSchema(schema) ChatModel` | §"Concrete Go Type Definitions" — `StructuredOutputs` interface |
| CORE-05 | Typed `StreamEvent` union with `Kind` enum + stable per-tool-call `Index` | §"Concrete Go Type Definitions" — `StreamEvent` struct + `StreamEventKind` enum + `ToolCallDelta.Index` |
| CORE-06 | `ProviderInfo` returned by `ChatModel.Info()` reflects the bound model's capabilities (per-(provider × model)) | §"Concrete Go Type Definitions" — `ProviderInfo` shape locked by D-02; `Info()` method on `ChatModel` |
| CORE-07 | Mock implementations (`ScriptedLLM`-style) for `ChatModel` + each capability | §"File-by-file breakdown" — `llm/scripted.go` + `llm/chat_only_mock.go` |
| CORE-08 | Existing `llm.Client` (v0.2 surface) remains callable, marked Deprecated with godoc + removal target | §"File-by-file breakdown" — `llm/legacy.go` rename + `// Deprecated:` comment template |
| CORE-09 | Migration guide in `docs/migration-v0.2-to-v0.3.md` — concrete diff examples | §"Migration Playbook" — Simple paradigm worked example + generic mapping table |

## Architecture Patterns

### System Architecture Diagram

```
                            User clones / GitHub PR
                                       │
                                       ▼
                  ┌────────────────────────────────────────┐
                  │  GitHub Actions on llm-agent PR        │
                  └──┬───────────────────┬─────────────────┘
                     │                   │
                     │                   │
         per-repo test.yml          umbrella.yml
         (this repo only)           (4 repos via go.work)
                     │                   │
                     │                   ├── checkout llm-agent (HEAD of PR)
                     │                   ├── checkout llm-agent-providers (main)
                     │                   ├── checkout llm-agent-otel (main)
                     │                   ├── checkout llm-agent-customer-support (main)
                     │                   ├── go work init (writes ../go.work)
                     │                   └── for each repo: cd <repo> && GOWORK=on go build/test ./...
                     │
                     ▼
            GOWORK=off go vet/build/test ./...
                     │
                     ▼
                 PR merge gated by both jobs green

       ╔════════════════════════════════════════════════════════╗
       ║          New `llm/` package (D-01 reboot)              ║
       ║                                                        ║
       ║   types.go   ── Message, Tool, ToolCall, Vector, Usage,║
       ║                  Request, Response, FinishReason       ║
       ║   info.go    ── ProviderInfo + Capabilities (D-02)     ║
       ║   chatmodel.go ── ChatModel (CORE-01)                  ║
       ║   capabilities.go ── ToolCaller (CORE-02),             ║
       ║                       Embedder (CORE-03),              ║
       ║                       StructuredOutputs (CORE-04)      ║
       ║   stream.go  ── StreamEvent, StreamEventKind,          ║
       ║                  ToolCallDelta, StreamReader (CORE-05) ║
       ║   errors.go  ── ErrCapabilityNotSupported,             ║
       ║                  ErrScriptExhausted, sentinel set      ║
       ║   scripted.go ── ScriptedLLM (CORE-07; full caps; D-03)║
       ║   chat_only_mock.go ── ChatOnlyMock (D-03)             ║
       ║   legacy.go  ── Client (renamed → LegacyClient,        ║
       ║                  CORE-08, // Deprecated:);             ║
       ║                  GenerateRequest, GenerateResponse,    ║
       ║                  StreamChunk, StreamUsage              ║
       ║   doc.go     ── package-level capability-negotiation   ║
       ║                  guide for adapter authors             ║
       ╚════════════════════════════════════════════════════════╝
```

Reader trace: a third-party adapter author clones `llm-agent`, reads `llm/doc.go` to learn the negotiation pattern, opens `llm/chatmodel.go` to see `ChatModel`, then `llm/capabilities.go` to see what they may optionally implement, then `llm/stream.go` for the streaming union, and finally `llm/scripted.go` to see a working reference implementation. The path from "what's the contract?" to "show me a working impl" is 5 files, ~400 LOC total.

### Recommended Project Structure

```
llm-agent/                          # core repo (this repo)
├── go.mod                          # stdlib-only, no go.sum
├── .gitignore                      # +go.work, +go.work.sum
├── LICENSE                         # unchanged
├── OWNERS                          # unchanged
├── CHANGELOG.md                    # +### Deprecated entry
├── DEPRECATIONS.md                 # NEW — Pitfall 15 enforcement; lists symbol → target version
├── docs/
│   └── migration-v0.2-to-v0.3.md   # NEW — CORE-09
├── llm/                            # rebooted package (D-01)
│   ├── doc.go                      # package overview + capability-negotiation guide
│   ├── chatmodel.go                # ChatModel
│   ├── capabilities.go             # ToolCaller, Embedder, StructuredOutputs
│   ├── stream.go                   # StreamEvent, StreamEventKind, StreamReader, ToolCallDelta
│   ├── info.go                     # ProviderInfo, Capabilities
│   ├── types.go                    # Message, Tool, ToolCall, Vector, Usage, Request, Response, FinishReason
│   ├── errors.go                   # ErrCapabilityNotSupported, ErrScriptExhausted, etc.
│   ├── scripted.go                 # ScriptedLLM (full-capability mock; promoted from _test.go)
│   ├── chat_only_mock.go           # ChatOnlyMock (capability-degrade testing)
│   ├── legacy.go                   # Client → LegacyClient (// Deprecated:) + old companion types
│   └── llm_test.go                 # interface satisfaction tests, ScriptedLLM tests
├── scripts/
│   └── workspace.sh                # NEW — go.work writer (INFRA-03 mirror; this repo runs the script too)
├── .github/
│   └── workflows/
│       ├── test.yml                # MODIFIED — add GOWORK=off env to existing steps
│       ├── umbrella.yml            # NEW — INFRA-05
│       └── release-precheck.yml    # NEW — INFRA-04
└── (rest of repo unchanged)
```

### Pattern 1: Small Interfaces + Type Assertion + ProviderInfo Hint

**What:** ChatModel is the smallest possible interface (Generate + Stream + Info); each capability is a separate interface that embeds ChatModel.
**When to use:** Always for v0.3 capability surface. This is K3.
**Source:** ARCHITECTURE.md §"Capability Negotiation Pattern" + Eino's BaseChatModel/ToolCallingChatModel pattern.

```go
// Caller idiom — exactly what Phase 3 agents will do:
if tc, ok := model.(llmpkg.ToolCaller); ok && model.Info().Capabilities.Tools {
    bound, err := tc.WithTools(tools)
    if err != nil { return err }
    return bound.Generate(ctx, req)
}
// Fall back to scratchpad templating
return model.Generate(ctx, scratchpadReq(req))
```

Both checks (type assertion AND `Capabilities.Tools`) are intentional: type assertion is the COMPILE-TIME signal ("does this concrete type expose the method?"); `Capabilities` is the RUNTIME signal ("for THIS bound model, does the capability actually work?"). Pitfall 6 documents why both are needed for Ollama (the Go type implements `ToolCaller` but `llama2` doesn't actually do tools — `Capabilities.Tools=false` for that model instance).

### Pattern 2: Functional WithTools / WithSchema (Immutable)

**What:** Capability-extending operations return a NEW model, never mutate the receiver.
**When to use:** Always for `WithTools`, `WithSchema`, or any future capability binding.
**Source:** ARCHITECTURE.md Pattern 2 + Eino's deprecated `BindTools` lesson (mutated state, race conditions).

```go
// CORRECT
bound, err := tc.WithTools(tools)  // bound is a NEW value
go bound.Generate(ctx, reqA)        // safe to call concurrently
go tc.WithTools(otherTools)         // independent — no shared mutable state

// WRONG (Eino BindTools mistake)
tc.BindTools(tools)                 // mutates receiver
go tc.Generate(ctx, reqA)           // race with sibling goroutine binding different tools
```

### Pattern 3: Iterator-Style StreamReader (Not `<-chan`)

**What:** `StreamReader` exposes `Next() (StreamEvent, error)` + `Close() error`. EOF is signaled by `io.EOF` from `Next`. NOT a `<-chan StreamEvent`.
**Why iterator over channel:**
1. **Cancellation semantics:** With a channel, the producer goroutine must observe `ctx.Done()` AND close the channel — two failure modes. With an iterator, `Close()` is the single explicit teardown call; `Next()` returns `ctx.Err()` directly when ctx is canceled.
2. **Error propagation:** Channels need a sidecar error field on every chunk OR a separate err channel; iterators return `(value, error)` from one call. Matches `bufio.Scanner`, `sql.Rows`, and the Anthropic SDK's `Stream.Next()` ergonomics.
3. **Goroutine leak prevention (Pitfall 3):** With `<-chan`, the producer goroutine survives until the channel is drained. If the consumer breaks out of the loop without draining (early return, error path), the goroutine leaks. Iterator with explicit `Close()` makes the cleanup contract obvious.
4. **Composes with retry SM (K4, Phase 2):** The retry state machine `Connecting → FirstByte → Streaming → Done` is easier to encode around `Next()` calls than around channel receives.

The current `llm.Client.GenerateStream` returns `<-chan StreamChunk` — this gets wrapped/replaced. `LegacyClient` retains it; new `ChatModel.Stream` returns `StreamReader`.

```go
type StreamReader interface {
    Next() (StreamEvent, error) // io.EOF when stream ends; ctx.Err() if cancelled
    Close() error               // idempotent; safe to call multiple times
}

// Caller idiom:
sr, err := model.Stream(ctx, req)
if err != nil { return err }
defer sr.Close()                  // ALWAYS, even on early return

for {
    ev, err := sr.Next()
    if errors.Is(err, io.EOF) { break }
    if err != nil { return err }  // includes ctx.Err()
    handle(ev)
}
```

### Anti-Patterns to Avoid

- **`<-chan StreamEvent` for the new interface:** revives Pitfall 3 footguns. Iterator is the chosen shape.
- **`Capabilities` as bitmask:** rejected by D-02. Struct fields are self-documenting and JSON-serializable.
- **Putting ScriptedLLM in `_test.go`:** rejected by D-03. Conformance suite (Phase 1) needs to import it.
- **Subpackage `llm/v2/`:** rejected by D-01. The `/v2` convention is for Go MODULE major bumps; in-repo subpackage versioning is non-idiomatic.
- **Modifying existing `Tool` / `ToolCall` / `Message` / `FinishReason` types:** they're already used by sister-repo-hostable code paths and the `LegacyClient`. Phase 0 keeps them unchanged in `legacy.go`; Phase 1 adapters import them or new equivalents from `llm/types.go`. Decision deferred to "Open Question Q1" below.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GitHub repo creation | Manual web-UI clicks | `gh repo create costa92/llm-agent-providers --public --add-readme=false` | Scriptable, idempotent, leaves audit trail. |
| `go.work` writer | Hand-write per-developer | `scripts/workspace.sh` (lifted verbatim from STACK.md §"Local development pattern" lines 148-161) | Already designed by research; just copy. |
| `replace` ban detection | Custom regex | `go mod edit -json \| jq '.Replace \| length' == 0` (lifted from PITFALLS Pitfall 12) | Official Go tooling parses go.mod; jq query is one line. |
| Stream iterator interface | Roll your own | Adopt anthropic-sdk-go's `Stream.Next() / Stream.Err() / Stream.Close()` shape | The major SDKs converged on this for a reason; Phase 1 adapters will match it natively. |
| Mock LLM | Inline anonymous struct | Promoted ScriptedLLM v2 (D-03) | Already half-built (in `scriptedllm_test.go`); promotion + capability-extension is mechanical. |
| Deprecation reminder | Just a code comment | `DEPRECATIONS.md` table-of-records (Pitfall 15) | The comment is invisible at release-tagging time; the file forces a sweep. |
| Sister-repo CI YAML | Reinvent | Mirror `.github/workflows/test.yml` shape from this repo | Existing test.yml already handles stdlib-only edge case (no go.sum); same logic applies to sister repos modulo go.sum-required-here. |

**Key insight:** Phase 0 has zero novel infrastructure. Every artifact has a precedent in this repo, in research/STACK.md, or in research/PITFALLS.md. The work is faithful translation of those precedents into actual Go files, YAML files, and gh CLI commands.

## Concrete Go Type Definitions for `llm/` Reboot

These are the planner's starting shapes. Field names follow Anthropic SDK's `partial_json` ergonomics for stream events and OpenAI SDK's `function.arguments` ergonomics for tool-call args, while preserving names from the existing `llm.Client` surface where it doesn't matter (`Provider`, `Model`, `FinishReason`, `Tool.Parameters`, `ToolCall.Arguments`).

### `llm/info.go` (CORE-06, ratifies D-02)

```go
package llm

// ProviderInfo describes a bound provider+model combination.
// Returned by ChatModel.Info(). Capabilities reflect THIS bound model,
// not the provider type generically (Pitfall 6).
type ProviderInfo struct {
    Provider     string       `json:"provider"`     // "openai", "anthropic", "ollama"
    Model        string       `json:"model"`        // "gpt-4o-mini", "claude-3-5-haiku", "llama3.1:8b"
    Capabilities Capabilities `json:"capabilities"`
}

// Capabilities is a value type — JSON-serializable for OTel attribute
// emission (gen_ai.provider.capabilities.tools etc., Phase 5).
// Per D-02, this is a struct (not a bitmask, not methods).
type Capabilities struct {
    Tools             bool `json:"tools"`               // Native function-calling supported
    Embeddings        bool `json:"embeddings"`          // Embed() returns vectors (NOT ErrNotSupported)
    StructuredOutputs bool `json:"structured_outputs"`  // WithSchema() applies a JSON schema constraint
    PromptCaching     bool `json:"prompt_caching"`      // Anthropic explicit / OpenAI auto
}
```

### `llm/chatmodel.go` (CORE-01)

```go
package llm

import "context"

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

**Decision: `Info()` lives on `ChatModel`, not on a separate `Identifier` interface.**

Rationale: every adapter implements `Info()` regardless of capability, and OTel decorators (Phase 5) need `Info()` on every wrapped value. Putting `Info()` on a separate interface forces a second type assertion at every span boundary — net loss. Being on `ChatModel` is the same shape openai-go uses for its own Client / model identity.

### `llm/capabilities.go` (CORE-02, CORE-03, CORE-04)

```go
package llm

import "context"

// ToolCaller is the capability for native tool/function-calling.
// WithTools is IMMUTABLE: it returns a new ToolCaller bound to the
// given tools; the receiver is unchanged. Safe for concurrent use
// (Pattern 2; rejects Eino's deprecated BindTools mutation pattern).
type ToolCaller interface {
    ChatModel
    WithTools(tools []Tool) (ToolCaller, error)
}

// Embedder is the capability for vector embeddings.
// Returns vectors in input order; len(vectors) == len(texts).
// Providers without embedding endpoints (Anthropic) do NOT implement
// this interface; callers detect via type assertion + Capabilities.Embeddings.
type Embedder interface {
    Embed(ctx context.Context, texts []string) (vectors []Vector, usage Usage, err error)
    EmbedDimensions() int
}

// StructuredOutputs is the capability for JSON-schema-constrained generation
// (OpenAI response_format, Anthropic tool-as-output trick).
// Like ToolCaller, WithSchema is IMMUTABLE.
type StructuredOutputs interface {
    ChatModel
    WithSchema(schema []byte) (ChatModel, error)
}
```

**Decision: `Embedder` does NOT embed `ChatModel`.** A pure embedding-only adapter (e.g., a future `voyageai` adapter) might implement `Embedder` without `ChatModel` — keeping them orthogonal preserves that option. By contrast, `ToolCaller` and `StructuredOutputs` are CHAT augmentations: `WithTools` returns "still a ChatModel, now with tools," so they MUST embed `ChatModel`. This split mirrors the conceptual reality (embeddings are a different operation, not a chat augmentation).

**Decision: `WithSchema(schema []byte)` returns `ChatModel`, not `StructuredOutputs`.** Calling `WithSchema` twice on the same value is meaningless (the second call would replace the first); the return type signals "this returned value is bound to a schema, you cannot meaningfully re-apply." Returning `ChatModel` keeps the call site honest. (Same justification could apply to `WithTools`, but `WithTools` returning `ToolCaller` enables agent-side patterns like `tc.WithTools(allowList).Generate(...)` where the returned tool-caller still advertises tool capability — useful for the K3 OTel rewrap.)

### `llm/stream.go` (CORE-05 — K1 keystone)

```go
package llm

import "io"

// StreamReader is the iterator-style interface for streaming responses.
// Next returns io.EOF when the stream ends cleanly, or ctx.Err() when
// the underlying context is cancelled. Close is idempotent and MUST
// be called by every consumer (typically via `defer sr.Close()`)
// to prevent goroutine leaks (Pitfall 3).
type StreamReader interface {
    Next() (StreamEvent, error)
    Close() error
}

// StreamEventKind enumerates the typed-union variants. Adapters emit
// their NATIVE granularity (OpenAI per-index deltas, Anthropic per-
// content-block deltas, Ollama whole-tool-call). Consumers that don't
// care about granularity use AccumulateStream below.
type StreamEventKind uint8

const (
    EventTextDelta         StreamEventKind = iota // adapter emitted text
    EventToolCallStart                            // tool_call begins; ToolCall.{Index, ID, Name} known
    EventToolCallArgsDelta                        // partial args JSON for an in-flight tool_call
    EventToolCallEnd                              // tool_call complete; consumer may dispatch
    EventThinkingDelta                            // optional; reasoning models / Anthropic
    EventDone                                     // terminal; Usage + FinishReason populated
)

// StreamEvent is the typed union. Field population is gated by Kind:
//
//   Kind = EventTextDelta:         Text != ""
//   Kind = EventToolCallStart:     ToolCall != nil; ToolCall.{Index,ID,Name} populated
//   Kind = EventToolCallArgsDelta: ToolCall != nil; ToolCall.{Index, ArgsDelta} populated
//   Kind = EventToolCallEnd:       ToolCall != nil; ToolCall.Index populated
//   Kind = EventThinkingDelta:     Text != ""
//   Kind = EventDone:               Usage != nil; FinishReason != ""
type StreamEvent struct {
    Kind         StreamEventKind
    Text         string         // EventTextDelta, EventThinkingDelta
    ToolCall     *ToolCallDelta // EventToolCall* kinds
    Usage        *Usage         // EventDone (when provider reports it)
    FinishReason FinishReason   // EventDone
}

// ToolCallDelta carries per-tool-call streaming state.
// Index is the STABLE per-tool-call key — across all chunks for a single
// tool call, Index is identical. This is what the agent-layer accumulator
// joins by, NOT Name (Pitfall 1: "OpenAI streaming tool_calls — losing
// chunks because you keyed by name instead of index").
//
// ID is the provider-side identifier (OpenAI tool_call_id, Anthropic
// content_block id) — used by the agent dedupe layer (Phase 3) keyed by
// (message_id, tool_use_id).
//
// Name is populated ONCE on the EventToolCallStart event for that Index.
// ArgsDelta is the partial JSON string; concatenation across chunks for
// a given Index yields the final arguments JSON (matches OpenAI's
// function.arguments delta string and Anthropic's input_json_delta.partial_json).
type ToolCallDelta struct {
    Index     int    // stable across chunks for a single tool call
    ID        string // provider-assigned ID; empty until provider emits it
    Name      string // populated on EventToolCallStart
    ArgsDelta string // partial JSON; concat all deltas for this Index to get final args
}

// AccumulateStream is a helper for consumers that don't care about
// streaming granularity — drains sr to completion and returns the
// equivalent non-streaming Response. Closes sr on exit.
func AccumulateStream(sr StreamReader) (Response, error) { /* ... */ }
```

**Decisions on field naming:**

- `ArgsDelta` (not `Args` or `Arguments`): emphasizes that the field is INCREMENTAL. Final args = concat of all `ArgsDelta` for a given `Index`. Matches OpenAI's `function.arguments` (which is also a delta string in streams) and Anthropic's `input_json_delta.partial_json` (literally a partial JSON string).
- `Index int` (not `uint`, not `ToolCallIndex` named type): OpenAI's API returns `int` (`tool_calls[].index`), Anthropic's returns `int` (`content_block` index). Adding a named type adds noise without preventing any real bug.
- Single `StreamEvent` (not `StreamEvent` + separate `ToolCallEvent`): the `Kind` enum carries the variant; pointer fields `ToolCall *ToolCallDelta` and `Usage *Usage` keep the struct cheap when not used. Two separate types would force the consumer to handle two channels / two readers — exactly the lowest-common-denominator anti-pattern from ARCHITECTURE.md.

### `llm/types.go` (shared Request/Response/Tool/Message types)

```go
package llm

import "encoding/json"

// Request is the new-surface request type. Replaces GenerateRequest at
// the new-interface layer; LegacyClient still uses GenerateRequest.
type Request struct {
    Messages         []Message      `json:"messages"`             // multi-turn dialog (preferred over Prompt)
    SystemPrompt     string         `json:"system_prompt,omitempty"` // lifted out of Messages for Anthropic top-level system
    MaxOutputTokens  int            `json:"max_output_tokens,omitempty"`
    Temperature      *float32       `json:"temperature,omitempty"` // pointer: nil = use provider default
    Metadata         map[string]any `json:"metadata,omitempty"`    // provider-specific extras (rare; prefer typed)
}

// Response is the new-surface response type.
type Response struct {
    Text         string       `json:"text"`
    FinishReason FinishReason `json:"finish_reason,omitempty"`
    Provider     string       `json:"provider"`
    Model        string       `json:"model,omitempty"`
    Usage        Usage        `json:"usage"`
    ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
}

// Message is a single turn in a conversation. Reused unchanged from
// LegacyClient — same Role/Content shape; system messages are lifted
// to Request.SystemPrompt before sending.
type Message struct {
    Role    string `json:"role"`    // "user", "assistant", "tool"
    Content string `json:"content"`
}

// Tool declares a function the model may call. Same as LegacyClient's Tool.
type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is what the model returns when it decides to invoke a Tool.
// Adds ID (vs LegacyClient's ToolCall) for dedupe (Pitfall 4: tool dedupe
// by (message_id, tool_use_id)).
type ToolCall struct {
    ID        string          `json:"id"`        // provider-assigned (NEW vs legacy)
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

// Vector is one embedding. Length matches Embedder.EmbedDimensions().
type Vector []float32

// Usage carries token accounting for one request. Source distinguishes
// reported (provider returned actual counts), estimated (computed from
// tokenizer), and unknown (mid-stream abort, no usage available).
// Source != "" is an invariant after Phase 2 lands (K4); for Phase 0
// the Source field exists but defaults to UsageUnknown.
type Usage struct {
    InputTokens  int         `json:"input_tokens"`
    OutputTokens int         `json:"output_tokens"`
    TotalTokens  int         `json:"total_tokens,omitempty"`
    Source       UsageSource `json:"source,omitempty"`
}

type UsageSource string

const (
    UsageReported  UsageSource = "reported"
    UsageEstimated UsageSource = "estimated"
    UsageUnknown   UsageSource = "unknown"
)

// FinishReason — same constants as LegacyClient; share underlying string type.
type FinishReason = legacyFinishReason  // alias, see legacy.go
```

**Decision on type sharing with LegacyClient:** `Message`, `Tool`, `FinishReason` constants stay identical; `ToolCall` ADDS an `ID` field but is structurally compatible (the `ID` is empty in v0.2 callers, who don't construct ToolCalls — only the LLM does, and LegacyClient adapters don't populate ID). `GenerateRequest` and `GenerateResponse` are reused only by `LegacyClient`; new code uses `Request` and `Response`.

### `llm/errors.go`

```go
package llm

import "errors"

// ErrCapabilityNotSupported is returned by methods on capability
// interfaces when the bound model does not actually support the
// capability — even though the Go type implements the interface.
//
// Canonical wrap pattern:
//   return nil, fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported)
//
// Callers detect with errors.Is(err, llm.ErrCapabilityNotSupported).
var ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")

// ErrScriptExhausted is returned by ScriptedLLM when the script runs
// out of pre-recorded responses. Test code matches with errors.Is.
var ErrScriptExhausted = errors.New("llm: scripted llm: script exhausted")
```

### `llm/legacy.go` (CORE-08)

```go
package llm

import (
    "context"
    "encoding/json"
)

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

// GenerateRequest, GenerateResponse, StreamChunk, StreamUsage —
// retained UNCHANGED from v0.2 so existing callers compile without
// edits. New code should use llm.Request / llm.Response / llm.StreamEvent.
//
// Deprecated: Use llm.Request instead.
type GenerateRequest struct {
    Prompt  string         `json:"prompt"`
    Context map[string]any `json:"context,omitempty"`
    Tools   []Tool         `json:"tools,omitempty"`
    History []Message      `json:"history,omitempty"`
}

// (... GenerateResponse, StreamChunk, StreamUsage all retained verbatim ...)

// legacyFinishReason is the underlying string type for FinishReason;
// see types.go for the alias and constants.
type legacyFinishReason string
```

**Decision: `type Client = LegacyClient` (alias).** This means EVERY existing caller that wrote `var c llm.Client = ...` continues to compile with ZERO changes during Phase 0. Without the alias, those callers would see "undefined: llm.Client" the moment Phase 0 lands. With the alias, the rename is purely additive: new code writes `llm.LegacyClient`, old code keeps writing `llm.Client`, both resolve to the same type. The alias itself is `// Deprecated:` so godoc surfaces the message.

This decision has a downside: it weakens the deprecation pressure (callers don't get a "you must rename" signal). Mitigation: `DEPRECATIONS.md` lists `Client` and `LegacyClient` both with target v0.4.0 removal; Phase 7 audit greps for both names.

### `llm/scripted.go` (CORE-07, D-03 ScriptedLLM v2)

```go
package llm

import (
    "context"
    "errors"
    "fmt"
    "io"
    "sync"
)

// ScriptedLLM is a deterministic full-capability mock. It implements
// ChatModel + ToolCaller + Embedder + StructuredOutputs and is used
// across the umbrella as the canonical reference: agent unit tests
// (this repo), conformance baseline (sister repos, Phase 1), example
// programs.
//
// Construction:
//   m := llm.NewScriptedLLM(
//       llm.WithProvider("scripted"),
//       llm.WithModel("test-1"),
//       llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true}),
//       llm.WithResponses(
//           llm.TextResponse("hello"),
//           llm.ToolCallResponse("calc", `{"a":2,"b":3}`),
//       ),
//   )
type ScriptedLLM struct {
    mu       sync.Mutex
    provider string
    model    string
    caps     Capabilities
    cursor   int
    resps    []Response
    embeds   [][]Vector // per-call batch responses for Embed
    tools    []Tool     // bound by WithTools (returns new ScriptedLLM)
}

// NewScriptedLLM constructs a ScriptedLLM with functional options.
// Default Capabilities = all true (full-capability default; Phase 3
// agent fallback tests use ChatOnlyMock instead).
func NewScriptedLLM(opts ...ScriptedOption) *ScriptedLLM { /* ... */ }

// Compile-time interface satisfaction.
var (
    _ ChatModel         = (*ScriptedLLM)(nil)
    _ ToolCaller        = (*ScriptedLLM)(nil)
    _ Embedder          = (*ScriptedLLM)(nil)
    _ StructuredOutputs = (*ScriptedLLM)(nil)
)

func (s *ScriptedLLM) Generate(ctx context.Context, req Request) (Response, error) { /* ... */ }
func (s *ScriptedLLM) Stream(ctx context.Context, req Request) (StreamReader, error) { /* synth StreamReader from current Response */ }
func (s *ScriptedLLM) Info() ProviderInfo {
    return ProviderInfo{Provider: s.provider, Model: s.model, Capabilities: s.caps}
}
func (s *ScriptedLLM) WithTools(tools []Tool) (ToolCaller, error) {
    cp := *s
    cp.tools = tools
    return &cp, nil
}
func (s *ScriptedLLM) Embed(ctx context.Context, texts []string) ([]Vector, Usage, error) { /* ... */ }
func (s *ScriptedLLM) EmbedDimensions() int { /* ... */ }
func (s *ScriptedLLM) WithSchema(schema []byte) (ChatModel, error) {
    cp := *s
    return &cp, nil  // ScriptedLLM doesn't validate schemas; honors the call as a no-op
}

// Functional options
type ScriptedOption func(*ScriptedLLM)
func WithProvider(p string) ScriptedOption { return func(s *ScriptedLLM) { s.provider = p } }
// ... etc.

// Convenience constructors for Response values
func TextResponse(text string) Response { /* ... */ }
func ToolCallResponse(name, argsJSON string) Response { /* ... */ }
```

### `llm/chat_only_mock.go` (D-03 ChatOnlyMock)

```go
package llm

import "context"

// ChatOnlyMock implements ONLY ChatModel — no ToolCaller, no Embedder,
// no StructuredOutputs. Used in Phase 3 agent tests to verify graceful
// capability degradation (ReAct falls back to scratchpad templating
// when model.(ToolCaller) fails).
type ChatOnlyMock struct {
    Provider string
    Model    string
    Resp     Response
}

// Compile-time: ONLY ChatModel — explicitly NOT the others.
var _ ChatModel = (*ChatOnlyMock)(nil)

func (m *ChatOnlyMock) Generate(ctx context.Context, req Request) (Response, error) { return m.Resp, nil }
func (m *ChatOnlyMock) Stream(ctx context.Context, req Request) (StreamReader, error) { /* synth from m.Resp */ }
func (m *ChatOnlyMock) Info() ProviderInfo {
    return ProviderInfo{
        Provider: m.Provider,
        Model:    m.Model,
        Capabilities: Capabilities{}, // ALL false — that's the point
    }
}
```

### `llm/llm_test.go`

```go
package llm

import (
    "context"
    "testing"
)

// TestInterfaceSatisfaction proves at compile time that the mocks
// implement the interfaces they claim. The actual var _ ... assertions
// are in scripted.go and chat_only_mock.go; this test holds the
// REVERSE assertion: ChatOnlyMock does NOT implement ToolCaller etc.
func TestChatOnlyMockExcludesCapabilities(t *testing.T) {
    var m ChatModel = &ChatOnlyMock{}
    if _, ok := m.(ToolCaller); ok {
        t.Fatal("ChatOnlyMock must not implement ToolCaller")
    }
    if _, ok := m.(Embedder); ok {
        t.Fatal("ChatOnlyMock must not implement Embedder")
    }
    if _, ok := m.(StructuredOutputs); ok {
        t.Fatal("ChatOnlyMock must not implement StructuredOutputs")
    }
}

// TestScriptedLLMSatisfiesAllCapabilities is enforced at compile time
// by var _ assertions; this test confirms runtime behaviour for each.
func TestScriptedLLM_Capabilities(t *testing.T) { /* table test per capability */ }

// TestStreamReaderClosesIdempotent — sr.Close() called twice does not panic.
func TestStreamReaderClosesIdempotent(t *testing.T) { /* ... */ }

// TestLegacyClientAlias — type Client = LegacyClient (D-01 source compatibility).
func TestLegacyClientAlias(t *testing.T) {
    var _ Client = (LegacyClient)(nil)
    var _ LegacyClient = (Client)(nil)
}
```

## File-by-File Breakdown — `llm-agent` Core Repo

### NEW files

| Path | Purpose | Approximate LOC |
|------|---------|-----------------|
| `llm/chatmodel.go` | `ChatModel` interface (CORE-01) | 30 |
| `llm/capabilities.go` | `ToolCaller`, `Embedder`, `StructuredOutputs` (CORE-02..04) | 40 |
| `llm/stream.go` | `StreamReader`, `StreamEvent`, `StreamEventKind`, `ToolCallDelta`, `AccumulateStream` (CORE-05) | 100 |
| `llm/info.go` | `ProviderInfo`, `Capabilities` (CORE-06; ratifies D-02) | 30 |
| `llm/types.go` | `Request`, `Response`, `Message`, `Tool`, `ToolCall`, `Vector`, `Usage`, `UsageSource`, `FinishReason` alias | 80 |
| `llm/errors.go` | `ErrCapabilityNotSupported`, `ErrScriptExhausted` | 15 |
| `llm/scripted.go` | `ScriptedLLM` v2 (full caps; CORE-07; D-03) | 200 |
| `llm/chat_only_mock.go` | `ChatOnlyMock` (capability-degrade testing; D-03) | 50 |
| `llm/doc.go` | Package overview + capability-negotiation guide for adapter authors | 80 |
| `llm/llm_test.go` | Interface satisfaction tests, ScriptedLLM tests | 200 |
| `docs/migration-v0.2-to-v0.3.md` | CORE-09: worked example (Simple) + mapping table | see Migration Playbook |
| `DEPRECATIONS.md` | Pitfall 15 enforcement: symbol → target version | 20 |
| `scripts/workspace.sh` | INFRA-03 — go.work writer | 20 (lifted from STACK.md) |
| `.github/workflows/umbrella.yml` | INFRA-05 — 4-repo build on every PR | 80 |
| `.github/workflows/release-precheck.yml` | INFRA-04 — `replace` ban on tagged-release branches | 40 |

### MODIFIED files

| Path | Change |
|------|--------|
| `llm/client.go` | RENAMED to `llm/legacy.go`. `type Client interface { ... }` → `type LegacyClient interface { ... }` + `type Client = LegacyClient` alias. All `// Deprecated:` godoc added. Type bodies unchanged. The companion types (`GenerateRequest`, `GenerateResponse`, `StreamChunk`, `StreamUsage`) move with it. `FinishReason` stays accessible (alias: `type FinishReason = legacyFinishReason`). `Tool`, `ToolCall`, `Message` — moved to `types.go`. (See Migration Playbook §"renames per-symbol".) |
| `scriptedllm_test.go` | Becomes a stub: keep the file (Go doesn't allow deleting test-only entry points without restructuring) but remove the type. Replace with a compile-time `_ = llm.ScriptedLLM{}` smoke test that verifies the new path works in this package's tests. The actual `scriptedLLM`/`newScriptedLLM`/`textResp` helpers used by `simple_test.go`, `react_test.go`, etc. become thin shims that delegate to `llm.ScriptedLLM`. (See Migration Playbook §"test-helper migration".) |
| `.github/workflows/test.yml` | Add `env: GOWORK: off` at job level (INFRA-02 — ensures CI never silently picks up a developer's workspace if one ever lands). No other changes. |
| `.gitignore` | Add `go.work` and `go.work.sum` (Pitfall 13; INFRA-02). |
| `CHANGELOG.md` | Add `## [Unreleased]` section with `### Deprecated` entry: `llm.Client` and `llm.LegacyClient` will be removed in v0.4.0; link to migration guide. |
| `examples/scriptedllm/scriptedllm.go` | RENAMED references: `llm.Client` → `llm.LegacyClient` (or no change if alias is preserved). Verify with `cd examples && go build ./...`. |
| `agent.go`, `simple.go`, `react.go`, `function_call.go`, `reflection.go`, `plan_solve.go` | UNCHANGED in Phase 0. They still consume `llm.Client` (now alias for `llm.LegacyClient`). Migration to `llm.ChatModel` is Phase 3 (CORE-10). |
| `rag/rag.go`, `bench/judge.go`, `bench/winrate.go`, `context/builder.go`, `rl/trainer_proxy.go`, `tool.go`, `registry.go` | UNCHANGED in Phase 0. Same reason. |

### UNCHANGED but reviewed

All other repo files are unaffected. Specifically: `orchestrate/`, `memory/`, `comm/`, `builtin/` don't reference `llm.*` types directly (verified via grep — only the listed files do).

## Sister Repo Skeleton Breakdown

Each sister repo gets the following files at the listed paths. All three repos are GitHub-public from day one (D-04).

### `llm-agent-providers` (skeleton)

```
llm-agent-providers/
├── go.mod
├── LICENSE
├── OWNERS
├── README.md
├── .gitignore
├── scripts/
│   └── workspace.sh
└── .github/
    └── workflows/
        ├── test.yml
        └── release-precheck.yml
```

**`go.mod`:**
```
module github.com/costa92/llm-agent-providers

go 1.26.0

require github.com/costa92/llm-agent v0.3.0-pre.1
```

**`LICENSE`:** Identical text to core repo's `LICENSE` (MIT, `Copyright (c) 2026 costa92`). Same year, same owner.

**`OWNERS`:**
```
# OWNERS — github.com/costa92/llm-agent-providers
#
# Code review and approval for the Go LLM agents framework's provider adapters
# (OpenAI / Anthropic / Ollama).
#
# Format: https://www.kubernetes.io/docs/contribute/participate/roles-and-responsibilities/

approvers:
  - costa92

reviewers:
  - costa92

labels:
  - area/providers
```

**`README.md`:** ~80 lines covering:
1. **Purpose:** "Provider adapters for `github.com/costa92/llm-agent`. Each adapter implements the capability interfaces from `llm-agent/llm` (ChatModel, ToolCaller, Embedder, StructuredOutputs)."
2. **Status:** "Phase 0 skeleton. Provider implementations land in Phases 1–4 per [llm-agent ROADMAP.md](https://github.com/costa92/llm-agent/blob/main/.planning/ROADMAP.md)."
3. **Install:** `go get github.com/costa92/llm-agent-providers/openai@v0.1.0` (note: tag will exist after Phase 1).
4. **Cross-repo iteration pattern (INFRA-06):**
   - Recommended: clone all 4 repos as siblings, run `./scripts/workspace.sh` from the parent directory, develop with `go.work`. Workspace is `.gitignore`d everywhere.
   - Escape hatch (NEVER tagged): `go mod edit -replace=github.com/costa92/llm-agent=../llm-agent` for ad-hoc iteration. `release-precheck.yml` rejects this on tagged-release branches.
5. **Versioning:** `v0.1.x` track for v0.3.x of core; bumps in lockstep with breaking core changes.
6. **Link to core:** "See [llm-agent CLAUDE.md](https://github.com/costa92/llm-agent/blob/main/CLAUDE.md) for the project's hard rules and design guide."

**`.gitignore`:**
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

# Multi-repo workspace (Pitfall 13)
go.work
go.work.sum
```

**`scripts/workspace.sh`:** Verbatim copy of the script in STACK.md §"Local development pattern" (lines 148-161 of ARCHITECTURE.md / STACK.md), unchanged. Each sister repo ships an identical copy so any of the 4 working dirs can bootstrap the workspace.

**`.github/workflows/test.yml`:** See §"CI YAML Sketches" below.

**`.github/workflows/release-precheck.yml`:** See §"CI YAML Sketches" below.

### `llm-agent-otel` (skeleton)

Same structure as `llm-agent-providers`, with these substitutions:

- `go.mod` module path: `github.com/costa92/llm-agent-otel`
- `OWNERS` `labels:` → `area/otel`
- `README.md` purpose: "OpenTelemetry decorator wrappers for `github.com/costa92/llm-agent`. `otelmodel.Wrap(ChatModel) ChatModel` + `otelagent.Wrap(Agent) Agent`. Lands in Phase 5."

### `llm-agent-customer-support` (skeleton)

Same structure, with:

- `go.mod` module path: `github.com/costa92/llm-agent-customer-support` (only requires `llm-agent v0.3.0-pre.1` for now; sister-repo deps added in Phase 6)
- `OWNERS` `labels:` → `area/refsvc`
- `README.md` purpose: "Reference customer-support service built on `llm-agent` + provider adapters + OTel adapter. **Demo only — production deployment requires hardening.** Lands in Phase 6."
- README banner: "**K8s manifests are NOT part of v0.3.** See [Pitfall 16](https://github.com/costa92/llm-agent/blob/main/.planning/research/PITFALLS.md#pitfall-16) for rationale."

## CI YAML Sketches

### Per-repo `test.yml` (sister repos)

Mirrors `llm-agent/.github/workflows/test.yml` but accommodates `go.sum` (sister repos take third-party deps; core does not).

```yaml
# llm-agent-providers/.github/workflows/test.yml
# (identical in llm-agent-otel and llm-agent-customer-support)
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
      - name: go mod tidy (drift check)
        run: |
          go mod tidy
          drift=$(git status --porcelain go.mod go.sum 2>/dev/null)
          if [ -n "$drift" ]; then
            echo "go mod tidy changed files — commit tidy changes first"
            echo "$drift"
            git --no-pager diff -- go.mod go.sum || true
            exit 1
          fi
      - name: go vet
        run: go vet ./...
      - name: go build
        run: go build ./...
      - name: go test
        run: go test ./...
```

Notes:
- `GOWORK: off` is at job-level `env:` so every `go` invocation in every step inherits it. The Go toolchain reads `GOWORK` from env regardless of subcommand. (Setting it on individual `go` calls would still work but is repetitive.)
- `cache: true` works fine for sister repos (they have go.sum); for core repo (no go.sum) it falls back gracefully.
- Phase 0 sister repos have NO Go source files yet (skeletons only). `go vet ./...` and `go build ./...` against a module with zero packages still succeed with exit 0 in Go 1.26 — empty modules are valid. (Verified: `go help vet` documents `./...` as expanding to "all packages in the current module," empty set is fine.)

### Umbrella CI (`llm-agent/.github/workflows/umbrella.yml`)

```yaml
# llm-agent/.github/workflows/umbrella.yml
# INFRA-05: builds all 4 repos against llm-agent HEAD on every PR.
name: umbrella

on:
  pull_request:
    branches: [main]
  workflow_dispatch:

concurrency:
  group: umbrella-${{ github.ref }}
  cancel-in-progress: true

jobs:
  cross-repo-build:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - name: Checkout llm-agent (this PR)
        uses: actions/checkout@v4
        with:
          path: llm-agent

      - name: Checkout llm-agent-providers (main)
        uses: actions/checkout@v4
        with:
          repository: costa92/llm-agent-providers
          path: llm-agent-providers
          ref: main

      - name: Checkout llm-agent-otel (main)
        uses: actions/checkout@v4
        with:
          repository: costa92/llm-agent-otel
          path: llm-agent-otel
          ref: main

      - name: Checkout llm-agent-customer-support (main)
        uses: actions/checkout@v4
        with:
          repository: costa92/llm-agent-customer-support
          path: llm-agent-customer-support
          ref: main

      - uses: actions/setup-go@v5
        with:
          go-version-file: llm-agent/go.mod
          cache: false

      - name: Initialize go.work pointing all 4 modules at this PR's llm-agent
        run: |
          go work init ./llm-agent ./llm-agent-providers ./llm-agent-otel ./llm-agent-customer-support
          # All 4 modules now resolve github.com/costa92/llm-agent to ./llm-agent
          # (this PR's HEAD), regardless of the version pinned in their go.mod files.

      - name: Build llm-agent (this PR)
        run: |
          cd llm-agent
          go vet ./...
          go build ./...
          go test ./...

      - name: Build llm-agent-providers against this PR's llm-agent
        run: |
          cd llm-agent-providers
          go build ./...
          go test ./...

      - name: Build llm-agent-otel against this PR's llm-agent
        run: |
          cd llm-agent-otel
          go build ./...
          go test ./...

      - name: Build llm-agent-customer-support against this PR's llm-agent
        run: |
          cd llm-agent-customer-support
          go build ./...
          go test ./...
```

**Notes:**
- `go.work` is created INSIDE the GHA runner working directory (the parent of all 4 checkouts). It is NEVER committed; the runner is ephemeral.
- `actions/checkout@v4` with `repository:` works for public repos without auth; sister repos are public per D-04.
- The umbrella job runs only on PR (and manual `workflow_dispatch`), NOT on every push to main — to keep main-branch CI fast. Cron-trigger could be added later if drift is observed.
- This trigger is on the `llm-agent` PR. Symmetric umbrella jobs on sister-repo PRs are NOT in Phase 0 — those would build sister-repo HEAD against `llm-agent` main. Defer until needed (likely Phase 1 when first adapter ships).

### Release-precheck (`release-precheck.yml`, identical in all 4 repos)

```yaml
# .github/workflows/release-precheck.yml
# INFRA-04: rejects `replace` directives on tagged-release branches.
# Lifted directly from PITFALLS.md Pitfall 12 §"How to avoid".
name: release-precheck

on:
  push:
    branches:
      - 'release/**'
  pull_request:
    branches:
      - 'release/**'

jobs:
  no-replace:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Reject `replace` directives in go.mod
        run: |
          replace_count=$(go mod edit -json | python3 -c '
          import json, sys
          data = json.load(sys.stdin)
          replaces = data.get("Replace") or []
          print(len(replaces))
          ')
          if [ "$replace_count" -ne 0 ]; then
            echo "::error::go.mod contains $replace_count replace directive(s) — these MUST be removed before tagging."
            echo "Detail:"
            go mod edit -json | python3 -m json.tool | grep -A 100 '"Replace"'
            exit 1
          fi
          echo "OK: go.mod has zero replace directives."
```

**Notes on implementation choice:**
- Pitfall 12 suggests `jq` (`go mod edit -json | jq '.Replace | length'`). I substituted Python because GitHub's `ubuntu-latest` runner ships Python 3 by default and avoids the conditional `apt-get install jq`. Both work; either is acceptable.
- Branch pattern `release/**` matches `release/v0.3.0`, `release/0.3.x`, etc. Tag-trigger (`on: push: tags:`) is an alternative but less ergonomic — the gate fires AFTER tag, when undoing is harder. Branch-trigger before tag is the right phase.
- This workflow is identical across all 4 repos. Could be a reusable workflow (`uses: costa92/llm-agent/.github/workflows/release-precheck.yml@main`) but reusable workflows add cross-repo dependency that's exactly the kind of subtle coupling Phase 0 is trying to avoid. Copy-paste 4 times is fine.

## Migration Playbook

### `docs/migration-v0.2-to-v0.3.md` Outline

```markdown
# Migrating from v0.2 to v0.3

The v0.3 release adds capability-aware interfaces (`llm.ChatModel`, `llm.ToolCaller`,
`llm.Embedder`, `llm.StructuredOutputs`) alongside the existing `llm.Client` (now
`llm.LegacyClient`). The old surface remains callable through v0.3.x; **it will be
removed in v0.4.0**.

## Quick reference: type renames

| v0.2 | v0.3 (new code) | v0.3 (legacy callers — alias) |
|------|-----------------|-------------------------------|
| `llm.Client` | `llm.ChatModel` | `llm.LegacyClient` (or `llm.Client`, alias) |
| `llm.Client.Generate(ctx, GenerateRequest) (GenerateResponse, error)` | `llm.ChatModel.Generate(ctx, Request) (Response, error)` | unchanged via alias |
| `llm.Client.GenerateStream(ctx, GenerateRequest) (<-chan StreamChunk, error)` | `llm.ChatModel.Stream(ctx, Request) (StreamReader, error)` | unchanged via alias |
| `llm.GenerateRequest` | `llm.Request` | unchanged via alias |
| `llm.GenerateResponse` | `llm.Response` | unchanged via alias |
| `llm.StreamChunk` (single channel value) | `llm.StreamEvent` (typed union with `Kind`) | unchanged via alias |
| (no equivalent) | `llm.ProviderInfo` returned by `Info()` | n/a — new in v0.3 |
| (no equivalent) | `llm.Capabilities` (struct on ProviderInfo) | n/a — new in v0.3 |
| `llm.Tool` | `llm.Tool` (unchanged shape) | unchanged |
| `llm.ToolCall` (Name + Arguments) | `llm.ToolCall` (now: ID + Name + Arguments) | unchanged shape; ID added |
| `llm.Message` | `llm.Message` (unchanged shape) | unchanged |
| `llm.FinishReason` + constants | unchanged | unchanged |

## Worked example: Simple paradigm

### v0.2 (current)
\`\`\`go
client := scriptedllm.New(scriptedllm.Text("hello"))
agent := agents.NewSimpleAgent(client, agents.SimpleOptions{})
res, err := agent.Run(ctx, "hi")
\`\`\`

### v0.3 transitional (no code change required — alias preserves source compatibility)
\`\`\`go
// SAME CODE WORKS; llm.Client is now an alias for llm.LegacyClient,
// and scriptedllm.New still returns the legacy contract for now.
client := scriptedllm.New(scriptedllm.Text("hello"))
agent := agents.NewSimpleAgent(client, agents.SimpleOptions{})
res, err := agent.Run(ctx, "hi")
\`\`\`

### v0.3 idiomatic (recommended for new code; `agents.NewSimpleAgent` accepts
ChatModel after Phase 3 / CORE-10 lands; until then stay on the legacy path)
\`\`\`go
// Phase 3+ — uses the new ChatModel directly.
model := llm.NewScriptedLLM(
    llm.WithProvider("scripted"),
    llm.WithModel("test-1"),
    llm.WithResponses(llm.TextResponse("hello")),
)
agent := agents.NewSimpleAgent(model, agents.SimpleOptions{})
res, err := agent.Run(ctx, "hi")
\`\`\`

## When to migrate

- **Now (v0.3.0):** new provider adapter authors target `llm.ChatModel` directly.
  Existing callers do nothing — alias preserves compilation.
- **Phase 3 (~v0.3.3):** internal agents migrate to `llm.ChatModel`. Examples + tests
  follow.
- **v0.4.0 (one minor cycle later):** `llm.Client` and `llm.LegacyClient` removed.
  All remaining callers MUST migrate before this tag.
```

### Internal `llm.Client` user inventory (verified by grep)

| File | Symbols Used | Migration Diff at Phase 0 | Migration Diff at Phase 3 (CORE-10) |
|------|--------------|---------------------------|--------------------------------------|
| `simple.go` | `llm.Client`, `llm.GenerateRequest` | NONE (alias preserves) | Field type changes to `llm.ChatModel`; `Generate(ctx, llm.GenerateRequest{Prompt})` → `Generate(ctx, llm.Request{Messages: []llm.Message{{Role:"user", Content:input}}})` |
| `react.go` | `llm.Client`, `llm.GenerateRequest` | NONE (alias) | Same shape as simple.go + capability negotiation (type-assert `ToolCaller`) |
| `function_call.go` | `llm.Client`, `llm.GenerateRequest`, `llm.GenerateResponse.ToolCalls` | NONE (alias) | Type change + REQUIRES `ToolCaller` (fail-fast at construction) |
| `reflection.go` | `llm.Client` | NONE (alias) | Type change to `ChatModel` |
| `plan_solve.go` | `llm.Client`, request types | NONE (alias) | Same as react.go |
| `tool.go` | `llm.Tool` (utility `AsLLMTool`) | NONE — `llm.Tool` shape unchanged | Likely unchanged; `Tool` is shared between legacy and v2 |
| `registry.go` | `llm.Tool`, `AsLLMTools` | NONE | UNCHANGED |
| `rag/rag.go` | `llm.Client`, `llm.GenerateRequest` | NONE (alias) | RAG `Options.LLM` field changes to `llm.ChatModel` |
| `bench/judge.go` | `llm.Client`, `llm.GenerateRequest` | NONE (alias) | Same as simple.go |
| `bench/winrate.go` | `llm.Client`, `llm.GenerateRequest` | NONE (alias) | Same |
| `context/builder.go` | `llm.Client`, `llm.Message` | NONE (alias) | Type change; `llm.Message` unchanged |
| `rl/trainer_proxy.go` | `llm.Client` (return type of `LoadModel`) | NONE (alias) | Return type changes to `llm.ChatModel` |
| `examples/scriptedllm/scriptedllm.go` | `llm.Client`, `llm.GenerateResponse` | NONE (alias) | Either: (a) keep legacy + mark deprecated, or (b) migrate to wrap `llm.NewScriptedLLM` and return new-shape responses. Decision deferred to Phase 3. |

**Phase 0 net diff to existing files:** ZERO functional changes. The alias `type Client = LegacyClient` makes the rename source-compatible. Phase 0 is purely additive at the call site.

### Test-helper migration (`scriptedllm_test.go` in repo root)

Today: `scriptedllm_test.go` defines `scriptedLLM`, `newScriptedLLM`, `textResp`, `errScriptExhausted` as test helpers used across `simple_test.go`, `react_test.go`, `function_call_test.go`, `reflection_test.go`, `plan_solve_test.go`, `example_simple_test.go`, `example_tool_use_test.go`, `example_multi_agent_test.go`.

Phase 0 plan:
1. Move the canonical `ScriptedLLM` to `llm/scripted.go` (D-03; full-capability v2 with options-based construction).
2. Replace `scriptedllm_test.go` body with a thin shim:
   ```go
   package agents

   import "github.com/costa92/llm-agent/llm"

   // scriptedLLM is a test-local alias for llm.ScriptedLLM.
   //
   // Deprecated: use llm.NewScriptedLLM directly. Retained until Phase 3
   // refactors the agent paradigms to consume llm.ChatModel.
   type scriptedLLM = llm.ScriptedLLM

   // Existing helpers continue to compile by delegating to llm package.
   var errScriptExhausted = llm.ErrScriptExhausted

   func newScriptedLLM(resps ...llm.GenerateResponse) *scriptedLLM {
       // Convert legacy GenerateResponse to v2 Response shape, then construct
       // ScriptedLLM. Or: have ScriptedLLM accept legacy responses as a
       // back-compat option for one phase. (Choose at plan time.)
       ...
   }

   func textResp(text string) llm.GenerateResponse {
       return llm.GenerateResponse{Text: text, FinishReason: llm.FinishReasonStop, Provider: "scripted"}
   }
   ```
3. Tests in `simple_test.go` etc. continue to compile and pass.

The `examples/scriptedllm/scriptedllm.go` package is a public-facing demo helper; it follows the same alias pattern.

## Pitfalls and How Phase 0 Design Prevents Each

| Pitfall | Phase 0 Mitigation |
|---------|---------------------|
| **6 (capability shape — type assertion vs bitmask)** | Resolved by D-02 (Capabilities struct on ProviderInfo) + dual signal (type assertion at compile time, Capabilities struct at runtime). `llm/doc.go` documents the canonical negotiation idiom (both checks) so Phase 1 adapter authors and Phase 3 agent refactor follow the same pattern. |
| **12 (`replace` directive forgotten in tagged release)** | `release-precheck.yml` in all 4 repos rejects any non-empty Replace block on `release/**` branches. Tested in Phase 0 by intentionally pushing a branch with a replace and verifying the gate fails. |
| **13 (`go.work` committed)** | `.gitignore` adds `go.work` + `go.work.sum` in all 4 repos. Per-repo CI sets `GOWORK=off` at job level. Sister-repo READMEs document the `scripts/workspace.sh` writer that places `go.work` in the PARENT directory (above any repo). |
| **14 (cross-repo break)** | Umbrella CI builds all 4 repos against PR HEAD on every llm-agent PR. Catches breakage at PR time, not after merge. Sister-repo README explicitly says "this CI runs against your PR if you change a public type." |
| **15 (deprecation kept forever)** | `DEPRECATIONS.md` (NEW file) lists every Deprecated symbol with target removal version. CHANGELOG entry for each Deprecated. Phase 7 (calendar-gated, in ROADMAP) removes them. The `// Deprecated:` godoc comment names `v0.4.0` so users see the deadline at every IDE hover. |
| **22 (architectural drift baseline)** | Phase 0 exit captures `go doc ./...` output as a baseline (committed at `docs/api-snapshot-v0.3.0-pre.1.txt` or similar). Future `/gsd-transition` calls diff against this. Provider Author Guide (Phase 1) is the litmus — if it grows conditionals, drift is happening. |

**Pitfalls explicitly NOT addressed in Phase 0** (deferred to other phases per the research bundle): 1, 2, 3, 4, 5 (streaming/tool semantics — Phase 1/2/3); 7, 8, 9, 10, 11 (OTel — Phase 5); 16, 17, 18 (refsvc — Phase 6); 19, 20, 21 (per-phase research/process discipline).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26) |
| Config file | none — `go test ./...` is the canonical invocation |
| Quick run command (this repo, this package only) | `go test ./llm/... -run . -count=1` |
| Quick run command (whole repo) | `go test ./... -count=1` |
| Full suite command | `go vet ./... && go build ./... && go test ./... && (cd examples && go vet ./... && go build ./...)` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CORE-01 | `ChatModel` interface compiles + has Generate/Stream/Info | type-level | `go vet ./llm/...` (compiles is enough) + `var _ llm.ChatModel = (*llm.ScriptedLLM)(nil)` in `llm/scripted.go` | ❌ Wave 0 |
| CORE-02 | `ToolCaller` defines `WithTools` returning new value (immutable) | unit | `go test ./llm/ -run TestToolCallerImmutable` | ❌ Wave 0 |
| CORE-03 | `Embedder.Embed` returns vectors + Usage + error | unit | `go test ./llm/ -run TestScriptedLLM_Embed` | ❌ Wave 0 |
| CORE-04 | `StructuredOutputs.WithSchema` returns `ChatModel` | unit | `go test ./llm/ -run TestStructuredOutputs_WithSchema` | ❌ Wave 0 |
| CORE-05 | `StreamEvent.Kind` enum has 6 variants; `ToolCallDelta.Index` int | type-level + unit | `go test ./llm/ -run TestStreamEventKindCount` (asserts iota max value) + `go test ./llm/ -run TestToolCallDelta_Index` | ❌ Wave 0 |
| CORE-06 | `ProviderInfo` has Provider/Model/Capabilities; `Capabilities` has 4 bool fields | unit | `go test ./llm/ -run TestProviderInfo_Shape` (uses reflect to assert fields) | ❌ Wave 0 |
| CORE-07 | `ScriptedLLM` satisfies all 4 capability interfaces | type-level | compile-time `var _` assertions in `llm/scripted.go` | ❌ Wave 0 |
| CORE-07 | `ChatOnlyMock` satisfies ChatModel ONLY (not the others) | runtime | `go test ./llm/ -run TestChatOnlyMockExcludesCapabilities` | ❌ Wave 0 |
| CORE-08 | `LegacyClient` is callable; `Client` alias resolves to it | type-level | compile-time `var _ llm.Client = (llm.LegacyClient)(nil)` + symmetric reverse | ❌ Wave 0 |
| CORE-08 | `// Deprecated:` godoc comment on `LegacyClient` and companion types | manual / linter | `go vet ./...` + grep check `grep -c "// Deprecated:" llm/legacy.go` ≥ 5 | partial (grep is automated) |
| CORE-09 | Migration guide exists with at least 1 worked example | manual | `test -f docs/migration-v0.2-to-v0.3.md && grep -c "## Worked example" docs/migration-v0.2-to-v0.3.md` ≥ 1 | ❌ Wave 0 |
| INFRA-01 | 4 modules exist (this repo + 3 sisters) | smoke | `gh repo view costa92/llm-agent-providers && gh repo view costa92/llm-agent-otel && gh repo view costa92/llm-agent-customer-support` (manual; CI cannot test repo creation) | n/a — manual verification |
| INFRA-02 | `go.work` is gitignored in all 4 repos | smoke | `git check-ignore go.work` returns 0 in each repo (in CI) | ❌ Wave 0 |
| INFRA-02 | CI runs with GOWORK=off | smoke | grep `GOWORK: off` in `.github/workflows/test.yml` of each repo | ❌ Wave 0 |
| INFRA-03 | `scripts/workspace.sh` writes a sibling-aware go.work | shell test | execute the script in a tmpdir with mock sibling dirs and assert `go.work` written | ❌ Wave 0 |
| INFRA-04 | release-precheck rejects `replace` directives | integration | push a `release/test` branch with a replace directive; assert workflow fails | ❌ Wave 0 (manual integration test at phase exit) |
| INFRA-05 | Umbrella CI builds all 4 repos | integration | open a PR to llm-agent that intentionally breaks a public type used by sisters; assert umbrella job fails | ❌ Wave 0 (manual integration test at phase exit) |
| INFRA-06 | Each sister-repo README documents iteration pattern | manual | grep "go.work" in each README; grep "replace" + "escape hatch" | ❌ Wave 0 |
| INFRA-07 | Versioning policy documented in all 4 repos | manual | grep version table in each README; CHANGELOG `### Breaking` template present | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./llm/... -count=1` (sub-second).
- **Per wave merge:** `go vet ./... && go build ./... && go test ./... && (cd examples && go vet ./... && go build ./...)`.
- **Phase gate (`/gsd-verify-work`):**
  1. Full suite green in core repo.
  2. `go build ./...` green in each of 3 sister repos when cloned individually with `GOWORK=off`.
  3. Umbrella workflow run on a fresh PR shows green (4-repo build).
  4. Manual: push a `release/test` branch with a `replace` directive; verify release-precheck workflow fails. Delete the test branch.
  5. `go doc ./...` snapshot captured to `docs/api-snapshot-v0.3.0-pre.1.txt` (Pitfall 22 baseline).
  6. Tag `v0.3.0-pre.1` on core; sister-repo `go.mod` `require` lines validated.

### Wave 0 Gaps

- [ ] `llm/llm_test.go` — interface satisfaction tests (CORE-01..07).
- [ ] `llm/scripted.go` — compile-time `var _ llm.ChatModel = (*ScriptedLLM)(nil)` etc.
- [ ] `llm/chat_only_mock.go` — same assertions in negative form.
- [ ] No new test framework needed; Go stdlib `testing` is the existing choice.
- [ ] `scripts/test-release-precheck.sh` (optional) — local script to verify release-precheck workflow against a tmp branch.

*(All listed items are net-new in Phase 0; existing test infrastructure covers everything once those files land.)*

## Security Domain

> Required when `security_enforcement` is enabled. Config `.planning/config.json` does not set this key, so default (enabled) applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Phase 0 has no runtime authentication surface (interfaces are pure type definitions). |
| V3 Session Management | no | Same. |
| V4 Access Control | no | Sister-repo creation uses `gh` CLI which honors GitHub's authn/authz; nothing app-level. |
| V5 Input Validation | partial | `Tool.Parameters` and `ToolCall.Arguments` are `json.RawMessage` — validation is the adapter's job (Phase 1). Phase 0 documents this as the contract. |
| V6 Cryptography | no | No cryptographic operations in Phase 0. The only "secret" handling consideration is GitHub repo creation (uses developer's `gh auth` token; never logged). |

### Known Threat Patterns for Phase 0 Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious `replace` directive in published module | Tampering | release-precheck CI gate (INFRA-04) |
| Committed `go.work` reroutes user's build at clone time | Tampering / DoS | `.gitignore` policy (INFRA-02) + `GOWORK=off` in CI |
| Deprecated public symbol kept indefinitely | Architectural / supply-chain integrity | DEPRECATIONS.md + scheduled removal phase (Pitfall 15) |
| Unreviewed merge to main breaks downstream sister repos | Tampering / availability | Umbrella CI (INFRA-05) + branch protection on `main` (manual GitHub-side setup; not in code) |
| Sister repo created with permissive defaults | Information disclosure (proprietary code) | All 3 sister repos public per D-04 — INTENTIONAL (not a vulnerability); LICENSE present in each |

**Threat-irrelevant areas** (no PII, no auth, no network in Phase 0): Phase 0 ships zero runtime code paths that handle user data or external inputs. The interfaces are pure type signatures; the multi-repo infra is build-time configuration. ASVS-grade risks materialize starting in Phase 1 (provider adapters making real HTTP calls).

## Project Constraints (from CLAUDE.md)

These directives are extracted from the project's `CLAUDE.md` and govern Phase 0 implementation. Treat with the same authority as locked decisions.

1. **Core repo (`llm-agent`) stays stdlib-only.** No `go.sum`, no non-stdlib deps. Phase 0 verified compliant — every new file uses only stdlib (`context`, `encoding/json`, `errors`, `io`, `sync`).
2. **No K8s in v0.3.** Phase 0 does not touch K8s; sister-repo `customer-support` README explicitly says "K8s manifests are NOT part of v0.3."
3. **No `replace` directives in tagged-release branches.** release-precheck CI enforces (INFRA-04).
4. **`go.work` is `.gitignore`d in every repo.** Phase 0 adds `.gitignore` entries everywhere; CI runs `GOWORK=off`.
5. **Capabilities are per-(provider × model).** Locked by D-02; `Capabilities` struct lives ON the model-bound `ProviderInfo`.
6. **Streaming events are a typed union.** `StreamEvent.Kind` + per-tool-call `Index`. Locked in `llm/stream.go`.
7. **OTel attaches as decorator wrappers.** Phase 0 ensures interfaces COMPOSE under wrapping. Specifically: `ToolCaller` embeds `ChatModel` so a wrapper that wants to expose tool-calling can type-assert on the inner and re-implement on the outer (the K3 rewrap pattern).
8. **Refsvc has hard caps + DISABLE_LLM=1 from Day 1.** Phase 0 doesn't touch refsvc beyond skeleton; the rule applies in Phase 6.

**Interface design check vs CLAUDE.md rule 7 (decorator composability):** A wrapper like `otelmodel.Wrap(inner ChatModel) ChatModel` must produce a value that EXPOSES `ToolCaller` if `inner` implements `ToolCaller`. With our interface shape, this works:
```go
type wrapper struct { inner ChatModel; ... }
type toolCallerWrapper struct { *wrapper; inner ToolCaller }

func (w *toolCallerWrapper) WithTools(t []Tool) (ToolCaller, error) {
    bound, err := w.inner.WithTools(t)
    if err != nil { return nil, err }
    return Wrap(bound).(ToolCaller), nil  // re-wrap; result still implements ToolCaller
}
```
The recursive rewrap is sound because `ToolCaller` embeds `ChatModel`, and `WithTools` returns `ToolCaller` (so we can assert on it). If `WithTools` returned `ChatModel`, the rewrap would lose tool-calling capability — that's why the interface embeds and the return type is `ToolCaller`. **This is the design check that justifies the embedding choice.** Verified at design time; concrete implementation is Phase 5.

## State of the Art

| Old Approach (v0.2 / pre-research) | Current Approach (v0.3 Phase 0) | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single `llm.Client` with optional `Tools` field — providers ignore unsupported tools | Capability interfaces (`ToolCaller` etc.) + `ProviderInfo` hint + `errors.Is(err, ErrCapabilityNotSupported)` | This phase | Agents can detect capability gaps at compile-time + runtime; degrade gracefully |
| `<-chan StreamChunk` streaming with optional `*ToolCall` field | Iterator `StreamReader.Next() (StreamEvent, error)` with typed `Kind` enum + per-tool-call `Index` | This phase | Native streaming granularity preserved per-provider; goroutine leaks easier to prevent |
| `BindTools(tools)` mutation pattern (Eino's deprecated approach) | `WithTools(tools) ToolCaller` immutable | This phase | Concurrent calls on the same model are race-free |
| Capabilities-as-bitmask | Capabilities-as-struct (D-02) | This phase | JSON-serializable for OTel; self-documenting in test failures |
| Single-repo monolith with `internal/providers/` | 4-repo umbrella (core stdlib-only + 3 sister deps-allowed) | This phase | Anyone can `go get llm-agent` and read every line; users opt into deps one package at a time |

**Deprecated/outdated:**
- `llm.Client`: superseded by `llm.ChatModel`; alias preserved through v0.3.x; removed v0.4.0.
- `llm.GenerateRequest` / `llm.GenerateResponse` / `llm.StreamChunk`: superseded by `llm.Request` / `llm.Response` / `llm.StreamEvent`; same removal schedule.
- ScriptedLLM-in-`_test.go`: superseded by `llm.ScriptedLLM` (non-test code). Old name remains as alias in `scriptedllm_test.go` shim until Phase 3.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Empty Go modules (no .go files yet) succeed at `go build ./...` and `go vet ./...` in Go 1.26. | CI YAML Sketches, sister-repo skeleton CI | LOW — verified by Go reference: `./...` with no packages = empty set + exit 0. If wrong, sister-repo skeleton CI needs a stub `doc.go` per repo. [VERIFIED: `go help build` documents the behavior; trivially testable] |
| A2 | `actions/checkout@v4` works for public repos without auth in `repository:` mode. | Umbrella CI sketch | LOW — public repos have always been accessible without explicit token to `actions/checkout`. [VERIFIED: GitHub Actions docs] |
| A3 | `release/**` branch trigger fires before tag push, allowing pre-tag `replace` ban. | Release-precheck CI sketch | MEDIUM — assumes the team uses a `release/v0.3.0` branch before tagging. If the workflow tags directly from `main` without a release branch, the gate misses entirely. **Mitigation:** add `tags:` trigger as a fallback. [ASSUMED — release process not documented yet; recommend adopting release-branch convention] |
| A4 | `gh repo create` from the costa92 user has rights to create public repos under that account. | Sister-repo creation step | LOW — costa92 owns the namespace; `gh auth status` confirms write access. [ASSUMED but trivially verifiable at execution time] |
| A5 | The `Capabilities` struct's 4 fields (Tools/Embeddings/StructuredOutputs/PromptCaching) cover all v0.3 capabilities. Future Phase 5 OTel emission of `gen_ai.provider.capabilities.*` matches these names. | `llm/info.go` | LOW — D-02 ratifies this exact shape. PromptCaching is the only one currently unused (Phase 1-4 don't exercise it); it's pre-allocated for the P2 differentiator DIFF-02 in Phase 6+. If a fifth capability is added later, the `Capabilities` struct extends additively (BC-safe). [VERIFIED by D-02] |
| A6 | Pre-release tag `v0.3.0-pre.1` resolves correctly via Go module proxy for sister-repo `require` lines. | Pre-release tag strategy | LOW — Go module proxy handles SemVer pre-release identifiers per the spec; `v0.3.0-pre.1` is a valid pre-release. [CITED: https://semver.org and https://go.dev/ref/mod#versions] |
| A7 | The reverse-rewrap pattern (otel wrapper re-implementing `ToolCaller`) compiles cleanly with the interface shape proposed. | Decorator composability check | LOW — ARCHITECTURE.md §"Pattern 1" includes an explicit code sketch using exactly this pattern; the embedding shape (`ToolCaller embeds ChatModel`) is what makes it work. [CITED: ARCHITECTURE.md lines 285-315] |
| A8 | `type Client = LegacyClient` (alias) preserves source compatibility for ALL existing callers including those that satisfy `llm.Client` from another package. | Migration playbook + legacy.go | MEDIUM — Go type aliases preserve method-set identity, so `var _ llm.Client = myImpl{}` compiles iff `myImpl` satisfies `llm.LegacyClient`'s methods. There is one edge case: if anyone wrote `func F() llm.Client` and external callers compare types via reflection, the reflected name might be `LegacyClient` not `Client`. Acceptable for v0.3.x. [VERIFIED: Go spec §Type identity for aliases] |

## Open Questions

1. **Q1: Should `llm.Tool`, `llm.Message`, `llm.FinishReason` remain shared between LegacyClient and new ChatModel, or should they be duplicated under different names?**
   - What we know: existing code (`tool.go`, `registry.go`, agent paradigms, RAG, bench, context) uses `llm.Tool` and `llm.Message` and `llm.FinishReason` directly. Sharing them avoids two parallel type systems.
   - What's unclear: if we ever want to evolve `Tool` (e.g., add `OutputSchema json.RawMessage`), shared types tie LegacyClient to that evolution. Duplicate types decouple but require conversion at every call site.
   - **Recommendation for planner:** Share. The shapes haven't needed evolution in 6 months of v0.2; `ToolCall` is the only one that grows (adds `ID` field; v0.2 callers don't construct `ToolCall` so the addition is back-compat). Document in `legacy.go` that LegacyClient and ChatModel share these helper types deliberately.

2. **Q2: Where does the `go doc ./...` baseline snapshot for Pitfall 22 live, and is its update automated?**
   - What we know: Pitfall 22 says capture the snapshot at every `/gsd-transition` and diff. Phase 0 exit creates the first one.
   - What's unclear: stable filename (e.g., `docs/api-snapshot.txt` overwritten each phase, vs `docs/api-snapshot-v0.3.0-pre.1.txt` versioned) and CI enforcement (failing if not regenerated when public API changes).
   - **Recommendation for planner:** Single rolling file `docs/api-snapshot.txt` regenerated by `make doc-snapshot` (or `go doc ./... > docs/api-snapshot.txt`). CI gate optional — for Phase 0, manual capture suffices; automated drift detection can land in Phase 7.

3. **Q3: When exactly is `v0.3.0-pre.1` tagged?**
   - What we know: sister repos must `require github.com/costa92/llm-agent v0.3.0-pre.1` for the pre-release period. The tag must exist before sister-repo CI can resolve it.
   - What's unclear: can sister-repo `go.mod` initially use a pseudo-version (`v0.3.0-pre.0.20260510...-abc123def456`), and only flip to `v0.3.0-pre.1` after Phase 0 lands? Or do we tag immediately on the Phase 0 merge?
   - **Recommendation for planner:** Tag `v0.3.0-pre.1` as the LAST step of Phase 0, after all 4 repos' Phase 0 commits land. Until then, sister-repo `require` lines use the pseudo-version pointing at the in-progress core SHA. The umbrella CI uses `go.work` to override the version anyway, so the require line is a hint for clean-clone consumers. Document this sequence in `00-PLAN.md`.

4. **Q4: Does the umbrella CI need to handle the case where one of the sister repos' main branch doesn't exist yet?**
   - What we know: GitHub Actions checkout will fail if the branch doesn't exist; sister repos are created in Phase 0 with an initial commit on `main`.
   - What's unclear: ordering. If umbrella.yml lands on llm-agent main BEFORE the sister repos exist, every umbrella run fails. If umbrella.yml lands AFTER, the gap window is unprotected.
   - **Recommendation for planner:** Plan ordering: (1) create + push 3 sister repos with skeletons FIRST, (2) then merge umbrella.yml to llm-agent. This is enforced by the plan dependency graph: `00-03 (sister skeletons)` runs before `00-05 (umbrella CI)`.

5. **Q5: Should `Embedder` be in a different file than `ToolCaller` and `StructuredOutputs` since it doesn't embed `ChatModel`?**
   - What we know: `capabilities.go` proposed in this research holds all three. Conceptually, `Embedder` is orthogonal to chat; the others are chat augmentations.
   - What's unclear: file separation as a documentation hint vs. unified-capabilities-file.
   - **Recommendation for planner:** Keep all three in `capabilities.go` — godoc renders them grouped, and the fact that `Embedder` doesn't embed `ChatModel` is locally visible from the type signature. Splitting introduces a `embedder.go` file with one type, which is more friction than the clarity gain.

6. **Q6: Should the `release-precheck.yml` workflow file be a single shared file (reusable workflow) or 4 copies?**
   - What we know: the YAML is identical across all 4 repos (verified by inspection). GitHub Actions supports `uses: <repo>/.github/workflows/<name>.yml@<ref>`.
   - What's unclear: shared workflow introduces an implicit dependency from sister repos onto the core repo's workflow file — exactly the kind of subtle coupling Phase 0 is trying to prevent. Copy-paste 4 times is sub-optimal but explicit.
   - **Recommendation for planner:** Copy-paste. Rationale: (a) 4 files is not a maintenance burden at this scale; (b) explicit is better than implicit per Go ethos; (c) the cost of breaking the shared workflow centrally would be unbounded — every sister repo's release gate stops working. Reusable workflow can be revisited in Phase 7 if duplication becomes painful.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building all 4 repos; running `go vet`, `go build`, `go test`, `go mod`, `go work` | ✓ | 1.26.0 | — |
| `gh` CLI | Creating sister GitHub repos (D-04) + scripted PR workflows | ✓ | 2.91.0 | Manual creation via GitHub web UI |
| GitHub Actions runners (`ubuntu-latest`) | All CI workflows | ✓ (GitHub-managed) | latest | — |
| Bash 4+ | `scripts/workspace.sh` | ✓ (system shell) | system default | sh-compatible rewrite |
| Python 3 | release-precheck workflow JSON parsing | ✓ (preinstalled on `ubuntu-latest`) | 3.x | substitute `jq` (preinstalled on `ubuntu-latest` too) |
| `git` | Repo operations | ✓ | system default | — |

**Missing dependencies with no fallback:** None. Phase 0 has no dependency that blocks execution.

**Missing dependencies with fallback:** None applicable.

## Sources

### Primary (HIGH confidence)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/phases/00-keystone-interfaces/00-CONTEXT.md` — D-01..D-04 locked decisions, canonical refs, code context
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/REQUIREMENTS.md` — INFRA-01..07, CORE-01..09 (Phase 0 surface)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/ROADMAP.md` §"Phase 0" — goal, success criteria, requirements mapping, pitfalls guarded
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/SUMMARY.md` — K1/K2/K3/K6 keystones, conflict resolutions
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/ARCHITECTURE.md` — capability negotiation pattern, streaming union, decorator pattern, multi-repo dependency graph
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/PITFALLS.md` Pitfalls 6, 12-15, 22 — exact prevention recipes adopted verbatim
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.planning/research/STACK.md` §"Multi-repo Layout" — go.work placement, scripts/workspace.sh, replace policy
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/llm/client.go` — current `Client`, `Tool`, `ToolCall`, `Message`, `StreamChunk`, `FinishReason` exact shapes (verified via Read)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/scriptedllm_test.go` — current ScriptedLLM design template
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/.github/workflows/test.yml` — current CI shape; mirror template for sister repos
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/CLAUDE.md` — hard rules (stdlib-only, no K8s, K1/K2/K3/K7 enforcement)

### Secondary (MEDIUM confidence)
- Go module reference: https://go.dev/ref/mod (replace directives, pre-release versioning)
- Go workspace tutorial: https://go.dev/doc/tutorial/workspaces (`GOWORK=off` semantics)
- GitHub Actions checkout docs: https://github.com/actions/checkout (cross-repo checkout with `repository:`)
- SemVer 2.0: https://semver.org (pre-release identifier shape `v0.3.0-pre.1`)

### Tertiary (LOW confidence — none in this research)
*(All claims either verified against codebase or cited from the locked decision set.)*

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only; no third-party deps in Phase 0.
- Concrete Go type definitions: HIGH on shape (D-01/D-02 lock direction); MEDIUM on field-naming details where Claude's discretion applies (per CONTEXT.md). Planner may adjust field names without changing semantics.
- File-by-file breakdown: HIGH — every file's purpose is traceable to a requirement ID + decision.
- Sister-repo skeletons: HIGH — content lifted from STACK.md and existing core-repo files (LICENSE, OWNERS, .gitignore patterns).
- CI YAML sketches: HIGH on the umbrella + per-repo shapes; MEDIUM on release-precheck (Q3 above — branch vs tag trigger choice has tradeoffs).
- Migration playbook: HIGH — exhaustive grep verified the 13-file inventory; alias trick eliminates Phase 0 functional diffs.
- Pitfalls coverage: HIGH — every Phase-0-relevant pitfall (6, 12, 13, 14, 15, 22) has an explicit mitigation tied to a deliverable.
- Validation Architecture: HIGH on test-mapping; LOW on integration-test automation for sister-repo CI behavior (some checks remain manual at phase exit, by design — repo creation cannot be unit-tested).

**Research date:** 2026-05-10
**Valid until:** 2026-06-09 (30 days — stable architectural research; no fast-moving libraries)
