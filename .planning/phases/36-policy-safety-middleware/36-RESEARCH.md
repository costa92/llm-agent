> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent` current docs.

# Phase 36 Research: Policy / safety middleware

**Researched:** 2026-05-21
**Phase:** 36 — policy / safety middleware (second v1.2 phase, follows Phase 35)
**Requirement:** CC-2
**Repo touched:** `llm-agent` (core only)
**Target tag:** `v0.6.1` (patch — strict-additive new `policy` sub-package)
**Upstream:**
- `.planning/research/v1.2-core-capability-deepening-SUMMARY.md` —
  KC-3 (policy mirrors `otelmodel.Wrap`; capability-preserving;
  typed `Gate` event union; sentinel `ErrBlocked`) and KC-5
  (additive only, no `/v2`, no edit to validated public types).
- `.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md` —
  §"Carry-forward notes" pins composition order:
  `policy.Wrap(budget.Wrap(otelmodel.Wrap(provider)))` for the future
  `budget.Wrap`; today the stack is
  `policy.Wrap(otelmodel.Wrap(provider))` with budget enforced at the
  `generateFromPrompt` chokepoint underneath all wrappers.

## Scope (CC-2 verbatim)

> A `policy` package is shipped in core with a capability-preserving
> `policy.Wrap(inner llm.ChatModel, gates ...Gate) llm.ChatModel`
> decorator that mirrors `otelmodel.Wrap` (K3 — handles `ToolCaller` /
> `Embedder` / `StructuredOutputs` assertions), a typed `Gate` event
> union (`PreGenerate`/`PostGenerate`/`PreStream`/`StreamDelta`/
> `PostStream`), a sentinel `policy.ErrBlocked`, and 3 built-in gates
> (`PIIRedactor`, `InjectionScanner`, `MaxInputLen`). The documented
> composition stack `policy.Wrap(otelmodel.Wrap(provider))` is verified
> by an integration test (capability assertions survive both wrappers;
> denied requests short-circuit before span open). **The core stays
> stdlib-only** — every built-in gate uses stdlib `regexp`; no rag
> import (the regex patterns are lifted to a separate file, not shared
> via import — KC-3). Phase 36.

One sentence: a stdlib-only `policy` package that wraps `llm.ChatModel`
as a capability-preserving decorator (mirroring `otelmodel.Wrap`), runs
a typed event union of `Gate`s at request/response/stream boundaries,
returns `policy.ErrBlocked` on a `Block` decision, and ships 3 regex /
length gates.

## User Constraints

No CONTEXT.md exists for Phase 36 — this research operates directly on
the v1.2 SUMMARY's KC-3 keystone (the design is pre-decided at the
milestone level). All hard rules below come from CLAUDE.md (project) +
KC-5 (additive-only ceiling) + KC-3 (decorator + mirror `otelmodel.Wrap`).

### Locked Decisions (from KC-3 + KC-5 + CLAUDE.md — DO NOT re-litigate)

- **Decorator shape — not hook system.** `policy.Wrap(inner
  llm.ChatModel, gates ...Gate) llm.ChatModel`. Lives at the model
  boundary; every agent paradigm benefits with zero edit. (KC-3.)
- **Capability-preserving.** Must propagate `ToolCaller`, `Embedder`,
  `StructuredOutputs` assertions exactly like `otelmodel.Wrap`. Same 8
  nested wrapper structs for the 2³ capability combinations. (KC-3 mirror
  clause + K3 precedent.)
- **Typed `Gate` event union, 5 kinds.** `PreGenerate`, `PostGenerate`,
  `PreStream`, `StreamDelta`, `PostStream`. (KC-3.)
- **Sentinel `policy.ErrBlocked`.** Wraps the deciding gate name; callers
  detect with `errors.Is(err, policy.ErrBlocked)`. (KC-3.)
- **Three built-in gates.** `PIIRedactor`, `InjectionScanner`, `MaxInputLen`.
  Regex patterns lifted from `llm-agent-rag/guard` where they overlap —
  copied into a new file in `policy/`, NOT imported from rag. (KC-3.)
- **Documented composition stack.** `policy.Wrap(otelmodel.Wrap(provider))` —
  outer-most policy denies before observed; middle observes; inner-most
  calls the network. (KC-3.)
- **Audit log via `OnDecision`.** Optional callback `func(Decision)`,
  attached via a `Config{OnDecision: ...}` option or `WithOnDecision`
  helper. Observation only, never interception. (KC-3.)
- **Stdlib-only.** `regexp`, `context`, `errors`, `fmt`, `strings`,
  `sync`, `unicode/utf8`, `io` — nothing else. (CLAUDE.md Rule 1; KC-5.)
- **No edit to `llm.ChatModel`, `agents.Agent`, `memory.Memory`,
  `orchestrate.NodeFunc[S]`.** New package + new optional types only.
  Tag is `v0.6.1` — strict patch / additive. (KC-5.)
- **Target tag: `v0.6.1`** (patch — additive). The v0.7.0 milestone-cap
  tag is Phase 38.

### Claude's Discretion (this research recommends)

- The exact `Decision` enum shape (Allow / Block / Redact / Replace).
  See §"Decision A: Gate return shape" below.
- Whether `StreamDelta` is enabled by default (recommendation: OFF — high
  per-delta cost, opt-in per gate via a gate option).
- Whether `MaxInputLen` measures runes, bytes, or characters
  (recommendation: bytes — cheap, stdlib `len()`, language-agnostic; ties
  back to provider input-limit which is also byte-based).
- The exact regex subset lifted from `llm-agent-rag/guard` (recommendation:
  email + phone + IPv4 from `redact.go`; instruction_override +
  prompt_exfiltration from `inject.go`; drop SSN + credit_card as US-locale-
  specific; see §"Decision E").
- Slice breakdown (5 slices recommended below) and wave ordering (strictly
  sequential: skeleton → built-in gates → compose-with-otel → example →
  exit gate).

### Deferred Ideas (OUT OF SCOPE for Phase 36)

- **OWASP / NIST safety category framework.** v1.2 SUMMARY out-of-scope
  table: a full taxonomy is a future expansion. Three built-ins are the
  v1.2 cap.
- **`policy.Wrap`-emitted spans / OTel attributes.** That's an
  `llm-agent-otel` sister-repo concern; v1.2 is core-only.
- **Per-request rate limiting / token-bucket gates.** A budget concern,
  not a policy concern; Phase 35's `budget` package owns it. If a future
  user wants "deny 100 req/s on this user_id", they can author a custom
  `Gate` — but no built-in ships in v1.2.
- **Schema validation as a gate.** `llm.StructuredOutputs` already handles
  this at the model level (`WithSchema`); no policy duplication.
- **A "policy.Wrap" as the budget enforcement point.** Decision 3 in
  Phase 35-RESEARCH.md is final: budget is at `generateFromPrompt`, not
  in a decorator. Phase 36 does NOT move budget into policy.
- **Cross-stream PII detection** (joining redacted text across streaming
  deltas to detect "ali...ce@…"). Per-delta best-effort only; the gate
  state is per-stream, not per-buffer.
- **Auto-redacting `Response.Text` based on `Redact` decision in
  streaming.** Streaming redaction is documented as best-effort; the
  decorator buffers the delta locally, applies the redaction, emits the
  redacted text. No buffer-the-whole-stream-then-emit policy. See
  Decision F.
- **Touching `llm-agent-rag/guard`.** v1.2 is core-only. Rag stays a
  fixed point (KS-5 freeze). Patterns are lifted by copy (separate
  source-of-truth in core/policy/), not by import.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CC-2 | `policy` package: capability-preserving `Wrap`, typed `Gate` event union, sentinel `ErrBlocked`, 3 built-in gates (PII, injection, max-input-length), `OnDecision` audit log, documented composition stack with `otelmodel.Wrap` verified by integration test, stdlib-only. | This entire document. See §"Standard Stack" for the locked surface, §"Architecture Patterns" for the 8-wrapper capability composition, §"Decision A-F" for design details, §"Slice Breakdown" for the 5-slice plan. |

## Constraint inventory

- **Stdlib-only core (CLAUDE.md Rule 1, KC-5).** `go.mod`
  (`/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/go.mod`)
  has exactly one `require` — `github.com/costa92/llm-agent-rag v1.0.1`
  for the RAG facade — and the policy package must add zero. Means no
  external regex engine; only stdlib `regexp`. Means no third-party PII
  pattern library; lift from `llm-agent-rag/guard` by copying source,
  not by `import`. [VERIFIED: read go.mod, exactly one require directive]
- **No `/v2` import path (KC-5).** v0.6.0 → v0.6.1 is a **patch**
  (additive) bump. Existing v0.6.0 callers must compile unchanged against
  v0.6.1. New package + new optional types are the only shapes allowed.
- **Mirror `otelmodel.Wrap`'s 8-wrapper composition.**
  `llm-agent-otel/otelmodel/otelmodel.go:14-49` ships the canonical
  capability-preserving decorator: a base `wrapper`, plus 7 nested
  wrappers covering `Tool * Embed * Schema` (2³ - 1 = 7 combinations
  beyond the bare ChatModel). `Wrap()` does a type-switch tree on the
  inner model to pick which wrapper to return. Policy MUST mirror this
  shape exactly — same 8 wrappers, same type-switch tree, same `WithTools`
  / `WithSchema` re-wrapping idiom. [VERIFIED: read otelmodel.go]
- **K1 (typed StreamEvent union) is locked.** Adding a new
  `StreamEvent.Kind` for "policy blocked mid-stream" is a v1.2 SUMMARY
  out-of-scope decision (already mirrored in Phase 35's Decision 4 — no
  new kinds in v1.2). Streaming block surfaces by closing the stream and
  returning `ErrBlocked` from the next `Next()` call. See Decision F.
- **K3 — OTel attaches as decorator, never hook.** Policy mirrors this
  shape. (Locked in CLAUDE.md hard rule 7.)
- **Compose with budget (Phase 35 ships first).** The documented stack
  is `policy.Wrap(otelmodel.Wrap(provider))` — budget is NOT in this
  stack; budget enforces at `generateFromPrompt` UNDERNEATH the wrapped
  model. The integration test must confirm: a budget-exhausted request
  short-circuits inside `generateFromPrompt` BEFORE any wrapped layer
  fires. (Cross-checked against 35-RESEARCH.md §"Carry-forward notes".)
- **Validated public types unchanged.** `llm.ChatModel`,
  `llm.StreamReader`, `llm.StreamEvent`, `agents.Agent`, `agents.Result` —
  none edited. The 4 capability interfaces (`ChatModel`, `ToolCaller`,
  `Embedder`, `StructuredOutputs`) are the contract; policy `Wrap` must
  preserve type-assertion compatibility for all of them.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Pre-/post-LLM request inspection | Decorator (model boundary) | — | KC-3 explicit: at the model boundary so every agent paradigm benefits without code edit |
| Capability preservation (Tool/Embed/Schema) | Decorator type-switch tree | Inner model | Mirrors `otelmodel.Wrap` — 8 nested wrappers covering 2³ capability combinations |
| Streaming-layer inspection | `StreamReader` decorator (composes with otelmodel's StreamReader) | Inner stream | Stream lifecycle is `PreStream → StreamDelta* → PostStream` — wraps the inner `llm.StreamReader.Next()` |
| Audit logging | `OnDecision(Decision)` callback | Caller's logger / observer | Observation only, never interception (KC-3) |
| Regex pattern source-of-truth | `policy/patterns.go` (copy of rag's) | — | Core stdlib-only; lift by copy, not by import (KS-5 rag freeze + KC-3 separation) |
| Composition with OTel | Outer `policy.Wrap(otelmodel.Wrap(provider))` | Outer `policy` (blocks before observed) | KC-3: outer-most policy denies before observed; middle observes; inner-most makes the call |
| Composition with budget | Underneath all wrappers, at `generateFromPrompt` chokepoint | — | Phase 35 already wired; budget exhaustion short-circuits BEFORE the wrapped model is reached |

## Standard Stack

### Core (new package shape — entirely stdlib)

| Symbol | Purpose | Why Standard |
|--------|---------|--------------|
| `policy.Wrap(inner llm.ChatModel, gates ...Gate) llm.ChatModel` | Capability-preserving decorator factory | Mirrors `otelmodel.Wrap` (KC-3, K3) — proven shape in the umbrella |
| `policy.Wrap` with `Config` option | Audit log via `Config.OnDecision`, future-extensibility | Mirrors `otelmodel.Wrap(model, Config{TracerProvider: ...})` shape exactly |
| `type Gate interface { Inspect(ctx, ev Event) Decision }` | The user-extension seam | Single-method interface; minimal surface |
| `type Event struct { Kind EventKind; Req *llm.Request; Resp *llm.Response; Delta *llm.StreamEvent }` | Typed event union (5 kinds) | KC-3 explicit; mirrors `llm.StreamEvent` kind+optional-pointer-fields pattern |
| `type Decision struct { Action DecisionAction; Reason string; Replacement string }` | Gate verdict | See Decision A below for the action enum |
| `type DecisionAction uint8` with `Allow / Block / Redact / Replace` constants | Action enum | See Decision A |
| `var ErrBlocked = errors.New("policy: blocked")` | Sentinel for blocked requests | KC-3 explicit; mirrors `budget.ErrBudgetExceeded` umbrella pattern (35-01) |
| `type BlockedError struct{ Gate, Reason string; Wrapped error }` with `Unwrap` and `errors.Is(ErrBlocked)` | Rich error carrying the deciding gate | Mirrors `llm.AuthError`/`llm.RateLimitError` shape in `llm/errors.go` |
| `PIIRedactor()` constructor → `Gate` | Built-in: regex PII redaction | Pattern set lifted from rag |
| `InjectionScanner()` constructor → `Gate` | Built-in: regex prompt-injection detection | Pattern set lifted from rag |
| `MaxInputLen(n int)` constructor → `Gate` | Built-in: byte-count cap on prompt + system | New (no rag counterpart) |

### Standard library imports allowed in `policy/`

| Stdlib package | Used for |
|----------------|----------|
| `context` | `Inspect(ctx, ev)` signature; thread cancellation through audit log |
| `errors` | `errors.New("policy: blocked")`, `errors.Is`/`Unwrap` plumbing |
| `fmt` | `BlockedError.Error()` string formatting |
| `io` | `io.EOF` recognition in the wrapped `StreamReader` |
| `regexp` | All three built-in gates' patterns |
| `strings` | Substring tests, lowercasing for case-insensitive checks |
| `sync` | Mutex for per-stream gate state (e.g., a redactor's per-stream buffer) |
| `unicode/utf8` | Optional, for `MaxInputLen`'s rune-mode variant if added later |

**Verification:**
- `regexp` is stdlib. `go doc regexp` works on any Go install. No third-party required. [VERIFIED: stdlib]
- `unicode/utf8` is stdlib. [VERIFIED: stdlib]
- No third-party "PII detection" library considered — KC-3 explicitly mandates regex from rag.

### Alternatives Considered (and rejected)

| Instead of | Could Use | Rejected because |
|------------|-----------|------------------|
| `regexp` patterns | `presidio` / `redact-go` / DLP API | All require non-stdlib deps; KC-5 absolute |
| Per-gate interface family (`RequestGate`, `ResponseGate`, `StreamGate`) | Single `Gate` interface with typed `Event` | KC-3 explicit: typed event union, single interface. Three interfaces fragment the registration story (users want one `[]Gate` slice, not three slices) |
| Channel-based event dispatch | Single-method interface | Matches the existing `llm.ChatModel` + `otelmodel.Wrap` shape; channels add concurrency without need (gates fire synchronously in the request path) |
| Mutation of `llm.Request` in-place by gates | Returning a `Replacement` field in `Decision` | KC-5 ceiling: `llm.Request` is locked. Gates return a new value; decorator applies it. Concurrent-safety: the request value flowing through is the decorator's local copy |

### Package Legitimacy Audit

Not applicable — Phase 36 introduces **zero new external dependencies**.
The `policy` package uses only stdlib (`regexp`, `context`, `errors`,
`fmt`, `strings`, `sync`, `io`, optionally `unicode/utf8`). No `npm` /
`pip` / `crates` / `go.sum` entries are added. `go.mod` remains
unchanged. The composition target `otelmodel.Wrap` lives in
`llm-agent-otel` (a sister repo, not pulled in by core); the
composition-test slice (36-03) verifies the stack WITHOUT importing
otelmodel into core — see Decision G for the test approach.

## Architecture Patterns

### System Architecture Diagram

```
                                  agent.Run(ctx, input)
                                          ↓
                            generateFromPrompt(ctx, model, ...)
                            ┌─────────────────────────────────┐
                            │ pre-call: budget.From(ctx).Charge   ← Phase 35 (already shipped)
                            │   on deny → return ErrBudgetExceeded
                            │ model.Generate(ctx, req) ↓
                            └─────────────────────────────────┘
                                          ↓
                            ╔═════════════════════════════════╗
                            ║ policy.Wrap (outer decorator)   ║  ← Phase 36 (NEW)
                            ║   Gates run on Event{PreGen}    ║
                            ║     Allow / Block / Redact /    ║
                            ║       Replace                   ║
                            ║   OnDecision(Decision) callback ║
                            ║   on Block → ErrBlocked         ║
                            ╚═════════════════════════════════╝
                                          ↓
                            ╔═════════════════════════════════╗
                            ║ otelmodel.Wrap (middle decorator)║  ← sister-repo, observation
                            ║   tracer.Start("chat <model>")  ║
                            ║   semconv attrs                 ║
                            ║   stream.Next() spans firstToken║
                            ╚═════════════════════════════════╝
                                          ↓
                            ╔═════════════════════════════════╗
                            ║ provider (inner, the network)   ║
                            ║   OpenAI / Anthropic / Ollama / ║
                            ║   DeepSeek / MiniMax adapter    ║
                            ╚═════════════════════════════════╝

   Capability preservation across all wrappers (decorator pattern):
   - Outer Wrap returns a type that satisfies inner's capability set
   - If inner is ToolCaller → outer is ToolCaller (and re-wraps WithTools result)
   - If inner is Embedder → outer is Embedder (and forwards Embed)
   - If inner is StructuredOutputs → outer is StructuredOutputs (and re-wraps WithSchema)
   - 2³ = 8 wrapper types for the 8 combinations; same shape as otelmodel
```

### Recommended Project Structure

```
llm-agent/                     # repo root (flat layout — package agents at root)
├── policy/                    # NEW — added in Phase 36
│   ├── doc.go                 # package-level doc, KC-3 / CC-2 citations
│   ├── policy.go              # Wrap, type-switch tree, 8 wrappers, Config
│   ├── policy_test.go         # capability-preservation, basic Generate/Stream
│   ├── gate.go                # Gate, Event, EventKind, Decision, DecisionAction, ErrBlocked, BlockedError
│   ├── gate_test.go           # Gate / Decision unit tests
│   ├── patterns.go            # regex sources (copied from rag/guard) — single source of truth in core
│   ├── pii.go                 # PIIRedactor gate
│   ├── pii_test.go
│   ├── injection.go           # InjectionScanner gate
│   ├── injection_test.go
│   ├── length.go              # MaxInputLen gate
│   ├── length_test.go
│   └── integration_test.go    # compose-with-otel + budget short-circuit + 5-paradigm smoke
└── examples/
    └── 07-policy/             # NEW — example using ScriptedLLM
        ├── main.go
        └── README.md
```

**Rationale for split-file structure (vs `otelmodel.go` single-file):**
`otelmodel.go` is 329 lines because it's one decorator + one Config +
the Wrap type-switch tree. Policy adds **3 built-in gates** on top of
the same decorator tree — that's another ~400 LOC of gate
implementations + their regex/length logic + their tests. Splitting by
gate keeps each file ≤200 LOC and lets each gate's regex table be the
canonical reference of patterns. The decorator tree itself
(`policy.go`) stays roughly co-sized with `otelmodel.go`.

### Pattern 1: Mirror-otelmodel capability-preserving decorator

**What:** A `Wrap(inner llm.ChatModel, gates ...Gate) llm.ChatModel`
that returns ONE of 8 concrete types depending on which of `ToolCaller`,
`Embedder`, `StructuredOutputs` the inner model satisfies.

**When to use:** Always — this is the whole point of the package.

**Example (skeleton, copying `otelmodel.go:14-49`):**

```go
// Source: mirrors llm-agent-otel/otelmodel/otelmodel.go:20-49 [VERIFIED: read]
package policy

func Wrap(model llm.ChatModel, gates ...Gate) llm.ChatModel {
    return WrapConfig(model, Config{Gates: gates})
}

func WrapConfig(model llm.ChatModel, cfg Config) llm.ChatModel {
    base := &wrapper{inner: model, gates: cfg.Gates, onDecision: cfg.OnDecision}
    if tc, ok := model.(llm.ToolCaller); ok {
        if emb, ok := model.(llm.Embedder); ok {
            if so, ok := model.(llm.StructuredOutputs); ok {
                return &toolEmbedSchemaWrapper{wrapper: base, toolCaller: tc, embedder: emb, structured: so}
            }
            return &toolEmbedWrapper{wrapper: base, toolCaller: tc, embedder: emb}
        }
        if so, ok := model.(llm.StructuredOutputs); ok {
            return &toolSchemaWrapper{wrapper: base, toolCaller: tc, structured: so}
        }
        return &toolWrapper{wrapper: base, toolCaller: tc}
    }
    if emb, ok := model.(llm.Embedder); ok {
        if so, ok := model.(llm.StructuredOutputs); ok {
            return &embedSchemaWrapper{wrapper: base, embedder: emb, structured: so}
        }
        return &embedWrapper{wrapper: base, embedder: emb}
    }
    if so, ok := model.(llm.StructuredOutputs); ok {
        return &schemaWrapper{wrapper: base, structured: so}
    }
    return base
}
```

This is verbatim from `otelmodel.go` with `tp: tp, tracer: ...` swapped
for `gates: ..., onDecision: ...`. The 8 wrapper types must be
identical to otelmodel's — same names (`toolWrapper`,
`embedSchemaWrapper`, etc.), same field layout. This is the contract
that guarantees capability preservation: if `wrapped.(llm.ToolCaller)`
returned `(_, true)` before wrapping, it MUST return `(_, true)` after.

The `WithTools` re-wrap path also mirrors otelmodel exactly:

```go
// Source: otelmodel.go:154-162
func (w *toolWrapper) WithTools(tools []llm.Tool) (llm.ToolCaller, error) {
    next, err := w.toolCaller.WithTools(tools)
    if err != nil {
        return nil, err
    }
    wrapped := w.wrap(next) // re-runs WrapConfig with same gates + onDecision
    tc, _ := wrapped.(llm.ToolCaller)
    return tc, nil
}
```

The `(*wrapper).wrap(next)` helper closes over the gates + onDecision
config so a `WithTools`-rebound child preserves the policy stack — the
canonical proof of capability-preservation across the immutable
`WithTools`/`WithSchema` pattern (K1's "immutable WithTools" rule).

### Pattern 2: Stream decorator (PreStream → StreamDelta* → PostStream)

**What:** A `StreamReader` decorator wrapping the inner stream. Fires
`PreStream` event once before the first `Next()`, fires `StreamDelta`
on each `EventTextDelta` / `EventToolCallArgsDelta` if a gate opts in,
fires `PostStream` on `EventDone` or `io.EOF`.

**When to use:** The decorator's `(w *wrapper).Stream(ctx, req)` returns
a `*streamReader` that wraps the inner. Single-stream-state per call —
not concurrent-safe across calls (which is fine: `StreamReader` is
single-consumer per spec, `llm/stream.go:13-16`).

**Example (skeleton, copying `otelmodel.go:102-148`):**

```go
// Source: mirrors otelmodel.go:102-148 [VERIFIED: read]
type streamReader struct {
    inner       llm.StreamReader
    gates       []Gate
    onDecision  func(Decision)
    req         llm.Request
    started     bool   // PreStream fired?
    closed      bool
    perStreamMu sync.Mutex // serializes per-stream gate state (some gates accumulate)
}

func (r *streamReader) Next() (llm.StreamEvent, error) {
    if !r.started {
        r.started = true
        // PreStream gate event
        ev := Event{Kind: PreStream, Req: &r.req}
        if d, blocked := runGates(r.gates, ev, r.onDecision); blocked {
            r.end()
            return llm.StreamEvent{}, &BlockedError{Gate: d.Gate, Reason: d.Reason}
        }
    }
    ev, err := r.inner.Next()
    if err != nil {
        if err == io.EOF {
            // PostStream fires before EOF surfaces (best-effort observe; cannot block here)
            postEv := Event{Kind: PostStream, Req: &r.req}
            _, _ = runGates(r.gates, postEv, r.onDecision)
        }
        r.end()
        return ev, err
    }
    if ev.Kind == llm.EventDone {
        postEv := Event{Kind: PostStream, Req: &r.req, Delta: &ev}
        _, _ = runGates(r.gates, postEv, r.onDecision)
        r.end()
        return ev, nil
    }
    // StreamDelta — opt-in per gate; default OFF (see Decision F)
    deltaEv := Event{Kind: StreamDelta, Req: &r.req, Delta: &ev}
    if d, blocked := runGates(r.gates, deltaEv, r.onDecision); blocked {
        r.end()
        return llm.StreamEvent{}, &BlockedError{Gate: d.Gate, Reason: d.Reason}
    }
    // Redact: rewrite ev.Text in place if d.Action == Redact
    return ev, nil
}
```

### Pattern 3: Built-in `Gate` implementations

**Common shape:** Each built-in is a value type (small struct or function-
returning-Gate) constructed via `NewXxx()` returning a `Gate` interface.
Each holds compiled regex sets / config; `Inspect` is the hot path.

**Example (`PIIRedactor`, lifted from `llm-agent-rag/guard/redact.go:67-94`):**

```go
// Source: lifted from llm-agent-rag/guard/redact.go [VERIFIED: read]
// Patterns COPIED into core/policy/patterns.go (not imported — KC-3, KS-5)
package policy

type piiRedactor struct {
    rules []rule
}

type rule struct {
    kind        string
    pattern     *regexp.Regexp
    placeholder string
}

func NewPIIRedactor() Gate {
    return &piiRedactor{rules: []rule{
        // email, phone, ipv4 lifted directly from rag (language-agnostic)
        {kind: "email", pattern: regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`), placeholder: "[REDACTED:EMAIL]"},
        {kind: "phone", pattern: regexp.MustCompile(`\+?\b\d[\d ()\-]{7,}\d\b`), placeholder: "[REDACTED:PHONE]"},
        {kind: "ipv4", pattern: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), placeholder: "[REDACTED:IPV4]"},
        // SSN, credit_card DROPPED (US-locale-specific — see Decision E)
    }}
}

func (p *piiRedactor) Inspect(ctx context.Context, ev Event) Decision {
    // Apply on PreGenerate (input) and PostGenerate (output)
    switch ev.Kind {
    case PreGenerate:
        if ev.Req == nil {
            return Decision{Action: Allow}
        }
        redacted, hit := p.redact(allInputText(ev.Req))
        if !hit {
            return Decision{Action: Allow}
        }
        return Decision{Action: Replace, Reason: "pii_redacted", Replacement: redacted}
    case PostGenerate:
        if ev.Resp == nil {
            return Decision{Action: Allow}
        }
        redacted, hit := p.redact(ev.Resp.Text)
        if !hit {
            return Decision{Action: Allow}
        }
        return Decision{Action: Redact, Reason: "pii_redacted", Replacement: redacted}
    }
    return Decision{Action: Allow}
}
```

### Anti-Patterns to Avoid

- **Don't add a hook system.** KC-3 is explicit — decorator, not hooks.
  Resist any "what if we let users register a callback on the agent
  itself" sketch.
- **Don't import `llm-agent-rag/guard`.** KS-5: rag is a frozen fixed
  point. KC-3: lift patterns by copy. Each package owns its own
  source-of-truth.
- **Don't add a new `StreamEvent.Kind` for "policy blocked"**. K1 is
  locked. Blocked surfaces via the next `Next()` returning
  `&BlockedError{...}` (see Decision F).
- **Don't mutate the request in place.** `llm.Request` is the caller's
  value; gates return a `Replacement`, the decorator passes a new
  request to the inner model. Concurrency-safe and KC-5-friendly.
- **Don't enable `StreamDelta` by default.** Per-delta regex is
  expensive (every text chunk runs every gate's pattern set). Default
  OFF; gates opt in via a `WithStreamDelta()` option or by simply
  returning `Allow` on `Kind == StreamDelta` if not interested.
- **Don't run gates concurrently.** Within a single request, gates run
  in registration order, sequentially. Concurrent gate runs introduce
  a happens-before ambiguity that the spec can't resolve (which
  `Decision` wins if two gates both return `Block`?). Sequential is
  predictable and the regex cost is small enough not to matter.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PII pattern matching | Custom DFA / token-scanner | `regexp` (stdlib) | Stdlib is fine; no DFA performance need for ≤4 patterns per request |
| Capability-preserving wrap | A new 8-wrapper tree from scratch | Copy structure from `otelmodel.go:14-49`, swap fields | Proven, audited, line-by-line testable against the reference |
| Sentinel error infrastructure | `interface{Is(error)bool}` ad-hoc | `errors.New + fmt.Errorf("%w: ...", base)` (mirror `budget.ErrBudgetExceeded`) | Stdlib `errors.Is` plumbing works out of the box; matches the 35-01 sentinel family pattern |
| Streaming `Next` plumbing | New iterator | Wrap the inner `llm.StreamReader.Next()` and surface `io.EOF` per the K1 contract | Same shape as `otelmodel/streamReader` — proven pattern |
| Audit-log emission | Custom logger / channels | `func(Decision)` callback (`OnDecision`) | One-function callback is the smallest possible interface; users wire in their own slog/zerolog |
| Test mocks | New mock LLM | `llm.NewScriptedLLM(...)` | Canonical mock per CLAUDE.md; populates Capabilities, Usage; already covers Tool/Embed/Schema combinations |

**Key insight:** This phase is 90% "mirror an existing, proven shape"
and 10% "add 3 regex gates that already exist in rag". The only
genuinely-new design is the `Decision{Action, Reason, Replacement}`
shape — see Decision A.

## Decision A: `Decision` action shape — 4 actions

**The question:** What can a `Gate` return?

**The answer (recommended):**

```go
type DecisionAction uint8

const (
    Allow   DecisionAction = iota // pass through; no change
    Block                          // short-circuit; return ErrBlocked
    Redact                         // mutate Response.Text (post-call) — leaves the request/call intact
    Replace                        // substitute the Request before the call (pre-call) — denies the original
)

type Decision struct {
    Action      DecisionAction
    Reason      string  // gate-defined; "pii_detected", "instruction_override", "length_exceeded"
    Replacement string  // populated when Action == Redact or Replace
    Gate        string  // populated by the decorator (the gate's package-defined Name); read-only to callers
}
```

**Why these four:**

| Action | When | Decorator behavior |
|--------|------|--------------------|
| `Allow` | Default. Gate doesn't care about this event. | Pass through unchanged |
| `Block` | Hard veto — e.g., injection scanner saw "ignore previous instructions" | Decorator returns `&BlockedError{...}` to caller; inner model is NOT invoked (on Pre*) or response is NOT returned (on Post*) |
| `Redact` | Soft scrub — e.g., PII in `Response.Text`. Don't deny; clean. | Decorator copies `Response`, replaces `Text` with `Replacement`, returns the cleaned response to caller. Caller sees a "successful" response |
| `Replace` | Soft scrub on input — e.g., PII in user prompt. Don't deny; clean before sending to model. | Decorator copies `Request`, replaces the prompt text with `Replacement`, passes the new request to the inner model |

**Confidence: HIGH.** [CITED: KC-3 names "PIIRedactor" — implies a soft action like Redact/Replace] [VERIFIED: rag's `redact.go` uses replace-with-placeholder pattern; the action shape matches]

**Alternative considered:** 3-action enum (Allow/Block/Modify) with a
single `Modify` for both directions. **Rejected** — the rewrite target
differs (Request before, Response after); a single `Modify` action
requires sniffing `ev.Kind` to know which to rewrite. Two actions is
clearer.

**Open question** (planner ratifies in 36-01): should `Allow` be the
zero-value? Recommended YES — a gate that forgets to return anything
defaults to non-interference; mirrors Go's idiomatic zero-value-is-safe
principle.

## Decision B: `Event` shape — typed union with optional pointers

**The question:** Five event kinds; how does data flow?

**The answer (mirrors `llm.StreamEvent`'s kind+optional-pointer
pattern from `llm/stream.go:41-47`):**

```go
type EventKind uint8

const (
    PreGenerate  EventKind = iota // before inner.Generate; ev.Req != nil
    PostGenerate                  // after inner.Generate (success); ev.Req + ev.Resp != nil
    PreStream                     // before first inner.Next; ev.Req != nil
    StreamDelta                   // each inner.Next chunk (opt-in); ev.Req + ev.Delta != nil
    PostStream                    // EventDone or io.EOF; ev.Req != nil, ev.Delta = the EventDone (or nil on EOF)
)

type Event struct {
    Kind  EventKind
    Req   *llm.Request     // populated on all 5 kinds
    Resp  *llm.Response    // populated on PostGenerate only
    Delta *llm.StreamEvent // populated on StreamDelta and PostStream(EventDone)
}
```

**Why pointers, not value-copies:**
- Mirrors `llm.StreamEvent` which uses pointers for the typed-union
  payload (`ToolCall *ToolCallDelta`, `Usage *Usage`).
- Allows zero-allocation `Allow` paths — gate inspects, makes no change,
  returns. The decorator owns the request value-copy; the pointer is
  read-only from the gate's POV.
- Future extension: if a kind needs a new payload, add a new pointer
  field; doesn't break the existing fields' callers.

**Confidence: HIGH.** Pattern is identical to `llm.StreamEvent` (K1).

**When does each fire (decorator pseudocode):**

```go
// Generate path:
1. decorator builds Event{Kind: PreGenerate, Req: &req}
2. for each gate g: g.Inspect(ctx, ev) → Decision
3. on Block: return &BlockedError{...} (inner.Generate NOT invoked)
4. on Replace: rewrite req with Replacement, continue
5. resp, err := inner.Generate(ctx, req)
6. if err: return err (no PostGenerate fires on error)
7. decorator builds Event{Kind: PostGenerate, Req: &req, Resp: &resp}
8. for each gate g: g.Inspect(ctx, ev) → Decision
9. on Block: return &BlockedError{...} (response discarded)
10. on Redact: rewrite resp.Text with Replacement, continue
11. return resp, nil

// Stream path: see Pattern 2 above (Architecture Patterns)
```

## Decision C: `OnDecision` audit log — synchronous, in-request-goroutine

**The question:** Sync or async? Where does it run?

**The answer (recommended): synchronous, in the request goroutine.**

```go
type Config struct {
    Gates      []Gate
    OnDecision func(Decision) // synchronous; called for every non-Allow Decision
}
```

**Rationale:**
- **Symmetry with `otelmodel.Wrap`'s tracer.** Otel spans are
  synchronous in-line with `Generate`; same for policy decisions.
- **Caller controls async if needed.** A user who wants async logging
  writes `cfg.OnDecision = func(d Decision) { go logger.Log(d) }`. The
  package doesn't force the choice.
- **Allow decisions are NOT reported** to `OnDecision`. Reporting every
  `Allow` would be 10× the log volume of every other action and is
  observability noise. Only `Block` / `Redact` / `Replace` decisions
  fire the callback.
- **Recoverable panics in `OnDecision` should not crash the request.**
  The decorator wraps the callback in a `defer recover()` and ignores
  panics (with a future option to surface them). Mirrors stdlib's
  `http.HandlerFunc` panic handling.
- **No goroutine spawn.** A user wanting async writes the goroutine
  themselves; the decorator stays simple.

**Confidence: HIGH.** Symmetric with otelmodel; idiomatic Go callback.

**Trade-off:** A slow `OnDecision` (e.g., logging to a file with
`fsync`) blocks the request. Document this in the package doc; users
who want async wrap their own goroutine.

## Decision D: `ErrBlocked` semantics — sentinel + rich error type

**The question:** What does a caller see when a gate blocks?

**The answer (mirrors `budget.ErrBudgetExceeded` + `llm.AuthError`
combined pattern):**

```go
var ErrBlocked = errors.New("policy: blocked")

type BlockedError struct {
    Gate    string // gate's name (e.g., "InjectionScanner")
    Reason  string // gate's reason (e.g., "instruction_override")
    Wrapped error  // nil unless the gate returned an underlying error
}

func (e *BlockedError) Error() string {
    return fmt.Sprintf("policy: blocked by %s: %s", e.Gate, e.Reason)
}

func (e *BlockedError) Is(target error) bool {
    return target == ErrBlocked
}

func (e *BlockedError) Unwrap() error { return e.Wrapped }
```

**Caller experience:**

```go
resp, err := wrappedModel.Generate(ctx, req)
if errors.Is(err, policy.ErrBlocked) {
    var be *policy.BlockedError
    errors.As(err, &be)
    log.Printf("blocked by %s: %s", be.Gate, be.Reason)
}
```

**Confidence: HIGH.** Mirrors `llm.AuthError`'s `Provider + Wrapped`
shape exactly; sentinel-`Is` plus rich-`As` is the idiomatic Go
combo (verified by reading `llm/errors.go:35-95`).

**Interaction with budget:** Budget enforces at `generateFromPrompt`
(inside the wrapped model from policy's POV). If both a policy gate and
a budget cap would trip on the same request, **budget fires first**
because the chokepoint charges BEFORE invoking `model.Generate` (which
is where the policy wrapper lives). A blocked-by-policy request was
already pre-charged on the call counter (`Charge(Usage{Calls: 1})`) —
the call is "spent" from budget's POV even though policy denied it.
This is the intended ordering: budget counts attempts, policy decides
what to do with the attempt that budget allowed.

**Trade-off:** If a user wants "blocked-by-policy refunds the budget
charge" they need to write a custom `Tracker` that exposes a
`Refund(Usage)` method. Out of scope for v1.2; documented as
carry-forward.

## Decision E: Built-in regex subset — drop US-locale-specific

**The question:** The rag-side `guard` ships 5 PII patterns (email,
phone, ipv4, ssn, credit_card) and 4 injection patterns. Which subset
ships in core?

**The answer (recommended):**

| rag pattern | core/policy? | Reason |
|-------------|--------------|--------|
| `email` | YES | Language-agnostic, RFC-5322 ASCII-ish, universal |
| `phone` | YES | Loose enough to catch international formats (`+?\b\d[\d ()-]{7,}\d\b`) |
| `ipv4` | YES | Universal (numeric) |
| `ssn` (US, `\d{3}-\d{2}-\d{4}`) | **NO** | US-locale-specific; would false-positive on date-like / ID-like strings in other locales |
| `credit_card` (`\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{1,4}`) | **NO** | High false-positive rate against any 12-19 digit string; needs Luhn check to be useful; a "PII demo" not a "PII protector" — recorded as v1.3 candidate with Luhn |
| `instruction_override` (injection) | YES | Universal — English idiom; matches the v1.2-SUMMARY "OWASP-style starter set" |
| `disregard_above` (injection) | YES | Universal |
| `role_override` (injection) | YES | Universal |
| `prompt_exfiltration` (injection) | YES | Universal |

**Result: 3 PII patterns + 4 injection patterns + 1 length gate = the
"3 built-in gates" of CC-2.** Each gate carries multiple patterns; the
gate count is 3, not the pattern count.

**Confidence: MEDIUM-HIGH.** [ASSUMED: that SSN/credit_card being
US-specific is undesirable for a "core" library; the rag package is
demo-grade per CLAUDE.md's "intentionally demo-grade" carry-forward
note]. The operator may want SSN/credit_card included anyway; record
as a planner question. [VERIFIED: rag's exact regex bodies via direct
read of `redact.go:67-94`].

**Carry-forward:** A v1.3 add can ship per-locale gate sets
(`NewUSLocalePIIRedactor`, `NewEULocalePIIRedactor`) without touching
the core gate. KC-5-friendly (additive).

## Decision F: Streaming semantics — buffer-and-redact OR close-and-block, never mutate-mid-flight

**The question:** When a `StreamDelta` gate fires mid-stream, what does
the user see?

**The answer (recommended): two distinct contracts depending on `Decision.Action`.**

| Gate's action on `StreamDelta` | Decorator behavior | User sees |
|-------------------------------|---------------------|-----------|
| `Allow` (default for gates that don't opt in) | Pass `ev` through unchanged | Original `StreamEvent` |
| `Block` | (1) Close the inner stream via `r.inner.Close()`. (2) Return `&BlockedError{...}` from the NEXT `Next()` call — current `Next()` already committed to returning an event, so the block surfaces on the FOLLOWING `Next()`. Alternative: surface the block immediately by returning `(StreamEvent{}, &BlockedError{...})` from this same `Next()` and dropping the event the inner produced. **Recommendation: surface immediately** — fewer races, no "leak the last event then block". | Caller gets a `BlockedError` instead of `ev`. The redacted text was never seen |
| `Redact` | Rewrite `ev.Text` (for `EventTextDelta`) or `ev.Delta.Text` field with `Replacement`. Continue. | Caller sees the redacted delta; the original was never emitted |
| `Replace` | Same as Redact for streaming — there's no separate "replace mid-stream" semantics. Document `Replace` on `StreamDelta` as equivalent to `Redact` | Same as Redact |

**Why "no buffer-and-redact across deltas":**
- A PII pattern that spans two text deltas (e.g., `"ali"` then
  `"ce@ex"` then `"ample.com"`) won't match the regex in any single
  delta. Buffering across deltas requires either (a) emitting on EOF
  (defeats streaming), (b) emitting on whitespace boundary (heuristic),
  (c) holding a per-stream lookback buffer of N bytes (state).
- v1.2 ships **per-delta best-effort only**. Each delta is regex'd
  against patterns; if it doesn't span across deltas, redaction works;
  if it does, the pattern misses. Document this limitation in the
  package doc — same as rag's "best-effort" guard limitation noted in
  PROJECT.md "Known Tech Debt".
- This decision aligns with rag's `guard` package (no cross-chunk
  detection); core inherits the same fundamental limit. [CITED: v1.2
  ROADMAP §"Known Carry-forward Debt"]

**Trade-off:** PII can leak through streaming with patterns straddling
deltas. The mitigation is "use post-call redaction on the final
response" — `PostGenerate` redactor fires on the assembled text. A
user wanting both per-delta and final redaction registers two gate
instances. Document this composition pattern.

**Confidence: HIGH** for the per-delta best-effort decision (matches
rag's shape, matches K1's "adapters emit native granularity" principle
in `llm/stream.go:21-22`). **MEDIUM** for the "surface immediately
vs. on next Next()" sub-decision — the planner ratifies in 36-01.

## Decision G: Composition test — verify without importing otelmodel into core

**The question:** The integration test must prove
`policy.Wrap(otelmodel.Wrap(provider))` works — but otelmodel lives in
a sister repo and importing it into core would violate the stdlib-only
rule.

**The answer (recommended):**

The integration test in `policy/integration_test.go` does NOT import
`otelmodel`. Instead, it constructs an in-test "observation wrapper"
that mimics the otelmodel decorator's shape — a struct that wraps
`llm.ChatModel`, satisfies the same 4 interfaces via type-switch,
calls a counter on each `Generate` / `Stream`. This proves:

1. **Capability assertions survive both wrappers.** Test asserts
   `policy.Wrap(observe.Wrap(scriptedLLM)).(llm.ToolCaller)` returns
   `(_, true)` when the scripted LLM has Tools capability.
2. **Blocked-by-policy short-circuits BEFORE the observer fires.** Test
   wraps a scripted LLM in an observer (with a `generateCount`
   counter), wraps that in `policy.Wrap(..., gateThatAlwaysBlocks())`,
   issues a request, asserts (a) caller sees `BlockedError`, (b)
   observer's `generateCount == 0` (proves the policy outer short-
   circuited before the observation layer was reached).
3. **Per-stream events flow through correctly.** Test wraps in both
   layers, issues `Stream(...)`, drains the stream, asserts observer
   counted N deltas + 1 done, and policy's `OnDecision` was called for
   gate firings.

The mimicked observer is ~20 LOC of test-only code; the real
verification (that policy.Wrap + otelmodel.Wrap compose correctly
*outside* the test) happens implicitly because both wrappers follow
the same `otelmodel.Wrap` shape — if policy mirrors it correctly, real
otelmodel composes naturally with no further work.

**Sister-repo CI integration (Phase 36 → llm-agent-otel):** Out of
scope for v1.2 (the sister repo bumps in v1.3 ecosystem alignment).
Surface as a known follow-up: when `llm-agent-otel` bumps from
v0.2.1 → matches-core-v0.6.x, the sister repo can add its own
`policy_compose_test.go` that DOES import policy + otelmodel and runs
the real-world stack. That work is NOT v1.2.

**Confidence: HIGH** for the in-test observer approach (preserves
stdlib-only). HIGH for the deferral of real-otel-compose CI to v1.3.

## Decision H: `MaxInputLen` — bytes (not runes)

**The question:** Measure runes, bytes, or characters?

**The answer (recommended): bytes.** Rationale:
- **Provider input limits are byte-based.** OpenAI's prompt cap is
  expressed in tokens internally, but the HTTP body byte limit is the
  hard ceiling. Anthropic's limits are similar.
- **Cheap.** `len(req.Messages[i].Content)` is O(1).
- **Language-agnostic** in the sense that it doesn't double-count
  multi-byte UTF-8 codepoints (one Chinese character ≈ 3 bytes; an
  emoji ≈ 4 bytes; that's correct from the network-cost POV).
- **Predictable for callers.** "Cap at 100 KB" is more intuitive than
  "cap at 100 K runes".

`MaxInputLen(n int)` accepts the byte count. A future
`MaxInputLenRunes(n int)` can be added in v1.3 if a real need surfaces
(KC-5-friendly additive).

**Confidence: HIGH.** [ASSUMED: provider-side byte budgets are the
operative cap — common knowledge but not formally cited; planner can
quibble the unit in 36-01 without breaking design].

## Common Pitfalls

### Pitfall 1: Capability assertion lost after `policy.Wrap`

**What goes wrong:** A user `policy.Wrap(provider)`s a model that
implements `ToolCaller`, then `wrapped.(llm.ToolCaller)` fails. The
ReAct agent's tool-call branch falls back to scratchpad templating
silently.

**Why it happens:** Forgot to return the `toolWrapper` (or its
combinatorial siblings) — defaulted to the base `wrapper` which doesn't
satisfy `ToolCaller`.

**How to avoid:** Compile-time assertions in `policy/policy.go`,
mirroring `otelmodel.go:300-321`:

```go
var (
    _ llm.ChatModel         = (*wrapper)(nil)
    _ llm.ChatModel         = (*toolWrapper)(nil)
    _ llm.ToolCaller        = (*toolWrapper)(nil)
    _ llm.ChatModel         = (*toolEmbedSchemaWrapper)(nil)
    _ llm.ToolCaller        = (*toolEmbedSchemaWrapper)(nil)
    _ llm.Embedder          = (*toolEmbedSchemaWrapper)(nil)
    _ llm.StructuredOutputs = (*toolEmbedSchemaWrapper)(nil)
    // ... all 8 wrappers with all their interfaces
)
```

**Warning signs:** `go vet` passes but `TestWrap_PreservesCapabilities`
fails. Equivalent test (mirror otelmodel_test.go:22-41) catches it.

### Pitfall 2: `WithTools` rebinding drops the gates

**What goes wrong:** `tc, _ := wrapped.(llm.ToolCaller); bound, _ :=
tc.WithTools(tools); bound.Generate(...)` — gates don't fire because
`bound` is the raw inner re-bound, not policy-wrapped.

**Why it happens:** `WithTools` implementation forgot to re-wrap the
result.

**How to avoid:** Every `WithTools` / `WithSchema` method in the 8
wrappers MUST end with `return w.wrap(next)` — the `(*wrapper).wrap()`
helper closes over the gates. Mirrors `otelmodel.go:98-100, 154-162,
198-204`.

**Warning signs:** `TestWithTools_PreservesGates` — a test that wraps,
calls `WithTools`, fires a gate-blocked request through the rebound
model, asserts `BlockedError` (not silent success).

### Pitfall 3: Concurrent `Generate` calls share gate state

**What goes wrong:** Two goroutines call `wrappedModel.Generate(ctx,
req)` concurrently. The gates each hold internal state (e.g., per-
stream buffers). Race condition: one goroutine's "first call?" flag
flips while the other is still in PreStream.

**Why it happens:** Gate impl uses a mutable field on the gate struct
itself rather than per-call state.

**How to avoid:** **Gate state lives per-call**, not on the gate value.
The decorator passes a per-stream state container to the StreamReader.
For per-Generate gates, no state is held — the gate is a value-type
inspector. The 3 built-in gates are stateless (regex / length checks).
`go test -race` is the gate; mirrors 35-01's race test pattern.

**Warning signs:** Flaky tests under `-race`. The 36-03 integration
slice runs the compose-with-observer test under `-race`.

### Pitfall 4: `Replace` action mid-stream silently leaks original

**What goes wrong:** A gate returns `Replace` on `StreamDelta`. The
decorator interprets `Replace` as "rewrite, continue". But the inner
model's downstream `Next()` returns the NEXT delta, not the replaced
content of THIS one. If the decorator forgot to actually substitute
`ev.Text`, the original (un-redacted) text leaks.

**Why it happens:** The decorator's `StreamDelta` handling is the
trickiest part of the implementation — pre/post are straightforward
but stream interception is "intercept this exact event".

**How to avoid:** The decorator's `streamReader.Next()` MUST treat
`Redact` and `Replace` identically on `StreamDelta`: copy `ev`, rewrite
`Text` (or `Delta.Text`) with `Replacement`, return the copy. Test:
`TestStreamRedactor_RewritesDelta` — a scripted LLM emits a known PII
delta; policy wraps a PII redactor; test drains and asserts every
delta's `Text` matches the redacted form.

**Warning signs:** A "raw" PII string appears in the streamed output
under unit test. The 36-02 PII gate test catches it.

### Pitfall 5: `OnDecision` blocking the request goroutine

**What goes wrong:** User wires `OnDecision: func(d) { logger.Send(d)
}` where `logger.Send` does a synchronous HTTP POST. Every blocked
request waits for the HTTP POST. Latency through the roof.

**Why it happens:** Decision C documented sync semantics; user didn't
read the doc.

**How to avoid:** Doc-comment on `OnDecision` field of `Config` says
"called synchronously in the request goroutine. Use `go ...` inside
your callback if you need async". Plus an example in `examples/07-policy/`
showing the async pattern.

**Warning signs:** User-reported "policy doubled my p99 latency".
Documented; no automated test (the choice belongs to the user).

## Runtime State Inventory

Not applicable — Phase 36 is **strict additive new code**. No rename /
refactor / migration. No stored data, no live-service config, no
OS-registered state, no secrets/env vars, no build artifacts that
embed any string this phase touches. The `policy` package is brand new
in `v0.6.1`; nothing pre-exists to migrate.

**For each category, explicit "nothing found":**

| Category | Status |
|----------|--------|
| Stored data | None — `policy` is stateless; no DB/file/cache state. Verified: no DB usage in any v0.6.0 core file (core is stdlib-only; no DB drivers). |
| Live service config | None — core is a library, not a service. The `customer-support` demo sister repo doesn't pre-exist `policy`. |
| OS-registered state | None — `policy` is a library, no daemon / task scheduler / cron / launchd entries. |
| Secrets / env vars | None — `policy` defines no env vars; gates carry their config in-process via the constructor args. |
| Build artifacts | None — adding a new package doesn't invalidate any existing artifact; the only artifact is `go.mod` which stays unchanged (no new require). |

## Code Examples

Verified patterns to copy / mirror in implementation:

### Example A: `Wrap` factory with 8-wrapper type-switch

```go
// Source: VERBATIM SHAPE from llm-agent-otel/otelmodel/otelmodel.go:20-49
// (swap tracer fields for gates fields)
func Wrap(model llm.ChatModel, gates ...Gate) llm.ChatModel {
    return WrapConfig(model, Config{Gates: gates})
}

func WrapConfig(model llm.ChatModel, cfg Config) llm.ChatModel {
    base := &wrapper{inner: model, gates: cfg.Gates, onDecision: cfg.OnDecision}
    // ... 7-deep nested ok-checks producing 1 of 8 wrapper types
}
```

### Example B: PII gate via lifted regex set

```go
// Source: llm-agent-rag/guard/redact.go:67-94 — COPIED to policy/patterns.go
// US-locale dropped per Decision E
func defaultPIIRules() []rule {
    return []rule{
        {kind: "email", pattern: regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`), placeholder: "[REDACTED:EMAIL]"},
        {kind: "phone", pattern: regexp.MustCompile(`\+?\b\d[\d ()\-]{7,}\d\b`), placeholder: "[REDACTED:PHONE]"},
        {kind: "ipv4", pattern: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), placeholder: "[REDACTED:IPV4]"},
    }
}
```

### Example C: Injection gate

```go
// Source: llm-agent-rag/guard/inject.go:49-68 — COPIED to policy/patterns.go
func defaultInjectionRules() []rule {
    return []rule{
        {name: "instruction_override", pattern: regexp.MustCompile(`(?i)ignore\s+(all\s+|the\s+)?(previous|prior|above)\s+(instructions|prompts?)`)},
        {name: "disregard_above",       pattern: regexp.MustCompile(`(?i)disregard\s+(everything\s+|all\s+)?(the\s+)?above`)},
        {name: "role_override",         pattern: regexp.MustCompile(`(?i)(you\s+are\s+now\b|new\s+instructions\s*:|forget\s+(everything|all\s+previous))`)},
        {name: "prompt_exfiltration",   pattern: regexp.MustCompile(`(?i)(reveal|print|show|repeat|display)\s+(your\s+|the\s+)?(system\s+)?(prompt|instructions)`)},
    }
}
```

### Example D: Capability-preservation test

```go
// Source: VERBATIM SHAPE from llm-agent-otel/otelmodel/otelmodel_test.go:22-41
func TestWrap_PreservesCapabilities(t *testing.T) {
    model := llm.NewScriptedLLM(
        llm.WithProvider("scripted"),
        llm.WithModel("full"),
        llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true}),
        llm.WithResponses(llm.TextResponse("hello")),
    )
    wrapped := policy.Wrap(model)
    if _, ok := wrapped.(llm.ToolCaller); !ok { t.Fatal("lost ToolCaller") }
    if _, ok := wrapped.(llm.Embedder); !ok { t.Fatal("lost Embedder") }
    if _, ok := wrapped.(llm.StructuredOutputs); !ok { t.Fatal("lost StructuredOutputs") }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hooks on the agent (`agents.OnLLMCall`) | Decorator at model boundary (`policy.Wrap`) | KC-3 ratifies (2026-05-20) | Decorator inverts the dependency — provider doesn't know about policy; agent doesn't know about policy. Only the user composing the model knows. |
| Ad-hoc safety inline in each agent paradigm | Single decorator, all 5 paradigms inherit | KC-3 (v1.2) | No edit to ReAct/Simple/etc.; 100% paradigm coverage with one wrap |
| Stream events expanded to carry "blocked" metadata | Block surfaces via `Next()` returning `BlockedError`; K1 typed union locked | KC-3 + K1 (v1.2) | Existing K1 consumers (every adapter, otelmodel) need no update for v1.2 |

**Deprecated / not applicable:**
- Nothing in core deprecated (this is an additive package).
- `llm-agent-rag/guard` continues to exist on the rag side, unchanged.
  The pattern reuse in core is **by copy** (separate source of truth)
  to preserve the rag freeze (KS-5) while letting core evolve.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | stdlib `testing` + `go test -race` |
| Config file | None — pure stdlib |
| Quick run command | `cd .../llm-agent && GOWORK=off go test ./policy/... -count=1` |
| Full suite command | `cd .../llm-agent && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1 && GOWORK=off go test -race ./policy/... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CC-2 | `policy.Wrap` preserves Tool/Embed/Schema capability assertions | unit | `go test ./policy/ -run TestWrap_PreservesCapabilities` | ❌ Wave 0 (36-01) |
| CC-2 | 8 wrappers compile-time assert against their interfaces (`var _ = (*toolEmbedSchemaWrapper)(nil)` etc.) | compile-time | `go vet ./policy/` | ❌ Wave 0 (36-01) |
| CC-2 | `WithTools` rebinding re-wraps with gates intact | unit | `go test ./policy/ -run TestWithTools_PreservesGates` | ❌ Wave 0 (36-01) |
| CC-2 | `PreGenerate` Block returns BlockedError before inner.Generate fires | unit | `go test ./policy/ -run TestBlock_ShortCircuits` | ❌ Wave 0 (36-01) |
| CC-2 | `PostGenerate` Redact rewrites Response.Text | unit | `go test ./policy/ -run TestRedact_RewritesResponse` | ❌ Wave 0 (36-01) |
| CC-2 | `PreGenerate` Replace rewrites Request before inner.Generate | unit | `go test ./policy/ -run TestReplace_RewritesRequest` | ❌ Wave 0 (36-01) |
| CC-2 | PIIRedactor: email / phone / ipv4 patterns redact | unit | `go test ./policy/ -run TestPIIRedactor` | ❌ Wave 0 (36-02) |
| CC-2 | InjectionScanner: 4 patterns trigger Block | unit | `go test ./policy/ -run TestInjectionScanner` | ❌ Wave 0 (36-02) |
| CC-2 | MaxInputLen: byte-count trigger | unit | `go test ./policy/ -run TestMaxInputLen` | ❌ Wave 0 (36-02) |
| CC-2 | `OnDecision` callback fires sync, once per non-Allow decision | unit | `go test ./policy/ -run TestOnDecision_Sync` | ❌ Wave 0 (36-01) |
| CC-2 | `ErrBlocked` sentinel via errors.Is + BlockedError via errors.As | unit | `go test ./policy/ -run TestErrBlocked` | ❌ Wave 0 (36-01) |
| CC-2 | Streaming: `PreStream` Block surfaces immediately on next Next() | unit | `go test ./policy/ -run TestStream_BlockedOnPreStream` | ❌ Wave 0 (36-01) |
| CC-2 | Streaming: `StreamDelta` Redact rewrites text in passing event | unit | `go test ./policy/ -run TestStream_RedactDelta` | ❌ Wave 0 (36-01) |
| CC-2 | Streaming: `PostStream` fires on EventDone and on io.EOF | unit | `go test ./policy/ -run TestStream_PostStreamFires` | ❌ Wave 0 (36-01) |
| CC-2 | Compose with mimicked-otel observer: capability survives both wraps | integration | `go test ./policy/ -run TestComposeWithObserver` | ❌ Wave 0 (36-03) |
| CC-2 | Compose: blocked-by-policy short-circuits BEFORE observer's Generate | integration | `go test ./policy/ -run TestComposeWithObserver_ShortCircuit` | ❌ Wave 0 (36-03) |
| CC-2 | Compose: budget exhausted at chokepoint short-circuits BEFORE policy wrapper | integration | `go test ./policy/ -run TestComposeWithBudget_BudgetWinsAtChokepoint` | ❌ Wave 0 (36-03) |
| CC-2 | Race: concurrent Generate / Stream under gates | race | `go test -race ./policy/...` | ❌ Wave 0 (36-01 + 36-03) |
| CC-2 | stdlib-only: `go list -deps ./policy/` shows only stdlib imports | shape | `go list -f '{{join .Imports "\n"}}' ./policy/ | grep -vE '^(stdlib|github.com/costa92/llm-agent/llm)$'` returns 0 lines | ❌ Wave 0 (36-05) |
| CC-2 | Example runs deterministically | example | `cd examples && go run ./07-policy` exits 0 | ❌ Wave 0 (36-04) |
| CC-2 | go.mod unchanged after Phase 36 | shape | `git diff go.mod | wc -l` is 0 | ❌ Wave 0 (36-05) |

### Sampling Rate

- **Per task commit:** `go test ./policy/... -count=1`
- **Per wave merge:** `go test ./... -count=1 && go test -race ./policy/...`
- **Phase gate:** `go vet ./... && go test ./... -count=1 && go test -race ./policy/... && go list -deps ./policy/ | check-stdlib-only` (the audit in 36-05)

### Wave 0 Gaps

- [ ] `policy/policy.go` — Wrap + 8 wrapper types + Config (36-01)
- [ ] `policy/gate.go` — Gate / Event / EventKind / Decision / DecisionAction / ErrBlocked / BlockedError (36-01)
- [ ] `policy/policy_test.go` — capability + Wrap + Config tests (36-01)
- [ ] `policy/gate_test.go` — Decision shape + ErrBlocked sentinel + BlockedError As/Is (36-01)
- [ ] `policy/patterns.go` — regex sources (copied from rag) (36-02)
- [ ] `policy/pii.go` + `policy/pii_test.go` — PIIRedactor gate (36-02)
- [ ] `policy/injection.go` + `policy/injection_test.go` — InjectionScanner gate (36-02)
- [ ] `policy/length.go` + `policy/length_test.go` — MaxInputLen gate (36-02)
- [ ] `policy/integration_test.go` — compose with mimicked observer + budget short-circuit + per-paradigm smoke (36-03)
- [ ] `examples/07-policy/main.go` + `README.md` — deterministic example via ScriptedLLM (36-04)
- [ ] `policy/doc.go` — package-level doc citing KC-3, CC-2, decisions (A-H) recorded in 36-01 (36-01)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Policy doesn't touch auth; provider adapters handle creds. KC-5: no edit to `llm.AuthError` shape. |
| V3 Session Management | no | Library, not session-state. |
| V4 Access Control | no | Library — caller wires access checks via custom Gates if desired. |
| V5 Input Validation | yes | `MaxInputLen` + `InjectionScanner` are inputs validators. Patterns documented; user can add custom Gates. |
| V6 Cryptography | no | No crypto in policy; KC-3 explicit scope is regex + length. |
| V7 Error Handling & Logging | yes | `BlockedError` exposes the gate name + reason (not the original blocked content, which the gate could redact). `OnDecision` audit log captures every Block/Redact/Replace decision sync. |
| V8 Data Protection | yes | `PIIRedactor` is the primary control. Patterns: email, phone, IPv4. SSN / credit_card dropped (Decision E — US-locale). Doc-commented as best-effort regex (Carry-forward debt — same fundamental limit as rag's `guard`). |
| V9 Communications | no | Network layer is the provider adapter's concern. |
| V11 Business Logic | partial | Custom Gates can express business rules (e.g., "never call this tool on Sundays"); not a built-in. |

### Known Threat Patterns for `llm.ChatModel` decorator

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt injection in user input | Tampering | `InjectionScanner` Gate — regex on the 4 documented patterns; returns `Block` decision; surfaces `BlockedError` |
| PII leakage to provider | Information Disclosure | `PIIRedactor` Gate — regex on email/phone/IPv4; returns `Replace` decision pre-call so PII never reaches the network |
| PII leakage to caller | Information Disclosure | `PIIRedactor` Gate — returns `Redact` decision post-call so PII in `Response.Text` is scrubbed before the caller sees it |
| Oversized input → provider 4xx + cost | Denial of Service + Cost | `MaxInputLen` Gate — pre-call byte-count check; `Block` decision; no network round-trip |
| Audit-log gap (no record of denied requests) | Repudiation | `OnDecision` callback fires synchronously for every non-Allow decision; caller wires their own logger |
| Capability-assertion drop | Spoofing (capability impersonation) | Compile-time `var _ llm.ToolCaller = (*toolWrapper)(nil)` assertions; `TestWrap_PreservesCapabilities` runtime check |

**Carry-forward debt (acknowledged):**
- Regex-based content safety is **best-effort** — catches known PII +
  injection patterns, not novel/obfuscated attacks. Inherited from
  rag's `guard` package; recorded in v1.2 ROADMAP §"Known Carry-forward
  Debt" and PROJECT.md "Known Tech Debt". A v1.x add can ship
  ML-classifier-backed gates outside core (in a sister repo, with
  non-stdlib deps).

## Project Constraints (from CLAUDE.md)

| Constraint | Source | Impact on Phase 36 |
|------------|--------|---------------------|
| Core repo stays stdlib-only | CLAUDE.md Rule 1 | `policy/` imports only stdlib + `github.com/costa92/llm-agent/llm`. Zero new `require` in `go.mod`; no `go.sum` created. |
| No K8s | CLAUDE.md Rule 2 | n/a — policy is a library |
| No `replace` in tagged release | CLAUDE.md Rule 3 | `v0.6.1` tag wave (Phase 36 → 38) must have no replace directives in `go.mod`. The CI dep-currency gate enforces. |
| `go.work` is `.gitignore`d | CLAUDE.md Rule 4 | All `go test` commands in `<verify>` blocks set `GOWORK=off`. |
| Capabilities per-(provider × model) | CLAUDE.md Rule 5 (K2) | `policy.Wrap` does NOT inspect `Info().Capabilities.Tools` to decide whether to wrap `WithTools` — instead it does Go-type assertion AT WRAP TIME (which is the K2 contract). Capability negotiation remains the caller's job. |
| StreamEvent typed union, stable Index | CLAUDE.md Rule 6 (K1) | No new `StreamEvent.Kind` in v1.2. Block on stream surfaces via `BlockedError` from `Next()`, not via a new kind. |
| OTel attaches as decorator | CLAUDE.md Rule 7 (K3) | Policy mirrors this — decorator, never hooks. |
| Refsvc hard caps + DISABLE_LLM=1 | CLAUDE.md Rule 8 (K7) | n/a — Phase 36 is core only; refsvc is `llm-agent-customer-support` (sister, not in v1.2 scope per CC-2). |
| Files NOT to touch | CLAUDE.md "Files you should NOT touch" | `LICENSE` / `OWNERS` / `.github/workflows/*` / `go.mod` (no new require) — none touched by Phase 36. |
| Use `ScriptedLLM` for examples | CLAUDE.md "When the user asks for code" | `examples/07-policy/main.go` MUST use `ScriptedLLM` (no real providers); deterministic. |
| Tests via `go vet ./... && go test ./...` | CLAUDE.md "When the user asks for code" | Phase exit gate is this command set, augmented with `-race` on `./policy/...`. |
| No `go.sum` by design | CLAUDE.md "When the user asks for code" | Exit-gate check: `git status --short go.sum` returns empty. |

## Slice Breakdown (recommended — planner ratifies in 36-01)

Recommended **5 slices**, all in the core repo `llm-agent`. Total
estimated effort: ~1100-1400 LOC including tests + example + docs.
Mirrors the Phase 35 5-slice shape (skeleton → integrate → ... →
example → exit gate) but specialized for the policy domain (the
"integrate" step is split across decorator slice + gates slice
because the decorator and the gates are independently testable).

| Slice | Wave | Type | Repo | Files modified | Requirements | Must-haves |
|---|---|---|---|---|---|---|
| **36-01** | 1 | execute | `llm-agent` | `policy/policy.go` (new); `policy/gate.go` (new); `policy/policy_test.go` (new); `policy/gate_test.go` (new); `policy/doc.go` (new) | CC-2 (decorator skeleton) | **Decorator + types only — no gates yet.** Surface: `Wrap`, `WrapConfig`, `Config{Gates, OnDecision}`, `Gate` interface, `Event{Kind, Req, Resp, Delta}`, 5 `EventKind` constants, `Decision{Action, Reason, Replacement, Gate}`, 4 `DecisionAction` constants (Allow/Block/Redact/Replace; Allow == 0), `ErrBlocked` sentinel, `BlockedError` rich error. All 8 wrapper types (`wrapper`, `toolWrapper`, …, `toolEmbedSchemaWrapper`) mirroring `otelmodel.go:14-49` line-for-line in shape. The `(*wrapper).wrap(next)` helper closes over gates+onDecision. The `streamReader` decorator handles PreStream/StreamDelta/PostStream lifecycle. Compile-time assertions for all 8 wrappers × their interfaces (lines mirror `otelmodel.go:300-321`). Tests: `TestWrap_PreservesCapabilities` (all 8 combinations using ScriptedLLM with varying Capabilities), `TestBlock_ShortCircuits` (gate returns Block on PreGenerate, inner.Generate never called), `TestRedact_RewritesResponse`, `TestReplace_RewritesRequest`, `TestStream_BlockedOnPreStream`, `TestStream_RedactDelta`, `TestStream_PostStreamFires`, `TestOnDecision_Sync` (callback fires exactly once per non-Allow decision; recoverable panic doesn't crash), `TestErrBlocked` (errors.Is + errors.As), `TestWithTools_PreservesGates` (rebind doesn't drop), `TestWithSchema_PreservesGates`. Race test: `go test -race`. Exit gate: `go vet ./policy/... && go test ./policy/... -count=1 && go test -race ./policy/... && go list -deps ./policy/ | grep -vE '^(stdlib-or-internal)$' | wc -l == 0`. |
| **36-02** | 2 | execute | `llm-agent` | `policy/patterns.go` (new — copied from rag); `policy/pii.go` (new); `policy/injection.go` (new); `policy/length.go` (new); `policy/pii_test.go`; `policy/injection_test.go`; `policy/length_test.go` | CC-2 (3 built-in gates) | Three gates as value types implementing `Gate`. (a) `NewPIIRedactor()` → fires on PreGenerate (Replace) and PostGenerate (Redact); regex subset per Decision E (email, phone, IPv4 only — DROP ssn / credit_card from rag's source). (b) `NewInjectionScanner()` → fires on PreGenerate only (returns Block on match); 4 patterns from rag's `inject.go` (instruction_override / disregard_above / role_override / prompt_exfiltration). (c) `NewMaxInputLen(n int)` → fires on PreGenerate only (returns Block when `byteLen(req.SystemPrompt) + sum(len(m.Content) for m in req.Messages) > n`). Each gate's test exercises the canonical positive and negative cases mirroring rag's test pattern (`redact_test.go`, `inject_test.go`). Stream tests for PII: streamed delta containing email → redacted in passing. Exit gate: `go vet ./policy/... && go test ./policy/... -count=1 && go test -race ./policy/...`. |
| **36-03** | 3 | execute | `llm-agent` | `policy/integration_test.go` (new) | CC-2 (compose-with-otel + compose-with-budget integration) | Per Decision G, an in-test **mimicked observer** wrapper (~20 LOC) that mirrors `otelmodel.Wrap`'s shape (counts Generate / Stream calls). Tests: (a) `TestCompose_CapabilityPreserved` — wrap ScriptedLLM (Tools+Embeds+Schema) in `observer.Wrap(...)` then `policy.Wrap(...)`; assert all 3 capability interface assertions still pass on the outer. (b) `TestCompose_BlockedByPolicyShortCircuits` — same stack with a gate that always Blocks; assert observer's generateCount == 0 (the outer policy short-circuited before the observer was reached). (c) `TestCompose_StreamingThroughBothLayers` — wrap, stream a 3-delta scripted response, drain; assert observer counted 3 deltas + 1 done AND policy's OnDecision callback fired for any non-Allow decisions. (d) `TestCompose_BudgetBeatsPolicyAtChokepoint` — wrap a model in `policy.Wrap(...)`, attach `Budget{MaxCalls:0}` (will deny pre-call) to ctx, run through `SimpleAgent`; assert `errors.Is(err, budget.ErrCallsExceeded)` AND NOT `errors.Is(err, policy.ErrBlocked)` — proves budget chokepoint fires BEFORE the policy decorator. (e) `TestCompose_PerParadigmSmoke` — table-driven over Simple/ReAct/Reflection/PlanSolve/FunctionCall × policy-wrapped scripted model; assert each paradigm propagates `BlockedError` upward without panic. Exit gate: `go vet ./policy/... && go test ./policy/... -count=1 && go test -race ./policy/...`. |
| **36-04** | 4 | execute | `llm-agent` | `examples/07-policy/main.go` (new); `examples/07-policy/README.md` (new) | CC-2 (example) | A deterministic example using `ScriptedLLM` (per CLAUDE.md). Three demo functions mirroring `examples/06-budget/main.go` shape: (i) `demoPIIRedaction` — user prompt contains an email; PIIRedactor `Replace`s in PreGenerate; show the rewritten request via a counting wrapper around the scripted LLM. (ii) `demoInjectionBlock` — user prompt contains "ignore previous instructions"; InjectionScanner Blocks; demo prints `errors.Is(err, policy.ErrBlocked) == true` and inspects `BlockedError.Gate` + `BlockedError.Reason`. (iii) `demoMaxInputLen` — oversized prompt; MaxInputLen Blocks before network. Also a `demoCompose` showing `policy.Wrap(otelmodel.Wrap(provider))` composition note in the README (NOT in main.go — main.go stays stdlib-only). README: ≤80 lines, shows the canonical `policy.Wrap(model, policy.NewPIIRedactor(), policy.NewInjectionScanner(), policy.NewMaxInputLen(4096))` setup + the composition stack with otel. Exit gate: `cd examples && GOWORK=off go run ./07-policy` exits 0; `go vet ./examples/07-policy/...`. |
| **36-05** | 5 | execute | `llm-agent` | (verify-only — no Go source mod) | CC-2 (exit gate) | Phase exit gate, run from `llm-agent/`. (a) `go vet ./... && go test ./... -count=1 && go test -race ./policy/... -count=1` — all green. (b) `go list -deps ./policy/` shows only stdlib + the package's own imports (`github.com/costa92/llm-agent/llm`); no new third-party module. (c) `git diff main -- go.mod go.sum` is empty (no new require; no `go.sum` created). (d) `git diff main -- llm/ agents/` (no edits) — KC-5 verification. (e) `git diff main -- agent_chatmodel.go agents.go simple.go react.go plan_solve.go reflection.go function_call.go` — zero edits (Phase 35 already wired budget; Phase 36 doesn't touch the chokepoint). (f) `go list -m all` — module set unchanged (only `github.com/costa92/llm-agent-rag v1.0.1`). (g) Tag and push `v0.6.1` from `main` (operator action — slice records the command but the operator runs it). |

Wave structure is strictly sequential. 36-01 → 36-02 is decorator-then-
built-in-gates (gates need the `Gate` interface from 36-01). 36-02 → 36-03
is gates-then-integration (integration test needs working gates). 36-03 →
36-04 is integration-then-example (example uses the verified compose
patterns). 36-04 → 36-05 is example-then-exit-gate. No parallelization
gain available within the phase.

**Sizing.** ~400 LOC decorator + 8-wrapper tree (36-01) + ~250 LOC for 3
gates (36-02) + ~150 LOC integration tests (36-03) + ~120 LOC example
(36-04) + 0 LOC exit gate (36-05). Plus ~600 LOC tests across the
slices. Total ~1100-1400 LOC. Matches the v1.2 SUMMARY's "M" estimate
(~700 LOC for the package alone, integration tests add the rest).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Decorator's 8-wrapper tree mirrors `otelmodel.go:14-49` exactly | Architecture Patterns | LOW — verified by direct file read; the otelmodel pattern is the K3 contract |
| A2 | `regexp` (stdlib) suffices for the 3 built-in gates' patterns | Standard Stack | LOW — verified by reading rag's `guard` which uses only `regexp` |
| A3 | Provider input limits are byte-based; `MaxInputLen` should measure bytes | Decision H | MEDIUM — common knowledge but not cited; planner may quibble unit (bytes vs runes vs tokens). [ASSUMED] |
| A4 | SSN + credit_card regex are US-locale-specific and should be dropped from core | Decision E | MEDIUM — operator preference call; planner can ratify "keep them anyway, document as US-only" instead. [ASSUMED] |
| A5 | `OnDecision` synchronous in the request goroutine is correct symmetry with otelmodel | Decision C | LOW — direct symmetry with `otelmodel.Wrap` which IS synchronous |
| A6 | Compose-with-otel CAN be tested without importing otelmodel via a mimicked observer | Decision G | LOW — the test verifies the SHAPE, and the shape is the otelmodel.Wrap shape. If policy mirrors the shape, real otel compose works |
| A7 | Per-delta streaming redaction is acceptable as best-effort (cross-delta PII can leak) | Decision F | MEDIUM — documented limitation; matches rag's `guard` known limit; if operator wants cross-delta buffering, the design changes [ASSUMED to be acceptable] |
| A8 | The composition `policy.Wrap(budget.Wrap(otelmodel.Wrap(provider)))` is the v1.3 stack; v1.2 stack is `policy.Wrap(otelmodel.Wrap(provider))` with budget at `generateFromPrompt` | Decision D, Constraint Inventory | LOW — verified by reading 35-RESEARCH.md §"Carry-forward notes" |
| A9 | `Replace` action zero-value is `Allow` (a forgetful gate is non-interfering) | Decision A | LOW — Go idiom; planner ratifies |
| A10 | Blocked-stream surfaces via the next `Next()` call returning `BlockedError`, not by mutating an existing `StreamEvent.Kind` | Decision F | LOW — K1 is locked; no new kinds in v1.2 |
| A11 | Budget exhaustion at the chokepoint fires BEFORE the policy decorator (because the decorator wraps `model.Generate`, but the chokepoint pre-charges before calling `model.Generate`) | Decision D | LOW — verified by reading the shipped `agent_chatmodel.go` |

## Open Questions

1. **Should `policy.Wrap` accept a single error from `OnDecision`?**
   - What we know: Decision C says sync, no error return. Otelmodel's
     callback (the tracer) doesn't return errors either.
   - What's unclear: If a user's audit logger fails (disk full), should
     the request fail? Symmetric with otel: no — observation never
     interferes with the request path.
   - Recommendation: `OnDecision func(Decision)` — no error return.
     Document that any errors in `OnDecision` are the user's to handle
     (e.g., panic + the decorator recovers).

2. **Drop SSN and credit_card patterns from the core PII set?**
   - What we know: Both are US-locale-specific; credit_card has
     high false-positive rate without Luhn check.
   - What's unclear: Operator may want them anyway for "demo
     completeness" — rag includes them.
   - Recommendation: Drop in v1.2 core; add `NewUSLocalePIIRedactor()`
     in v1.3 as additive (KC-5-friendly). **Planner asks operator in
     36-01 before writing the regex set.**

3. **`MaxInputLen` — measure bytes or runes?**
   - What we know: Provider HTTP byte budgets are byte-based; runes
     give better cross-language fairness.
   - What's unclear: Which dimension is the user actually trying to
     cap?
   - Recommendation: Bytes for v1.2. Add `MaxInputLenRunes(n int)` in
     v1.3 if needed.

4. **`StreamDelta` enabled by default for which gates?**
   - What we know: Per-delta regex is expensive; per-call regex is
     cheap.
   - What's unclear: Should the PIIRedactor's default behavior include
     `StreamDelta` inspection, or only `PreGenerate`/`PostGenerate`?
   - Recommendation: Default `StreamDelta` OFF for all 3 built-in
     gates. Gates opt in via a constructor option:
     `NewPIIRedactor(policy.WithStreamRedaction())`. Documented as
     best-effort per-delta with the "spans-deltas can leak" caveat
     (Decision F).

5. **Should `BlockedError.Wrapped` carry the original `Decision`?**
   - What we know: The error carries `Gate` and `Reason` strings.
   - What's unclear: Sometimes callers want the full Decision struct
     for introspection (e.g., "what did the gate try to Replace
     it with?").
   - Recommendation: Add a `BlockedError.Decision Decision` field
     (struct copy). Planner ratifies in 36-01.

6. **Order of gates when multiple return non-Allow?**
   - What we know: Gates run sequentially in registration order.
   - What's unclear: If gate-0 returns `Replace` and gate-1 returns
     `Block`, does the Block win? Or does Replace happen first,
     then gate-1 sees the rewritten request?
   - Recommendation: **"First non-Allow wins."** Gate-0's `Replace`
     rewrites the request; gate-1 then sees the rewritten request and
     may still Block. If gate-0 returns `Block`, the chain
     short-circuits before gate-1 even runs. Document this order
     contract in the package doc.

7. **`go.work` interaction.** The repo's `go.work` is `.gitignore`d
   (CLAUDE.md Rule 4). Phase 36 tests must run under `GOWORK=off` to
   ensure the policy package doesn't accidentally pull from a local
   `replace` directive into `llm-agent-otel`. All `<verify>` blocks
   above include `GOWORK=off`. No change.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` toolchain | All slices | ✓ (assumed) | go 1.26.0 from `go.mod` | — |
| stdlib `regexp` | gates | ✓ | stdlib | — |
| stdlib `testing` | tests | ✓ | stdlib | — |
| `git` | exit gate (`git diff`) | ✓ | system | — |
| `llm-agent-otel` (sister) | NOT REQUIRED | n/a | — | Decision G — in-test mimicked observer |
| `llm-agent-rag/guard` source files (for copying patterns) | 36-02 | ✓ — verified by direct read | v1.0.1 (tagged) | — |
| `ScriptedLLM` (in `llm/`) | tests + example | ✓ — verified | v0.6.0 | — |
| `agentstest` (sub-package) | tests (if needed) | ✓ — shipped 2026-05-21 | v0.6.0 | not strictly needed; built-in gates' tests don't need stub Tools |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** `llm-agent-otel` (Decision G
provides the in-test mimicked observer; real otel compose is
out-of-scope sister-repo work for v1.3).

## Sources

### Primary (HIGH confidence — direct file reads, this session)

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-otel/otelmodel/otelmodel.go` — the K3 decorator pattern that policy mirrors (lines 14-49 Wrap factory; 102-148 streamReader; 300-321 compile-time assertions)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-otel/otelmodel/otelmodel_test.go` — capability-preservation test shape (lines 22-41 `TestWrap_PreservesCapabilities`)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/llm/chatmodel.go` — `ChatModel` interface contract
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/llm/capabilities.go` — `ToolCaller` / `Embedder` / `StructuredOutputs` interfaces (the 4 capability interfaces policy must preserve)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/llm/stream.go` — `StreamReader` interface + `StreamEvent` typed union (K1 locked)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/llm/types.go` — `Request` / `Response` / `Usage` shapes
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/llm/errors.go` — `ErrCapabilityNotSupported`, `AuthError`, `RateLimitError` (the sentinel + rich-error pattern policy mirrors for `ErrBlocked` + `BlockedError`)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/budget/budget.go` — the immediately-prior sibling package (sentinel family `ErrBudgetExceeded` + dim wraps, Tracker interface, Q1/Q2 design memos)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/budget/doc.go` — package-doc style template
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/budget/budget_test.go` — test patterns (race test, table-driven, errors.Is umbrella checks)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/agent_chatmodel.go` — verified the chokepoint is shipped (Phase 35-02); budget enforcement happens BEFORE `model.Generate` (which is where the policy decorator wraps), proving policy DOES NOT need to budget-aware
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-rag/guard/redact.go` — PII regex sources to lift (email/phone/IPv4/SSN/credit_card)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-rag/guard/inject.go` — injection regex sources to lift (4 patterns)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent-rag/guard/redact_test.go` and `inject_test.go` — test pattern templates
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/examples/06-budget/main.go` — example shape template
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md` — sibling-phase research template; Decision-1/2/3/4 voice + structure
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/phases/35-budget-and-cancellation-context/35-01-PLAN.md` — sibling-phase plan template
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/v1.2-REQUIREMENTS.md` — CC-2 verbatim
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/v1.2-ROADMAP.md` — Phase 36 planned work
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/.planning/research/v1.2-core-capability-deepening-SUMMARY.md` — KC-3 keystone (the locked design)
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/CLAUDE.md` — hard rules

### Secondary (MEDIUM confidence)

- ASVS category mapping — extrapolated from common library responsibilities; not cited to a specific ASVS doc URL (Phase 36 is library code, not service code, so most ASVS categories don't apply).

### Tertiary (LOW confidence — none, flagged for validation)

- None. Every claim above is grounded in direct file reads from this session.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — every type / function name is grounded in
  KC-3's verbatim spec + the read of `otelmodel.go`. The 8-wrapper
  tree shape is verbatim from the K3 reference.
- Architecture (decorator + 8 wrappers + streamReader): HIGH — mirror
  proven `otelmodel.Wrap` line-for-line; capability preservation is the
  K3 contract, audited and shipped.
- Built-in gate regex set: MEDIUM-HIGH — verified by direct read of
  rag's `guard`. The drop-SSN/credit-card decision is HIGH-confidence
  for "core should be locale-agnostic" but MEDIUM-confidence the
  operator agrees. **Planner asks operator in 36-01 before locking the
  regex set.**
- Streaming semantics (Decision F): MEDIUM — per-delta best-effort
  matches rag's known limit; the "surface block immediately vs. next
  Next()" sub-decision is recorded for planner ratification.
- Composition with budget (Decision D): HIGH — verified by direct
  read of the shipped `agent_chatmodel.go`. Budget fires at the
  chokepoint (before model.Generate); policy wraps model.Generate.
- Slice breakdown (5 slices): HIGH — mirrors the proven Phase 35
  shape; sequencing is dependency-forced.

**Research date:** 2026-05-21
**Valid until:** 2026-06-20 (30 days — stable domain; the only mover
is the operator's ratification of A3-A7 in 36-01)

---

*Researched 2026-05-21 by the Phase 36 research spawn. Voice + structure
mirror `.planning/phases/35-budget-and-cancellation-context/35-RESEARCH.md`.
Every file path cited verified by direct read against the working tree
at `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem`.
The decorator design mirrors `otelmodel.Wrap` per KC-3 line-for-line
in shape; the 3 built-in gates' patterns are lifted by copy from
`llm-agent-rag/guard/{redact,inject}.go` with SSN + credit_card dropped
per Decision E (planner to ratify with operator in 36-01).*

## RESEARCH COMPLETE
