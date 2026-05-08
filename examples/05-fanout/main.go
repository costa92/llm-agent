// Demo 05: pkg/fanout — bounded-parallelism task runner.
//
// fanout.Run executes []fanout.Task[T] with a maxConcurrency cap, recovers
// panics into ErrTaskPanic, preserves task order via Result[T].Index, and
// supports optional WithFailFast(). Use it any time you need
// "concurrent-but-bounded with deterministic ordering."
//
// This demo simulates 5 parallel HTTP fetches with maxConcurrency=2.
//
// Run:
//
//	cd examples/05-fanout && go run .
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/costa92/llm-agent/pkg/fanout"
)

func main() {
	tasks := []fanout.Task[string]{
		fakeFetch("alpha", 80*time.Millisecond),
		fakeFetch("bravo", 30*time.Millisecond),
		fakeFetch("charlie", 60*time.Millisecond),
		fakeFetch("delta", 40*time.Millisecond),
		fakeFetch("echo", 20*time.Millisecond),
	}

	start := time.Now()
	results, err := fanout.Run(context.Background(), 2, tasks)
	if err != nil {
		log.Fatalf("fanout: %v", err)
	}

	fmt.Printf("=== fanout(maxConcurrency=2) finished in %v ===\n", time.Since(start).Round(time.Millisecond))
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  idx=%d ERR %v\n", r.Index, r.Err)
			continue
		}
		fmt.Printf("  idx=%d %s\n", r.Index, r.Value)
	}
}

func fakeFetch(name string, delay time.Duration) fanout.Task[string] {
	return func(ctx context.Context) (string, error) {
		select {
		case <-time.After(delay):
			return fmt.Sprintf("fetched %s in %v", name, delay), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}
