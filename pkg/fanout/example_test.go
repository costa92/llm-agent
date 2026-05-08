// pkg/fanout/example_test.go
package fanout_test

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/costa92/llm-agent/pkg/fanout"
)

func ExampleRun() {
	tasks := []fanout.Task[int]{
		func(ctx context.Context) (int, error) { return 1, nil },
		func(ctx context.Context) (int, error) { return 2, nil },
		func(ctx context.Context) (int, error) { return 0, errors.New("third failed") },
	}

	results, err := fanout.Run(context.Background(), 2, tasks)
	if err != nil {
		fmt.Println("ctx err:", err)
		return
	}

	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("idx %d: ERR %v\n", r.Index, r.Err)
		} else {
			fmt.Printf("idx %d: %d\n", r.Index, r.Value)
		}
	}

	// Output:
	// idx 0: 1
	// idx 1: 2
	// idx 2: ERR third failed
}

func ExampleRun_failFast() {
	tasks := []fanout.Task[string]{
		func(ctx context.Context) (string, error) { return "", errors.New("fail") },
		func(ctx context.Context) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
	}

	results, _ := fanout.Run(context.Background(), 2, tasks, fanout.WithFailFast())

	msgs := []string{
		fmt.Sprintf("idx 0: %v", results[0].Err),
		fmt.Sprintf("idx 1: %v", results[1].Err),
	}
	sort.Strings(msgs)
	for _, m := range msgs {
		fmt.Println(m)
	}

	// Output:
	// idx 0: fail
	// idx 1: context canceled
}
