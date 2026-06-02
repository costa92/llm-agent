package policy

import (
	"context"
	"strings"
	"testing"

	"github.com/costa92/llm-agent-contract/llm"
)

// TestMaxInputLen_Name verifies the canonical Name() identifier.
func TestMaxInputLen_Name(t *testing.T) {
	g := NewMaxInputLen(100)
	if got := g.Name(); got != "MaxInputLen" {
		t.Fatalf("Name() = %q, want %q", got, "MaxInputLen")
	}
}

// TestMaxInputLen_UnderCap_Allow verifies a request comfortably under
// the cap Allows.
func TestMaxInputLen_UnderCap_Allow(t *testing.T) {
	g := NewMaxInputLen(100)
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: strings.Repeat("a", 50)},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow", dec.Action)
	}
}

// TestMaxInputLen_AtCap_Allow verifies the cap is inclusive on the
// upper bound — exactly N bytes is OK; only `> N` Blocks.
func TestMaxInputLen_AtCap_Allow(t *testing.T) {
	g := NewMaxInputLen(50)
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: strings.Repeat("a", 50)},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (cap is inclusive at 50 bytes)", dec.Action)
	}
}

// TestMaxInputLen_OverCap_Block verifies 1 byte over the cap fires.
func TestMaxInputLen_OverCap_Block(t *testing.T) {
	g := NewMaxInputLen(50)
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: strings.Repeat("a", 51)},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Block {
		t.Fatalf("Action = %v, want Block", dec.Action)
	}
	if dec.Reason != "length_exceeded" {
		t.Fatalf("Reason = %q, want %q", dec.Reason, "length_exceeded")
	}
}

// TestMaxInputLen_CountsSystemPrompt verifies the SystemPrompt bytes
// are added to the message total (provider HTTP byte budgets count
// system + user together).
func TestMaxInputLen_CountsSystemPrompt(t *testing.T) {
	g := NewMaxInputLen(20)
	req := llm.Request{
		SystemPrompt: "hello", // 5 bytes
		Messages: []llm.Message{
			{Role: "user", Content: "world is long enough"}, // 20 bytes
		},
	}
	// total = 25 bytes > 20 → Block
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Block {
		t.Fatalf("Action = %v, want Block (5 + 20 = 25 > 20)", dec.Action)
	}
}

// TestMaxInputLen_MultipleMessages verifies the message total is
// summed across all turns (multi-turn dialog with no SystemPrompt).
func TestMaxInputLen_MultipleMessages(t *testing.T) {
	mkReq := func() llm.Request {
		return llm.Request{Messages: []llm.Message{
			{Role: "user", Content: strings.Repeat("a", 10)},
			{Role: "assistant", Content: strings.Repeat("b", 10)},
			{Role: "user", Content: strings.Repeat("c", 10)},
		}}
	}

	// total = 30 bytes; cap 25 → Block.
	g25 := NewMaxInputLen(25)
	r1 := mkReq()
	if dec := g25.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &r1}); dec.Action != Block {
		t.Fatalf("cap=25 Action = %v, want Block (sum=30)", dec.Action)
	}

	// total = 30 bytes; cap 35 → Allow.
	g35 := NewMaxInputLen(35)
	r2 := mkReq()
	if dec := g35.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &r2}); dec.Action != Allow {
		t.Fatalf("cap=35 Action = %v, want Allow (sum=30)", dec.Action)
	}
}

// TestMaxInputLen_ByteSemantics_Q3 enforces Q3 — len() returns bytes,
// not runes. The string "中文" is 6 bytes in UTF-8 (3 bytes per
// Chinese character) but only 2 runes. A 4-byte cap MUST Block;
// a 10-byte cap MUST Allow. This is the documented Q3 invariant.
//
// A future rune-mode gate (v1.3 NewMaxInputLenRunes) is the additive
// path for callers who want grapheme-cluster semantics; that gate
// would see this string as length 2 and Allow at cap 4.
func TestMaxInputLen_ByteSemantics_Q3(t *testing.T) {
	const chinese = "中文" // 6 bytes UTF-8, 2 runes

	g4 := NewMaxInputLen(4)
	r1 := llm.Request{Messages: []llm.Message{{Role: "user", Content: chinese}}}
	if dec := g4.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &r1}); dec.Action != Block {
		t.Fatalf("cap=4 Action = %v, want Block (Q3 byte semantics — 6 bytes > 4)", dec.Action)
	}

	g10 := NewMaxInputLen(10)
	r2 := llm.Request{Messages: []llm.Message{{Role: "user", Content: chinese}}}
	if dec := g10.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &r2}); dec.Action != Allow {
		t.Fatalf("cap=10 Action = %v, want Allow (Q3 byte semantics — 6 bytes <= 10)", dec.Action)
	}
}

// TestMaxInputLen_ZeroCap_NoEnforcement verifies that cap == 0 is
// treated as "no cap" (defensive — the zero value of the gate's
// internal field should be a safe no-op).
func TestMaxInputLen_ZeroCap_NoEnforcement(t *testing.T) {
	g := NewMaxInputLen(0)
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: strings.Repeat("a", 1_000_000)},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (cap=0 means no enforcement)", dec.Action)
	}
}

// TestMaxInputLen_NegativeCap_NoEnforcement verifies that a negative
// cap is also a no-op (defensive against bad caller config).
func TestMaxInputLen_NegativeCap_NoEnforcement(t *testing.T) {
	g := NewMaxInputLen(-1)
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: strings.Repeat("a", 100)},
	}}
	dec := g.Inspect(context.Background(), Event{Kind: PreGenerate, Req: &req})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (negative cap is no-op)", dec.Action)
	}
}

// TestMaxInputLen_PostGenerate_Allow verifies post-call events are
// no-ops — length is a REQUEST-SIDE gate (the network has already paid
// the cost; capping the response is a different concern, deferred).
func TestMaxInputLen_PostGenerate_Allow(t *testing.T) {
	g := NewMaxInputLen(10)
	resp := llm.Response{Text: strings.Repeat("a", 1_000_000)}
	dec := g.Inspect(context.Background(), Event{Kind: PostGenerate, Resp: &resp})

	if dec.Action != Allow {
		t.Fatalf("Action = %v, want Allow (PostGenerate is not the length gate's surface)", dec.Action)
	}
}
