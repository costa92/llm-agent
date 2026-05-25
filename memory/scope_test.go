package memory

import (
	"context"
	"testing"
)

func TestScope_IsZero(t *testing.T) {
	if !(Scope{}).IsZero() {
		t.Error("zero Scope{} should report IsZero=true")
	}
	cases := []Scope{
		{User: "alice"},
		{Project: "x"},
		{Session: "s"},
		{User: "a", Project: "p"},
		{User: "a", Project: "p", Session: "s"},
	}
	for _, s := range cases {
		if s.IsZero() {
			t.Errorf("Scope %+v should report IsZero=false", s)
		}
	}
}

func TestScope_Equal(t *testing.T) {
	a := Scope{User: "alice", Project: "p1", Session: "s1"}
	b := Scope{User: "alice", Project: "p1", Session: "s1"}
	c := Scope{User: "alice", Project: "p1"}
	if !a.Equal(b) {
		t.Error("identical scopes should be Equal")
	}
	if a.Equal(c) {
		t.Error("scopes differing on Session should not be Equal")
	}
}

func TestScope_Matches_Wildcard(t *testing.T) {
	filter := Scope{User: "alice"}
	concrete := Scope{User: "alice", Project: "x", Session: "y"}
	if !filter.Matches(concrete) {
		t.Error("partial filter should match richer concrete scope")
	}
	// zero filter matches anything.
	if !(Scope{}).Matches(concrete) {
		t.Error("zero (wildcard) filter should match any concrete scope")
	}
	if !(Scope{}).Matches(Scope{}) {
		t.Error("zero filter should match zero concrete scope")
	}
}

func TestScope_Matches_Mismatch(t *testing.T) {
	filter := Scope{User: "alice"}
	if filter.Matches(Scope{User: "bob"}) {
		t.Error("filter {alice} should not match concrete {bob}")
	}
	if (Scope{User: "alice", Project: "x"}).Matches(Scope{User: "alice", Project: "y"}) {
		t.Error("mismatching Project should fail to match")
	}
}

func TestScope_Matches_LegacyData(t *testing.T) {
	// legacy (pre-scope) items carry a zero concrete scope.
	// scoped queries (non-zero filter) must NOT see legacy data.
	filter := Scope{User: "alice"}
	if filter.Matches(Scope{}) {
		t.Error("non-zero filter should not match legacy (zero-concrete) data")
	}
}

func TestWithScope_RoundTrip(t *testing.T) {
	ctx := context.Background()
	if got := ScopeFrom(ctx); !got.IsZero() {
		t.Errorf("absent: got %+v, want zero", got)
	}
	want := Scope{User: "alice", Project: "p", Session: "s"}
	ctx2 := WithScope(ctx, want)
	got := ScopeFrom(ctx2)
	if !got.Equal(want) {
		t.Errorf("round-trip: got %+v, want %+v", got, want)
	}
	// parent ctx unchanged.
	if !ScopeFrom(ctx).IsZero() {
		t.Error("WithScope must not mutate parent ctx")
	}
}
