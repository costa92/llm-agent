package orchestrate

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/costa92/llm-agent"
)

func parseTasks(plan agents.Result) ([]PlannedTask, error) {
	lines := strings.Split(strings.TrimSpace(plan.Answer), "\n")
	out := make([]PlannedTask, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			return nil, errors.New("invalid plan format")
		}
		out = append(out, PlannedTask{
			Name:   parts[0],
			Worker: parts[1],
			Input:  parts[2],
		})
	}
	return out, nil
}

func TestFanOutFanIn_Success(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "research-1|researcher|topic:a\nresearch-2|researcher|topic:b"
	}}
	worker := &stubAgent{name: "researcher", transform: func(s string) string { return "done:" + s }}
	aggregator := &stubAgent{name: "aggregator", transform: func(s string) string {
		if !strings.Contains(s, "done:topic:a") || !strings.Contains(s, "done:topic:b") {
			t.Fatalf("aggregator input missing worker results: %q", s)
		}
		return "summary"
	}}

	f := NewFanOutFanIn("research", FanOutFanInOptions{
		Planner:    planner,
		Workers:    map[string]agents.Agent{"researcher": worker},
		Aggregator: aggregator,
		ParsePlan:  parseTasks,
	})

	res, err := f.Run(context.Background(), "investigate")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.FinalAnswer != "summary" {
		t.Fatalf("FinalAnswer = %q, want summary", res.FinalAnswer)
	}
	if len(res.WorkerResults) != 2 {
		t.Fatalf("got %d worker results, want 2", len(res.WorkerResults))
	}
	if res.WorkerResults[0].Task.Name != "research-1" || res.WorkerResults[1].Task.Name != "research-2" {
		t.Fatalf("worker result order not preserved: %+v", res.WorkerResults)
	}
	if res.TotalUsage.LLMCalls != 4 {
		t.Fatalf("LLMCalls = %d, want 4", res.TotalUsage.LLMCalls)
	}
}

func TestFanOutFanIn_DefaultSingleWorkerRouting(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "t1||alpha"
	}}
	worker := &stubAgent{name: "solo", transform: func(s string) string { return "w:" + s }}
	aggregator := &stubAgent{name: "aggregator", transform: func(s string) string { return s }}

	f := NewFanOutFanIn("", FanOutFanInOptions{
		Planner:    planner,
		Workers:    map[string]agents.Agent{"solo": worker},
		Aggregator: aggregator,
		ParsePlan:  parseTasks,
	})

	res, err := f.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := res.WorkerResults[0].Result.Answer; got != "w:alpha" {
		t.Fatalf("worker result = %q, want w:alpha", got)
	}
	if f.Name() != "fanout-fanin" {
		t.Fatalf("default name = %q", f.Name())
	}
}

func TestFanOutFanIn_DefaultParserAndRoundRobinRouting(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "alpha\nbeta\ncharlie"
	}}
	w1 := &stubAgent{name: "a", transform: func(s string) string { return "a:" + s }}
	w2 := &stubAgent{name: "b", transform: func(s string) string { return "b:" + s }}
	aggregator := &stubAgent{name: "agg", transform: func(s string) string { return s }}

	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"b": w2,
			"a": w1,
		},
		Aggregator: aggregator,
	})
	res, err := f.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := res.WorkerResults[0].Result.Answer; got != "a:alpha" {
		t.Fatalf("first worker result = %q, want a:alpha", got)
	}
	if got := res.WorkerResults[1].Result.Answer; got != "b:beta" {
		t.Fatalf("second worker result = %q, want b:beta", got)
	}
	if got := res.WorkerResults[2].Result.Answer; got != "a:charlie" {
		t.Fatalf("third worker result = %q, want a:charlie", got)
	}
}

func TestFanOutFanIn_PlannerError(t *testing.T) {
	want := errors.New("planner boom")
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner:    &stubAgent{name: "planner", err: want},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "aggregator"},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if !errors.Is(err, want) {
		t.Fatalf("expected planner error, got %v", err)
	}
}

func TestFanOutFanIn_ParseError(t *testing.T) {
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner:    &stubAgent{name: "planner", transform: func(string) string { return "bad-plan" }},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "aggregator"},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "parse plan") {
		t.Fatalf("expected parse-plan error, got %v", err)
	}
}

func TestFanOutFanIn_EmptyTasks(t *testing.T) {
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner:    &stubAgent{name: "planner", transform: func(string) string { return "" }},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "aggregator"},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if !errors.Is(err, ErrFanOutNoTasks) {
		t.Fatalf("expected ErrFanOutNoTasks, got %v", err)
	}
}

func TestFanOutFanIn_WorkerError(t *testing.T) {
	want := errors.New("worker boom")
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner: &stubAgent{name: "planner", transform: func(string) string {
			return "t1|w|alpha"
		}},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w", err: want}},
		Aggregator: &stubAgent{name: "aggregator"},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if !errors.Is(err, want) {
		t.Fatalf("expected worker error, got %v", err)
	}
}

func TestFanOutFanIn_AggregatorError(t *testing.T) {
	want := errors.New("agg boom")
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner: &stubAgent{name: "planner", transform: func(string) string {
			return "t1|w|alpha"
		}},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "agg", err: want},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if !errors.Is(err, want) {
		t.Fatalf("expected aggregator error, got %v", err)
	}
}

func TestFanOutFanIn_UnknownWorker(t *testing.T) {
	f := NewFanOutFanIn("x", FanOutFanInOptions{
		Planner: &stubAgent{name: "planner", transform: func(string) string {
			return "t1|ghost|alpha"
		}},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "agg"},
		ParsePlan:  parseTasks,
	})
	_, err := f.Run(context.Background(), "x")
	if !errors.Is(err, ErrFanOutUnknownWorker) {
		t.Fatalf("expected ErrFanOutUnknownWorker, got %v", err)
	}
}

func TestFanOutFanIn_NilPrerequisites(t *testing.T) {
	_, err := NewFanOutFanIn("x", FanOutFanInOptions{}).Run(context.Background(), "x")
	if !errors.Is(err, ErrFanOutNilPlanner) {
		t.Fatalf("expected ErrFanOutNilPlanner, got %v", err)
	}

	_, err = NewFanOutFanIn("x", FanOutFanInOptions{
		Planner:    &stubAgent{name: "planner"},
		Workers:    map[string]agents.Agent{"w": &stubAgent{name: "w"}},
		Aggregator: &stubAgent{name: "agg"},
	}).Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("default parser should allow nil ParsePlan, got %v", err)
	}
}

// orderedFailWorker: 第 0 个 task 立即报错,其余 task 阻塞 ctx.Done。
// 用 atomic counter 校验 sibling 是否被 fail-fast cancel 提前退出。
type orderedFailWorker struct {
	name      string
	bootCount *atomic.Int32 // 每次进入 Run 自增,用来判断有几个 task 实际启动了
	doneCount *atomic.Int32 // 完成完整 2s 等待的 task 数(理想 = 0)
}

func (w *orderedFailWorker) Name() string { return w.name }
func (w *orderedFailWorker) Run(ctx context.Context, input string) (agents.Result, error) {
	idx := w.bootCount.Add(1) - 1
	if idx == 0 {
		return agents.Result{}, errors.New("idx-0 boom")
	}
	select {
	case <-ctx.Done():
		return agents.Result{}, ctx.Err()
	case <-time.After(2 * time.Second):
		w.doneCount.Add(1)
		return agents.Result{Answer: "ok", Usage: agents.Usage{LLMCalls: 1}}, nil
	}
}
func (w *orderedFailWorker) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("orderedFailWorker: stream not implemented")
}

func TestFanOutFanIn_WorkerFailureAbortsRemainingViaFailFast(t *testing.T) {
	var bootCount, doneCount atomic.Int32
	worker := &orderedFailWorker{name: "fw", bootCount: &bootCount, doneCount: &doneCount}

	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "t1|fw|in1\nt2|fw|in2\nt3|fw|in3\nt4|fw|in4\nt5|fw|in5"
	}}
	aggregator := &stubAgent{name: "aggregator", transform: func(s string) string { return "agg:" + s }}

	f := NewFanOutFanIn("ff", FanOutFanInOptions{
		Planner:        planner,
		Workers:        map[string]agents.Agent{"fw": worker},
		Aggregator:     aggregator,
		ParsePlan:      parseTasks,
		MaxConcurrency: 5,
	})

	_, err := f.Run(context.Background(), "input")
	if err == nil {
		t.Fatal("expected non-nil err from runWorkers, got nil")
	}
	if !strings.Contains(err.Error(), "idx-0 boom") {
		t.Errorf("err = %v, want containing %q", err, "idx-0 boom")
	}
	if !strings.Contains(err.Error(), `fanout "ff": worker "fw"`) {
		t.Errorf("err = %v, want fanout/worker wrap", err)
	}
	if dc := doneCount.Load(); dc != 0 {
		t.Errorf("doneCount = %d, want 0 (siblings should have been cancelled)", dc)
	}
}

// alwaysWaitWorker: 任何输入都阻塞 ctx.Done 后返回 ctx.Err(永不主动错)
type alwaysWaitWorker struct{ name string }

func (w *alwaysWaitWorker) Name() string { return w.name }
func (w *alwaysWaitWorker) Run(ctx context.Context, _ string) (agents.Result, error) {
	<-ctx.Done()
	return agents.Result{}, ctx.Err()
}
func (w *alwaysWaitWorker) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("alwaysWaitWorker: stream not implemented")
}

func TestFanOutFanIn_OuterCtxCancelledReturnsCtxErr(t *testing.T) {
	planner := &stubAgent{name: "planner", transform: func(string) string {
		return "t1|w|in1\nt2|w|in2\nt3|w|in3"
	}}
	worker := &alwaysWaitWorker{name: "w"}
	aggregator := &stubAgent{name: "agg"}

	f := NewFanOutFanIn("ff", FanOutFanInOptions{
		Planner:    planner,
		Workers:    map[string]agents.Agent{"w": worker},
		Aggregator: aggregator,
		ParsePlan:  parseTasks,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消

	_, err := f.Run(ctx, "input")
	if err == nil {
		t.Fatal("expected non-nil err for cancelled ctx, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want errors.Is context.Canceled", err)
	}
}
