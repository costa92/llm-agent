package memory

import (
	"context"
	"testing"

	"github.com/costa92/llm-agent-contract/llm"
)

// --- Disabled is filtered from Search across all 3 memory types ------------

func TestSearch_SkipsDisabled_Working(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()

	keptID, _ := w.Add(ctx, MemoryItem{Content: "go modules dependency", Importance: 0.5})
	disabledID, _ := w.Add(ctx, MemoryItem{Content: "go modules dependency", Importance: 0.5})

	if err := w.Update(ctx, disabledID, func(it *MemoryItem) { SetDisabled(it, true) }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	results, err := w.Search(ctx, "go modules", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range results {
		if r.Item.ID == disabledID {
			t.Errorf("disabled item leaked into Search results: %v", r.Item.ID)
		}
	}
	found := false
	for _, r := range results {
		if r.Item.ID == keptID {
			found = true
		}
	}
	if !found {
		t.Error("expected non-disabled item to still be returned")
	}
}

func TestSearch_SkipsDisabled_Episodic(t *testing.T) {
	m := newEpisodic(t)
	ctx := context.Background()

	keptID, _ := m.Add(ctx, MemoryItem{Content: "event alpha", Importance: 0.5})
	disabledID, _ := m.Add(ctx, MemoryItem{Content: "event alpha", Importance: 0.5})
	_ = m.Update(ctx, disabledID, func(it *MemoryItem) { SetDisabled(it, true) })

	results, _ := m.Search(ctx, "event alpha", 10)
	for _, r := range results {
		if r.Item.ID == disabledID {
			t.Errorf("disabled episodic item leaked: %v", r.Item.ID)
		}
	}
	found := false
	for _, r := range results {
		if r.Item.ID == keptID {
			found = true
		}
	}
	if !found {
		t.Error("expected non-disabled episodic item to be returned")
	}
}

func TestSearch_SkipsDisabled_Semantic(t *testing.T) {
	m := newSemantic(t)
	ctx := context.Background()

	keptID, _ := m.Add(ctx, MemoryItem{Content: "fact one", Tags: []string{"k"}, Importance: 0.5})
	disabledID, _ := m.Add(ctx, MemoryItem{Content: "fact one", Tags: []string{"k"}, Importance: 0.5})
	_ = m.Update(ctx, disabledID, func(it *MemoryItem) { SetDisabled(it, true) })

	results, _ := m.Search(ctx, "fact one", 10)
	for _, r := range results {
		if r.Item.ID == disabledID {
			t.Errorf("disabled semantic item leaked: %v", r.Item.ID)
		}
	}
	found := false
	for _, r := range results {
		if r.Item.ID == keptID {
			found = true
		}
	}
	if !found {
		t.Error("expected non-disabled semantic item to be returned")
	}
}

// --- Forget skips pinned items ---------------------------------------------

func TestForget_SkipsPinned_ByImportance(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()

	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "low one", Importance: 0.1})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "low two", Importance: 0.1})
	pinnedID, _ := mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "low pinned", Importance: 0.1})
	if err := mgr.Update(ctx, KindEpisodic, pinnedID, func(it *MemoryItem) { SetPinned(it, true) }); err != nil {
		t.Fatalf("Update pinned: %v", err)
	}

	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByImportance, Threshold: 0.5})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	// Two unpinned-low items deleted, pinned-low survives.
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if _, err := mgr.Get(ctx, KindEpisodic, pinnedID); err != nil {
		t.Errorf("pinned item was forgotten: %v", err)
	}
}

func TestForget_SkipsPinned_ByCapacity(t *testing.T) {
	mgr := newManager(t)
	ctx := context.Background()

	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "a", Importance: 0.8})
	_, _ = mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "b", Importance: 0.9})
	pinnedID, _ := mgr.Add(ctx, KindEpisodic, MemoryItem{Content: "p", Importance: 0.1})
	_ = mgr.Update(ctx, KindEpisodic, pinnedID, func(it *MemoryItem) { SetPinned(it, true) })

	count, err := mgr.Forget(ctx, KindEpisodic, ForgetOptions{Strategy: ForgetByCapacity, Keep: 1})
	if err != nil {
		t.Fatalf("Forget: %v", err)
	}
	// Pinned is excluded entirely from the candidate set. 2 unpinned ⇒ Keep=1
	// ⇒ evict 1. Plus pinned still present.
	if count != 1 {
		t.Errorf("count = %d, want 1 (one unpinned evicted)", count)
	}
	if _, err := mgr.Get(ctx, KindEpisodic, pinnedID); err != nil {
		t.Errorf("pinned item was evicted by capacity: %v", err)
	}
	if mgr.StatsAll()[KindEpisodic].Count != 2 {
		t.Errorf("after forget, count = %d, want 2 (pinned + 1 highest unpinned)",
			mgr.StatsAll()[KindEpisodic].Count)
	}
}

// --- SavedBoost multiplies score for pinned / user_saved items -------------

func TestSavedBoost_BoostsPinned_Working(t *testing.T) {
	w, err := NewWorking(llm.NewScriptedLLM(llm.WithEmbedDimensions(64)),
		WorkingOptions{Capacity: 10, SavedBoost: 2.0})
	if err != nil {
		t.Fatalf("NewWorking: %v", err)
	}
	ctx := context.Background()
	plainID, _ := w.Add(ctx, MemoryItem{Content: "go modules notes", Importance: 0.5})
	pinnedID, _ := w.Add(ctx, MemoryItem{Content: "go modules notes", Importance: 0.5})
	_ = w.Update(ctx, pinnedID, func(it *MemoryItem) { SetPinned(it, true) })

	results, err := w.Search(ctx, "go modules", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Item.ID != pinnedID {
		t.Errorf("top = %v, want pinned %v (SavedBoost=2.0 should win)",
			results[0].Item.ID, pinnedID)
	}
	if results[1].Item.ID != plainID {
		t.Errorf("second = %v, want plain %v", results[1].Item.ID, plainID)
	}
}

func TestSavedBoost_DefaultNoOp(t *testing.T) {
	// SavedBoost zero value ⇒ no special multiplier; pinned and non-pinned
	// identical-content items should tie (or differ only by base scoring).
	w, err := NewWorking(llm.NewScriptedLLM(llm.WithEmbedDimensions(64)),
		WorkingOptions{Capacity: 10}) // SavedBoost unset
	if err != nil {
		t.Fatalf("NewWorking: %v", err)
	}
	ctx := context.Background()
	plainID, _ := w.Add(ctx, MemoryItem{Content: "x y z keyword", Importance: 0.5})
	pinnedID, _ := w.Add(ctx, MemoryItem{Content: "x y z keyword", Importance: 0.5})
	_ = w.Update(ctx, pinnedID, func(it *MemoryItem) { SetPinned(it, true) })

	results, err := w.Search(ctx, "keyword", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Their scores should be within a tiny epsilon of each other.
	const eps = 1e-9
	delta := results[0].Score - results[1].Score
	if delta < -eps || delta > eps {
		t.Errorf("with SavedBoost=0 (no-op), pinned and plain scores diverged by %v (>%v)",
			delta, eps)
	}
	// Sanity: both IDs present.
	seen := map[string]bool{}
	for _, r := range results {
		seen[r.Item.ID] = true
	}
	if !seen[pinnedID] || !seen[plainID] {
		t.Errorf("expected both ids in results; got %v", seen)
	}
}
