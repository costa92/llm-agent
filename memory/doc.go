// Package memory implements 3 in-process Memory types + a Manager
// + an agents.Tool adapter:
//
//   - WorkingMemory  — capacity-bounded, time-decay, "what's recent + active"
//   - EpisodicMemory — long-term, recency-weighted, "what happened over time"
//   - SemanticMemory — K-V with tag-aware ranking, "facts I know"
//
// All three satisfy the Memory interface. Manager coordinates Add /
// Search across kinds and adds Consolidate (working→episodic by
// importance) + Forget (3 strategies).
//
// AsTool wraps a Manager so any pkg/llm/agents Agent can call memory
// operations through the standard Tool surface.
//
// # Score formulas (per spec §6.3)
//
//   Working   = (vec×0.7 + keyword×0.3) × time_decay × (0.8 + importance×0.4)
//   Episodic  = (vec×0.8 + recency×0.2) ×              (0.8 + importance×0.4)
//   Semantic  = (vec×0.7 + tag_overlap×0.3) ×          (0.8 + importance×0.4)
//
// Vector scoring uses pkg/llm/agents/rag.Embedder. Phase 2 ships
// HashEmbedder (FNV bucket, deterministic, low-quality semantic). Real
// embedders land in Phase 3 — drop them in via the same interface.
//
// # Portability
//
// memory inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package memory
