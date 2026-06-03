// Demo 06: Budget / cancellation context.
//
// Wires budget.WithBudget into a SimpleAgent and demonstrates the three
// budget dimensions enforced by the agents.generateFromPrompt chokepoint
// (Phase 35, CC-1):
//
//   - MaxCalls : pre-call deny — the LLM is NOT reached on the denied attempt.
//   - MaxTokens: post-call deny — the LLM IS reached and a valid response is
//     produced; the chokepoint returns the sentinel after charging.
//   - MaxWall  : ctx-deadline cancellation — WithBudget installs a
//     context.WithDeadline, so the LLM's Generate sees ctx.Done() and the
//     err surface is context.DeadlineExceeded (no new StreamEvent.Kind).
//
// The whole demo is deterministic — the canonical scriptedllm mock (per
// CLAUDE.md) returns pre-recorded responses, no network is touched. The
// MaxWall demo uses a 200 ms sleep against a 50 ms deadline (4x margin) so
// the wall-clock deadline fires reliably on any machine.
//
// Run:
//
//	cd examples && go run ./06-budget
package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent/examples/scriptedllm"
	"github.com/costa92/llm-agent-contract/llm"
)

func main() {
	demoMaxCalls()
	demoMaxTokens()
	demoMaxWall()
	fmt.Println("OK")
}

// ---------------------------------------------------------------------------
// MaxCalls — pre-call deny
// ---------------------------------------------------------------------------
//
// Budget{MaxCalls: 3} caps attempts (Q2 — counts attempts, not successes).
// The chokepoint pre-call Charge fires on the 4th attempt and returns the
// sentinel BEFORE the LLM is invoked — so the LLM call count stops at 3.
func demoMaxCalls() {
	fmt.Println("--- MaxCalls (pre-call deny) ---")

	inner := scriptedllm.New(
		tokenText("r1", 10),
		tokenText("r2", 10),
		tokenText("r3", 10),
		tokenText("r4", 10), // never reached
	)
	counted := &countingLLM{inner: inner}

	ctx, t := budget.WithBudget(context.Background(), budget.Budget{MaxCalls: 3})
	agent := agents.NewSimpleAgent(counted, agents.SimpleOptions{Name: "demo"})

	var lastErr error
	for i := 1; i <= 4; i++ {
		_, err := agent.Run(ctx, fmt.Sprintf("call %d", i))
		if err != nil {
			lastErr = err
			fmt.Printf("call %d: denied — %v\n", i, err)
			break
		}
		fmt.Printf("call %d: ok\n", i)
	}

	fmt.Printf("4th denied with errors.Is(err, budget.ErrCallsExceeded) = %v\n",
		errors.Is(lastErr, budget.ErrCallsExceeded))
	fmt.Printf("4th denied with errors.Is(err, budget.ErrBudgetExceeded) = %v\n",
		errors.Is(lastErr, budget.ErrBudgetExceeded))
	fmt.Printf("LLM Generate calls reaching the model: %d (denied attempt never reaches LLM)\n",
		counted.calls())
	fmt.Printf("tracker snapshot: %+v\n", t.Snapshot())
	fmt.Println()
}

// ---------------------------------------------------------------------------
// MaxTokens — post-call deny (response + sentinel both returned)
// ---------------------------------------------------------------------------
//
// Budget{MaxTokens: 100} with 3 scripted 60-token responses. Call 1 commits
// 60 tokens. Call 2's pre-call passes (Calls is uncapped); the LLM returns
// a valid 60-token response; the chokepoint post-call Charge then refuses
// to commit (would overflow → 120) and surfaces (resp, ErrTokensExceeded).
// Decision 3 (35-RESEARCH.md): the response IS valid and IS returned.
//
// SimpleAgent.Run collapses to (Result{}, err) on any error from
// generateFromPrompt — so the example proves the response surfaced by
// counting the model's Generate calls: the LLM WAS reached even on the
// denied attempt. (The lower-level test in agent_chatmodel_test.go
// asserts the (resp, err) pair directly; SimpleAgent's public surface
// is enough to demonstrate the contract for a reader.)
func demoMaxTokens() {
	fmt.Println("--- MaxTokens (post-call deny) ---")

	inner := scriptedllm.New(
		tokenText("r1", 60),
		tokenText("r2", 60), // reaches LLM, then chokepoint denies on post-charge
		tokenText("r3", 60), // reaches LLM again, denied again — same contract
	)
	counted := &countingLLM{inner: inner}

	ctx, t := budget.WithBudget(context.Background(), budget.Budget{MaxTokens: 100})
	agent := agents.NewSimpleAgent(counted, agents.SimpleOptions{Name: "demo"})

	for i := 1; i <= 3; i++ {
		_, err := agent.Run(ctx, fmt.Sprintf("call %d", i))
		switch {
		case err == nil:
			fmt.Printf("call %d: ok\n", i)
		case errors.Is(err, budget.ErrTokensExceeded):
			fmt.Printf("call %d: valid response but exhausted — %v\n", i, err)
		default:
			fmt.Printf("call %d: unexpected err — %v\n", i, err)
		}
	}

	fmt.Printf("LLM Generate calls reaching the model: %d (network call succeeded all 3 times — deny is post-call)\n",
		counted.calls())
	fmt.Printf("tracker snapshot: %+v (no-commit-on-deny: only the successful 60 tokens are recorded)\n", t.Snapshot())
	fmt.Println()
}

// ---------------------------------------------------------------------------
// MaxWall — ctx-deadline cancellation
// ---------------------------------------------------------------------------
//
// Budget{MaxWall: 50ms} causes WithBudget to derive a
// context.WithDeadline(parent, time.Now().Add(50ms)). The slowLLM wrapper
// sleeps 200ms in Generate honoring ctx.Done(), so the deadline fires
// before the response is returned. The chokepoint surfaces the raw
// context.DeadlineExceeded — wall-clock has zero new error surface
// (Decision 4 in 35-RESEARCH.md).
func demoMaxWall() {
	fmt.Println("--- MaxWall (ctx.DeadlineExceeded) ---")

	inner := scriptedllm.New(tokenText("never-returned", 10))
	slow := &slowLLM{inner: inner, pause: 200 * time.Millisecond}

	ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxWall: 50 * time.Millisecond})
	agent := agents.NewSimpleAgent(slow, agents.SimpleOptions{Name: "demo"})

	start := time.Now()
	_, err := agent.Run(ctx, "trigger wall-clock cap")
	elapsed := time.Since(start)

	fmt.Printf("call: errors.Is(err, context.DeadlineExceeded) = %v (wall-clock fires via ctx, not a budget sentinel)\n",
		errors.Is(err, context.DeadlineExceeded))
	fmt.Printf("deadline fired before response: elapsed < 4x deadline? %v\n",
		elapsed < 200*time.Millisecond) // sanity: cancelled long before slowLLM would have returned
	fmt.Println()
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// tokenText builds a scripted response with the given text and a populated
// Usage.TotalTokens so the chokepoint's post-call charge has something to
// count. The canonical scriptedllm.Text helper leaves Usage.TotalTokens
// at zero, which would defeat the MaxTokens demo.
func tokenText(text string, tokens int) llm.Response {
	r := scriptedllm.Text(text)
	r.Usage.TotalTokens = tokens
	return r
}

// countingLLM wraps an llm.ChatModel and counts how many times Generate
// reaches it. Used to show that pre-call deny short-circuits before the
// LLM is invoked, and that post-call deny does invoke the LLM.
type countingLLM struct {
	inner llm.ChatModel
	n     int64 // atomic
}

func (c *countingLLM) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	atomic.AddInt64(&c.n, 1)
	return c.inner.Generate(ctx, req)
}
func (c *countingLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return c.inner.Stream(ctx, req)
}
func (c *countingLLM) Info() llm.ProviderInfo { return c.inner.Info() }
func (c *countingLLM) calls() int             { return int(atomic.LoadInt64(&c.n)) }

// slowLLM wraps an llm.ChatModel and sleeps `pause` before delegating
// Generate. It honors ctx.Done() so the MaxWall demo's deadline fires
// cleanly. Mirrors the real-provider pattern (HTTP client respects
// request context cancellation).
type slowLLM struct {
	inner llm.ChatModel
	pause time.Duration
}

func (s *slowLLM) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	select {
	case <-time.After(s.pause):
		return s.inner.Generate(ctx, req)
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
}
func (s *slowLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	return s.inner.Stream(ctx, req)
}
func (s *slowLLM) Info() llm.ProviderInfo { return s.inner.Info() }
