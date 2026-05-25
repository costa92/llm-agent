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
// # Scope and ScopedManager
//
// Scope partitions memory by three axes: User, Project, Session.
// Each axis is a free-form string; an empty axis is a wildcard at
// query time. The Scope{} zero value matches every item, which is
// the pre-v0.7 default behavior — existing callers that never call
// WithScope see no observable change.
//
// Scope propagates through context.Context:
//
//	ctx := memory.WithScope(parent, memory.Scope{User: "alice"})
//	sm.Add(ctx, memory.KindWorking, item)        // stamps {user:"alice"} into Metadata["_scope"]
//	sm.Search(ctx, memory.KindWorking, "q", 10)  // returns only alice's items
//
// ScopedManager is a *Manager decorator. NewScopedManager(inner)
// returns a *ScopedManager that mirrors the 9 public *Manager methods
// (Add / Get / Update / Remove / Search / SearchAll / Consolidate /
// Forget / StatsAll). The first six honor the ctx scope; the last
// three are explicit v0.7 limitations:
//
//   - Consolidate / Forget / StatsAll DO NOT honor scope; they
//     operate on the entire inner Manager. This is deliberate — they
//     bypass the Memory abstraction to access the underlying
//     scoredStore directly. Future work may add scope-aware variants.
//
// On mismatched scope, Get / Update / Remove return ErrNotFound
// rather than a "wrong scope" error so callers cannot probe for
// cross-scope IDs.
//
// # Listing and pagination
//
// The Lister interface is an OPTIONAL capability the three concrete
// Memory types implement. It is intentionally NOT embedded in the
// Memory interface so foreign Memory implementations are unaffected:
//
//	type Lister interface {
//	    List(ctx, filter ListFilter, pageSize int, cursor string) (ListPage, error)
//	}
//
// ListFilter narrows the result set across Scope, Source, Category,
// Tags (any-of), PinnedOnly, MinImportance, and IncludeDisabled.
// Order is deterministic: (CreatedAt DESC, ID ASC). Cursors are
// opaque base64(JSON{after_created_at, after_id}) blobs — callers
// pass back NextCursor verbatim to fetch the next page. End-of-stream
// is signaled by an empty NextCursor.
//
// Manager.ListAll fans the operation across all active kinds; the
// cursors argument is per-kind (map[Kind]string). ScopedManager.ListAll
// applies the ctx scope on top of the filter — a non-zero ctx scope
// OVERRIDES filter.Scope (the ctx scope is the trust boundary).
//
// # Sanitizer (privacy hook)
//
// WithSanitizer(inner, chain...) returns a Memory decorator that runs
// the chain on every Add. Each Sanitizer returns:
//
//	(newItem, true,  nil)    keep (possibly redacted)
//	(_,       false, nil)    silently reject → Add returns
//	                         ErrRejectedByPolicy
//	(_,       _,     err)    propagate err to the Add caller
//
// The chain short-circuits at the first reject. Read paths
// (Get / Search / Update / Remove / Stats) bypass the chain so the
// audit trail and lookup semantics stay independent of policy
// mutations. SanitizerFunc adapts a plain function.
//
// LIMITATION (v0.7): WithSanitizer returns a Memory interface value
// — it does NOT satisfy the concrete *WorkingMemory / *EpisodicMemory
// / *SemanticMemory types ManagerOptions expects. Callers wanting
// Sanitizer + Manager fan-out must compose at a higher layer (e.g.
// invoke the sanitizer chain themselves before calling Manager.Add)
// or apply the sanitizer at the Tool surface. Direct embedding in
// ManagerOptions is deferred to a future release.
//
// # Persistence
//
// Each concrete Memory type implements two optional capability
// interfaces:
//
//	type Exporter interface {
//	    Export(ctx) (Snapshot, error)
//	}
//	type Importer interface {
//	    Import(ctx, snap Snapshot, mode ImportMode) (ImportReport, error)
//	}
//
// Snapshot is a JSON-serializable dump (Version + Kind + []SnapshotItem)
// where each SnapshotItem inlines the MemoryItem AND its cached
// embedding vector. On Import the vectors are reused as-is so the
// receiving Memory does NOT need to re-embed any restored content.
//
// ImportMode controls merge semantics:
//
//	ImportReplace — wipe target then load every snapshot item.
//	ImportMerge   — load only items whose ID is not already present
//	                (Skipped counter ticks for collisions).
//	ImportUpsert  — load everything; collisions overwrite
//	                (Replaced counter ticks).
//
// ImportReport returns Loaded / Skipped / Replaced counts so callers
// can observe what changed. Version mismatch returns
// ErrSnapshotVersionMismatch; Kind mismatch (e.g. importing a
// KindEpisodic snapshot into a *WorkingMemory) returns
// ErrSnapshotKindMismatch.
//
// SnapshotStore is the pluggable persistence backend:
//
//	type SnapshotStore interface {
//	    Save(ctx, key string, snap Snapshot) error
//	    Load(ctx, key string) (Snapshot, error)
//	    Delete(ctx, key string) error
//	    List(ctx) ([]string, error)
//	}
//
// FilesystemStore is the stdlib-only default. NewFilesystemStore(dir)
// writes one JSON file per (key, kind) tuple: <key>__<kind>.json.
// Keys are sanitized (every character outside [a-zA-Z0-9_-] is
// replaced with '_') so path traversal is impossible regardless of
// caller input. Saves are atomic (os.CreateTemp in the same dir +
// os.Rename). FilesystemStore.LoadKind(ctx, key, kind) is the typed
// variant used by Manager.ImportAll.
//
// Downstream repos (e.g. llm-agent-otel, llm-agent-providers) can
// inject SQLite/Postgres/S3/Redis stores by implementing
// SnapshotStore. Core stays stdlib-only.
//
// # Restore constructors
//
// RestoreWorking / RestoreEpisodic / RestoreSemantic construct the
// corresponding concrete Memory type AND immediately import a
// snapshot in ImportReplace mode:
//
//	dst, err := memory.RestoreWorking(embedder, snap, memory.WorkingOptions{})
//
// They still require an Embedder (Search and subsequent Add use it),
// but they do NOT re-embed the items already in the snapshot — those
// vectors are restored byte-for-byte. Returns ErrEmbedderRequired on
// nil embedder, ErrSnapshotKindMismatch on wrong kind, and
// ErrSnapshotVersionMismatch on unknown version.
//
// Manager.ExportAll and Manager.ImportAll fan the operation across
// all active kinds. ManagerOptions.SnapshotStore is optional; when
// set, ExportAll(ctx, persistKey) also writes each kind's snapshot to
// the store, and ImportAll(ctx, nil, persistKey, mode) reads each
// kind back. ImportAll with an inline snaps map bypasses the store
// (snaps wins). When persistKey != "" but SnapshotStore is nil, both
// methods return ErrSnapshotStoreNotConfigured.
//
// # Tool actions
//
// AsTool exposes the full Manager surface as a single agents.Tool
// with an `action` discriminator. The v0.7 action set is:
//
//	add / search / get / update / remove   — basic CRUD
//	consolidate / forget / stats           — lifecycle / introspection
//	list                                   — enumerate items (per-kind or fan-out)
//	pin / unpin                            — toggle the _pinned metadata flag
//	disable / enable                       — toggle the _disabled metadata flag
//	export / import                        — Snapshot persistence via SnapshotStore
//
// The export action wraps Manager.ExportAll; pass snapshot_key to
// persist via the configured SnapshotStore (omit it to return
// in-memory snapshots only). The import action wraps
// Manager.ImportAll and defaults to ImportMerge (the safest mode) if
// import_mode is omitted; pass "replace" or "upsert" explicitly to
// change semantics.
//
// Pre-v0.7 callers see no behavior change — the schema is additive
// (new optional fields: filter, page_size, cursor, cursors,
// snapshot_key, import_mode) and the existing action enum entries
// are unchanged.
//
// # Portability
//
// memory inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package memory
