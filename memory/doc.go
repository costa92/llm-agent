// Package memory is a thin adapter over the leaf memory contract
// github.com/costa92/llm-agent-contract/memory.
//
// It does two things:
//
//   - Re-exports the contract's data + interface surface (MemoryItem,
//     SearchResult, Stats, Kind, Scope, Source, Category, ListFilter,
//     ListPage, Snapshot, ImportMode, the Memory / Lister / Exporter /
//     Importer / SnapshotStore / Manager interfaces, the sentinel errors,
//     and the eight metadata helpers) under bare names, so in-module
//     callers (e.g. the context package) keep compiling unchanged. See
//     aliases.go.
//
//   - Exposes AsTool, which wraps any contract memory.Manager as an
//     agents.Tool. This is the one coupling that intentionally stays in
//     this module: memory → agents (the framework root package).
//
// The memory ENGINE — the concrete WorkingMemory / EpisodicMemory /
// SemanticMemory types, the *Manager / ScopedManager implementations,
// scoring, persistence (FilesystemStore), and the constructors
// (NewManager / NewWorking / ...) — has moved out of this module into
// github.com/costa92/llm-agent-memory/v2. Construct a Manager there and
// pass it to AsTool here. Construction is a clean break (the constructors
// are deliberately NOT re-exported, to avoid reintroducing the engine
// dependency); types remain alias-compatible.
//
// # AsTool
//
// AsTool exposes the full Manager surface as a single agents.Tool with an
// `action` discriminator. The action set is:
//
//	add / search / get / update / remove   — basic CRUD
//	consolidate / forget / stats           — lifecycle / introspection
//	list                                   — enumerate items (per-kind or fan-out)
//	pin / unpin                            — toggle the _pinned metadata flag
//	disable / enable                       — toggle the _disabled metadata flag
//	export / import                        — Snapshot persistence via SnapshotStore
//
// The export action wraps Manager.ExportAll; pass snapshot_key to persist
// via the Manager's configured SnapshotStore (omit it to return in-memory
// snapshots only). The import action wraps Manager.ImportAll and defaults
// to ImportMerge (the safest mode) if import_mode is omitted; pass
// "replace" or "upsert" explicitly to change semantics.
//
// The list action's filter is the flat JSON wire form of ListFilter:
// scope is flattened into user_scope / project_scope / session_scope, plus
// source, category, tags (any-of), pinned_only, include_disabled, and
// min_importance. Order is deterministic and pagination is cursor-based:
// callers pass back NextCursor verbatim; an empty NextCursor signals
// end-of-stream.
//
// # Portability
//
// memory inherits the agents / contract portability contract — no
// internal/*, no project pkg/*, no business vocabulary. Its only
// non-stdlib dependency is the stdlib-only contract leaf.
package memory
