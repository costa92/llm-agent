package policy

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

// TestDecision_ZeroValueIsAllow ratifies Q1 from 36-RESEARCH.md: a
// gate that returns the zero Decision must be non-interfering, so
// Allow MUST be the zero DecisionAction.
func TestDecision_ZeroValueIsAllow(t *testing.T) {
	var d Decision
	if d.Action != Allow {
		t.Fatalf("zero Decision.Action = %d, want Allow (%d)", d.Action, Allow)
	}
	if Allow != 0 {
		t.Fatalf("Allow = %d, want 0 (Q1 ratification: zero-value = non-interfering)", Allow)
	}
	// And PreGenerate must be iota = 0 so a forgotten Kind defaults
	// to the most common case.
	if PreGenerate != 0 {
		t.Fatalf("PreGenerate = %d, want 0 (the most-common-case default)", PreGenerate)
	}
}

// TestErrBlocked_SentinelDetection verifies errors.Is plumbing: a
// freshly constructed *BlockedError matches ErrBlocked but NOT
// unrelated sentinels (e.g., io.EOF).
func TestErrBlocked_SentinelDetection(t *testing.T) {
	err := &BlockedError{Gate: "g", Reason: "r"}
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("errors.Is(err, ErrBlocked) = false, want true; err=%v", err)
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("errors.Is(err, io.EOF) = true, want false; err=%v", err)
	}
}

// TestErrBlocked_AsRich verifies errors.As extracts a *BlockedError
// from a wrapped error chain — the canonical rich-error pattern from
// llm/errors.go (AuthError shape).
func TestErrBlocked_AsRich(t *testing.T) {
	inner := &BlockedError{Gate: "InjectionScanner", Reason: "instruction_override"}
	wrapped := fmt.Errorf("upstream: %w", inner)
	var be *BlockedError
	if !errors.As(wrapped, &be) {
		t.Fatalf("errors.As(wrapped, &be) = false, want true; wrapped=%v", wrapped)
	}
	if be.Gate != "InjectionScanner" {
		t.Fatalf("extracted Gate = %q, want %q", be.Gate, "InjectionScanner")
	}
	if be.Reason != "instruction_override" {
		t.Fatalf("extracted Reason = %q, want %q", be.Reason, "instruction_override")
	}
	// Sentinel detection still works through the wrap.
	if !errors.Is(wrapped, ErrBlocked) {
		t.Fatalf("errors.Is(wrapped, ErrBlocked) = false through fmt.Errorf %%w wrap")
	}
}

// TestBlockedError_UnwrapsWrapped verifies the Unwrap chain composes
// with errors.Is for any underlying error captured in Wrapped.
func TestBlockedError_UnwrapsWrapped(t *testing.T) {
	err := &BlockedError{Gate: "g", Reason: "r", Wrapped: io.ErrUnexpectedEOF}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("errors.Is(err, io.ErrUnexpectedEOF) = false through Unwrap chain; err=%v", err)
	}
	// And the umbrella sentinel still matches.
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("errors.Is(err, ErrBlocked) = false; err=%v", err)
	}
}

// TestBlockedError_ErrorString verifies the formatted message exactly
// matches the 36-RESEARCH.md §"Decision D" template.
func TestBlockedError_ErrorString(t *testing.T) {
	got := (&BlockedError{Gate: "InjectionScanner", Reason: "instruction_override"}).Error()
	want := "policy: blocked by InjectionScanner: instruction_override"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// TestBlockedError_DecisionField ratifies Q5: BlockedError carries a
// struct copy of the deciding Decision so callers can introspect the
// full decision rather than just Gate + Reason.
func TestBlockedError_DecisionField(t *testing.T) {
	d := Decision{
		Action:      Block,
		Reason:      "instruction_override",
		Gate:        "InjectionScanner",
		Replacement: "",
	}
	err := &BlockedError{Gate: d.Gate, Reason: d.Reason, Decision: d}
	if err.Decision.Action != Block {
		t.Fatalf("Decision.Action = %d, want Block (%d)", err.Decision.Action, Block)
	}
	if err.Decision.Reason != "instruction_override" {
		t.Fatalf("Decision.Reason = %q, want %q", err.Decision.Reason, "instruction_override")
	}
	// Mutating the local Decision after construction MUST NOT affect
	// the BlockedError's copy (value semantics — Q5 rationale).
	d.Action = Allow
	if err.Decision.Action != Block {
		t.Fatalf("after local mutation Decision.Action = %d, want Block (value copy invariant)", err.Decision.Action)
	}
}
