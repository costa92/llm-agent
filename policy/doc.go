// Package policy provides a capability-preserving llm.ChatModel decorator that runs typed Gate events at request, response, and stream boundaries.
//
// The package ships Wrap / WrapConfig and an 8-wrapper type-switch
// tree mirroring otelmodel.Wrap; a typed Gate event union (5
// EventKinds: PreGenerate / PostGenerate / PreStream / StreamDelta /
// PostStream); a Decision shape with 4 actions (Allow / Block /
// Redact / Replace); the ErrBlocked + BlockedError sentinel + rich
// error pair (errors.Is umbrella + errors.As detail); and the
// documented composition stack policy.Wrap(otelmodel.Wrap(provider))
// where outer denies before observed, middle observes, inner calls.
//
// # Decisions ratified by this package (do not re-litigate)
//
// Q1 — Config.OnDecision returns no error. Callbacks fire
// synchronously in the request goroutine for every non-Allow
// decision; nil-safe; panics are recovered by the decorator. Symmetric
// with otelmodel's tracer callback — observation never blocks the
// request path.
//
// Q2 — The PII pattern set ships email + phone + ipv4 only. The
// US-locale-specific ssn and credit_card patterns are deferred to a
// v1.3 NewUSLocalePIIRedactor additive (KC-5-friendly). Built-in
// gates land in slice 36-02; this slice ships surface only.
//
// Q3 — MaxInputLen measures bytes (len(string)), not runes. Bytes
// are the operative cap for provider HTTP budgets, O(1), and
// language-agnostic. A future MaxInputLenRunes(n int) is a v1.3
// additive candidate.
//
// Q4 — StreamDelta is opt-in (default OFF) for all 3 built-in gates.
// Per-delta regex is expensive and cross-delta PII can leak by design
// (best-effort, matches rag's known limit). Users wanting cross-delta
// detection register a PostGenerate redactor that fires once on the
// assembled response.
//
// Q5 — BlockedError.Decision is shipped (a struct copy of the
// deciding Decision). Callers introspect via errors.As rather than
// just Gate + Reason; value semantics avoid post-block mutation.
//
// # Gate ordering
//
// Gates run sequentially in registration order. The first Block wins
// (short-circuits the chain). Replace rewrites the request and
// continues — subsequent gates may still Block. Redact rewrites the
// response or delta in place; subsequent gates see the rewrite.
//
// # Streaming semantics
//
// PreStream fires once on the first wrapped Next(); Block closes the
// inner stream and surfaces a BlockedError from the same Next() call
// (Decision F surface-immediately variant — no race window).
// StreamDelta fires per inner event (gates opt in; default Allow);
// Block surfaces immediately on the current Next(). PostStream fires
// on EventDone or io.EOF; observation only — Block on PostStream is
// a no-op because the stream is already terminal.
//
// # Composition with budget
//
// The v1.2 stack is policy.Wrap(otelmodel.Wrap(provider)). Budget
// enforces at agents.generateFromPrompt UNDERNEATH the wrapped model
// — a budget-exhausted request short-circuits BEFORE policy is
// reached. BlockedError.Wrapped may carry a budget error in
// cascading scenarios.
//
// # References
//
//   - CC-2 — Policy / safety middleware (v1.2 milestone).
//   - KC-3 — decorator mirrors otelmodel.Wrap; typed Gate event
//     union; sentinel ErrBlocked; regex gates lifted from rag by copy.
//   - KC-5 — core stays stdlib-only; imports nothing outside
//     context/errors/fmt/io/sync + github.com/costa92/llm-agent/llm.
//   - K1 — typed StreamEvent union; no new Kind for blocked stream.
//   - 36-RESEARCH.md §Decisions A-H — internal design decisions.
package policy
