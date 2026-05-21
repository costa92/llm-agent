> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.

# Phase 35 Research: Budget / cancellation context

**Researched:** 2026-05-20
**Phase:** 35 — budget / cancellation context (first v1.2 phase)
**Requirement:** CC-1
**Repo touched:** `llm-agent` (core only)
**Upstream:** `.planning/research/v1.2-core-capability-deepening-SUMMARY.md` —
keystone KC-4 (budget = ctx-keyed propagation + `Tracker` enforcement,
cost-table opt-in) and KC-5 (additive only, no `/v2`, no edit to
`llm.ChatModel` / `agents.Agent`). v1.1 closed clean — no prior-phase
dependency in flight.

## Scope (CC-1 verbatim)

> A `budget` package is shipped in core with `budget.WithBudget(ctx,
> *Tracker) context.Context`, `budget.From(ctx) *Tracker` (safe no-op on
> absent budget), `type Tracker interface{ Charge(Usage) error; Remaining()
> Budget }`, built-in trackers `budget.NewStrict` and `budget.NewSoft`,
> and is integrated at the `generateFromPrompt` chokepoint so every
> existing agent paradigm (Simple/ReAct/Reflection/PlanSolve/FunctionCall)
> honors it with zero behavior change when no budget is set. Cost is
> opt-in / outside core — core ships `Budget.Cost float64` plumbing only;
> no provider→$ table in core. The core stays stdlib-only (`go list -deps
> ./...` lists zero third-party modules; no edit to `llm.ChatModel` or
> `agents.Agent` — KC-5).

One sentence: a stdlib-only `budget` package that propagates by ctx and
enforces at the one helper every paradigm shares, with no edit to the
validated public types.

## Constraint inventory

- **Stdlib-only core (CLAUDE.md Rule 1, KC-5).** `go.mod`
  (`/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent/go.mod`)
  has exactly one `require` — `github.com/costa92/llm-agent-rag v1.0.1`
  for the RAG facade — and the budget package must add zero. Means no
  tokenizer dep (tiktoken / sentencepiece) in core; means no cost-table
  lookup library.
- **No `/v2` import path (KC-5).** v0.5.1 → v0.6.0 is a **minor**
  (additive) bump. Existing v0.5.1 callers must compile unchanged against
  v0.6.0. New package + new optional interfaces are the only shapes
  allowed.
- **Compose with K1 (typed StreamEvent union) — but in practice K1
  is currently unexercised by agent paradigms.** Every paradigm calls
  `generateFromPrompt` → `model.Generate(...)`, **not** `model.Stream(...)`
  (verified by grep, see Chokepoint section). So Phase 35 has to *be
  ready for streaming* but does not have to wire it through any existing
  agent in this milestone.
- **Compose with K2 (per-(provider × model) capabilities).** `Tracker`
  has no opinion about which model — but if cost-charging is wired in
  (KC-4 says it's opt-in), the per-model rate comes from a user-supplied
  `CostMapper`, not from core. Core ships `Budget.Cost float64` plumbing
  only.
- **Compose with K3 (OTel decorator pattern).** `otelmodel.Wrap(inner
  ChatModel) ChatModel` is the canonical capability-preserving
  decorator. A `budget.Wrap(inner)` decorator that mirrors this shape is
  *natural* but **not the chokepoint of choice** — see Decision 3.
- **Validated public types unchanged.** `llm.ChatModel`,
  `llm.StreamReader`, `llm.StreamEvent`, `agents.Agent`,
  `agents.Result` — none edited. `agents.Usage` (`agent.go:52`) is the
  shipped per-Run accumulator and already carries `LLMCalls + Tokens` —
  Phase 35 may add a *new* `agents.Usage` field only if it's `omitempty`
  and additive; preferred path is to keep the budget plumbing **inside
  `budget.Usage`** (a distinct type) so `agents.Usage` is untouched.
- **`MaxSteps`/`MaxTurns`/`MaxParallel` per-paradigm caps remain.** KC-4
  is explicit: per-agent step caps and cross-agent budgets coexist; no
  deprecation. The budget enforces independently of and concurrently with
  `MaxSteps`.

## Chokepoint discovery (correction to v1.2 SUMMARY)

The v1.2 research and CC-1 wording both name `generateFromPrompt` as the
integration chokepoint. The orchestrator's pre-flight grep claimed this
symbol does NOT exist in core. **The pre-flight grep is wrong; the
symbol DOES exist.** Verified:

| Location | Line | Definition / use |
|---|---|---|
| `llm-agent/agent_chatmodel.go` | 10 | **definition** — `func generateFromPrompt(ctx context.Context, model llm.ChatModel, systemPrompt, prompt string) (llm.Response, error)`; body calls `model.Generate(ctx, req)` (line 17) |
| `llm-agent/simple.go` | 52 | SimpleAgent uses it for its one call |
| `llm-agent/react.go` | 104 | ReAct scratchpad loop |
| `llm-agent/react.go` | 160 | ReAct native-tools branch |
| `llm-agent/reflection.go` | 79 | Reflection initial draft |
| `llm-agent/reflection.go` | 92 | Reflection critique |
| `llm-agent/reflection.go` | 108 | Reflection revise |
| `llm-agent/plan_solve.go` | 79 | PlanSolve planning |
| `llm-agent/plan_solve.go` | 100 | PlanSolve per-step |
| `llm-agent/plan_solve.go` | 117 | PlanSolve synthesis |
| `llm-agent/function_call.go` | 81 | FunctionCall tool-binding path |

Why the pre-flight grep missed it: the file `agent_chatmodel.go` sits
**at the repo root** (the core repo uses a *flat* layout — `package
agents` declared in files at `/llm-agent/*.go`), not under an
`agents/` directory. The grep over `agents/` `llm/` `orchestrate/`
naturally returned nothing.

**There is ONE chokepoint, not five.** All 5 paradigms × 10 call sites
funnel through `generateFromPrompt`. This is unambiguously the correct
integration point — single edit, all paradigms covered.

**Other Generate call sites that DO bypass `generateFromPrompt`** (out of
scope for CC-1 but recorded as carry-forward):

| Location | Purpose |
|---|---|
| `llm-agent/bench/winrate.go:139` | Bench harness — judges LLM A vs B |
| `llm-agent/bench/judge.go:121` | Bench-as-judge LLM call |
| `llm-agent/rag/rag.go:479` | RAG facade `Ask` answer generation |
| `llm-agent/context/compress.go:101` | GSSC context-compression |

These are auxiliary subsystems (bench/eval, rag-facade, context
pipeline), not agent paradigms. CC-1 says "every existing agent
paradigm honors it" — these are not agent paradigms and are
deliberately out of scope. Decision 3 (decorator option) addresses
the *user-side* path to enforce here without expanding Phase 35.

**Stream is unexercised in core agent paradigms.** Verified:
`grep -E "Stream\(ctx" *.go` returns hits only in `RunStream(ctx
context.Context, …)` (the agent's *channel-based* emitter that emits
`StepEvent`s) and in `llm/chatmodel.go` (the interface declaration).
**No agent paradigm calls `model.Stream(ctx, req)` today** — all 5
emit `StepEvent`s synthesized from non-streaming `model.Generate`
calls (`agent.go:98-125`'s `runStreamFromBlocking` wraps the blocking
runInternal in a goroutine + channel; the LLM call itself is still
`Generate`). This dramatically simplifies Decision 4.

## Decision 1: Budget shape — value type, no cost-table

`Budget` is a **plain value struct** (not an interface). Fields, locked:

```go
type Budget struct {
    MaxTokens int           // 0 = no token cap
    MaxCalls  int           // 0 = no call-count cap
    MaxWall   time.Duration // 0 = no wall-clock cap; if > 0, plumbed via context.WithDeadline
    MaxCost   float64       // 0 = no cost cap; units are user-defined (USD or otherwise)
}
```

| Field | Why this shape |
|---|---|
| `MaxTokens int` | Discrete; matches `llm.Usage.TotalTokens`. Already a count, no estimation needed when the provider reports it. |
| `MaxCalls int` | Cross-agent counter (Supervisor in Phase 37 needs this — `MaxSteps` is per-agent). KC-4 explicitly: "`MaxSteps` is per-agent; `Budget.Calls` is cross-agent." |
| `MaxWall time.Duration` | Composes with stdlib via `context.WithDeadline` — the strict tracker calls `context.WithDeadline(parent, time.Now().Add(b.MaxWall))` at attach time so `ctx.Err()` fires naturally on expiry. **Rationale for not reusing `context.Deadline`**: a Budget might bound wall-clock more tightly than the surrounding ctx; the tracker honors the tighter of the two. |
| `MaxCost float64` | Plumbing only. **Core ships no `CostMapper`**; the caller passes `Usage.Cost` to `Tracker.Charge` if and only if they're tracking dollars. KC-4: cost-table is opt-in / outside core. |

**Counted Usage type** is a *new* struct in the `budget` package,
distinct from `llm.Usage` and `agents.Usage`:

```go
type Usage struct {
    Tokens int     // typically resp.Usage.TotalTokens
    Calls  int     // typically 1 per Generate
    Wall   time.Duration // optional; tracker computes if zero
    Cost   float64 // 0 unless caller has wired a CostMapper
}
```

Why a *new* `budget.Usage` instead of reusing `llm.Usage`: `llm.Usage`
carries `InputTokens/OutputTokens/TotalTokens/Source` — provider-side
accounting. `agents.Usage` carries `LLMCalls + Tokens` — per-Run rollup.
Budget needs `Calls + Tokens + Wall + Cost` — a third shape. Trying to
unify forces edits to validated types (KC-5 violation). Three distinct
types, three distinct concerns. **`budget.Charge` accepts the new shape;
the chokepoint adapts.**

**Confidence:** HIGH. Each field maps to a concrete enforcement need
identified in the v1.2 SUMMARY's Candidate-4 audit; nothing speculative.

## Decision 2: Token counting in stdlib-only core — `Usage.TotalTokens` passthrough; **no estimator interface in v1.2**

The honest answer here: **core does not need a tokenizer.** The
providers already return token counts in `llm.Response.Usage`
(`InputTokens / OutputTokens / TotalTokens / Source`). Every shipped
provider adapter populates these from the provider's API response — the
`Source` field distinguishes `UsageReported` (provider's count) from
`UsageEstimated` (rare; fallback). `ScriptedLLM` also populates `Usage`
on the scripted response.

Therefore the budget tracker **does not estimate** — it charges what the
response reports. The flow at the chokepoint:

```
1. before LLM call: tracker.Charge(budget.Usage{Calls: 1, Tokens: 0})
   → tracker decides: deny if Remaining().MaxCalls <= 0
2. issue model.Generate(ctx, req)
3. after LLM call (success): tracker.Charge(budget.Usage{
       Tokens: resp.Usage.TotalTokens,
       Calls:  0,  // already charged in step 1
   })
4. honor any ctx.Err() or budget.ErrExhausted that fell out
```

Pre/post charge separates the *call cap* (decided before the call, so a
deny does not waste tokens) from the *token total* (only known
post-response). This is the same shape that `pytest-timeout` and most
rate-limiters use.

**No `Estimator` interface ships in v1.2.** Candidates (c) and (d) from
the prompt — caller-provided or per-(provider × model) `Estimator
interface { Estimate(string) int }` — are tempting but **add surface
without adding capability** today: every provider already reports actual
counts post-call, so pre-call estimation is only useful if Phase 35 wants
to deny a request *before* sending it based on a guess. CC-1 does not
require that, and adding the surface in v1.2 then needing to break it in
v1.3 violates KC-5. **Carry-forward note** (record in Phase 38 audit):
v1.x can add `budget.WithEstimator(ctx, e Estimator)` as a strictly
additive optional interface later if the need surfaces.

**Confidence:** HIGH for the "no estimator in v1.2" decision (every
provider reports usage, verified by reading providers/{openai,
anthropic,ollama,deepseek,minimax} where the prompt allowed reading and
by `ScriptedLLM.Generate` returning the scripted `Response.Usage` as-is
— `llm/scripted.go:67-77`). MEDIUM for the deferral being permanent —
some users may want a "deny before sending" gate based on prompt size;
we can ship that when the need arrives.

## Decision 3: Integration mechanism — **ctx-keyed propagation + enforcement at `generateFromPrompt`**; `budget.Wrap` decorator is **deferred to follow-up**, NOT shipped in 35

KC-4 already pre-decides this: "ctx-keyed for propagation + a
`Tracker` interface for enforcement". This research confirms — the
chokepoint exists (`generateFromPrompt`), it covers all 5 paradigms, and
ctx-keyed propagation is the Go-idiomatic match.

**The shipped surface:**

```go
package budget

// WithBudget attaches a Tracker to ctx. If b.MaxWall > 0, ALSO returns
// a context.WithDeadline-decorated ctx so wall-clock fires via ctx.Err().
func WithBudget(parent context.Context, t Tracker) context.Context

// From returns the tracker keyed on ctx, or a no-op Tracker if absent.
// Always safe to call — never returns nil.
func From(ctx context.Context) Tracker

// Tracker is the enforcement seam. Charge MUST be safe for concurrent
// use (the chokepoint may be entered from multiple goroutines).
type Tracker interface {
    Charge(Usage) error    // returns ErrExhausted (wrapping a Limit field) if cap hit
    Remaining() Budget     // never-decreasing snapshot of how much is left
}

// Built-in trackers (KC-4 — both shipped in 35):
func NewStrict(b Budget) Tracker  // deny on exhaustion — Charge returns ErrExhausted
func NewSoft(b Budget, onExhausted func(Usage)) Tracker  // warn-only; never returns ErrExhausted

// Sentinel.
var ErrExhausted = errors.New("budget: exhausted")
```

**The integration edit (the only Go file change to existing code):**

```go
// agent_chatmodel.go — edit body only; signature unchanged
func generateFromPrompt(ctx context.Context, model llm.ChatModel, systemPrompt, prompt string) (llm.Response, error) {
    t := budget.From(ctx)  // safe no-op when absent
    if err := t.Charge(budget.Usage{Calls: 1}); err != nil {
        return llm.Response{}, err
    }
    req := llm.Request{Messages: []llm.Message{{Role: "user", Content: prompt}}}
    if systemPrompt != "" {
        req.SystemPrompt = systemPrompt
    }
    resp, err := model.Generate(ctx, req)
    if err != nil {
        return resp, err  // ctx.Err() from MaxWall surfaces here naturally
    }
    // Charge actual token cost post-call. Calls already charged.
    if cerr := t.Charge(budget.Usage{Tokens: resp.Usage.TotalTokens}); cerr != nil {
        return resp, cerr
    }
    return resp, nil
}
```

**Why not also ship `budget.Wrap(inner) ChatModel` decorator in
Phase 35:**

- The chokepoint already gives 100% paradigm coverage. A decorator
  duplicates enforcement at a different layer.
- The decorator path *also* enforces on bench/rag/context-compress
  Generate calls — but those are out of scope per CC-1.
- KC-4 says "single integration chokepoint at `generateFromPrompt`";
  shipping a decorator additionally would expand the surface beyond
  what KC-4 ratifies.
- Capability-preserving decorator code is **40-60% of `otelmodel.Wrap`'s
  body** (the 8 nested wrapper structs covering 2³ capability
  combinations). Adding that to v1.2 is the kind of "stub four" move the
  research warned against.

**However: ship `budget.Wrap` is a strong v1.3 follow-up.** Recorded in
"Carry-forward" below. Users who want budget on a non-agent Generate
call (e.g., `rag.System.Ask` or a bench harness) can construct a small
local wrapper in v1.2; v1.3 ships the standardized capability-preserving
version.

**`budget.NewStrict` and wall-clock plumbing:** `NewStrict(b)` returns a
tracker whose `WithBudget` call (the moment of attachment to ctx) does
`if b.MaxWall > 0 { ctx, _ = context.WithDeadline(parent, time.Now().Add(b.MaxWall)) }`
internally. The deadline cancellation surfaces as `ctx.Err() == context.DeadlineExceeded`
from inside `model.Generate(ctx, ...)` — *which providers already
respect via the http.Client they own*. Zero new surface for wall-clock;
it reuses stdlib's deadline machinery. Charge for tokens still happens
through the tracker; wall-clock fires through ctx independently.

**Concurrency note for `Charge`:** because `Charge` runs from the
chokepoint and the chokepoint is called from agent-paradigm goroutines
(notably FunctionCall's `MaxParallel` runner spawns N concurrent tool
executions — though they each call `tool.Execute`, not
`generateFromPrompt`, so today the chokepoint isn't concurrent within a
single Run; that may change with Supervisor in Phase 37). Strict tracker
implementation: `sync.Mutex` around a counter — simple, stdlib, correct.

**Confidence:** HIGH for the chokepoint choice (verified). HIGH for the
"defer the decorator" call (concrete reasons, follow-up well-defined).
HIGH for wall-clock-via-context.WithDeadline (textbook stdlib pattern,
already used by every provider).

## Decision 4: Streaming + budget — **no new StreamEvent kind in v1.2**; rely on `ctx.Err()` + post-stream `Charge(EventDone.Usage)`

The v1.2 SUMMARY raised: "how does enforcement interact with the typed
stream union (K1)? Mid-stream cancellation = a new `StreamEvent.Kind`
(e.g., `StreamEventBudgetExceeded`) or `context.Cancel`?"

**Answer for Phase 35: neither, because no agent paradigm streams
today.** Every paradigm goes `agent.RunStream → runStreamFromBlocking →
runInternal → generateFromPrompt → model.Generate`. Stream events
emitted are *agent-layer* `StepEvent`s (Thought/Action/Observation),
not *model-layer* `StreamEvent`s. The agent's RunStream is "blocking
inner, emit step events" — the LLM is still being called via
`Generate`. So the streaming integration question is *theoretical* in
Phase 35.

**The Phase 35 streaming contract:**

1. **Wall-clock budget** fires via `ctx.Err()`. If a future caller calls
   `model.Stream(ctx, req)` with a ctx that has `WithBudget(... MaxWall:
   X ...)` attached, the underlying provider's Stream loop honors
   `ctx.Done()` exactly as today — there is nothing new to wire.
2. **Token budget** is post-hoc when streaming. The recommended call
   shape (documented in `examples/budget/`) is:
   ```go
   sr, _ := model.Stream(ctx, req)
   defer sr.Close()
   for { ev, err := sr.Next(); ... }
   // when ev.Kind == EventDone:
   _ = budget.From(ctx).Charge(budget.Usage{Tokens: ev.Usage.TotalTokens})
   ```
   The tracker won't be able to *prevent* a stream that overshoots
   `MaxTokens` mid-flight (the model has already generated them); but
   the *next* `Charge` from that ctx (e.g., the next agent call in the
   same Run) will deny.
3. **Adding a new `StreamEvent.Kind` (e.g., `EventBudgetExceeded`) is
   explicitly OUT of v1.2.** Reasons: (a) K1 names the existing six
   kinds as a stable typed union — adding a kind is a behavior change
   any K1 consumer (otelmodel's streamReader, every provider adapter)
   must update; that's not additive in any reasonable sense. (b) The
   `Charge` interface is the seam, not the StreamEvent; surfacing budget
   exhaustion via `Tracker.Charge` returning `ErrExhausted` to whoever
   accumulated the stream is the correct layering. (c) Providers
   shouldn't have to know about budget at all — they're stateless
   request/response/stream machines.
4. **`Index` field (the K1 stable per-tool-call key) is unaffected.**
   Budget doesn't tag streams or join by index; it counts and denies.

**The actual `Charge` semantics for streaming when it eventually exists
in core agents:** the chokepoint (or a future stream-aware chokepoint
helper, e.g., `generateStreamFromPrompt`) would call
`tracker.Charge(Usage{Calls: 1})` before opening the stream, then
`tracker.Charge(Usage{Tokens: ev.Usage.TotalTokens})` when
`EventDone` arrives. Identical to the non-stream path; just the timing
of the second charge moves.

**Confidence:** HIGH for "no new StreamEvent.Kind in v1.2" (preserves
K1's locked union). HIGH for "rely on ctx.Err() for cancellation" (it's
how stdlib does it). MEDIUM for "agent paradigms still don't stream" —
this is *today's* state; if a future phase wires streaming through
agents (e.g., Phase 37 Supervisor with streaming workers), a
`generateStreamFromPrompt` helper will need the same Charge calls
mirrored. Recorded as carry-forward.

## Open questions

1. **`budget.Usage` field name collision with `llm.Usage` / `agents.Usage`
   in client code.** Three types named `Usage` is ugly. Mitigation
   options: (a) rename to `budget.Charge` (the type), but then
   `Charge(Charge)` reads strangely; (b) rename to `budget.Tick` —
   describes "one tick of resource consumption" — but is jargon; (c)
   keep `budget.Usage` and document the disambiguation in package
   docstring. **Recommendation:** keep `budget.Usage`. Callers will
   typically write `budget.Usage{Tokens: …}` with the package qualifier;
   ambiguity with `llm.Usage` is resolved by the package selector. The
   planner should ratify in 35-01.

2. **Does `MaxCalls` count *Charge invocations* or *successful LLM calls*?**
   The chokepoint charges `Calls: 1` BEFORE the call (Decision 3) so a
   deny doesn't waste a network round-trip — meaning a denied call
   counts against `MaxCalls`. That feels right (the user asked for "at
   most N calls; the N+1th is denied"). But: what if `model.Generate`
   errors out for a non-budget reason (network timeout)? The call was
   charged but no work happened. **Recommendation:** charge stays;
   error-rollback is a v1.3 feature if anyone asks. Document the
   semantics in the package doc explicitly: "MaxCalls counts attempts,
   not successes."

3. **`budget.NewSoft`'s `OnExhausted` callback signature.** Two
   candidates:
   - `func(over Usage, b Budget)` — "you went over by this much, against
     this budget"
   - `func(t Tracker)` — "here's the tracker, query it"
   The first is more useful for logging; the second is more flexible.
   **Recommendation:** `func(over Usage, b Budget)`. The tracker can be
   recovered from ctx separately if needed.

4. **Does the chokepoint's pre-call `Charge(Calls: 1)` also need to
   surface the existing `agents.Usage.LLMCalls` accumulator?** Today
   each paradigm increments `usage.LLMCalls++` after `generateFromPrompt`
   returns (e.g., `react.go:108`, `reflection.go:83`). This is
   independent of budget — the budget counter is on the tracker; the
   agent's per-Run counter is on `Result.Usage`. **No collision; both
   stay.** The planner should explicitly NOT touch the
   `usage.LLMCalls++` lines in the 5 paradigm files — that's a
   separate concern.

5. **Should `Budget` (the struct) ship `Validate() error`?** E.g., reject
   negative values. **Recommendation:** yes, with `NewStrict` /
   `NewSoft` calling it internally; surfaces typos at construction time
   rather than first `Charge`.

## Slice breakdown

Recommended **4 slices**, all in the core repo. Total estimated effort:
~700-900 LOC including tests + example + docs. Mirrors the v1.2 SUMMARY's
"35-01..03" outline but splits the example out and adds an explicit
exit-gate verification slice.

| Slice | Wave | Type | Repo | Files modified | Requirements | Must-haves |
|---|---|---|---|---|---|---|
| **35-01** | 1 | execute | `llm-agent` | `budget/budget.go` (new); `budget/budget_test.go` (new); `budget/doc.go` (new) | CC-1 (skeleton) | Package skeleton: `Budget`, `Usage`, `Tracker`, `WithBudget`, `From`, `NewStrict`, `NewSoft`, `ErrExhausted`, `Budget.Validate`. `Tracker` has `sync.Mutex` for concurrent `Charge`. `From(ctx)` returns no-op tracker when absent. Wall-clock plumbed via `context.WithDeadline` inside `WithBudget`. **Tests use `ScriptedLLM`-shape mocks (no `ChatModel` involvement yet); pure budget-package tests.** Exit gate: `go vet ./budget/... && go test ./budget/... && go list -deps ./budget` shows zero third-party modules. |
| **35-02** | 2 | execute | `llm-agent` | `agent_chatmodel.go` (edit `generateFromPrompt`); `agent_chatmodel_test.go` (new) | CC-1 (chokepoint integration) | Edit `generateFromPrompt` to charge pre + post (Decision 3 sketch). Add table-driven test exercising all 5 paradigms × {no budget, strict budget hit, strict budget miss, soft budget over} using `ScriptedLLM` — proves zero behavior change when no budget set + uniform enforcement when set. **No edit to the 5 paradigm files themselves** (KC-5). Exit gate: full `go test ./...` green + no `agents.Usage`/`llm.Usage` field added (verified by `git diff` review). |
| **35-03** | 3 | execute | `llm-agent` | `examples/06-budget/main.go` (new) | CC-1 (example + docs) | A deterministic example: `ScriptedLLM` with 3 scripted responses, ReAct with `MaxTokens` cap that fires on the 2nd call. Output mirrors `examples/01-simple-agent/main.go` style. **Stdlib-only** (no rag import, no otel). Also a small `examples/budget/README.md` (≤ 80 lines) showing the canonical `ctx = budget.WithBudget(ctx, budget.NewStrict(...)); agent.Run(ctx, input)` pattern, the no-op safety, and the wall-clock-deadline composition. Exit gate: `go run examples/06-budget/main.go` exits 0 with the expected ErrExhausted on the budgeted ReAct. |
| **35-04** | 4 | execute | `llm-agent` | `budget/integration_test.go` (new); brief planning-doc updates if needed | CC-1 (verification + close) | Wide integration test: one test per agent paradigm (Simple/ReAct/Reflection/PlanSolve/FunctionCall) — wire `budget.NewStrict(Budget{MaxCalls: 2})`, run, assert `ErrExhausted` on the 3rd internal Generate; assert paradigm-specific behavior is unaffected when budget absent. Phase exit gate (run from `llm-agent/`): `go vet ./... && go test ./... && go list -deps ./...` — the deps list must remain `{llm-agent-rag v1.0.1, stdlib only}`; assert with a small `grep -v 'costa92/llm-agent-rag'` filter that no new third-party module entered `go.sum`. |

Wave structure is strictly sequential (each builds on the previous).
35-01 → 35-02 is package-then-integration; 35-02 → 35-03 is
integration-then-example; 35-03 → 35-04 is example-then-wide-verify.
No parallelization gain available.

**Sizing.** ~150 LOC for 35-01 + ~80 LOC for 35-02 edit + ~80 LOC for
35-03 example + ~150-200 LOC for 35-04 integration tests = ~500
production LOC + ~300 test LOC = ~800 LOC total. Matches the v1.2
SUMMARY's "M" estimate (600 LOC for the budget package alone, +
integration is the planner's true M-L).

## Out of scope

- **`budget.Wrap(inner ChatModel) ChatModel` decorator** — deferred to
  v1.3 (Decision 3). The chokepoint gives 100% paradigm coverage; the
  decorator is for non-agent callers (bench, rag.System, context
  pipeline). v1.x can add the decorator additively.
- **`Estimator interface` / pre-call token estimation** — deferred to
  v1.3 (Decision 2). Every provider reports actual usage; pre-call
  estimation has no real customer in v1.2.
- **Cost-table (provider→$/token)** — KC-4 explicitly says
  outside-core. Users wire `Budget.Cost` from their own `CostMapper`;
  the providers sister repo may ship one in a future milestone.
- **Mid-stream `StreamEvent.EventBudgetExceeded` kind** — Decision 4.
  K1's typed union is locked; budget exhaustion surfaces via
  `Tracker.Charge` return value, not a new event kind.
- **Touching `llm.Usage`, `agents.Usage`, `agents.Result`, `llm.ChatModel`,
  `agents.Agent`, `orchestrate.NodeFunc[S]`** — KC-5 absolute. Validated
  public types unchanged.
- **Wiring budget into `bench/`, `rag/`, `context/compress.go`** —
  out-of-scope (those aren't agent paradigms). Carry-forward note:
  v1.3 `budget.Wrap` decorator addresses these uniformly.
- **`MaxSteps` / `MaxTurns` deprecation** — KC-4 explicitly: per-agent
  step caps and cross-agent budgets coexist; no deprecation. No edit to
  any paradigm option struct.
- **Supervisor budget enforcement** — Phase 37 work. Phase 35 ships the
  primitives; Phase 37 consumes them (rounds count against `MaxCalls`;
  ctx-propagation to workers).
- **OTel emission of budget events** — `otelmodel` is a sister repo
  (`llm-agent-otel`), out of v1.2 scope. v1.3 ecosystem alignment may
  add `otelmodel.Wrap` budget-event awareness.
- **K8s / Helm packaging** — standing hard-rule non-goal.
- **Any non-stdlib dependency in core** — CLAUDE.md Rule 1, absolute.

## Carry-forward notes (for Phase 36, 37, 38)

- **Phase 36 (policy middleware) — composition pattern with budget.**
  KC-3 says `policy.Wrap(otelmodel.Wrap(provider))` is the documented
  composition stack. Budget integrates a level deeper (at
  `generateFromPrompt`, *before* the request hits any wrapped model).
  Phase 36's tests should cover the case where a budget exhausts mid-Run
  through a policy-wrapped model — assert `budget.ErrExhausted`
  short-circuits before policy gates fire on the (denied) request. This
  is the right ordering: budget first (cheap; checked locally), policy
  second (input scanning; possibly expensive), model last (network /
  cost). A short note in Phase 36's RESEARCH.md should pin this
  ordering.

- **Phase 37 (Supervisor) — budget propagation to workers.** CC-3 says
  "Supervisor honors **CC-1**'s ctx-keyed budget (rounds count against
  `Budget.Calls`; ctx propagates to workers)". The mechanism is free if
  Supervisor passes the parent ctx unchanged to each worker's `Run` —
  the tracker is on ctx, `From` finds it inside each worker's
  `generateFromPrompt`, and each worker's LLM calls charge against the
  same tracker. Phase 37 must avoid the trap of `context.WithoutCancel`
  or detached child contexts that would break the propagation. **Hard
  rule for Phase 37:** workers receive `supervisorCtx`, not a freshly
  built ctx.

- **Phase 38 (milestone close) — exit-gate proof for stdlib-only.** The
  Phase-38 audit needs to assert that `go list -deps ./...` on the v0.6.0
  tag shows the same module set as v0.5.1 *plus* zero new third-party
  modules. The `budget` package's `go list -deps ./budget` should list
  only stdlib modules (`context`, `errors`, `fmt`, `sync`, `time`). Add
  this as an explicit check in the milestone audit doc.

- **Future: `budget.Wrap` decorator.** When v1.3 implements it, mirror
  `otelmodel.Wrap`'s 8-wrapper capability-preservation pattern
  (`otelmodel.go:14-49`). Composition order with otelmodel and policy:
  `policy.Wrap(budget.Wrap(otelmodel.Wrap(provider)))` — budget under
  policy (cheaper check first), under otelmodel (so budget denials are
  observed in spans), over provider (so the network call only happens
  if all wrappers pass).

- **Future: streaming chokepoint helper.** If a future phase wires
  `model.Stream` through agent paradigms, add
  `generateStreamFromPrompt(ctx, model, ...)` alongside
  `generateFromPrompt`. Both helpers `budget.From(ctx).Charge`
  identically — pre-Calls + post-Tokens. Recorded so the path is
  obvious when the need surfaces.

- **Test infrastructure.** The whole milestone uses `ScriptedLLM`
  (`llm/scripted.go:53-77`) — already populates `Response.Usage`. Phase
  35 needs no new mock infrastructure. Phases 36 / 37 can reuse the same
  budget-aware ctx setup the 35-04 integration tests establish.

---

*Researched 2026-05-20. Voice + structure mirror
`.planning/phases/34-umbrella-coherence-gate-and-milestone-close/34-RESEARCH.md`.
Every file path cited verified by direct read against the working tree at
`/home/hellotalk/code/go/src/github.com/costa92/llm-agent-ecosystem/llm-agent`.
The pre-flight grep miss (claiming `generateFromPrompt` does not exist in
core) corrected in §"Chokepoint discovery" — the symbol DOES exist at
`agent_chatmodel.go:10` and funnels all 5 paradigms × 10 call sites.*
