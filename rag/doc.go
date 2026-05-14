// Package rag is the compatibility facade for Retrieval-Augmented Generation
// inside the main llm-agent repository.
//
// The implementation source of truth now lives in the standalone SDK:
//
//   - github.com/costa92/llm-agent-rag
//
// This package continues to expose the historical llm-agent API so existing
// callers do not need to migrate immediately:
//
//   - Embedder
//   - Chunker
//   - VectorStore
//   - RAGSystem
//   - AsTool
//
// Most implementation logic is delegated to the standalone SDK through
// compatibility wrappers and adapters.
//
// See:
//
//   - docs/2026-05-13-standalone-rag-sdk-design.md
//   - docs/2026-05-13-rag-sdk-migration-status.md
package rag
