// PIIRedactor is the built-in PII-removal Gate.
//
// Behavior summary:
//
//   - PreGenerate  → if the last user-role message Content matches any
//     rule, return Decision{Action: Replace, Reason: "pii_redacted",
//     Replacement: <redacted last-user content>}. The decorator
//     (policy.go::applyReplace) substitutes Replacement into the last
//     user message; SystemPrompt rewrite is the documented v1.3 follow-
//     up. Simpler v1.2 contract per 36-02-PLAN.md.
//   - PostGenerate → if resp.Text matches any rule, return Decision{
//     Action: Redact, Reason: "pii_redacted", Replacement: <redacted
//     resp.Text>}.
//   - StreamDelta  → Allow by default (Q4); when constructed with
//     WithStreamRedaction(), returns Redact on each delta whose Text
//     matches. Cross-delta PII can leak — best-effort, documented limit
//     (matches rag's known constraint).
//   - PreStream / PostStream → Allow (no payload to inspect).
//
// Constructor takes functional options. The 3 built-in rules (email,
// phone, ipv4) are closed over at construct time from defaultPIIRules()
// — the gate is stateless across Inspect calls (Pitfall 3 from
// 36-RESEARCH.md: concurrent Inspect on one gate value is race-clean).
//
// Q-trace: Q2 (PII subset — 3 patterns, ssn/credit_card dropped) is
// enforced by patterns.go::defaultPIIRules; Q4 (StreamDelta default
// OFF) is enforced by the piiRedactor.streamDelta field defaulting to
// false.

package policy

import (
	"context"

	"github.com/costa92/llm-agent/llm"
)

// piiRedactor is the unexported implementation. Holds the immutable
// rule slice + streamDelta opt-in flag. No per-call state — safe to
// share one value across many goroutines (Pitfall 3).
type piiRedactor struct {
	rules       []piiRule
	streamDelta bool
}

// PIIOption is the functional-option type for NewPIIRedactor.
//
// Only WithStreamRedaction is exported today; the option pattern is
// the additive-evolution path (v1.3 may add WithCustomRules, etc.
// without breaking callers).
type PIIOption func(*piiRedactor)

// WithStreamRedaction enables per-delta redaction in the StreamDelta
// branch of Inspect. Default is OFF (Q4 ratification — per-delta regex
// is expensive on hot streaming paths; cross-delta PII may leak). Use
// only when delta-grain redaction is required by audit policy.
func WithStreamRedaction() PIIOption {
	return func(p *piiRedactor) { p.streamDelta = true }
}

// NewPIIRedactor constructs a PIIRedactor gate with the built-in
// pattern set from patterns.go::defaultPIIRules (3 patterns: email,
// phone, ipv4 per Q2). Options are applied in order.
//
// Returned as Gate (the interface) — callers compose via policy.Wrap.
func NewPIIRedactor(opts ...PIIOption) Gate {
	p := &piiRedactor{rules: defaultPIIRules()}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the stable identifier surfaced as Decision.Gate +
// BlockedError.Gate via the decorator audit path.
func (p *piiRedactor) Name() string { return "PIIRedactor" }

// Inspect is the Gate-interface entry point. Dispatches on ev.Kind.
//
// PreGenerate: redacts the last user-role Message.Content; if no
// user-role message exists, falls back to ev.Req.SystemPrompt. Returns
// Allow when no rule matches.
//
// PostGenerate: redacts ev.Resp.Text.
//
// StreamDelta: opt-in only (p.streamDelta); redacts ev.Delta.Text.
//
// PreStream / PostStream / unknown kinds: Allow (no-op).
func (p *piiRedactor) Inspect(_ context.Context, ev Event) Decision {
	switch ev.Kind {
	case PreGenerate:
		if ev.Req == nil {
			return Decision{Action: Allow}
		}
		target := lastUserContent(ev.Req)
		redacted, hit := p.redact(target)
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

	case StreamDelta:
		if !p.streamDelta || ev.Delta == nil {
			return Decision{Action: Allow}
		}
		redacted, hit := p.redact(ev.Delta.Text)
		if !hit {
			return Decision{Action: Allow}
		}
		return Decision{Action: Redact, Reason: "pii_redacted", Replacement: redacted}

	default:
		return Decision{Action: Allow}
	}
}

// redact applies every rule's regex in order; returns (out, hit) where
// hit is true iff at least one rule actually matched (so the gate can
// short-circuit to Allow when there was no PII). Mirrors rag's
// guard/redact.go:45-59 algorithm: MatchString to detect hit, then
// ReplaceAllString to rewrite.
func (p *piiRedactor) redact(in string) (string, bool) {
	out := in
	hit := false
	for _, rule := range p.rules {
		if rule.pattern == nil {
			continue
		}
		if rule.pattern.MatchString(out) {
			hit = true
			out = rule.pattern.ReplaceAllString(out, rule.placeholder)
		}
	}
	return out, hit
}

// lastUserContent finds the last user-role Message.Content in the
// Request. Falls back to SystemPrompt when there are no user-role
// messages — this lets the decorator's applyReplace target the right
// field via the same precedence rule (policy.go::applyReplace).
func lastUserContent(req *llm.Request) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return req.Messages[i].Content
		}
	}
	return req.SystemPrompt
}

// Compile-time assertion — piiRedactor implements Gate.
var _ Gate = (*piiRedactor)(nil)
