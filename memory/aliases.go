package memory

import (
	contractmem "github.com/costa92/llm-agent-contract/memory"
)

// This file is the back-compat shim that collapses the public memory
// surface onto the leaf contract github.com/costa92/llm-agent-contract/memory.
//
// The memory ENGINE (WorkingMemory / EpisodicMemory / SemanticMemory /
// Manager / ScopedManager / persistence / scoring) has moved out of this
// module into github.com/costa92/llm-agent-memory/v2; the data + interface
// contract lives in the contract leaf. What remains here is purely an
// adapter: the type re-exports below let tool.go and the context package
// keep compiling against bare names, while AsTool (tool.go) wraps any
// contract Manager as an agents.Tool.
//
// Construction (NewManager / NewWorking / ...) is intentionally NOT
// re-exported — those constructors moved to llm-agent-memory/v2 and
// re-exporting them would reintroduce the engine dependency. This is a
// clean break for construction, alias-compatible for types.

// --- data + interface type aliases ---------------------------------------

type Kind = contractmem.Kind
type MemoryItem = contractmem.MemoryItem
type SearchResult = contractmem.SearchResult
type Stats = contractmem.Stats
type Scope = contractmem.Scope
type Source = contractmem.Source
type Category = contractmem.Category
type ListFilter = contractmem.ListFilter
type ListPage = contractmem.ListPage
type ForgetStrategy = contractmem.ForgetStrategy
type ConsolidateOptions = contractmem.ConsolidateOptions
type ForgetOptions = contractmem.ForgetOptions
type Snapshot = contractmem.Snapshot
type SnapshotItem = contractmem.SnapshotItem
type ImportMode = contractmem.ImportMode
type ImportReport = contractmem.ImportReport
type Embedder = contractmem.Embedder

type Memory = contractmem.Memory
type Lister = contractmem.Lister
type Exporter = contractmem.Exporter
type Importer = contractmem.Importer
type SnapshotStore = contractmem.SnapshotStore
type Manager = contractmem.Manager

// --- const re-exports -----------------------------------------------------

const (
	KindWorking  = contractmem.KindWorking
	KindEpisodic = contractmem.KindEpisodic
	KindSemantic = contractmem.KindSemantic

	CategoryUser      = contractmem.CategoryUser
	CategoryFeedback  = contractmem.CategoryFeedback
	CategoryProject   = contractmem.CategoryProject
	CategoryReference = contractmem.CategoryReference

	SnapshotVersion = contractmem.SnapshotVersion

	ImportReplace = contractmem.ImportReplace
	ImportMerge   = contractmem.ImportMerge
	ImportUpsert  = contractmem.ImportUpsert

	ForgetByImportance = contractmem.ForgetByImportance
	ForgetByAge        = contractmem.ForgetByAge
	ForgetByCapacity   = contractmem.ForgetByCapacity

	SourceUserSaved     = contractmem.SourceUserSaved
	SourceAgentInferred = contractmem.SourceAgentInferred
	SourceSystem        = contractmem.SourceSystem
	SourceUnknown       = contractmem.SourceUnknown
)

// --- sentinel re-exports --------------------------------------------------

var (
	ErrNotFound                   = contractmem.ErrNotFound
	ErrEmptyQuery                 = contractmem.ErrEmptyQuery
	ErrEmbedderRequired           = contractmem.ErrEmbedderRequired
	ErrKindDisabled               = contractmem.ErrKindDisabled
	ErrSnapshotVersionMismatch    = contractmem.ErrSnapshotVersionMismatch
	ErrSnapshotKindMismatch       = contractmem.ErrSnapshotKindMismatch
	ErrSnapshotStoreNotConfigured = contractmem.ErrSnapshotStoreNotConfigured
)

// --- metadata helper forwards ---------------------------------------------
//
// The eight metadata helpers forward to the contract impl so in-package
// callers (tool.go's Update closures) keep calling the bare names with
// identical behavior.

func GetSource(it MemoryItem) Source            { return contractmem.GetSource(it) }
func SetSource(it *MemoryItem, src Source)      { contractmem.SetSource(it, src) }
func GetCategory(it MemoryItem) Category        { return contractmem.GetCategory(it) }
func SetCategory(it *MemoryItem, cat Category)  { contractmem.SetCategory(it, cat) }
func IsPinned(it MemoryItem) bool               { return contractmem.IsPinned(it) }
func SetPinned(it *MemoryItem, pinned bool)     { contractmem.SetPinned(it, pinned) }
func IsDisabled(it MemoryItem) bool             { return contractmem.IsDisabled(it) }
func SetDisabled(it *MemoryItem, disabled bool) { contractmem.SetDisabled(it, disabled) }
