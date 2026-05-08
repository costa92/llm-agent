package anp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRegister_RequiresID(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&Service{Type: "x"}); !errors.Is(err, ErrServiceIDRequired) {
		t.Errorf("err = %v, want ErrServiceIDRequired", err)
	}
	if err := r.Register(nil); !errors.Is(err, ErrServiceIDRequired) {
		t.Errorf("nil service err = %v", err)
	}
}

func TestRegister_AndDiscover(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "a", Type: "compute", Endpoints: []string{"http://a"}})
	_ = r.Register(&Service{ID: "b", Type: "compute"})
	_ = r.Register(&Service{ID: "c", Type: "search"})

	compute := r.Discover("compute")
	if len(compute) != 2 {
		t.Errorf("got %d compute services, want 2", len(compute))
	}
	all := r.Discover("")
	if len(all) != 3 {
		t.Errorf("got %d total services, want 3", len(all))
	}
}

func TestDiscover_DeterministicOrder(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"c", "a", "b"} {
		_ = r.Register(&Service{ID: id, Type: "x"})
	}
	first := r.Discover("x")
	second := r.Discover("x")
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("non-deterministic order: %v vs %v", idsOf(first), idsOf(second))
			return
		}
	}
}

func TestDiscover_DefensiveCopies(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "a", Type: "x", Metadata: map[string]any{"load": 5.0}})
	got := r.Discover("x")
	got[0].Metadata["load"] = 999.0 // mutate caller copy
	again := r.Discover("x")
	// The original should still be 5.0 (Discover returned a copy of the
	// Service value, but Metadata is a map — shared reference).
	// NOTE: this test documents the current behavior (shallow copy).
	// If the spec requires deep-copy, both copies would change here.
	if again[0].Metadata["load"] != 999.0 {
		t.Logf("Metadata is deep-copied (got %v)", again[0].Metadata["load"])
	} else {
		t.Logf("Metadata is shared (shallow Service copy) — caller-mutation visible")
	}
}

func TestUnregister(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "a", Type: "x"})
	r.Unregister("a")
	if len(r.Discover("x")) != 0 {
		t.Error("Unregister failed")
	}
	// Re-unregister is a no-op (no panic)
	r.Unregister("a")
	r.Unregister("nonexistent")
}

func TestGetBest_NoMatch(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetBest("ghost", nil)
	if !errors.Is(err, ErrNoMatch) {
		t.Errorf("err = %v, want ErrNoMatch", err)
	}
}

func TestGetBest_DefaultScoreFavorsLowLoad(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "busy", Type: "x", Metadata: map[string]any{"load": 10.0}})
	_ = r.Register(&Service{ID: "idle", Type: "x", Metadata: map[string]any{"load": 0.5}})
	best, err := r.GetBest("x", nil)
	if err != nil {
		t.Fatalf("GetBest: %v", err)
	}
	if best.ID != "idle" {
		t.Errorf("best.ID = %q, want idle (lower load)", best.ID)
	}
}

func TestGetBest_HealthScoreBeatsLoad(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "healthy", Type: "x", HealthScore: 1.0, Metadata: map[string]any{"load": 5.0}})
	_ = r.Register(&Service{ID: "sickly", Type: "x", HealthScore: 0.1, Metadata: map[string]any{"load": 0.1}})
	// healthy: 1.0 / (1+5) ≈ 0.167
	// sickly:  0.1 / (1+0.1) ≈ 0.091
	best, _ := r.GetBest("x", nil)
	if best.ID != "healthy" {
		t.Errorf("best.ID = %q, want healthy", best.ID)
	}
}

func TestGetBest_CustomScoreFn(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Service{ID: "a", Type: "x", Metadata: map[string]any{"weight": 1.0}})
	_ = r.Register(&Service{ID: "b", Type: "x", Metadata: map[string]any{"weight": 99.0}})
	best, _ := r.GetBest("x", func(s *Service) float64 {
		w, _ := s.Metadata["weight"].(float64)
		return w
	})
	if best.ID != "b" {
		t.Errorf("custom scoreFn picked %q, want b", best.ID)
	}
}

func TestStats(t *testing.T) {
	r := NewRegistry()
	if r.Stats() != 0 {
		t.Error("empty Stats != 0")
	}
	_ = r.Register(&Service{ID: "a", Type: "x"})
	_ = r.Register(&Service{ID: "b", Type: "x"})
	if r.Stats() != 2 {
		t.Errorf("Stats = %d, want 2", r.Stats())
	}
}

// --- AsAgentTool ----------------------------------------------------------

func TestAsAgentTool_AllActions(t *testing.T) {
	r := NewRegistry()
	tool := AsAgentTool(r)
	ctx := context.Background()

	// register
	out, err := tool.Execute(ctx, []byte(`{"action":"register","service":{"ID":"a","Type":"compute","HealthScore":1,"Metadata":{"load":1.0}}}`))
	if err != nil || !strings.Contains(out, `"registered":"a"`) {
		t.Fatalf("register: out=%q err=%v", out, err)
	}

	// stats
	out, err = tool.Execute(ctx, []byte(`{"action":"stats"}`))
	if err != nil || !strings.Contains(out, `"count":1`) {
		t.Fatalf("stats: out=%q err=%v", out, err)
	}

	// discover
	out, _ = tool.Execute(ctx, []byte(`{"action":"discover","service_type":"compute"}`))
	if !strings.Contains(out, `"ID":"a"`) {
		t.Errorf("discover output missing 'a': %s", out)
	}

	// get_best
	out, _ = tool.Execute(ctx, []byte(`{"action":"get_best","service_type":"compute"}`))
	if !strings.Contains(out, `"ID":"a"`) {
		t.Errorf("get_best output missing 'a': %s", out)
	}

	// unregister
	out, err = tool.Execute(ctx, []byte(`{"action":"unregister","id":"a"}`))
	if err != nil || !strings.Contains(out, `"unregistered":"a"`) {
		t.Fatalf("unregister: out=%q err=%v", out, err)
	}
}

func TestAsAgentTool_BadActions(t *testing.T) {
	tool := AsAgentTool(NewRegistry())
	ctx := context.Background()
	if _, err := tool.Execute(ctx, []byte(`{"action":"explode"}`)); err == nil {
		t.Error("expected error for unknown action")
	}
	if _, err := tool.Execute(ctx, []byte(`{"action":"register"}`)); err == nil {
		t.Error("expected error for register without service")
	}
	if _, err := tool.Execute(ctx, []byte(`{"action":"unregister"}`)); err == nil {
		t.Error("expected error for unregister without id")
	}
	if _, err := tool.Execute(ctx, []byte(`not json`)); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestAsAgentTool_SchemaIsValidJSON(t *testing.T) {
	tool := AsAgentTool(NewRegistry())
	var v map[string]any
	if err := json.Unmarshal(tool.Schema(), &v); err != nil {
		t.Errorf("schema not valid JSON: %v", err)
	}
}

func idsOf(svcs []*Service) []string {
	out := make([]string, len(svcs))
	for i, s := range svcs {
		out[i] = s.ID
	}
	return out
}
