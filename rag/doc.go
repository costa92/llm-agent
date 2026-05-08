// Package rag implements Retrieval-Augmented Generation primitives:
//
//   - Embedder         — text→vector seam (HashEmbedder fallback ships zero-deps)
//   - Chunker          — text→chunks seam (CharChunker prefers paragraph breaks)
//   - VectorStore      — vector index seam (InMemoryStore ships, swap in pgvector etc.)
//   - RAGSystem        — façade: AddText (chunk+embed+upsert) / Search / Ask
//   - Advanced retrieval (default off): MQE (multi-query expansion) + HyDE
//   - AsTool           — wrap a RAGSystem as agents.Tool
//
// # Choosing implementations
//
//   - HashEmbedder is deterministic + free but semantically poor (no
//     synonym awareness). Sufficient for unit tests + learning demos.
//     Swap in a real Embedder (Ollama, OpenAI, vLLM) for production.
//   - InMemoryStore is O(N) per Search — fine up to a few thousand
//     chunks. Beyond that, drop in a backend that does ANN.
//
// # Portability
//
// rag inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package rag
