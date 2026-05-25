package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestWithSanitizer_EmptyChainNoOp(t *testing.T) {
	w := newWorking(t)
	got := WithSanitizer(w)
	// identity expected — no allocation when chain is empty
	if got != Memory(w) {
		t.Errorf("empty chain should return inner verbatim; got %T, want *WorkingMemory", got)
	}
}

func TestSanitizer_KeepRedactedItem(t *testing.T) {
	w := newWorking(t)
	redactor := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		it.Content = strings.ReplaceAll(it.Content, "secret", "[REDACTED]")
		return it, true, nil
	})
	mem := WithSanitizer(w, redactor)

	id, err := mem.Add(context.Background(), MemoryItem{Content: "this has a secret token", Importance: 0.5})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := mem.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "this has a [REDACTED] token" {
		t.Errorf("Content = %q, want redacted form", got.Content)
	}
}

func TestSanitizer_RejectKeepFalse(t *testing.T) {
	w := newWorking(t)
	rejector := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		return it, false, nil
	})
	mem := WithSanitizer(w, rejector)

	id, err := mem.Add(context.Background(), MemoryItem{Content: "anything", Importance: 0.5})
	if !errors.Is(err, ErrRejectedByPolicy) {
		t.Errorf("err = %v, want ErrRejectedByPolicy", err)
	}
	if id != "" {
		t.Errorf("rejected Add returned id = %q, want empty", id)
	}
	if w.Stats().Count != 0 {
		t.Errorf("inner Count = %d, want 0 (nothing should be stored)", w.Stats().Count)
	}
}

func TestSanitizer_PropagatesError(t *testing.T) {
	w := newWorking(t)
	wantErr := errors.New("network down")
	failing := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		return it, false, wantErr
	})
	mem := WithSanitizer(w, failing)
	_, err := mem.Add(context.Background(), MemoryItem{Content: "anything", Importance: 0.5})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestSanitizer_ChainShortCircuits(t *testing.T) {
	w := newWorking(t)
	calls := []string{}
	first := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		calls = append(calls, "first")
		return it, true, nil
	})
	second := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		calls = append(calls, "second")
		return it, false, nil
	})
	third := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		calls = append(calls, "third")
		return it, true, nil
	})
	mem := WithSanitizer(w, first, second, third)
	_, err := mem.Add(context.Background(), MemoryItem{Content: "x", Importance: 0.5})
	if !errors.Is(err, ErrRejectedByPolicy) {
		t.Errorf("err = %v, want ErrRejectedByPolicy", err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Errorf("calls = %v, want [first second] (third should not run)", calls)
	}
}

func TestSanitizer_OtherMethodsPassThrough(t *testing.T) {
	w := newWorking(t)
	// inserted directly into the inner store, so the sanitizer cannot affect it.
	id, err := w.Add(context.Background(), MemoryItem{Content: "raw secret", Importance: 0.5})
	if err != nil {
		t.Fatalf("inner Add: %v", err)
	}

	calls := 0
	hot := SanitizerFunc(func(_ context.Context, _ Kind, it MemoryItem) (MemoryItem, bool, error) {
		calls++
		return it, true, nil
	})
	mem := WithSanitizer(w, hot)

	// Type passes through.
	if mem.Type() != KindWorking {
		t.Errorf("Type = %q, want working", mem.Type())
	}
	// Get passes through (sanitizer NOT called on read).
	if _, err := mem.Get(context.Background(), id); err != nil {
		t.Errorf("Get: %v", err)
	}
	// Search passes through.
	res, err := mem.Search(context.Background(), "raw secret", 5)
	if err != nil {
		t.Errorf("Search: %v", err)
	}
	if len(res) == 0 {
		t.Error("Search returned no results")
	}
	// Update passes through.
	if err := mem.Update(context.Background(), id, func(it *MemoryItem) { it.Importance = 0.9 }); err != nil {
		t.Errorf("Update: %v", err)
	}
	// Stats passes through.
	if mem.Stats().Count != 1 {
		t.Errorf("Stats.Count = %d, want 1", mem.Stats().Count)
	}
	// Remove passes through.
	if err := mem.Remove(context.Background(), id); err != nil {
		t.Errorf("Remove: %v", err)
	}
	if calls != 0 {
		t.Errorf("sanitizer was called %d times on read-only paths, want 0", calls)
	}
}
