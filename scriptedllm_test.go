package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is the agents-package test helper that returns pre-set
// llm.Response values in order on each Generate call. After the script
// is exhausted it wraps llm.ErrScriptExhausted. Concurrent-safe via mu.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.Response
}

func newScriptedLLM(resps ...llm.Response) *scriptedLLM {
	return &scriptedLLM{resps: resps}
}

func (s *scriptedLLM) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.Response{}, fmt.Errorf("scriptedLLM: %w", llm.ErrScriptExhausted)
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

func (s *scriptedLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	resp, err := s.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	return llm.NewScriptedLLM(llm.WithResponses(resp)).Stream(ctx, llm.Request{})
}

func (s *scriptedLLM) Info() llm.ProviderInfo {
	return llm.ProviderInfo{
		Provider: "scripted",
		Model:    "test",
		Capabilities: llm.Capabilities{
			Tools: false,
		},
	}
}

// callCount returns how many times Generate was invoked.
func (s *scriptedLLM) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// textResp builds a single text-only llm.Response.
func textResp(text string) llm.Response {
	return llm.Response{
		Text:         text,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
		Usage:        llm.Usage{Source: llm.UsageReported},
	}
}

var _ llm.ChatModel = (*scriptedLLM)(nil)
