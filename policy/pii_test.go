package policy

import (
	"context"
	"strings"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

// TestPIIRedactor_Name verifies the canonical Name() identifier — this
// string surfaces as Decision.Gate in audit logs (set by the decorator
// in policy.go::runGates) so it MUST be the camel-case form (Q-ratified
// in 36-02-PLAN.md).
func TestPIIRedactor_Name(t *testing.T) {
	g := NewPIIRedactor()
	if got := g.Name(); got != "PIIRedactor" {
		t.Fatalf("Name() = %q, want %q", got, "PIIRedactor")
	}
}

// TestPIIRedactor_PreGenerate_Replace verifies that a user message
// containing email + phone triggers Replace with both placeholders
// substituted in.
func TestPIIRedactor_PreGenerate_Replace(t *testing.T) {
	g := NewPIIRedactor()
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "Email me at alice@example.com or call 555-123-4567"},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Replace {
		t.Fatalf("Action = %v, want Replace", dec.Action)
	}
	if dec.Reason != "pii_redacted" {
		t.Fatalf("Reason = %q, want %q", dec.Reason, "pii_redacted")
	}
	if !strings.Contains(dec.Replacement, "[REDACTED:EMAIL]") {
		t.Fatalf("Replacement missing [REDACTED:EMAIL]: %q", dec.Replacement)
	}
	if !strings.Contains(dec.Replacement, "[REDACTED:PHONE]") {
		t.Fatalf("Replacement missing [REDACTED:PHONE]: %q", dec.Replacement)
	}
	if strings.Contains(dec.Replacement, "alice@example.com") {
		t.Fatalf("raw email leaked into Replacement: %q", dec.Replacement)
	}
}

// TestPIIRedactor_PostGenerate_Redact verifies the response-side
// behavior: PostGenerate returns Redact (not Replace) so the decorator
// rewrites resp.Text in place.
func TestPIIRedactor_PostGenerate_Redact(t *testing.T) {
	g := NewPIIRedactor()
	req := llm.Request{}
	resp := llm.Response{Text: "Sure, contact me at alice@example.com"}
	dec := g.Inspect(context.Background(), Event{Kind: PostGenerate, Req: &req, Resp: &resp})

	if dec.Action != Redact {
		t.Fatalf("Action = %v, want Redact", dec.Action)
	}
	if dec.Reason != "pii_redacted" {
		t.Fatalf("Reason = %q, want %q", dec.Reason, "pii_redacted")
	}
	if !strings.Contains(dec.Replacement, "[REDACTED:EMAIL]") {
		t.Fatalf("Replacement missing [REDACTED:EMAIL]: %q", dec.Replacement)
	}
}

// TestPIIRedactor_CleanText_Allow verifies that text with no PII passes
// through Allow with no Replacement payload (the gate is a no-op).
func TestPIIRedactor_CleanText_Allow(t *testing.T) {
	g := NewPIIRedactor()
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "The quick brown fox jumps over the lazy dog"},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow", dec.Action)
	}
	if dec.Replacement != "" {
		t.Fatalf("Replacement = %q, want empty", dec.Replacement)
	}
}

// TestPIIRedactor_DroppedPatterns_Q2 enforces Q2 — the SSN-specific
// `[REDACTED:SSN]` placeholder and credit-card-specific
// `[REDACTED:CREDIT_CARD]` placeholder from rag's full set are NOT
// produced by the core default set.
//
// Note: the broad `phone` pattern (`\+?\b\d[\d ()\-]{7,}\d\b`) overlaps
// the SSN format (`123-45-6789` is 11 chars of digit+dash) and the
// space-separated credit-card format (`4111 1111 1111 1111` is 19
// chars of digit+space). So the input WILL produce a Replace
// (the phone rule fires) — but the SSN/CC placeholders must be absent.
// That is the Q2 invariant: rag's *placeholders* don't show up.
func TestPIIRedactor_DroppedPatterns_Q2(t *testing.T) {
	g := NewPIIRedactor()
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "My SSN is 123-45-6789 and card 4111 1111 1111 1111"},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	// rag's SSN + CC placeholders must NOT appear — they are not in
	// defaultPIIRules per Q2 ratification.
	if strings.Contains(dec.Replacement, "[REDACTED:SSN]") {
		t.Fatalf("Q2 violated — [REDACTED:SSN] placeholder produced: %q", dec.Replacement)
	}
	if strings.Contains(dec.Replacement, "[REDACTED:CREDIT_CARD]") {
		t.Fatalf("Q2 violated — [REDACTED:CREDIT_CARD] placeholder produced: %q", dec.Replacement)
	}
}

// TestPIIRedactor_IPv4_Redacted verifies the IPv4 rule fires on the
// canonical RFC-1918 form.
func TestPIIRedactor_IPv4_Redacted(t *testing.T) {
	g := NewPIIRedactor()
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "Server at 192.168.1.1"},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Replace {
		t.Fatalf("Action = %v, want Replace", dec.Action)
	}
	if !strings.Contains(dec.Replacement, "[REDACTED:IPV4]") {
		t.Fatalf("Replacement missing [REDACTED:IPV4]: %q", dec.Replacement)
	}
	if strings.Contains(dec.Replacement, "192.168.1.1") {
		t.Fatalf("raw IPv4 leaked into Replacement: %q", dec.Replacement)
	}
}

// TestPIIRedactor_StreamDelta_DefaultOff enforces Q4 — gates built
// WITHOUT WithStreamRedaction() return Allow on StreamDelta even if
// the delta text contains PII. This is the documented best-effort
// limit (per-delta regex is expensive; cross-delta PII can leak).
func TestPIIRedactor_StreamDelta_DefaultOff(t *testing.T) {
	g := NewPIIRedactor()
	delta := llm.StreamEvent{Kind: llm.EventTextDelta, Text: "alice@example.com"}
	dec := g.Inspect(context.Background(), Event{Kind: StreamDelta, Delta: &delta})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (Q4 — StreamDelta default OFF)", dec.Action)
	}
}

// TestPIIRedactor_StreamDelta_OptIn enforces Q4's opt-in path —
// WithStreamRedaction() flips the gate into per-delta redact mode and
// matches against ev.Delta.Text.
func TestPIIRedactor_StreamDelta_OptIn(t *testing.T) {
	g := NewPIIRedactor(WithStreamRedaction())
	delta := llm.StreamEvent{Kind: llm.EventTextDelta, Text: "alice@example.com"}
	dec := g.Inspect(context.Background(), Event{Kind: StreamDelta, Delta: &delta})

	if dec.Action != Redact {
		t.Fatalf("Action = %v, want Redact (StreamDelta opt-in)", dec.Action)
	}
	if !strings.Contains(dec.Replacement, "[REDACTED:EMAIL]") {
		t.Fatalf("Replacement missing [REDACTED:EMAIL]: %q", dec.Replacement)
	}
}

// TestPIIRedactor_PreStream_PostStream_Allow verifies the stream-
// envelope events (PreStream + PostStream) are no-ops — PII detection
// has no payload to inspect at those boundaries.
func TestPIIRedactor_PreStream_PostStream_Allow(t *testing.T) {
	g := NewPIIRedactor(WithStreamRedaction())
	req := llm.Request{Messages: []llm.Message{{Role: "user", Content: "alice@example.com"}}}

	if dec := g.Inspect(context.Background(), Event{Kind: PreStream, Req: &req}); dec.Action != Allow {
		t.Fatalf("PreStream Action = %v, want Allow", dec.Action)
	}
	if dec := g.Inspect(context.Background(), Event{Kind: PostStream, Req: &req}); dec.Action != Allow {
		t.Fatalf("PostStream Action = %v, want Allow", dec.Action)
	}
}
