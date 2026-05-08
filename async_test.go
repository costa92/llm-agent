package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/costa92/llm-agent/pkg/fanout"
)

// counterTool: increments counter on each Execute, returns its index.
type counterTool struct {
	name    string
	counter *atomic.Int32
}

func (c counterTool) Name() string            { return c.name }
func (c counterTool) Description() string     { return "counter" }
func (c counterTool) Schema() json.RawMessage { return json.RawMessage(`{}`) }
func (c counterTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	v := c.counter.Add(1)
	return fmt.Sprintf("call-%d", v), nil
}

// failingTool: always returns err.
type failingTool struct{ name string }

func (f failingTool) Name() string            { return f.name }
func (f failingTool) Description() string     { return "fail" }
func (f failingTool) Schema() json.RawMessage { return json.RawMessage(`{}`) }
func (f failingTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "", errors.New("boom")
}

// panicTool: panics with a fixed value when executed.
type panicTool struct{ name string }

func (p panicTool) Name() string            { return p.name }
func (p panicTool) Description() string     { return "panic" }
func (p panicTool) Schema() json.RawMessage { return json.RawMessage(`{}`) }
func (p panicTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	panic("panic-tool-boom")
}

// slowTool: sleeps until ctx cancels, returns ctx.Err().
type slowTool struct{ name string }

func (s slowTool) Name() string            { return s.name }
func (s slowTool) Description() string     { return "slow" }
func (s slowTool) Schema() json.RawMessage { return json.RawMessage(`{}`) }
func (s slowTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(2 * time.Second):
		return "ok", nil
	}
}

func TestAsyncRunner_AllSucceed(t *testing.T) {
	c := &atomic.Int32{}
	tasks := []Task{
		{Tool: counterTool{name: "a", counter: c}, Args: json.RawMessage(`{}`)},
		{Tool: counterTool{name: "b", counter: c}, Args: json.RawMessage(`{}`)},
		{Tool: counterTool{name: "c", counter: c}, Args: json.RawMessage(`{}`)},
	}
	r := NewAsyncRunner(0)
	res, err := r.Execute(context.Background(), tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("len = %d", len(res))
	}
	for i, tr := range res {
		if tr.Index != i {
			t.Errorf("res[%d].Index = %d", i, tr.Index)
		}
		if tr.Err != nil {
			t.Errorf("res[%d].Err = %v", i, tr.Err)
		}
		if !strings.HasPrefix(tr.Output, "call-") {
			t.Errorf("res[%d].Output = %q", i, tr.Output)
		}
	}
}

func TestAsyncRunner_OneFailureDoesNotKillOthers(t *testing.T) {
	c := &atomic.Int32{}
	tasks := []Task{
		{Tool: counterTool{name: "a", counter: c}, Args: json.RawMessage(`{}`)},
		{Tool: failingTool{name: "b"}, Args: json.RawMessage(`{}`)},
		{Tool: counterTool{name: "c", counter: c}, Args: json.RawMessage(`{}`)},
	}
	r := NewAsyncRunner(0)
	res, err := r.Execute(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Execute err = %v, want nil (single failures don't fail func)", err)
	}
	if res[0].Err != nil || res[2].Err != nil {
		t.Errorf("non-failing tasks should have nil err: %v, %v", res[0].Err, res[2].Err)
	}
	if res[1].Err == nil {
		t.Error("res[1].Err should be set")
	}
}

func TestAsyncRunner_PanicCapturedAsError(t *testing.T) {
	c := &atomic.Int32{}
	tasks := []Task{
		{Tool: counterTool{name: "a", counter: c}, Args: json.RawMessage(`{}`)},
		{Tool: panicTool{name: "b"}, Args: json.RawMessage(`{}`)},
		{Tool: counterTool{name: "c", counter: c}, Args: json.RawMessage(`{}`)},
	}
	r := NewAsyncRunner(0)

	res, err := r.Execute(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Execute err = %v, want nil", err)
	}
	if len(res) != 3 {
		t.Fatalf("len(res) = %d, want 3", len(res))
	}

	if res[0].Err != nil || res[2].Err != nil {
		t.Errorf("sibling errs not nil: res[0].Err=%v, res[2].Err=%v", res[0].Err, res[2].Err)
	}

	var panicErr *fanout.ErrTaskPanic
	if !errors.As(res[1].Err, &panicErr) {
		t.Fatalf("res[1].Err = %v (%T), want *fanout.ErrTaskPanic", res[1].Err, res[1].Err)
	}
	if panicErr.Recovered != "panic-tool-boom" {
		t.Errorf("Recovered = %v, want %q", panicErr.Recovered, "panic-tool-boom")
	}
	if len(panicErr.Stack) == 0 {
		t.Error("Stack is empty")
	}
}

func TestAsyncRunner_CtxCancel(t *testing.T) {
	tasks := []Task{
		{Tool: slowTool{name: "a"}, Args: json.RawMessage(`{}`)},
		{Tool: slowTool{name: "b"}, Args: json.RawMessage(`{}`)},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := NewAsyncRunner(0)
	res, err := r.Execute(ctx, tasks)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
	for i, tr := range res {
		if tr.Err != nil && !errors.Is(tr.Err, context.DeadlineExceeded) {
			t.Errorf("res[%d].Err = %v", i, tr.Err)
		}
	}
}
