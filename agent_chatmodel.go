package agents

import (
	"context"
	"fmt"

	"github.com/costa92/llm-agent/llm"
)

func generateFromPrompt(ctx context.Context, model llm.ChatModel, systemPrompt, prompt string) (llm.Response, error) {
	req := llm.Request{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	}
	if systemPrompt != "" {
		req.SystemPrompt = systemPrompt
	}
	return model.Generate(ctx, req)
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
