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

// TestPIIRedactor_DroppedPatterns_Q2 enforces Q2 — the SSN
// `123-45-6789` and credit-card `4111 1111 1111 1111` patterns from
// rag's full set are NOT in the default core set. A user message
// containing only those (plus no email/phone/ipv4) must Allow.
//
// Note: the phone pattern (`\+?\b\d[\d ()\-]{7,}\d\b`) is broad — it
// matches the spaces-separated 16-digit credit card AND the SSN format
// as a 9-digit run. To verify Q2 cleanly we use a card without spaces
// and an SSN-shaped string broken by whitespace so the phone regex
// doesn't catch them.
func TestPIIRedactor_DroppedPatterns_Q2(t *testing.T) {
	g := NewPIIRedactor()
	// `4111-1111-1111-1111` (with dashes) would match rag's
	// credit_card pattern but NOT our phone pattern (the inner class
	// `[\d ()\-]` requires the 1-character break to be space/paren/dash
	// — dashes are valid, so this DOES match phone too). To get a
	// cleanly Q2-only test, use a number that ONLY rag's credit_card
	// would catch — a 4444555566667777 contiguous run (16 digits, no
	// spaces, no dashes). The phone pattern requires at least one
	// non-digit separator inside, so it cannot match.
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "card: 4444555566667777 ssn-shaped: 12345 67 89"},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (Q2 — ssn + credit_card dropped); Replacement=%q",
			dec.Action, dec.Replacement)
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
