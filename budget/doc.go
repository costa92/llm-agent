// Package budget provides ctx-keyed token/call/wall-clock/cost budgets for llm-agent.
//
// The package ships a Budget value type, a distinct Usage value type, a
// concurrency-safe Tracker interface with two built-in constructors
// (NewStrict, NewSoft), and a sentinel-error family. Budgets are attached
// to a context.Context via WithBudget (the common case) or WithTracker
// (for callers building a soft tracker explicitly), then extracted at the
// chokepoint via From.
//
// In Phase 35 this package is data + plumbing only — the integration
// site is agent_chatmodel.go::generateFromPrompt, which is wired in slice
// 35-02. A 35-02-and-later caller calls From(ctx); when no tracker is
// attached the chokepoint short-circuits (the "zero behavior change when
// no budget is set" guarantee). This slice (35-01) ships nothing outside
// budget/.
//
// # Q1 — three Usage types coexist by design (operator-confirmed 2026-05-20)
//
// budget.Usage, llm.Usage, and agents.Usage are intentionally separate
// types living in separate packages. They model different concerns:
// budget.Usage is a "what to charge the tracker" payload; llm.Usage is
// the provider-reported token accounting on a single ChatModel response;
// agents.Usage is the per-run aggregation surfaced on agents.Result. The
// package selector (budget.Usage{...} vs llm.Usage{...}) disambiguates at
// every call site — no merging, no shared base type, no mutual import.
// Future readers: do not re-litigate this decision; it is recorded in
// 35-RESEARCH.md §"Open questions" Q1.
//
// # Q2 — MaxCalls counts attempts, not successes (operator-confirmed 2026-05-20)
//
// Tracker.Charge fires on the pre-call path. The Calls counter is
// incremented before the LLM call is dispatched, so a denied charge
// (one that the LLM never sees) still costs against the cap. This is
// the only ordering that makes a strict cap actually a cap — counting
// successes lets a runaway retry loop ignore the budget. Recorded in
// 35-RESEARCH.md §"Open questions" Q2.
//
// # References
//
//   - CC-1 — Budget/Cancellation Context requirement (v1.2 milestone).
//   - KC-4 — cost is provider-priced upstream; this package only holds
//     a float64 "cost so far" with no pricing logic.
//   - KC-5 — core stays stdlib-only; this package imports nothing
//     outside context/errors/fmt/sync/sync-atomic/time.
package budget
