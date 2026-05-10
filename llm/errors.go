package llm

import "errors"

// Sentinel errors for the llm package. Callers detect via errors.Is.
// Both sentinels MUST survive `fmt.Errorf("...: %w", sentinel)` wrapping.
var (
	// ErrCapabilityNotSupported is returned by methods on capability
	// interfaces when the bound model does not actually support the
	// capability — even though the Go type implements the interface.
	//
	// Canonical wrap pattern:
	//   return nil, fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported)
	//
	// Callers detect with errors.Is(err, llm.ErrCapabilityNotSupported).
	ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")

	// ErrScriptExhausted is returned by ScriptedLLM when the script runs
	// out of pre-recorded responses. Test code matches with errors.Is.
	ErrScriptExhausted = errors.New("llm: scripted llm: script exhausted")
)
