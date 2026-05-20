package agents

import (
	"context"
	"fmt"

	"github.com/costa92/llm-agent/budget"
	"github.com/costa92/llm-agent/llm"
)

func generateFromPrompt(ctx context.Context, model llm.ChatModel, systemPrompt, prompt string) (llm.Response, error) {
	req := llm.Request{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	}
	if systemPrompt != "" {
		req.SystemPrompt = systemPrompt
	}

	// Pre-call charge: MaxCalls counts attempts (Q2 — operator-confirmed
	// 2026-05-20). A denied call never reaches the network. When no
	// tracker is installed on ctx, budget.From returns (nil, false) and
	// this branch is a no-op — the load-bearing backwards-compat guarantee.
	t, hasBudget := budget.From(ctx)
	if hasBudget {
		if err := t.Charge(budget.Usage{Calls: 1}); err != nil {
			return llm.Response{}, err
		}
	}

	resp, err := model.Generate(ctx, req)
	if err != nil {
		// Upstream error path unchanged. ctx.Err() from Budget.MaxWall
		// (the WithBudget-installed deadline) surfaces here naturally.
		return resp, err
	}

	// Post-call charge: token cost from resp.Usage.TotalTokens.
	// Cost is NOT charged here in v1.2 — llm.Response.Usage has no Cost
	// field. v1.3 adds CostMapper (Estimator) to derive Cost from tokens
	// × pricing. Tracked as a v1.2 → v1.3 gap.
	if hasBudget {
		if cerr := t.Charge(budget.Usage{Tokens: resp.Usage.TotalTokens}); cerr != nil {
			// Decision 3: post-call deny returns BOTH the response AND
			// the sentinel. The response IS valid; the next call through
			// the chokepoint will be denied (either pre-call on Calls,
			// or post-call again on Tokens). Callers may discriminate
			// via errors.Is(err, budget.ErrBudgetExceeded).
			return resp, cerr
		}
	}

	return resp, nil
}

func nativeToolCaller(model llm.ChatModel) (llm.ToolCaller, bool) {
	tc, ok := model.(llm.ToolCaller)
	if !ok {
		return nil, false
	}
	if !model.Info().Capabilities.Tools {
		return nil, false
	}
	return tc, true
}

func toolCapabilityError(model llm.ChatModel) error {
	info := model.Info()
	return fmt.Errorf("%s/%s: tools: %w", info.Provider, info.Model, llm.ErrCapabilityNotSupported)
}
