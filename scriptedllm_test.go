package agents

import (
	"context"
	"errors"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is a test helper that returns pre-set GenerateResponse values
// in order on each call. After the script is exhausted it returns
// errScriptExhausted. Concurrent-safe via mu.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.GenerateResponse
}

var errScriptExhausted = errors.New("scriptedLLM: script exhausted")

func newScriptedLLM(resps ...llm.GenerateResponse) *scriptedLLM {
	return &scriptedLLM{resps: resps}
}

func (s *scriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.GenerateResponse{}, errScriptExhausted
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

func (s *scriptedLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, errors.New("scriptedLLM: streaming not supported")
}

// callCount returns how many times Generate was invoked.
func (s *scriptedLLM) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// textResp builds a single text-only GenerateResponse.
func textResp(text string) llm.GenerateResponse {
	return llm.GenerateResponse{
		Text:         text,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
	}
}
