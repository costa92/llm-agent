package agents

import (
	"errors"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	tool := fixedTool{name: "calc", desc: "calc"}
	if err := r.Register(tool); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := r.Get("calc")
	if !ok {
		t.Fatal("Get(calc) should hit")
	}
	if got.Name() != "calc" {
		t.Errorf("Name = %q", got.Name())
	}
}

func TestRegistry_RegisterDuplicate_ReturnsErr(t *testing.T) {
	r := NewRegistry(fixedTool{name: "a"})
	err := r.Register(fixedTool{name: "a"})
	if !errors.Is(err, ErrToolAlreadyRegistered) {
		t.Errorf("err = %v, want ErrToolAlreadyRegistered", err)
	}
}

func TestRegistry_List_Sorted(t *testing.T) {
	r := NewRegistry(
		fixedTool{name: "c"},
		fixedTool{name: "a"},
		fixedTool{name: "b"},
	)
	got := r.List()
	want := []string{"a", "b", "c"}
	for i, tool := range got {
		if tool.Name() != want[i] {
			t.Errorf("List[%d] = %q, want %q", i, tool.Name(), want[i])
		}
	}
}

func TestRegistry_AsLLMTools_PreservesOrder(t *testing.T) {
	r := NewRegistry(fixedTool{name: "z"}, fixedTool{name: "a"})
	got := r.AsLLMTools()
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "a" || got[1].Name != "z" {
		t.Errorf("order: %q,%q", got[0].Name, got[1].Name)
	}
}

func TestRegistry_PromptDescription(t *testing.T) {
	r := NewRegistry(
		fixedTool{name: "calc", desc: "do math"},
		fixedTool{name: "web", desc: "search web"},
	)
	got := r.PromptDescription()
	want := "- calc: do math\n- web: search web\n"
	if got != want {
		t.Errorf("PromptDescription =\n%q\nwant\n%q", got, want)
	}
}

func TestRegistry_PromptDescription_Empty(t *testing.T) {
	r := NewRegistry()
	if got := r.PromptDescription(); got != "(none)\n" {
		t.Errorf("empty registry = %q, want (none)", got)
	}
}
