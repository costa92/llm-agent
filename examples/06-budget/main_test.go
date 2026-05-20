// TestExample_RunsToCompletion captures stdout, runs main(), and asserts
// the deterministic transcript covers all three budget dimensions. The
// example is in the standard examples/ test suite — running it via
// `cd examples && go test ./06-budget/...` provides a CI smoke check
// that the chokepoint contract holds end-to-end without any provider
// import or network.
//
// Adding a test to a demo is a deliberate divergence from examples 01-05
// (none of which ship tests today). The precedent is recorded so future
// examples that demonstrate enforcement contracts (rather than purely
// illustrative wiring) may follow the same pattern.
package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestExample_RunsToCompletion(t *testing.T) {
	// Redirect stdout into a pipe so we can assert what main() prints.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	// Drain the pipe in a goroutine; ReadAll on the close.
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	// Run the demo with stdout captured.
	main()

	// Restore stdout and close the writer so the goroutine returns.
	_ = w.Close()
	os.Stdout = origStdout
	out := <-done

	mustContain(t, out,
		"MaxCalls",
		"4th denied with errors.Is(err, budget.ErrCallsExceeded) = true",
		"budget.ErrBudgetExceeded) = true",
		"LLM Generate calls reaching the model: 3", // pre-call deny short-circuits
		"MaxTokens",
		"valid response but exhausted",
		"LLM Generate calls reaching the model: 3", // post-call deny — all 3 reach the LLM
		"MaxWall",
		"context.DeadlineExceeded) = true",
		"deadline fired before response",
		"OK",
	)
}

func mustContain(t *testing.T, out string, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(out, f) {
			t.Errorf("captured stdout missing fragment %q.\nFull output:\n%s", f, out)
		}
	}
}
