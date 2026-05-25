package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// SnapshotVersion is the current Snapshot schema version. Older versions on
// import may need migration; Importers reject unknown versions.
const SnapshotVersion = 1

// Snapshot is a portable dump of one Memory's contents. JSON-serializable
// (encoding/json). Vectors are inlined; on Restore they are reused as-is so
// the receiving Memory does not need to re-embed existing content.
type Snapshot struct {
	Version int            `json:"version"`
	Kind    Kind           `json:"kind"`
	Items   []SnapshotItem `json:"items"`
}

// SnapshotItem pairs a MemoryItem with its cached embedding vector.
type SnapshotItem struct {
	Item   MemoryItem `json:"item"`
	Vector []float32  `json:"vector"`
}

// ImportMode controls how Importer merges incoming items with existing ones.
type ImportMode string

const (
	// ImportReplace wipes the target memory then loads the snapshot.
	ImportReplace ImportMode = "replace"
	// ImportMerge adds unseen items; skips items whose ID already exists.
	ImportMerge ImportMode = "merge"
	// ImportUpsert adds unseen items; overwrites items whose ID already exists.
	ImportUpsert ImportMode = "upsert"
)

// ImportReport summarizes the outcome of an Import call.
type ImportReport struct {
	Loaded   int     `json:"loaded"`
	Skipped  int     `json:"skipped"`
	Replaced int     `json:"replaced"`
	Errors   []error `json:"-"` // not serialized; surfaces via Error()
}

// Exporter dumps a Memory to a Snapshot.
type Exporter interface {
	Export(ctx context.Context) (Snapshot, error)
}

// Importer restores from a Snapshot, replacing or merging existing content per
// mode. The Snapshot's Kind must match the receiving Memory's Type.
type Importer interface {
	Import(ctx context.Context, snap Snapshot, mode ImportMode) (ImportReport, error)
}

// --- sentinel errors ---

// ErrSnapshotVersionMismatch is returned by Import when snap.Version is unknown.
var ErrSnapshotVersionMismatch = errors.New("memory: snapshot version mismatch")

// ErrSnapshotKindMismatch is returned by Import when snap.Kind != receiving Memory.Type().
var ErrSnapshotKindMismatch = errors.New("memory: snapshot kind mismatch")

// ErrSnapshotStoreNotConfigured is returned by Manager.ExportAll / ImportAll
// when ManagerOptions.SnapshotStore was not set.
var ErrSnapshotStoreNotConfigured = errors.New("memory: SnapshotStore not configured on Manager")

// --- shared store-level Export/Import ---

// exportFromStore captures items + vectors from a scoredStore into a
// JSON-serializable Snapshot. Items are emitted in (CreatedAt ASC, ID ASC)
// order so the JSON byte output is stable across runs.
func exportFromStore(s *scoredStore, kind Kind) Snapshot {
	items, vecs := s.snapshot()
	snap := Snapshot{Version: SnapshotVersion, Kind: kind, Items: make([]SnapshotItem, 0, len(items))}
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		ai, aj := items[ids[i]], items[ids[j]]
		if !ai.CreatedAt.Equal(aj.CreatedAt) {
			return ai.CreatedAt.Before(aj.CreatedAt)
		}
		return ids[i] < ids[j]
	})
	for _, id := range ids {
		snap.Items = append(snap.Items, SnapshotItem{Item: items[id], Vector: vecs[id]})
	}
	return snap
}

// importIntoStore writes a Snapshot into a scoredStore per the given mode.
// Caller must validate snap.Kind matches the receiving Memory.Type() before
// calling — this helper only enforces version, not kind.
func importIntoStore(s *scoredStore, snap Snapshot, mode ImportMode) (ImportReport, error) {
	if snap.Version != SnapshotVersion {
		return ImportReport{}, fmt.Errorf("%w: got %d, want %d", ErrSnapshotVersionMismatch, snap.Version, SnapshotVersion)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var rpt ImportReport
	if mode == ImportReplace {
		s.items = make(map[string]MemoryItem, len(snap.Items))
		s.vectors = make(map[string][]float32, len(snap.Items))
	}
	for _, si := range snap.Items {
		id := si.Item.ID
		if id == "" {
			rpt.Errors = append(rpt.Errors, fmt.Errorf("memory: snapshot item with empty ID, skipped"))
			continue
		}
		_, exists := s.items[id]
		switch mode {
		case ImportMerge:
			if exists {
				rpt.Skipped++
				continue
			}
			s.items[id] = si.Item
			if si.Vector != nil {
				cp := make([]float32, len(si.Vector))
				copy(cp, si.Vector)
				s.vectors[id] = cp
			}
			rpt.Loaded++
		case ImportUpsert:
			s.items[id] = si.Item
			if si.Vector != nil {
				cp := make([]float32, len(si.Vector))
				copy(cp, si.Vector)
				s.vectors[id] = cp
			} else {
				delete(s.vectors, id)
			}
			if exists {
				rpt.Replaced++
			} else {
				rpt.Loaded++
			}
		case ImportReplace:
			s.items[id] = si.Item
			if si.Vector != nil {
				cp := make([]float32, len(si.Vector))
				copy(cp, si.Vector)
				s.vectors[id] = cp
			}
			rpt.Loaded++
		default:
			return rpt, fmt.Errorf("memory: unknown import mode %q", mode)
		}
	}
	return rpt, nil
}
