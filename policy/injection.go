// InjectionScanner is the built-in prompt-injection detection Gate.
//
// Behavior summary:
//
//   - PreGenerate → scan a concatenation of ev.Req.SystemPrompt and
//     every ev.Req.Messages[i].Content. The first rule (in
//     defaultInjectionRules slice order) whose pattern matches returns
//     Decision{Action: Block, Reason: <rule.name>}. The Reason is the
//     well-known pattern name (instruction_override / disregard_above /
//     role_override / prompt_exfiltration) — callers may match on it.
//   - PostGenerate / PreStream / StreamDelta / PostStream → Allow.
//     Injection is a REQUEST-SIDE concern: a model whose response
//     happens to contain "ignore previous instructions" is not itself
//     attempting injection. Best-effort and consistent with rag's
//     PatternScanner shape (input-only).
//
// The gate is stateless — the rule slice is closed over at construct
// time. Safe for concurrent Inspect across goroutines (Pitfall 3).

package policy

import (
	"context"
	"strings"
)

// injectionScanner holds the immutable rule slice. No per-call state.
type injectionScanner struct {
	rules []injectionRule
}

// NewInjectionScanner constructs an InjectionScanner gate with the 4
// built-in patterns from patterns.go::defaultInjectionRules. Returned
// as Gate so callers compose via policy.Wrap.
func NewInjectionScanner() Gate {
	return &injectionScanner{rules: defaultInjectionRules()}
}

// Name returns the stable identifier surfaced as Decision.Gate +
// BlockedError.Gate via the decorator audit path.
func (s *injectionScanner) Name() string { return "InjectionScanner" }

// Inspect dispatches on ev.Kind. Only PreGenerate is meaningful —
// every other kind returns Allow.
//
// The scan surface is SystemPrompt + all Message.Content joined by
// "\n" — this is the same shape a downstream model receives, so
// detection mirrors what the model would actually see.
//
// Rules iterate in slice order; first match wins (Decision.Reason is
// the rule's name).
func (s *injectionScanner) Inspect(_ context.Context, ev Event) Decision {
	if ev.Kind != PreGenerate || ev.Req == nil {
		return Decision{Action: Allow}
	}

	// Build the scan surface. strings.Builder avoids per-message
	// allocations on the hot path.
	var b strings.Builder
	if ev.Req.SystemPrompt != "" {
		b.WriteString(ev.Req.SystemPrompt)
	}
	for _, msg := range ev.Req.Messages {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(msg.Content)
	}
	input := b.String()

	for _, rule := range s.rules {
		if rule.pattern == nil {
			continue
		}
		if rule.pattern.MatchString(input) {
			return Decision{Action: Block, Reason: rule.name}
		}
	}
	return Decision{Action: Allow}
}

// Compile-time assertion — injectionScanner implements Gate.
var _ Gate = (*injectionScanner)(nil)
