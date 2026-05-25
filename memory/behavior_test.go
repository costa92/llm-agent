package memory

import (
	"context"
	"testing"
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
