package memory

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(ManagerOptions{
		Working:  newWorking(t),
		Episodic: newEpisodic(t),
		Semantic: newSemantic(t),
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

func TestManager_RejectsAllNil(t *testing.T) {
	if _, err := NewManager(ManagerOptions{}); !errors.Is(err, ErrNoMemories) {
		t.Errorf("err = %v, want ErrNoMemories", err)
	}
}

func TestManager_RoutesByKind(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	id, err := mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "an event", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := mgr.Get(ctx, KindEpisodic, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "an event" {
		t.Errorf("Content = %q", got.Content)
	}
}

func TestManager_DisabledKindErrors(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{Working: newWorking(t)})
	_, err := mgr.Add(context.Background(), KindEpisodic, MemoryItem{Content: "x"})
	if !errors.Is(err, ErrKindDisabled) {
		t.Errorf("err = %v, want ErrKindDisabled", err)
	}
}

func TestManager_SearchAllFansOut(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "go modules", Importance: 0.5})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "go modules history", Importance: 0.5})
	_, _ = mgr.Add(ctx, KindSemantic, MemoryItem{Content: "go modules guide", Tags: []string{"go"}, Importance: 0.5})

	out, err := mgr.SearchAll(ctx, "go modules", 5)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		if len(out[kind]) == 0 {
			t.Errorf("kind %s returned no results", kind)
		}
	}
}

func TestManager_SearchAllSkipsDisabledKinds(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{Working: newWorking(t)}) // only working
	out, err := mgr.SearchAll(context.Background(), "anything", 5)
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if _, has := out[KindEpisodic]; has {
		t.Error("episodic should be absent (disabled)")
	}
}

func TestManager_StatsAll(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{Working: newWorking(t)})
	stats := mgr.StatsAll()
	if _, has := stats[KindWorking]; !has {
		t.Error("StatsAll missing working")
	}
	if _, has := stats[KindEpisodic]; has {
		t.Error("StatsAll should not have episodic (disabled)")
	}
}

// --- consolidate ----------------------------------------------------------

func TestManager_ConsolidateCopiesHighImportanceToEpisodic(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "trivial", Importance: 0.2})
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "important fact", Importance: 0.9})
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "another important", Importance: 0.85})

	count, err := mgr.Consolidate(ctx, ConsolidateOptions{Threshold: 0.7})
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (importance ≥ 0.7)", count)
	}
	// Episodic should now have those 2; working is unchanged.
	if mgr.StatsAll()[KindEpisodic].Count != 2 {
		t.Errorf("episodic count = %d, want 2", mgr.StatsAll()[KindEpisodic].Count)
	}
	if mgr.StatsAll()[KindWorking].Count != 3 {
		t.Errorf("working count = %d, want 3 (source not deleted)", mgr.StatsAll()[KindWorking].Count)
	}
}

func TestManager_ConsolidateRequiresBothMemories(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{Working: newWorking(t)}) // no episodic
	_, err := mgr.Consolidate(context.Background(), ConsolidateOptions{Threshold: 0.5})
	if !errors.Is(err, ErrConsolidateUnavailable) {
		t.Errorf("err = %v, want ErrConsolidateUnavailable", err)
	}
}

// --- forget ---------------------------------------------------------------

func TestManager_ForgetByImportance(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "low", Importance: 0.1})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "mid", Importance: 0.5})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "high", Importance: 0.9})

	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByImportance, Threshold: 0.4})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (only 'low' < 0.4)", count)
	}
	if mgr.StatsAll()[KindEpisodic].Count != 2 {
		t.Errorf("after forget, count = %d, want 2", mgr.StatsAll()[KindEpisodic].Count)
	}
}

func TestManager_ForgetByAge(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	id1, _ := mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "old", Importance: 0.5})
	_ = mgr.Update(ctx, KindEpisodic, id1, func(it *MemoryItem) {
		it.CreatedAt = time.Now().Add(-48 * time.Hour)
	})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "fresh", Importance: 0.5})

	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByAge, MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (only old > 24h)", count)
	}
}

func TestManager_ForgetByCapacity(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "low", Importance: 0.1})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "mid", Importance: 0.5})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "high", Importance: 0.9})

	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByCapacity, Keep: 1})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (kept 1)", count)
	}
	// The kept one should be the highest-importance.
	res, _ := mgr.Search(ctx, KindEpisodic, "high", 5)
	if len(res) != 1 || res[0].Item.Content != "high" {
		t.Errorf("kept item should be 'high'; got %v", res)
	}
}

func TestManager_ForgetByCapacity_KeepZeroNoop(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "x", Importance: 0.5})
	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByCapacity, Keep: 0})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 (Keep=0 is no-op)", count)
	}
}

func TestManager_ForgetUnknownStrategy(t *testing.T) {
	mgr := newManager(t)
	_, err := mgr.Forget(context.Background(), KindEpisodic, ForgetOptions{Strategy: "garbage"})
	if err == nil {
		t.Error("expected error for unknown strategy")
	}
}

// --- ExportAll / ImportAll ------------------------------------------------

func TestManager_ExportAll_NoStorePersistFails(t *testing.T) {
	mgr := newManager(t)
	_, err := mgr.ExportAll(context.Background(), "some-key")
	if !errors.Is(err, ErrSnapshotStoreNotConfigured) {
		t.Errorf("err = %v, want ErrSnapshotStoreNotConfigured", err)
	}
}

func TestManager_ExportAll_WithStorePersists(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	mgr, err := NewManager(ManagerOptions{
		Working:       newWorking(t),
		Episodic:      newEpisodic(t),
		Semantic:      newSemantic(t),
		SnapshotStore: fs,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "w"})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "e"})
	_, _ = mgr.Add(ctx, KindSemantic, MemoryItem{Content: "s"})

	out, err := mgr.ExportAll(ctx, "k1")
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("ExportAll returned %d kinds, want 3", len(out))
	}
	// Verify each kind landed on disk.
	for _, kind := range []Kind{KindWorking, KindEpisodic, KindSemantic} {
		snap, err := fs.LoadKind(ctx, "k1", kind)
		if err != nil {
			t.Errorf("LoadKind %s: %v", kind, err)
			continue
		}
		if snap.Kind != kind {
			t.Errorf("loaded Kind = %q, want %q", snap.Kind, kind)
		}
	}
}

func TestManager_ImportAll_Inline(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()
	_, _ = mgr.Add(ctx, KindWorking, MemoryItem{Content: "w"})
	out, err := mgr.ExportAll(ctx, "")
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}

	// Build a fresh manager and import the snaps inline.
	mgr2 := newManager(t)
	rpts, err := mgr2.ImportAll(ctx, out, "", ImportReplace)
	if err != nil {
		t.Fatalf("ImportAll: %v", err)
	}
	if rpts[KindWorking].Loaded != 1 {
		t.Errorf("Working Loaded = %d, want 1", rpts[KindWorking].Loaded)
	}
	if mgr2.StatsAll()[KindWorking].Count != 1 {
		t.Errorf("working count after import = %d, want 1", mgr2.StatsAll()[KindWorking].Count)
	}
}

func TestManager_ImportAll_FromStore(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	src, err := NewManager(ManagerOptions{
		Working:       newWorking(t),
		Episodic:      newEpisodic(t),
		Semantic:      newSemantic(t),
		SnapshotStore: fs,
	})
	if err != nil {
		t.Fatalf("src NewManager: %v", err)
	}
	ctx := context.Background()
	_, _ = src.Add(ctx, KindWorking, MemoryItem{Content: "w"})
	_, _ = src.Add(ctx, KindSemantic, MemoryItem{Content: "s"})
	if _, err := src.ExportAll(ctx, "k1"); err != nil {
		t.Fatalf("ExportAll: %v", err)
	}

	dst, err := NewManager(ManagerOptions{
		Working:       newWorking(t),
		Episodic:      newEpisodic(t),
		Semantic:      newSemantic(t),
		SnapshotStore: fs,
	})
	if err != nil {
		t.Fatalf("dst NewManager: %v", err)
	}
	rpts, err := dst.ImportAll(ctx, nil, "k1", ImportReplace)
	if err != nil {
		t.Fatalf("ImportAll: %v", err)
	}
	if rpts[KindWorking].Loaded != 1 {
		t.Errorf("Working Loaded = %d, want 1", rpts[KindWorking].Loaded)
	}
	if rpts[KindSemantic].Loaded != 1 {
		t.Errorf("Semantic Loaded = %d, want 1", rpts[KindSemantic].Loaded)
	}
	// Episodic had no items; it may be absent (no file) or present with 0
	// items — both are acceptable behaviors. Just assert dst received the
	// 2 we wrote.
	if dst.StatsAll()[KindWorking].Count != 1 {
		t.Errorf("dst working count = %d, want 1", dst.StatsAll()[KindWorking].Count)
	}
	if dst.StatsAll()[KindSemantic].Count != 1 {
		t.Errorf("dst semantic count = %d, want 1", dst.StatsAll()[KindSemantic].Count)
	}
}

func TestManager_ImportAll_PersistKeyWithoutStoreFails(t *testing.T) {
	mgr := newManager(t)
	_, err := mgr.ImportAll(context.Background(), nil, "k1", ImportReplace)
	if !errors.Is(err, ErrSnapshotStoreNotConfigured) {
		t.Errorf("err = %v, want ErrSnapshotStoreNotConfigured", err)
	}
}
