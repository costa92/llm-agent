package agents

import (
	"context"
	"encoding/json"

	"github.com/costa92/llm-agent/pkg/fanout"
)

// Task pairs a Tool with its args for a single async invocation.
type Task struct {
	Tool Tool
	Args json.RawMessage
}

// TaskResult carries one Task's outcome.
type TaskResult struct {
	Index  int    // position in the input tasks slice
	Output string
	Err    error
}

// AsyncRunner executes Tasks in parallel.
//
//   - Single Task failures are captured in TaskResult.Err and do not abort
//     other Tasks (no fail-fast).
//   - The function-level error is set only when ctx is cancelled / times out.
//   - A Task panic is recovered and surfaced as TaskResult.Err
//     (*fanout.ErrTaskPanic), so a single misbehaving Tool no longer crashes
//     the process.
type AsyncRunner struct {
	maxParallel int // 0 = unlimited
}

// NewAsyncRunner returns an AsyncRunner. maxParallel <= 0 means unlimited.
func NewAsyncRunner(maxParallel int) *AsyncRunner {
	return &AsyncRunner{maxParallel: maxParallel}
}

// Execute fans out tasks via pkg/fanout, waits for all, returns results
// indexed by input position.
func (r *AsyncRunner) Execute(ctx context.Context, tasks []Task) ([]TaskResult, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	fanoutTasks := make([]fanout.Task[string], len(tasks))
	for i, t := range tasks {
		t := t
		fanoutTasks[i] = func(ctx context.Context) (string, error) {
			return t.Tool.Execute(ctx, t.Args)
		}
	}
	results, err := fanout.Run(ctx, r.maxParallel, fanoutTasks)
	out := make([]TaskResult, len(results))
	for i, res := range results {
		out[i] = TaskResult{Index: res.Index, Output: res.Value, Err: res.Err}
	}
	return out, err
}
