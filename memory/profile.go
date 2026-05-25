package memory

// This file adds "ChatGPT-style" profile metadata helpers on top of
// MemoryItem.Metadata WITHOUT changing the MemoryItem struct or any
// existing Memory interface method. All state lives inside the
// existing map[string]any under a reserved "_"-prefixed namespace so
// it does not collide with caller-supplied metadata keys.
//
// PR 1/4 for v0.7.0. Subsequent PRs (ScopedManager, decay/promote
// tuning, MemoryTool actions) extend this metadata namespace.

// Source classifies how a memory entered the system.
type Source string

const (
	// SourceUserSaved marks a memory the user explicitly asked the agent
	// to remember ("Remember that I ..."). Constructors default Pinned=true.
	SourceUserSaved Source = "user_saved"

	// SourceAgentInferred marks a memory the agent decided to record
	// from conversation without an explicit save instruction.
	SourceAgentInferred Source = "agent_inferred"

	// SourceSystem marks a memory injected by the platform (defaults,
	// onboarding bootstrap, etc.).
	SourceSystem Source = "system"

	// SourceUnknown is the zero value (no _source metadata set).
	SourceUnknown Source = ""
)

// Category is the user-facing taxonomy. Mirrors ChatGPT's
// "Manage memories" filters (User / Feedback / Project / Reference).
// Callers can store additional category strings; these constants are
// just the canonical names.
type Category string

const (
	CategoryUser      Category = "user"
	CategoryFeedback  Category = "feedback"
	CategoryProject   Category = "project"
	CategoryReference Category = "reference"
)

// Reserved metadata keys. The leading "_" namespace avoids collision
// with caller-supplied Metadata keys. Kept package-private so callers
// must go through the typed accessors below.
const (
	metaKeyScope    = "_scope" // reserved for PR 2 (ScopedManager)
	metaKeySource   = "_source"
	metaKeyCategory = "_category"
	metaKeyPinned   = "_pinned"
	metaKeyDisabled = "_disabled"
)

// --- Constructors ----------------------------------------------------------

// NewSavedMemory builds a MemoryItem with ChatGPT-style "user-saved"
// defaults: high Importance, Pinned, Source=SourceUserSaved.
//
// The caller may further mutate the returned value (set Tags, override
// Importance, etc.) before handing to Memory.Add.
func NewSavedMemory(content string, cat Category) MemoryItem {
	it := MemoryItem{
		Content:    content,
		Importance: 0.9,
		Metadata:   map[string]any{},
	}
	SetSource(&it, SourceUserSaved)
	SetCategory(&it, cat)
	SetPinned(&it, true)
	return it
}

// NewInferredMemory builds a MemoryItem the agent inferred from
// conversation. Confidence ∈ [0,1] is clamped and stored as Importance;
// Source=SourceAgentInferred; not pinned.
func NewInferredMemory(content string, cat Category, confidence float64) MemoryItem {
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	it := MemoryItem{
		Content:    content,
		Importance: confidence,
		Metadata:   map[string]any{},
	}
	SetSource(&it, SourceAgentInferred)
	SetCategory(&it, cat)
	return it
}

// --- Getters ---------------------------------------------------------------
//
// All getters return the zero value when Metadata is nil, the key is
// missing, or the stored value has the wrong type. They never panic.

// GetSource returns the Source recorded on the item, or SourceUnknown
// when absent / wrong type.
func GetSource(it MemoryItem) Source {
	if it.Metadata == nil {
		return SourceUnknown
	}
	switch v := it.Metadata[metaKeySource].(type) {
	case Source:
		return v
	case string:
		return Source(v)
	default:
		return SourceUnknown
	}
}

// GetCategory returns the Category recorded on the item, or empty when
// absent / wrong type.
func GetCategory(it MemoryItem) Category {
	if it.Metadata == nil {
		return ""
	}
	switch v := it.Metadata[metaKeyCategory].(type) {
	case Category:
		return v
	case string:
		return Category(v)
	default:
		return ""
	}
}

// IsPinned reports whether the item is marked pinned. Pinned items are
// excluded from Forget strategies and (when SavedBoost is configured)
// receive a multiplicative score boost during Search.
func IsPinned(it MemoryItem) bool {
	if it.Metadata == nil {
		return false
	}
	v, ok := it.Metadata[metaKeyPinned].(bool)
	return ok && v
}

// IsDisabled reports whether the item is marked disabled. Disabled
// items remain in storage (Get / Stats / Forget still see them) but
// are filtered out of Search results.
func IsDisabled(it MemoryItem) bool {
	if it.Metadata == nil {
		return false
	}
	v, ok := it.Metadata[metaKeyDisabled].(bool)
	return ok && v
}

// --- Setters ---------------------------------------------------------------
//
// All setters initialize Metadata when nil. They write the strongly-typed
// value so subsequent getters take the typed branch.

// SetSource writes the Source onto the item's Metadata.
func SetSource(it *MemoryItem, source Source) {
	ensureMetadata(it)
	it.Metadata[metaKeySource] = source
}

// SetCategory writes the Category onto the item's Metadata.
func SetCategory(it *MemoryItem, cat Category) {
	ensureMetadata(it)
	it.Metadata[metaKeyCategory] = cat
}

// SetPinned writes the pinned flag onto the item's Metadata.
func SetPinned(it *MemoryItem, pinned bool) {
	ensureMetadata(it)
	it.Metadata[metaKeyPinned] = pinned
}

// SetDisabled writes the disabled flag onto the item's Metadata.
func SetDisabled(it *MemoryItem, disabled bool) {
	ensureMetadata(it)
	it.Metadata[metaKeyDisabled] = disabled
}

func ensureMetadata(it *MemoryItem) {
	if it.Metadata == nil {
		it.Metadata = map[string]any{}
	}
}
