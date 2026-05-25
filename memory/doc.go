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
//   Working   = (vec×0.7 + keyword×0.3) × time_decay × imp × saved
//   Episodic  = (vec×0.8 + recency×0.2) ×              imp × saved
//   Semantic  = (vec×0.7 + tag_overlap×0.3) ×          imp × saved
//
// where
//   imp   = 0.8 + Importance × 0.4 (importanceMultiplier)
//   saved = SavedBoost when IsPinned(it) || GetSource(it)==SourceUserSaved,
//           else 1.0 (savedBoostMultiplier). Non-positive SavedBoost is
//           treated as 1.0 (no-op) so the zero value preserves pre-v0.7
//           scoring behavior.
//
// Vector scoring uses llm.Embedder. The bundled tests use ScriptedLLM's
// deterministic embedding capability; production embedders drop in via
// the same interface.
//
// # ChatGPT-style profile metadata
//
// MemoryItem carries an existing Metadata map[string]any. The
// "profile" helpers in profile.go layer a few well-known keys (under a
// reserved "_"-prefixed namespace) on top of that map WITHOUT changing
// the MemoryItem struct or the Memory interface:
//
//   - Source     — user_saved | agent_inferred | system (via GetSource / SetSource)
//   - Category   — user | feedback | project | reference (via GetCategory / SetCategory)
//   - Pinned     — survives Forget and (with SavedBoost) ranks higher in Search
//                  (via IsPinned / SetPinned)
//   - Disabled   — hidden from Search results but still stored; can be re-enabled
//                  (via IsDisabled / SetDisabled)
//
// Constructors NewSavedMemory and NewInferredMemory bundle the
// ChatGPT-style defaults (high importance + pinned + user_saved for
// "Remember that ..." flows; agent_inferred with confidence-as-
// importance for autonomous captures).
//
// SavedBoost on WorkingOptions / EpisodicOptions / SemanticOptions
// turns the pinned/user_saved flag into a multiplicative score boost
// at Search time. The zero value is a strict no-op so existing
// callers see no scoring change.
//
// # Portability
//
// memory inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package memory
