package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// --- SnapshotStore + FilesystemStore -------------------------------------

// SnapshotStore is the pluggable persistence backend. Implementations are
// keyed (the key identifies a logical snapshot, often a session/user ID).
// Stdlib-only impl in core is FilesystemStore; downstream repos can inject
// SQLite/Postgres/S3/Redis stores without core taking a dep.
type SnapshotStore interface {
	Save(ctx context.Context, key string, snap Snapshot) error
	Load(ctx context.Context, key string) (Snapshot, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]string, error)
}

// FilesystemStore writes one JSON file per (key, kind) into dir. File names
// are sanitized: <key>__<kind>.json where any character outside
// [a-zA-Z0-9_-] in the key is replaced with '_'. The on-disk format is
// the default encoding/json output (one JSON document per file).
//
// FilesystemStore is goroutine-safe; concurrent Save on different keys are
// safe; same-key concurrent Save uses os.Rename for atomicity (only the
// last write wins, no half-written files are observable to Load).
type FilesystemStore struct {
	dir string
}

// NewFilesystemStore creates the store; dir is created with 0755 if missing.
// Returns the error from os.MkdirAll if creation fails.
func NewFilesystemStore(dir string) (*FilesystemStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("memory: filesystem store mkdir: %w", err)
	}
	return &FilesystemStore{dir: dir}, nil
}

func (fs *FilesystemStore) path(key string, kind Kind) string {
	return filepath.Join(fs.dir, sanitizeKey(key)+"__"+string(kind)+".json")
}

// sanitizeKey replaces every character outside [a-zA-Z0-9_-] with '_'.
// Empty/all-bad keys become "_". This makes path traversal impossible
// regardless of caller input.
func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

// Save writes snap to <dir>/<sanitized_key>__<kind>.json atomically (tmp +
// rename). Returns an error if snap.Kind is empty (kind is part of the
// filename and we refuse to guess).
func (fs *FilesystemStore) Save(_ context.Context, key string, snap Snapshot) error {
	if snap.Kind == "" {
		return errors.New("memory: snapshot kind is required for Save")
	}
	p := fs.path(key, snap.Kind)
	// write to tmp in the SAME directory, then rename for atomicity
	// (rename is atomic when src and dst are on the same filesystem).
	tmp, err := os.CreateTemp(fs.dir, "snap-*.tmp")
	if err != nil {
		return fmt.Errorf("memory: filesystem store create temp: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op if rename succeeded
	enc := json.NewEncoder(tmp)
	if err := enc.Encode(snap); err != nil {
		tmp.Close()
		return fmt.Errorf("memory: filesystem store encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("memory: filesystem store close: %w", err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return fmt.Errorf("memory: filesystem store rename: %w", err)
	}
	return nil
}

// Load returns the first snapshot found for key across the three kinds (in
// working → episodic → semantic order). Callers wanting a specific kind
// should call LoadKind. Returns an error wrapping os.ErrNotExist when no
// snapshot exists for any kind.
func (fs *FilesystemStore) Load(ctx context.Context, key string) (Snapshot, error) {
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		snap, err := fs.LoadKind(ctx, key, kind)
		if err == nil {
			return snap, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return Snapshot{}, err
		}
	}
	return Snapshot{}, fmt.Errorf("memory: filesystem store: no snapshot for key %q: %w", key, os.ErrNotExist)
}

// LoadKind loads a snapshot for a specific (key, kind) tuple. Returns an
// error wrapping os.ErrNotExist if the file does not exist.
func (fs *FilesystemStore) LoadKind(_ context.Context, key string, kind Kind) (Snapshot, error) {
	p := fs.path(key, kind)
	f, err := os.Open(p)
	if err != nil {
		return Snapshot{}, err
	}
	defer f.Close()
	var snap Snapshot
	dec := json.NewDecoder(f)
	if err := dec.Decode(&snap); err != nil {
		if errors.Is(err, io.EOF) {
			return Snapshot{}, fmt.Errorf("memory: filesystem store: empty snapshot file %q", p)
		}
		return Snapshot{}, fmt.Errorf("memory: filesystem store decode: %w", err)
	}
	return snap, nil
}

// Delete removes the snapshot files for all three kinds at key. Missing
// files are NOT an error; only the first I/O error is returned.
func (fs *FilesystemStore) Delete(_ context.Context, key string) error {
	var firstErr error
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		p := fs.path(key, kind)
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// List returns the sorted set of unique keys discovered under dir. Each
// key may correspond to one or more kind files; the key appears once in
// the returned slice.
func (fs *FilesystemStore) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("memory: filesystem store list: %w", err)
	}
	keySet := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		trimmed := strings.TrimSuffix(name, ".json")
		idx := strings.LastIndex(trimmed, "__")
		if idx <= 0 {
			continue
		}
		key := trimmed[:idx]
		keySet[key] = struct{}{}
	}
	out := make([]string, 0, len(keySet))
	for k := range keySet {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}
