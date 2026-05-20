package agents

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent/llm"
)

// tokenResp builds a single text-only llm.Response with a specified
// Usage.TotalTokens. The canonical textResp helper (scriptedllm_test.go)
// leaves Usage empty; the budget chokepoint tests need Usage populated
// to exercise the post-call Tokens charge.
func tokenResp(text string, totalTokens int) llm.Response {
	return llm.Response{
		Text:         text,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
		Usage: llm.Usage{
			TotalTokens: totalTokens,
			Source:      llm.UsageReported,
		},
	}
}

// slowScriptedLLM is a test-local wrapper around scriptedLLM that sleeps
// for `delay` before delegating to Generate. Used by the MaxWall test to
// drive ctx.Done() via the deadline installed by budget.WithBudget.
type slowScriptedLLM struct {
	inner *scriptedLLM
	delay time.Duration
}

func newSlowScriptedLLM(delay time.Duration, resps ...llm.Response) *slowScriptedLLM {
	return &slowScriptedLLM{inner: newScriptedLLM(resps...), delay: delay}
}

func (s *slowScriptedLLM) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	// Honor ctx cancellation while sleeping — this is what real
	// provider HTTP clients do (cf. net/http honors ctx.Done()).
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
	return s.inner.Generate(ctx, req)
}

func (s *slowScriptedLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return s.inner.Stream(ctx, req)
}

func (s *slowScriptedLLM) Info() llm.ProviderInfo { return s.inner.Info() }

var _ llm.ChatModel = (*slowScriptedLLM)(nil)

// TestGenerateFromPrompt_NoBudget_Passthrough proves zero behavior
// change when no budget is installed on ctx — the load-bearing
// backwards-compat guarantee. The scriptedLLM call counter must advance
// exactly N times and every response must be returned byte-identical to
// what was scripted.
func TestGenerateFromPrompt_NoBudget_Passthrough(t *testing.T) {
	want := []llm.Response{
		tokenResp("one", 10),
		tokenResp("two", 20),
		tokenResp("three", 30),
		tokenResp("four", 40),
		tokenResp("five", 50),
	}
	s := newScriptedLLM(want...)
	ctx := context.Background()
	for i, w := range want {
		got, err := generateFromPrompt(ctx, s, "", "hi")
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if got.Text != w.Text || got.Usage.TotalTokens != w.Usage.TotalTokens {
			t.Errorf("call %d: got %+v, want %+v", i, got, w)
		}
	}
	if got := s.callCount(); got != len(want) {
		t.Errorf("callCount = %d, want %d", got, len(want))
	}
	// Confirm From(ctx) on a budget-less ctx returns (nil, false) — the
	// invariant the chokepoint relies on.
	if _, ok := budget.From(ctx); ok {
		t.Errorf("budget.From on background ctx returned ok=true, want false")
	}
}

// TestGenerateFromPrompt_MaxCalls_PreCallDeny proves the pre-call
// Charge(Calls: 1) short-circuits the network round-trip. With
// MaxCalls=3, the 4th call must return ErrCallsExceeded with a zero
// llm.Response, and the scriptedLLM call counter must read 3 — proving
// the LLM was NOT invoked on the denied attempt.
func TestGenerateFromPrompt_MaxCalls_PreCallDeny(t *testing.T) {
	s := newScriptedLLM(
		tokenResp("r1", 5),
		tokenResp("r2", 5),
		tokenResp("r3", 5),
		tokenResp("r4-never-served", 5),
	)
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
	for i := 0; i < 3; i++ {
		if _, err := generateFromPrompt(ctx, s, "", "hi"); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	resp, err := generateFromPrompt(ctx, s, "", "hi")
	if !errors.Is(err, budget.ErrCallsExceeded) {
		t.Fatalf("4th call err = %v, want errors.Is(..., ErrCallsExceeded)", err)
	}
	if !errors.Is(err, budget.ErrBudgetExceeded) {
		t.Errorf("ErrCallsExceeded should wrap ErrBudgetExceeded")
	}
	// Can't compare llm.Response directly (contains a slice). Check the
	// scalar fields that ScriptedLLM would populate on a real call.
	if resp.Text != "" || resp.FinishReason != "" || resp.Provider != "" || resp.Usage.TotalTokens != 0 {
		t.Errorf("denied call returned non-zero Response %+v, want zero (proves no network round-trip)", resp)
	}
	if got := s.callCount(); got != 3 {
		t.Errorf("callCount = %d, want 3 (the denied attempt MUST NOT reach ScriptedLLM)", got)
	}
}

// TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth proves the
// post-call charge denies AFTER the response was generated, and the
// chokepoint returns BOTH the response AND the sentinel (Decision 3).
//
// Setup: MaxTokens=100, each scripted response carries 60 tokens.
//   - Call 1: pre-call Calls=1 OK; network returns 60-token resp; post-
//     call cumulative tokens = 60 ≤ 100 → returns (resp, nil).
//   - Call 2: pre-call Calls=2 OK; network returns 60-token resp; post-
//     call cumulative tokens = 120 > 100 → returns (resp, ErrTokensExceeded).
//     The response itself is valid; the network IS called.
//   - Call 3: pre-call Calls=3 OK (MaxCalls is 0=no cap); network returns
//     60-token resp; post-call cumulative tokens = 180 > 100 → returns
//     (resp, ErrTokensExceeded). v1.2 has no Estimator so pre-call
//     deny on tokens is impossible — deferred to v1.3.
func TestGenerateFromPrompt_MaxTokens_PostCallDeny_ReturnsBoth(t *testing.T) {
	s := newScriptedLLM(
		tokenResp("r1", 60),
		tokenResp("r2", 60),
		tokenResp("r3", 60),
	)
	ctx, tracker := budget.WithBudget(context.Background(), budget.Budget{MaxTokens: 100})

	// Call 1: under cap.
	resp1, err1 := generateFromPrompt(ctx, s, "", "hi")
	if err1 != nil {
		t.Fatalf("call 1: unexpected err: %v", err1)
	}
	if resp1.Text != "r1" {
		t.Errorf("call 1: text=%q want r1", resp1.Text)
	}
	if snap := tracker.Snapshot(); snap.Tokens != 60 {
		t.Errorf("after call 1: Snapshot.Tokens=%d want 60", snap.Tokens)
	}

	// Call 2: post-call deny — response IS returned, sentinel ALSO returned.
	resp2, err2 := generateFromPrompt(ctx, s, "", "hi")
	if !errors.Is(err2, budget.ErrTokensExceeded) {
		t.Fatalf("call 2: err = %v, want ErrTokensExceeded", err2)
	}
	if !errors.Is(err2, budget.ErrBudgetExceeded) {
		t.Errorf("ErrTokensExceeded should wrap ErrBudgetExceeded")
	}
	if resp2.Text != "r2" {
		t.Errorf("call 2: resp.Text=%q want r2 (response IS produced on post-call deny)", resp2.Text)
	}
	if resp2.Usage.TotalTokens != 60 {
		t.Errorf("call 2: resp.Usage.TotalTokens=%d want 60", resp2.Usage.TotalTokens)
	}
	// Snapshot must remain at 60 — the strict tracker does NOT commit on a denied charge.
	if snap := tracker.Snapshot(); snap.Tokens != 60 {
		t.Errorf("after call 2 (denied): Snapshot.Tokens=%d want 60 (no-commit-on-deny)", snap.Tokens)
	}

	// Call 3: still produces resp + sentinel (v1.2 post-hoc semantics).
	resp3, err3 := generateFromPrompt(ctx, s, "", "hi")
	if !errors.Is(err3, budget.ErrTokensExceeded) {
		t.Fatalf("call 3: err = %v, want ErrTokensExceeded", err3)
	}
	if resp3.Text != "r3" {
		t.Errorf("call 3: resp.Text=%q want r3", resp3.Text)
	}

	// Scripted called 3 times: all attempts reached the LLM (Calls was uncapped).
	if got := s.callCount(); got != 3 {
		t.Errorf("callCount = %d, want 3 (no MaxCalls — all network calls happen)", got)
	}
}

// TestGenerateFromPrompt_MaxWall_ContextDeadline proves wall-clock
// enforcement happens via ctx.Done(), not via a tracker.Charge(Wall:).
// The chokepoint adds no wall-clock surface; WithBudget's installed
// context.WithDeadline cancels the slow ScriptedLLM.
func TestGenerateFromPrompt_MaxWall_ContextDeadline(t *testing.T) {
	slow := newSlowScriptedLLM(200*time.Millisecond, tokenResp("never-served", 0))
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxWall: 50 * time.Millisecond})
	start := time.Now()
	_, err := generateFromPrompt(ctx, slow, "", "hi")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected non-nil error from wall-clock deadline, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want errors.Is(..., context.DeadlineExceeded)", err)
	}
	if elapsed > 180*time.Millisecond {
		t.Errorf("elapsed=%v — deadline should fire ~50ms, not wait full 200ms", elapsed)
	}
}

// TestGenerateFromPrompt_Concurrent_Race proves the chokepoint is
// race-clean: 20 goroutines × 10 calls each = 200 attempts against
// MaxCalls=50. Tracker semantics guarantee exactly 50 successes; the
// remaining 150 attempts return ErrCallsExceeded pre-call. Must pass
// under `go test -race`.
func TestGenerateFromPrompt_Concurrent_Race(t *testing.T) {
	const goroutines = 20
	const perGoroutine = 10
	const maxCalls = 50
	const total = goroutines * perGoroutine

	// Each call needs an available scripted response or the LLM returns
	// errScriptExhausted (which is itself an error). To avoid script
	// exhaustion swallowing the test signal, pre-load total responses.
	resps := make([]llm.Response, total)
	for i := range resps {
		resps[i] = tokenResp("r", 1)
	}
	s := newScriptedLLM(resps...)
	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: maxCalls})

	var (
		succ   int64
		denied int64
		other  int64
		wg     sync.WaitGroup
	)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				_, err := generateFromPrompt(ctx, s, "", "hi")
				switch {
				case err == nil:
					atomic.AddInt64(&succ, 1)
				case errors.Is(err, budget.ErrCallsExceeded):
					atomic.AddInt64(&denied, 1)
				default:
					atomic.AddInt64(&other, 1)
				}
			}
		}()
	}
	wg.Wait()

	if other != 0 {
		t.Errorf("unexpected non-budget errors: %d", other)
	}
	if succ != maxCalls {
		t.Errorf("succ = %d, want exactly %d (MaxCalls cap)", succ, maxCalls)
	}
	if succ+denied != total {
		t.Errorf("succ+denied = %d, want %d (every attempt accounted for)", succ+denied, total)
	}
	if got := s.callCount(); int64(got) != succ {
		t.Errorf("scripted.callCount = %d, want %d (denied attempts must NOT reach LLM)", got, succ)
	}
}
