package memory

import (
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

