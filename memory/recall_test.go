package memory

import (
	"context"
	"testing"
	"time"
)

// --- matchesFilter unit tests --------------------------------------------

func TestListFilter_Match_ZeroFilterAcceptsNonDisabled(t *testing.T) {
	it := MemoryItem{Content: "x"}
	if !matchesFilter(it, ListFilter{}) {
		t.Error("zero filter should accept plain item")
	}
}

func TestListFilter_Match_HidesDisabledByDefault(t *testing.T) {
	it := MemoryItem{Content: "x", Metadata: map[string]any{}}
	SetDisabled(&it, true)
	if matchesFilter(it, ListFilter{}) {
		t.Error("disabled item should be hidden by default")
	}
	if !matchesFilter(it, ListFilter{IncludeDisabled: true}) {
		t.Error("disabled item should appear when IncludeDisabled=true")
	}
}

func TestListFilter_Match_PinnedOnly(t *testing.T) {
	plain := MemoryItem{Content: "p"}
	pinned := MemoryItem{Content: "q"}
	SetPinned(&pinned, true)
	if matchesFilter(plain, ListFilter{PinnedOnly: true}) {
		t.Error("plain item should fail PinnedOnly")
	}
	if !matchesFilter(pinned, ListFilter{PinnedOnly: true}) {
		t.Error("pinned item should pass PinnedOnly")
	}
}

func TestListFilter_Match_SourceAndCategory(t *testing.T) {
	it := MemoryItem{Content: "x"}
	SetSource(&it, SourceUserSaved)
	SetCategory(&it, CategoryUser)
	if !matchesFilter(it, ListFilter{Source: SourceUserSaved}) {
		t.Error("source match should pass")
	}
	if matchesFilter(it, ListFilter{Source: SourceSystem}) {
		t.Error("source mismatch should fail")
	}
	if !matchesFilter(it, ListFilter{Category: CategoryUser}) {
		t.Error("category match should pass")
	}
	if matchesFilter(it, ListFilter{Category: CategoryFeedback}) {
		t.Error("category mismatch should fail")
	}
}

func TestListFilter_Match_TagsAnyOf(t *testing.T) {
	it := MemoryItem{Content: "x", Tags: []string{"go", "modules"}}
	if !matchesFilter(it, ListFilter{Tags: []string{"go"}}) {
		t.Error("tag any-of should match")
	}
	if !matchesFilter(it, ListFilter{Tags: []string{"missing", "modules"}}) {
		t.Error("tag any-of should match when at least one tag matches")
	}
	if matchesFilter(it, ListFilter{Tags: []string{"missing"}}) {
		t.Error("tag any-of should fail when none match")
	}
}

func TestListFilter_Match_MinImportance(t *testing.T) {
	it := MemoryItem{Content: "x", Importance: 0.5}
	if !matchesFilter(it, ListFilter{MinImportance: 0.4}) {
		t.Error("importance 0.5 should pass MinImportance=0.4")
	}
	if matchesFilter(it, ListFilter{MinImportance: 0.6}) {
		t.Error("importance 0.5 should fail MinImportance=0.6")
	}
}

func TestListFilter_Match_Scope(t *testing.T) {
	it := MemoryItem{Content: "x"}
	stampScope(&it, Scope{User: "alice"})
	if !matchesFilter(it, ListFilter{Scope: Scope{User: "alice"}}) {
		t.Error("scope match should pass")
	}
	if matchesFilter(it, ListFilter{Scope: Scope{User: "bob"}}) {
		t.Error("scope mismatch should fail")
	}
}

// --- cursor encode/decode ------------------------------------------------

func TestCursor_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Nanosecond)
	c := listCursor{AfterCreatedAt: now, AfterID: "id_42"}
	s, err := encodeCursor(c)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := decodeCursor(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.AfterCreatedAt.Equal(c.AfterCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.AfterCreatedAt, c.AfterCreatedAt)
	}
	if got.AfterID != c.AfterID {
		t.Errorf("AfterID = %q, want %q", got.AfterID, c.AfterID)
	}
}

func TestCursor_EmptyDecodesToZero(t *testing.T) {
	c, err := decodeCursor("")
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if !c.AfterCreatedAt.IsZero() || c.AfterID != "" {
		t.Errorf("empty cursor should decode to zero, got %+v", c)
	}
}

func TestCursor_BadCursorErrors(t *testing.T) {
	if _, err := decodeCursor("not-base64-!!!"); err == nil {
		t.Error("garbage cursor should error")
	}
	if _, err := decodeCursor("aGVsbG8"); err == nil {
		// "hello" base64'd; not JSON
		t.Error("non-JSON cursor should error")
	}
}

// --- Lister interface conformance ----------------------------------------

func TestLister_ImplementedByThreeMemoryTypes(t *testing.T) {
	var _ Lister = (*WorkingMemory)(nil)
	var _ Lister = (*EpisodicMemory)(nil)
	var _ Lister = (*SemanticMemory)(nil)
}

// --- listFromStore behavior ----------------------------------------------

// addBackdated inserts an item into the given working memory with the
// caller-controlled CreatedAt so List sort ordering is testable.
func addBackdated(t *testing.T, w *WorkingMemory, content string, created time.Time) string {
	t.Helper()
	ctx := context.Background()
	id, err := w.Add(ctx, MemoryItem{Content: content, Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := w.Update(ctx, id, func(it *MemoryItem) {
		it.CreatedAt = created
	}); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	return id
}

func TestListFromStore_OrdersByCreatedAtDescIDAsc(t *testing.T) {
	w := newWorking(t)
	base := time.Now().Add(-time.Hour).UTC()
	// distinct CreatedAt: t1 < t2 < t3 — expected DESC: t3, t2, t1
	id1 := addBackdated(t, w, "a1", base)
	id2 := addBackdated(t, w, "a2", base.Add(time.Minute))
	id3 := addBackdated(t, w, "a3", base.Add(2*time.Minute))

	page, err := w.List(context.Background(), ListFilter{}, 10, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(page.Items))
	}
	if page.Items[0].ID != id3 || page.Items[1].ID != id2 || page.Items[2].ID != id1 {
		t.Errorf("order = [%s,%s,%s], want [%s,%s,%s]",
			page.Items[0].ID, page.Items[1].ID, page.Items[2].ID, id3, id2, id1)
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty (end of stream)", page.NextCursor)
	}
}

func TestListFromStore_SameCreatedAtBreaksByIDAsc(t *testing.T) {
	w := newWorking(t)
	same := time.Now().UTC()
	idA := addBackdated(t, w, "ax", same)
	idB := addBackdated(t, w, "bx", same)
	// IDs are monotonic by seq inside one store, so idA < idB (string compare on the same prefix).
	if idA > idB {
		idA, idB = idB, idA
	}
	page, err := w.List(context.Background(), ListFilter{}, 10, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(page.Items))
	}
	if page.Items[0].ID != idA || page.Items[1].ID != idB {
		t.Errorf("tie-break order = [%s,%s], want [%s,%s]",
			page.Items[0].ID, page.Items[1].ID, idA, idB)
	}
}

func TestListFromStore_PaginationCursor(t *testing.T) {
	w := newWorking(t)
	base := time.Now().Add(-time.Hour).UTC()
	// 5 items, strictly ascending CreatedAt → returned in reverse
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = addBackdated(t, w, "item", base.Add(time.Duration(i)*time.Minute))
	}
	// expected order: ids[4], ids[3], ids[2], ids[1], ids[0]
	ctx := context.Background()
	all := []string{}
	cursor := ""
	pages := 0
	for {
		page, err := w.List(ctx, ListFilter{}, 2, cursor)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		pages++
		for _, it := range page.Items {
			all = append(all, it.ID)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3 (2+2+1)", pages)
	}
	want := []string{ids[4], ids[3], ids[2], ids[1], ids[0]}
	if len(all) != 5 {
		t.Fatalf("got %d ids across pages, want 5: %v", len(all), all)
	}
	for i, id := range want {
		if all[i] != id {
			t.Errorf("page %d position %d: got %s, want %s", i/2, i, all[i], id)
		}
	}
}

func TestListFromStore_FiltersDisabledByDefault(t *testing.T) {
	w := newWorking(t)
	ctx := context.Background()
	id1, _ := w.Add(ctx, MemoryItem{Content: "visible", Importance: 0.5})
	id2, _ := w.Add(ctx, MemoryItem{Content: "hidden", Importance: 0.5})
	_ = w.Update(ctx, id2, func(it *MemoryItem) {
		SetDisabled(it, true)
	})

	page, err := w.List(ctx, ListFilter{}, 10, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != id1 {
		t.Errorf("default page = %v, want [%s]", page.Items, id1)
	}

	page2, _ := w.List(ctx, ListFilter{IncludeDisabled: true}, 10, "")
	if len(page2.Items) != 2 {
		t.Errorf("IncludeDisabled page count = %d, want 2", len(page2.Items))
	}
}

func TestListFromStore_BadCursorErrors(t *testing.T) {
	w := newWorking(t)
	_, _ = w.Add(context.Background(), MemoryItem{Content: "x", Importance: 0.5})
	if _, err := w.List(context.Background(), ListFilter{}, 5, "garbage!!!"); err == nil {
		t.Error("bad cursor should error")
	}
}

