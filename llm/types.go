package llm

import "encoding/json"

// Request is the new-surface request type. Replaces GenerateRequest at
// the new-interface (ChatModel) layer. LegacyClient continues to use
// GenerateRequest (defined in legacy.go).
//
// Why a separate type: the v0.2 GenerateRequest used Prompt+History;
// the v0.3 surface is messages-only with SystemPrompt lifted out so
// Anthropic's top-level system parameter has a clean home and OpenAI's
// system-role message can be derived from it.
type Request struct {
	Messages        []Message      `json:"messages"`                    // multi-turn dialog (preferred over Prompt)
	SystemPrompt    string         `json:"system_prompt,omitempty"`     // lifted out of Messages for Anthropic top-level system
	MaxOutputTokens int            `json:"max_output_tokens,omitempty"` // 0 = use provider default
	Temperature     *float32       `json:"temperature,omitempty"`       // pointer: nil = use provider default
	Metadata        map[string]any `json:"metadata,omitempty"`          // provider-specific extras (rare; prefer typed)
}

// Response is the new-surface response type. Replaces GenerateResponse
// at the ChatModel layer.
type Response struct {
	Text         string       `json:"text"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
	Provider     string       `json:"provider"`
	Model        string       `json:"model,omitempty"`
	Usage        Usage        `json:"usage"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
}

// Message is a single turn in a conversation. Reused unchanged from
// the v0.2 surface — same Role/Content shape. System messages are
// lifted to Request.SystemPrompt before sending to providers; the
// "system" role string remains valid for callers that prefer
// embedding it in Messages (LegacyClient flow).
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "tool", "system"
	Content string `json:"content"`
}

// Tool declares a function the model may call. Parameters is a raw
// JSON Schema document — this package doesn't validate it (the
// upstream provider does) so callers can use whatever schema dialect
// their provider expects.
//
// Shape unchanged from v0.2; shared between LegacyClient and ChatModel
// surfaces deliberately (the field names haven't needed to evolve in
// 6 months of v0.2; sharing avoids two parallel type systems).
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is what the model returns when it decides to invoke a Tool.
// The ID field is NEW vs. v0.2's ToolCall — it's the provider-assigned
// identifier (OpenAI tool_call_id, Anthropic content_block.id) that
// the agent dedupe layer (Phase 3) uses as one half of the
// (message_id, tool_use_id) dedupe key (Pitfall 4).
//
// v0.2 callers that READ ToolCalls keep working because the model is
// the only producer of ToolCall values — v0.2 did not construct
// ToolCalls explicitly, so adding an ID field is back-compat.
type ToolCall struct {
	ID        string          `json:"id,omitempty"` // provider-assigned; NEW vs v0.2
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Vector is one embedding. Length matches Embedder.EmbedDimensions().
type Vector []float32

// Usage carries token accounting for one request. Source distinguishes
// reported (provider returned actual counts), estimated (computed from
// tokenizer), and unknown (mid-stream abort, no usage available).
//
// Source != "" is an invariant after Phase 2 lands (K4); for Phase 0
// the Source field exists but defaults to UsageUnknown when the zero
// value is used.
type Usage struct {
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TotalTokens  int         `json:"total_tokens,omitempty"`
	Source       UsageSource `json:"source,omitempty"`
}

// UsageSource enumerates the provenance of token counts in a Usage.
// Reported = provider returned actual counts; Estimated = computed
// from a tokenizer; Unknown = mid-stream abort, no usage available
// (NOT zero-tokens — Pitfall 5).
type UsageSource string

const (
	UsageReported  UsageSource = "reported"
	UsageEstimated UsageSource = "estimated"
	UsageUnknown   UsageSource = "unknown"
)

// FinishReason is an alias for the underlying legacyFinishReason
// string type defined in legacy.go. The alias means LegacyClient and
// ChatModel callers see the same FinishReason name and the same
// constant set (FinishReasonStop, FinishReasonLength, etc.) — type
// identity is preserved across the v0.2 / v0.3 surfaces.
type FinishReason = legacyFinishReason
