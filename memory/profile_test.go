package memory

import (
	"testing"
)

// --- Source / Category / Pinned / Disabled getter/setter round-trip --------

func TestProfile_SourceRoundTrip(t *testing.T) {
	var it MemoryItem
	if got := GetSource(it); got != SourceUnknown {
		t.Errorf("zero MemoryItem Source = %q, want SourceUnknown (empty)", got)
	}
	SetSource(&it, SourceUserSaved)
	if got := GetSource(it); got != SourceUserSaved {
		t.Errorf("after SetSource: GetSource = %q, want SourceUserSaved", got)
	}
	if it.Metadata == nil {
		t.Error("SetSource on nil-Metadata item must initialize the map")
	}
	SetSource(&it, SourceAgentInferred)
	if got := GetSource(it); got != SourceAgentInferred {
		t.Errorf("re-SetSource: GetSource = %q, want SourceAgentInferred", got)
	}
}

func TestProfile_CategoryRoundTrip(t *testing.T) {
	var it MemoryItem
	if got := GetCategory(it); got != Category("") {
		t.Errorf("zero Category = %q, want empty", got)
	}
	SetCategory(&it, CategoryUser)
	if got := GetCategory(it); got != CategoryUser {
		t.Errorf("after SetCategory: GetCategory = %q, want CategoryUser", got)
	}
	if it.Metadata == nil {
		t.Error("SetCategory must initialize Metadata when nil")
	}
}

func TestProfile_PinnedRoundTrip(t *testing.T) {
	var it MemoryItem
	if IsPinned(it) {
		t.Error("zero MemoryItem should not be pinned")
	}
	SetPinned(&it, true)
	if !IsPinned(it) {
		t.Error("after SetPinned(true), IsPinned should be true")
	}
	if it.Metadata == nil {
		t.Error("SetPinned must initialize Metadata when nil")
	}
	SetPinned(&it, false)
	if IsPinned(it) {
		t.Error("after SetPinned(false), IsPinned should be false")
	}
}

func TestProfile_DisabledRoundTrip(t *testing.T) {
	var it MemoryItem
	if IsDisabled(it) {
		t.Error("zero MemoryItem should not be disabled")
	}
	SetDisabled(&it, true)
	if !IsDisabled(it) {
		t.Error("after SetDisabled(true), IsDisabled should be true")
	}
	if it.Metadata == nil {
		t.Error("SetDisabled must initialize Metadata when nil")
	}
	SetDisabled(&it, false)
	if IsDisabled(it) {
		t.Error("after SetDisabled(false), IsDisabled should be false")
	}
}

// --- Getters on zero/missing/type-mismatched metadata return zero ----------

func TestProfile_Getters_ZeroValueOnMissingMetadata(t *testing.T) {
	it := MemoryItem{Metadata: map[string]any{"unrelated": 42}}
	if GetSource(it) != SourceUnknown {
		t.Error("missing _source key should yield SourceUnknown")
	}
	if GetCategory(it) != Category("") {
		t.Error("missing _category key should yield empty Category")
	}
	if IsPinned(it) {
		t.Error("missing _pinned key should yield false")
	}
	if IsDisabled(it) {
		t.Error("missing _disabled key should yield false")
	}
}

func TestProfile_Getters_TypeMismatchYieldsZero(t *testing.T) {
	it := MemoryItem{Metadata: map[string]any{
		"_source":   42,        // not a string / Source
		"_category": []int{1},  // not a string / Category
		"_pinned":   "yes",     // not a bool
		"_disabled": 1,         // not a bool
	}}
	if GetSource(it) != SourceUnknown {
		t.Error("type-mismatch _source should yield SourceUnknown")
	}
	if GetCategory(it) != Category("") {
		t.Error("type-mismatch _category should yield empty Category")
	}
	if IsPinned(it) {
		t.Error("type-mismatch _pinned should yield false")
	}
	if IsDisabled(it) {
		t.Error("type-mismatch _disabled should yield false")
	}
}

// --- Constructors: NewSavedMemory / NewInferredMemory ----------------------

func TestProfile_NewSavedMemoryDefaults(t *testing.T) {
	it := NewSavedMemory("user prefers concise replies", CategoryUser)
	if it.Content != "user prefers concise replies" {
		t.Errorf("Content = %q", it.Content)
	}
	if it.Importance != 0.9 {
		t.Errorf("Importance = %v, want 0.9", it.Importance)
	}
	if !IsPinned(it) {
		t.Error("saved memory should be pinned by default")
	}
	if GetSource(it) != SourceUserSaved {
		t.Errorf("Source = %q, want SourceUserSaved", GetSource(it))
	}
	if GetCategory(it) != CategoryUser {
		t.Errorf("Category = %q, want CategoryUser", GetCategory(it))
	}
}

func TestProfile_NewInferredMemoryClampsConfidence(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.5, 0.5},
		{-0.3, 0},
		{1.7, 1.0},
		{0.0, 0.0},
		{1.0, 1.0},
	}
	for _, tc := range cases {
		it := NewInferredMemory("the user lives in SF", CategoryUser, tc.in)
		if it.Importance != tc.want {
			t.Errorf("confidence=%v → Importance=%v, want %v", tc.in, it.Importance, tc.want)
		}
		if GetSource(it) != SourceAgentInferred {
			t.Errorf("Source = %q, want SourceAgentInferred", GetSource(it))
		}
		if GetCategory(it) != CategoryUser {
			t.Errorf("Category = %q, want CategoryUser", GetCategory(it))
		}
		if IsPinned(it) {
			t.Error("inferred memory should NOT be pinned by default")
		}
	}
}
