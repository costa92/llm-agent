package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is a test helper that returns pre-set GenerateResponse
// values in order on each Generate call. After the script is exhausted
// it returns errScriptExhausted. Concurrent-safe via mu.
//
// As of v0.3 (Phase 0), this type is a thin shim that re-uses the
// canonical sentinel error from the new llm package. It still satisfies
// the v0.2 llm.Client (= LegacyClient) interface so existing agent
// paradigm tests in this package continue to compile unchanged. Phase 3
// will migrate agent paradigms to llm.ChatModel and this shim will go
// away.
//
// Deprecated: New tests should use llm.NewScriptedLLM directly. Retained
// until Phase 3 refactors agent paradigms.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.GenerateResponse
}

// errScriptExhausted aliases llm.ErrScriptExhausted so existing tests
// matching with errors.Is(err, errScriptExhausted) continue to work.
var errScriptExhausted = llm.ErrScriptExhausted

func newScriptedLLM(resps ...llm.GenerateResponse) *scriptedLLM {
	return &scriptedLLM{resps: resps}
}

// Generate returns the next scripted GenerateResponse or wraps
// errScriptExhausted when the script is exhausted.
func (s *scriptedLLM) Generate(_ context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.GenerateResponse{}, fmt.Errorf("scriptedLLM: %w", errScriptExhausted)
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

// GenerateStream returns an error — streaming was not supported by the
// v0.2 helper either. Phase 2 streaming work uses llm.ScriptedLLM
// directly via the v0.3 surface.
func (s *scriptedLLM) GenerateStream(_ context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("scriptedLLM: streaming not supported (use llm.ScriptedLLM via v0.3 surface)")
}

// callCount returns how many times Generate was invoked. Preserved for
// backwards-compat with existing tests that assert on call count.
func (s *scriptedLLM) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// textResp builds a single text-only GenerateResponse (v0.2 shape).
func textResp(text string) llm.GenerateResponse {
	return llm.GenerateResponse{
		Text:         text,
		FinishReason: llm.FinishReasonStop,
		Provider:     "scripted",
	}
}

// Compile-time: scriptedLLM satisfies the v0.2 LegacyClient (= Client) contract.
var _ llm.Client = (*scriptedLLM)(nil)
