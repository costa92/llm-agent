package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemStore_SaveLoad_RoundTrip(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	src := newWorking(t)
	ctx := context.Background()
	_, _ = src.Add(ctx, MemoryItem{Content: "alpha", Importance: 0.5})
	snap, _ := src.Export(ctx)

	if err := fs.Save(ctx, "k1", snap); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := fs.Load(ctx, "k1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Kind != KindWorking {
		t.Errorf("Kind = %q, want working", got.Kind)
	}
	if len(got.Items) != len(snap.Items) {
		t.Errorf("Items count = %d, want %d", len(got.Items), len(snap.Items))
	}
}

func TestFilesystemStore_LoadKind(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	w := newWorking(t)
	_, _ = w.Add(ctx, MemoryItem{Content: "w1"})
	wSnap, _ := w.Export(ctx)

	e := newEpisodic(t)
	_, _ = e.Add(ctx, MemoryItem{Content: "e1"})
	eSnap, _ := e.Export(ctx)

	if err := fs.Save(ctx, "k1", wSnap); err != nil {
		t.Fatalf("Save w: %v", err)
	}
	if err := fs.Save(ctx, "k1", eSnap); err != nil {
		t.Fatalf("Save e: %v", err)
	}

	gotW, err := fs.LoadKind(ctx, "k1", KindWorking)
	if err != nil {
		t.Fatalf("LoadKind working: %v", err)
	}
	if gotW.Kind != KindWorking {
		t.Errorf("Working Kind = %q", gotW.Kind)
	}
	gotE, err := fs.LoadKind(ctx, "k1", KindEpisodic)
	if err != nil {
		t.Fatalf("LoadKind episodic: %v", err)
	}
	if gotE.Kind != KindEpisodic {
		t.Errorf("Episodic Kind = %q", gotE.Kind)
	}
}

func TestFilesystemStore_LoadMissingReturnsNotExist(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	_, err = fs.Load(context.Background(), "absent-key")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want wrapping os.ErrNotExist", err)
	}
}

func TestFilesystemStore_DeleteAllKinds(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	w := newWorking(t)
	_, _ = w.Add(ctx, MemoryItem{Content: "x"})
	wSnap, _ := w.Export(ctx)
	if err := fs.Save(ctx, "k1", wSnap); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := fs.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	keys, err := fs.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, k := range keys {
		if k == "k1" {
			t.Errorf("List still contains deleted key %q", k)
		}
	}
}

func TestFilesystemStore_List(t *testing.T) {
	fs, err := NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	w := newWorking(t)
	_, _ = w.Add(ctx, MemoryItem{Content: "x"})
	wSnap, _ := w.Export(ctx)
	for _, k := range []string{"alpha", "bravo", "charlie"} {
		if err := fs.Save(ctx, k, wSnap); err != nil {
			t.Fatalf("Save %q: %v", k, err)
		}
	}
	keys, err := fs.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("len(keys) = %d, want 3", len(keys))
	}
	// sorted
	if !(keys[0] == "alpha" && keys[1] == "bravo" && keys[2] == "charlie") {
		t.Errorf("keys = %v, want sorted [alpha bravo charlie]", keys)
	}
}

func TestFilesystemStore_SanitizesKey(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystemStore(dir)
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	w := newWorking(t)
	_, _ = w.Add(ctx, MemoryItem{Content: "x"})
	wSnap, _ := w.Export(ctx)
	if err := fs.Save(ctx, "../../etc/passwd", wSnap); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// File should land inside dir, not escape it.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no files written")
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "/") || strings.Contains(e.Name(), "\\") {
			t.Errorf("entry name contains path separator: %q", e.Name())
		}
		if strings.Contains(e.Name(), "..") {
			t.Errorf("entry name contains '..': %q", e.Name())
		}
	}
	// And nothing should have been written outside dir.
	bad := filepath.Join(dir, "..", "..", "etc", "passwd")
	if _, err := os.Stat(bad); err == nil {
		t.Errorf("path traversal succeeded: file exists at %q", bad)
	}
}

func TestFilesystemStore_AtomicSave(t *testing.T) {
	// Two sequential Saves to the same key should not leave a half-written
	// file: the final read must succeed and the contents must be the
	// LATER snapshot's contents.
	dir := t.TempDir()
	fs, err := NewFilesystemStore(dir)
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	w1 := newWorking(t)
	id1, _ := w1.Add(ctx, MemoryItem{Content: "first"})
	snap1, _ := w1.Export(ctx)

	w2 := newWorking(t)
	id2, _ := w2.Add(ctx, MemoryItem{Content: "second"})
	snap2, _ := w2.Export(ctx)

	if err := fs.Save(ctx, "k1", snap1); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if err := fs.Save(ctx, "k1", snap2); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	got, err := fs.LoadKind(ctx, "k1", KindWorking)
	if err != nil {
		t.Fatalf("LoadKind: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(got.Items))
	}
	if got.Items[0].Item.ID != id2 {
		t.Errorf("ID = %q, want %q (second save should win, first id was %q)", got.Items[0].Item.ID, id2, id1)
	}

	// And no stray .tmp files left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("stray temp file: %q", e.Name())
		}
	}
}
