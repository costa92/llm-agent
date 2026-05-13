package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/costa92/llm-agent/llm"
)

// scriptedLLM is a test helper that returns pre-set llm.Response
// values in order on each Generate call. After the script is exhausted
// it returns errScriptExhausted. Concurrent-safe via mu.
//
// As of v0.3 (Phase 3), this type is a thin shim that re-uses the
// canonical sentinel error from the new llm package while satisfying
// llm.ChatModel for the legacy package-local tests that still prefer a
// lightweight scripted helper.
//
// Deprecated: New tests should use llm.NewScriptedLLM directly. Retained
// until Phase 3 refactors agent paradigms.
type scriptedLLM struct {
	mu    sync.Mutex
	calls int
	resps []llm.Response
}

// errScriptExhausted aliases llm.ErrScriptExhausted so existing tests
// matching with errors.Is(err, errScriptExhausted) continue to work.
var errScriptExhausted = llm.ErrScriptExhausted

func newScriptedLLM(resps ...llm.Response) *scriptedLLM {
	return &scriptedLLM{resps: resps}
}

// Generate returns the next scripted Response or wraps errScriptExhausted
// when the script is exhausted.
func (s *scriptedLLM) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.resps) {
		s.calls++
		return llm.Response{}, fmt.Errorf("scriptedLLM: %w", errScriptExhausted)
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

// callCount returns how many times Generate was invoked. Preserved for
// backwards-compat with existing tests that assert on call count.
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
