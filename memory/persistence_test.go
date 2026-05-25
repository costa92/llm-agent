package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/costa92/llm-agent/llm"
)

// --- round-trip ----------------------------------------------------------

func TestSnapshot_RoundTrip_WorkingMemory(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	_, _ = w.Add(ctx, MemoryItem{Content: "alpha", Importance: 0.5})
	_, _ = w.Add(ctx, MemoryItem{Content: "beta", Importance: 0.5})

	snap, err := w.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if snap.Version != SnapshotVersion {
		t.Errorf("Version = %d, want %d", snap.Version, SnapshotVersion)
	}
	if snap.Kind != KindWorking {
		t.Errorf("Kind = %q, want working", snap.Kind)
	}
	if len(snap.Items) != 2 {
		t.Fatalf("Items count = %d, want 2", len(snap.Items))
	}
	// vectors should be present
	for _, si := range snap.Items {
		if len(si.Vector) == 0 {
			t.Errorf("Vector missing for %q", si.Item.ID)
		}
	}
	// items sorted by CreatedAt ASC then ID ASC: alpha first
	if snap.Items[0].Item.Content != "alpha" {
		t.Errorf("first item Content = %q, want alpha (CreatedAt ASC ordering)", snap.Items[0].Item.Content)
	}
}

func TestSnapshot_RoundTrip_Episodic(t *testing.T) {
	m := newEpisodic(t)
	ctx := context.Background()
	_, _ = m.Add(ctx, MemoryItem{Content: "ev1", Importance: 0.5})
	_, _ = m.Add(ctx, MemoryItem{Content: "ev2", Importance: 0.5})

	snap, err := m.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if snap.Kind != KindEpisodic {
		t.Errorf("Kind = %q, want episodic", snap.Kind)
	}
	if len(snap.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(snap.Items))
	}
}

func TestSnapshot_RoundTrip_Semantic(t *testing.T) {
	m := newSemantic(t)
	ctx := context.Background()
	_, _ = m.Add(ctx, MemoryItem{Content: "fact1", Tags: []string{"t1"}, Importance: 0.5})

	snap, err := m.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if snap.Kind != KindSemantic {
		t.Errorf("Kind = %q, want semantic", snap.Kind)
	}
	if len(snap.Items) != 1 {
		t.Errorf("Items count = %d, want 1", len(snap.Items))
	}
}

// --- version / kind guards ------------------------------------------------

func TestImport_VersionMismatchRejected(t *testing.T) {
	w := newWorking(t)
	bad := Snapshot{Version: 99, Kind: KindWorking}
	_, err := w.Import(context.Background(), bad, ImportReplace)
	if !errors.Is(err, ErrSnapshotVersionMismatch) {
		t.Errorf("err = %v, want ErrSnapshotVersionMismatch", err)
	}
}

func TestImport_KindMismatchRejected(t *testing.T) {
	w := newWorking(t)
	bad := Snapshot{Version: SnapshotVersion, Kind: KindEpisodic}
	_, err := w.Import(context.Background(), bad, ImportReplace)
	if !errors.Is(err, ErrSnapshotKindMismatch) {
		t.Errorf("err = %v, want ErrSnapshotKindMismatch", err)
	}
}

// --- import modes -------------------------------------------------------

// buildSnap creates a Snapshot from src by Exporting it. Helper used to
// keep the mode-dispatch tests focused on import semantics rather than
// raw struct literal noise.
func buildSnap(t *testing.T, kind Kind, items []MemoryItem) Snapshot {
	t.Helper()
	// Build a temp memory of the right kind, Add each item, then Export.
	switch kind {
	case KindWorking:
		w := newWorking(t)
		ctx := context.Background()
		for _, it := range items {
			if _, err := w.Add(ctx, it); err != nil {
				t.Fatalf("buildSnap Add: %v", err)
			}
		}
		snap, _ := w.Export(ctx)
		return snap
	case KindEpisodic:
		m := newEpisodic(t)
		ctx := context.Background()
		for _, it := range items {
			if _, err := m.Add(ctx, it); err != nil {
				t.Fatalf("buildSnap Add: %v", err)
			}
		}
		snap, _ := m.Export(ctx)
		return snap
	case KindSemantic:
		m := newSemantic(t)
		ctx := context.Background()
		for _, it := range items {
			if _, err := m.Add(ctx, it); err != nil {
				t.Fatalf("buildSnap Add: %v", err)
			}
		}
		snap, _ := m.Export(ctx)
		return snap
	}
	t.Fatalf("buildSnap: unknown kind %q", kind)
	return Snapshot{}
}

func TestImport_ReplaceWipesAndLoads(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	_, _ = w.Add(ctx, MemoryItem{Content: "a"})
	_, _ = w.Add(ctx, MemoryItem{Content: "b"})

	snap := buildSnap(t, KindWorking, []MemoryItem{
		{Content: "c"}, {Content: "d"},
	})
	rpt, err := w.Import(ctx, snap, ImportReplace)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rpt.Loaded != 2 {
		t.Errorf("Loaded = %d, want 2", rpt.Loaded)
	}
	if w.Stats().Count != 2 {
		t.Errorf("after replace, Count = %d, want 2 (old items wiped)", w.Stats().Count)
	}
}

func TestImport_MergeSkipsExisting(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	// Add an item with a known ID.
	if _, err := w.Add(ctx, MemoryItem{ID: "fixed-a", Content: "original"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Build a snapshot that re-uses ID "fixed-a" plus a new item.
	snap := buildSnap(t, KindWorking, []MemoryItem{
		{ID: "fixed-a", Content: "replacement"},
		{ID: "fixed-b", Content: "newcomer"},
	})
	rpt, err := w.Import(ctx, snap, ImportMerge)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rpt.Loaded != 1 {
		t.Errorf("Loaded = %d, want 1 (only b)", rpt.Loaded)
	}
	if rpt.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (a already exists)", rpt.Skipped)
	}
	got, _ := w.Get(ctx, "fixed-a")
	if got.Content != "original" {
		t.Errorf("Content = %q, want 'original' (merge must not overwrite)", got.Content)
	}
}

func TestImport_UpsertOverwrites(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	if _, err := w.Add(ctx, MemoryItem{ID: "fixed-a", Content: "original"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	snap := buildSnap(t, KindWorking, []MemoryItem{
		{ID: "fixed-a", Content: "replacement"},
		{ID: "fixed-b", Content: "newcomer"},
	})
	rpt, err := w.Import(ctx, snap, ImportUpsert)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rpt.Loaded != 1 {
		t.Errorf("Loaded = %d, want 1 (only b is new)", rpt.Loaded)
	}
	if rpt.Replaced != 1 {
		t.Errorf("Replaced = %d, want 1 (a overwritten)", rpt.Replaced)
	}
	got, _ := w.Get(ctx, "fixed-a")
	if got.Content != "replacement" {
		t.Errorf("Content = %q, want 'replacement' (upsert overwrites)", got.Content)
	}
}

func TestImport_RestoresVectors(t *testing.T) {
	// After Export → Import on a fresh memory, Search with the same query
	// should return the same top result. If vectors were dropped on
	// import the receiver would have to re-embed (which we deliberately
	// do not do; we reuse the inlined vectors).
	src := newSemantic(t)
	ctx := context.Background()
	_, _ = src.Add(ctx, MemoryItem{Content: "alpha bravo charlie", Importance: 0.5})
	_, _ = src.Add(ctx, MemoryItem{Content: "delta echo foxtrot", Importance: 0.5})

	srcRes, _ := src.Search(ctx, "alpha bravo", 1)
	if len(srcRes) != 1 {
		t.Fatalf("src search returned %d results", len(srcRes))
	}

	snap, _ := src.Export(ctx)
	dst, err := NewSemantic(llm.NewScriptedLLM(llm.WithEmbedDimensions(64)), SemanticOptions{})
	if err != nil {
		t.Fatalf("NewSemantic: %v", err)
	}
	if _, err := dst.Import(ctx, snap, ImportReplace); err != nil {
		t.Fatalf("Import: %v", err)
	}
	dstRes, _ := dst.Search(ctx, "alpha bravo", 1)
	if len(dstRes) != 1 {
		t.Fatalf("dst search returned %d results", len(dstRes))
	}
	if dstRes[0].Item.ID != srcRes[0].Item.ID {
		t.Errorf("top ID = %q, want %q (vector restore broken)", dstRes[0].Item.ID, srcRes[0].Item.ID)
	}
}
