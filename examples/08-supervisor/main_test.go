package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestExample_RunsToCompletion(t *testing.T) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	main()
	_ = w.Close()
	os.Stdout = orig
	out := <-done
	mustContain(t, out, "--- Basic:", "--- Budget", "budget.ErrCallsExceeded", "--- Compose", "OK")
}

func mustContain(t *testing.T, out string, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(out, f) {
			t.Fatalf("missing fragment %q\nfull output:\n%s", f, out)
		}
	}
}
