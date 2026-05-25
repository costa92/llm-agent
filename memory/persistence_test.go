package memory

import (
	"context"
	"errors"
	"testing"
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
