package policy

import (
	"context"
	"errors"
	"fmt"

	"github.com/costa92/llm-agent-contract/llm"
)

// EventKind discriminates the lifecycle position at which a Gate fires.
//
// Field-population rules (consult ev.Req / ev.Resp / ev.Delta accordingly):
//
//   - PreGenerate : ev.Req != nil
//   - PostGenerate: ev.Req != nil; ev.Resp != nil
//   - PreStream   : ev.Req != nil
//   - StreamDelta : ev.Req != nil; ev.Delta != nil
//   - PostStream  : ev.Req != nil; ev.Delta != nil on EventDone, nil on io.EOF
//
// PreGenerate is iota=0 so a forgotten Kind defaults to the most-common
// case. See 36-RESEARCH.md §"Decision B" for the typed-union pattern.
type EventKind uint8

const (
	// PreGenerate fires before the wrapped model.Generate call. Gates
	// may Block, Replace (rewrite ev.Req), Redact (no-op in this kind —
	// no Response yet), or Allow. Zero-value Kind.
	PreGenerate EventKind = iota

	// PostGenerate fires after a successful wrapped model.Generate.
	// Gates may Block (discard the response and surface BlockedError),
	// Redact (rewrite ev.Resp.Text), or Allow.
	PostGenerate

	// PreStream fires once before the first inner Next() of a wrapped
	// stream. Gates may Block (close inner, return BlockedError from
	// the first Next() call), Replace (rewrite ev.Req for the inner
	// stream — best-effort; inner.Stream may have already started), or
	// Allow.
	PreStream

	// StreamDelta fires per inner StreamEvent (excluding EventDone and
	// io.EOF, which fire PostStream). Opt-in by default — see Q4
	// ratification in doc.go. Gates may Block (surface immediately on
	// the current Next() — Decision F surface-immediately variant),
	// Redact (rewrite ev.Delta.Text in place), or Allow.
	StreamDelta

	// PostStream fires once on EventDone OR io.EOF. Best-effort
	// observation only; Block on PostStream is a no-op (the stream is
	// already terminal).
	PostStream
)

// Event is the typed-union payload passed to Gate.Inspect. Field
// population is gated by Kind — see EventKind doc comments above.
// Pointer fields permit zero-allocation Allow paths; gates MUST treat
// the pointed-to values as read-only (decorator owns the value copy).
type Event struct {
	Kind  EventKind
	Req   *llm.Request
	Resp  *llm.Response
	Delta *llm.StreamEvent
}

// DecisionAction enumerates the four verdicts a Gate may return.
//
// Allow is the zero-value (Q1 ratification, 36-RESEARCH.md §"Open
// Questions" Q1): a Gate that returns the zero Decision is
// non-interfering by default.
type DecisionAction uint8

const (
	// Allow passes the event through unchanged. Zero-value — Q1
	// ratification: non-interfering default; a Gate that forgets to
	// return anything Allows.
	Allow DecisionAction = iota

	// Block short-circuits the request path. On Pre* the wrapped
	// model is NOT invoked; on Post* the response is discarded.
	// Surfaces as &BlockedError{...} which is errors.Is(err,
	// ErrBlocked).
	Block

	// Redact rewrites the response (PostGenerate: ev.Resp.Text) or the
	// stream delta (StreamDelta: ev.Delta.Text) with
	// Decision.Replacement. Caller sees a "successful" but cleaned
	// response/event.
	Redact

	// Replace rewrites the request before the wrapped model is
	// invoked. Decorator substitutes Decision.Replacement into the
	// last user-role Message.Content (else Request.SystemPrompt).
	// On StreamDelta, Replace is equivalent to Redact.
	Replace
)

// Decision is the verdict returned by Gate.Inspect.
//
// Action selects one of the four decorator behaviors. Reason is a
// gate-defined string (e.g., "pii_redacted", "instruction_override")
// surfaced in BlockedError and OnDecision callbacks. Replacement is
// populated when Action is Redact or Replace. Gate is populated by the
// decorator from Gate.Name() — gates returning a non-empty Gate field
// have it OVERWRITTEN; the field is read-only to callers via the
// decorator's audit path.
type Decision struct {
	Action      DecisionAction
	Reason      string
	Replacement string
	Gate        string
}

// Gate is the user-extension seam. Implementations inspect an Event
// and return a Decision. Name returns the gate's stable identity
// string, surfaced in BlockedError.Gate and Decision.Gate via the
// decorator's audit path.
//
// Implementations MUST be safe for concurrent invocation: the decorator
// may call Inspect from multiple goroutines on the same Gate value
// when callers invoke wrapped.Generate concurrently.
type Gate interface {
	Inspect(ctx context.Context, ev Event) Decision
	Name() string
}

// ErrBlocked is the umbrella sentinel returned (wrapped in
// *BlockedError) when a Gate's Decision.Action == Block.
//
// Canonical detection:
//
//	if errors.Is(err, policy.ErrBlocked) {
//	    var be *policy.BlockedError
//	    if errors.As(err, &be) {
//	        log.Printf("blocked by %s: %s", be.Gate, be.Reason)
//	    }
//	}
//
// Callers detect with errors.Is(err, policy.ErrBlocked). The richer
// *BlockedError carries the deciding gate name, reason, and a copy of
// the Decision (Q5 ratification).
var ErrBlocked = errors.New("policy: blocked")

// BlockedError is the rich error returned by the decorator when a
// Gate's Decision.Action == Block. It satisfies the errors.Is contract
// against ErrBlocked and the errors.Unwrap contract for any underlying
// error captured in Wrapped (e.g., a cascading budget error).
//
// Decision is a struct copy of the deciding Decision (Q5 ratification —
// callers may introspect the full decision rather than just Gate +
// Reason). Value semantics avoid post-block mutation of the audit
// record.
type BlockedError struct {
	Gate     string
	Reason   string
	Decision Decision
	Wrapped  error
}

// Error formats as `policy: blocked by <Gate>: <Reason>`. Matches the
// 36-RESEARCH.md §"Decision D" template.
func (e *BlockedError) Error() string {
	return fmt.Sprintf("policy: blocked by %s: %s", e.Gate, e.Reason)
}

// Is reports whether the target is ErrBlocked, satisfying the
// errors.Is contract. Use errors.Is(err, policy.ErrBlocked) on the
// surfaced error value to detect a policy block.
func (e *BlockedError) Is(target error) bool { return target == ErrBlocked }

// Unwrap returns the underlying Wrapped error (nil unless the gate or
// decorator chain captured an upstream error — e.g., a cascading
// budget exhaustion).
func (e *BlockedError) Unwrap() error { return e.Wrapped }
