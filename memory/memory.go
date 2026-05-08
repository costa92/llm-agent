// Package memory implements 3 in-process Memory types (Working /
// Episodic / Semantic) plus a Manager that coordinates Consolidate
// and Forget. Each type satisfies the Memory interface so a single
// Agent can route writes/reads by Kind.
//
// Backend dependency: pkg/llm/agents/rag.Embedder for vector scoring.
// Phase 2 ships HashEmbedder (zero-deps); Phase 3 adds real backends.
//
// # Portability
//
// memory inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package memory

import (
	"context"
	"errors"
	"time"
)

// Kind identifies which of the three Memory types an item belongs to.
type Kind string

const (
	KindWorking  Kind = "working"
	KindEpisodic Kind = "episodic"
	KindSemantic Kind = "semantic"
)

// MemoryItem is the unit of storage. Importance ∈ [0,1] drives forget
// + consolidate. Tags are arbitrary labels (used by Semantic for
// overlap scoring; cosmetic in Working/Episodic).
type MemoryItem struct {
	ID         string
	Content    string
	Tags       []string
	Importance float64
	CreatedAt  time.Time
	AccessedAt time.Time
	Metadata   map[string]any
}

// SearchResult pairs a MemoryItem with its query-relevance Score.
// Score is a domain-specific composite (per Memory type) in [0, +∞).
type SearchResult struct {
	Item  MemoryItem
	Score float64
}

// Stats summarizes a Memory's contents — useful for debugging + the
// MemoryTool stats action.
type Stats struct {
	Count         int
	Capacity      int           // 0 = unlimited
	OldestAge     time.Duration // duration since the oldest item's CreatedAt
	AvgImportance float64
}

// Memory is the contract every memory type satisfies. All methods are
// goroutine-safe in the bundled implementations.
type Memory interface {
	Type() Kind
	Add(ctx context.Context, item MemoryItem) (string, error) // returns generated ID
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	Get(ctx context.Context, id string) (MemoryItem, error)
	Update(ctx context.Context, id string, fn func(*MemoryItem)) error
	Remove(ctx context.Context, id string) error
	Stats() Stats
}

// --- sentinel errors ------------------------------------------------------

// ErrNotFound is returned by Get / Update / Remove when id is absent.
var ErrNotFound = errors.New("memory: item not found")

// ErrEmptyQuery is returned by Search when query is whitespace-only.
var ErrEmptyQuery = errors.New("memory: query is required")

// ErrEmbedderRequired is returned by constructors when Embedder is nil.
var ErrEmbedderRequired = errors.New("memory: embedder is required")
