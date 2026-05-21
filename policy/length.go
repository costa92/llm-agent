// MaxInputLen is the built-in input-length cap Gate.
//
// Behavior summary:
//
//   - PreGenerate → compute total bytes = len(SystemPrompt) +
//     Σ len(Message.Content). If total > cap, return Decision{Action:
//     Block, Reason: "length_exceeded"}. Otherwise Allow. Cap is
//     INCLUSIVE on the upper bound: total == cap Allows.
//   - PostGenerate / PreStream / StreamDelta / PostStream → Allow.
//     Length is a REQUEST-SIDE gate — capping the response or each
//     delta is a different concern (and would interact badly with
//     streaming where the total isn't known until the end). Deferred.
//
// Q3 ratification (36-RESEARCH.md Decision H): cap is measured in
// BYTES. len(string) is O(1); provider HTTP byte budgets are the
// operative cap; one Chinese character ≈ 3 bytes; one emoji ≈ 4 bytes.
// A future NewMaxInputLenRunes(n int) constructor is the documented
// additive path for callers who want rune-mode (Unicode grapheme)
// semantics — NOT in this slice.
//
// Defensive: cap <= 0 is treated as "no enforcement" (Allow always).
// This protects against bad caller config (NewMaxInputLen(0) or a
// negative literal) and makes the zero-value gate safe.
//
// The gate is stateless. Concurrent Inspect across goroutines is
// race-clean (Pitfall 3).

package policy

import "context"

// maxInputLen holds the immutable byte cap. No per-call state.
type maxInputLen struct {
	cap int
}

// NewMaxInputLen constructs a MaxInputLen gate with the given byte cap.
// A non-positive cap (n <= 0) yields a no-op gate that always Allows.
// Returned as Gate so callers compose via policy.Wrap.
func NewMaxInputLen(n int) Gate {
	return &maxInputLen{cap: n}
}

// Name returns the stable identifier surfaced as Decision.Gate +
// BlockedError.Gate via the decorator audit path.
func (m *maxInputLen) Name() string { return "MaxInputLen" }

// Inspect dispatches on ev.Kind. Only PreGenerate is meaningful —
// every other kind returns Allow.
//
// Cap semantics: total > cap Blocks; total == cap Allows (inclusive
// upper bound). Negative or zero cap disables enforcement.
func (m *maxInputLen) Inspect(_ context.Context, ev Event) Decision {
	if ev.Kind != PreGenerate || ev.Req == nil {
		return Decision{Action: Allow}
	}
	if m.cap <= 0 {
		return Decision{Action: Allow}
	}

	total := len(ev.Req.SystemPrompt)
	for _, msg := range ev.Req.Messages {
		total += len(msg.Content)
	}

	if total > m.cap {
		return Decision{Action: Block, Reason: "length_exceeded"}
	}
	return Decision{Action: Allow}
}

// Compile-time assertion — maxInputLen implements Gate.
var _ Gate = (*maxInputLen)(nil)
