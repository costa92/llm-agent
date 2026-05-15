// This file pins, at compile time, the public facade surface of
// `github.com/costa92/llm-agent/rag`. Consumers of the core agents
// framework that use the RAG facade depend on these symbols by name —
// any rename or removal breaks them silently otherwise.
//
// The mirror file `contract/contract_test.go` in the standalone repo
// (`github.com/costa92/llm-agent-rag`) pins the consumed surface from
// the other direction. Together, the two files form an explicit
// cross-repo contract that `go test` enforces.
//
// Adding to this file is a deliberate act; removing from it is a
// breaking change for consumers and must be coordinated through a
// version bump + CHANGELOG entry.
//
// See `llm-agent-rag/docs/core-compatibility.md` for the higher-level
// discussion.

package rag

import (
	"testing"
)

// TestContract_PublicFacade pins every exported symbol on this
// package's facade surface. Compile success is the gate; this test has
// no runtime assertions.
func TestContract_PublicFacade(t *testing.T) {
	// Types —
	var (
		_ RAGSystem
		_ Options
		_ SearchOptions
		_ Document
		_ SearchHit
		_ VectorStore
		_ StoreStats
		_ InMemoryStore
	)

	// Constructors —
	var (
		_ = New
		_ = NewInMemoryStore
		_ = AsTool
	)

	// Errors —
	var (
		_ = ErrEmptyQuery
		_ = ErrLLMRequired
		_ = ErrStoreNotFound
		_ = ErrDimMismatch
	)

	// Methods on RAGSystem — pinned via method-value references so a
	// signature change breaks compilation.
	var rs *RAGSystem
	var (
		_ = rs.AddText
		_ = rs.Search
		_ = rs.Ask
		_ = rs.Remove
		_ = rs.Stats
	)

	// Methods on InMemoryStore —
	var ims *InMemoryStore
	var (
		_ = ims.Upsert
		_ = ims.Search
		_ = ims.Get
		_ = ims.Remove
		_ = ims.Stats
	)

	_ = t // keep the test body referenced
}
