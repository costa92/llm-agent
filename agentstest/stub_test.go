package agentstest_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	agents "github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent/agentstest"
)

func TestStubTool_ZeroValueDefaults(t *testing.T) {
	var tool agentstest.StubTool

	if got := tool.Name(); got != "stub" {
		t.Errorf("Name() = %q, want %q", got, "stub")
	}
	if got := tool.Description(); got != "" {
		t.Errorf("Description() = %q, want empty", got)
	}
	if got := string(tool.Schema()); got != `{"type":"object"}` {
		t.Errorf("Schema() = %q, want default object schema", got)
	}
	out, err := tool.Execute(context.Background(), nil)
	if err != nil || out != "ok" {
		t.Errorf("Execute() = (%q, %v), want (\"ok\", nil)", out, err)
	}
}

func TestStubTool_CustomFields(t *testing.T) {
	tool := agentstest.StubTool{
		NameValue:        "lookup",
		DescriptionValue: "do a lookup",
		SchemaValue:      json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
		OutputValue:      "found",
	}
	if tool.Name() != "lookup" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() != "do a lookup" {
		t.Errorf("Description() = %q", tool.Description())
	}
	if !strings.Contains(string(tool.Schema()), `"q"`) {
		t.Errorf("Schema() lost properties: %s", tool.Schema())
	}
	out, _ := tool.Execute(context.Background(), nil)
	if out != "found" {
		t.Errorf("Execute() = %q, want %q", out, "found")
	}
}

func TestStubTool_ExecuteFnDrivesOutput(t *testing.T) {
	tool := agentstest.StubTool{
		ExecuteFn: func(_ context.Context, args json.RawMessage) (string, error) {
			var p struct {
				S string `json:"s"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", err
			}
			return strings.ToUpper(p.S), nil
		},
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"s":"hello"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out != "HELLO" {
		t.Errorf("Execute() = %q, want HELLO", out)
	}
}

func TestNewStubTool(t *testing.T) {
	tool := agentstest.NewStubTool("ping", "pong")
	if tool.Name() != "ping" {
		t.Errorf("Name() = %q", tool.Name())
	}
	out, err := tool.Execute(context.Background(), nil)
	if err != nil || out != "pong" {
		t.Errorf("Execute() = (%q, %v)", out, err)
	}
}

func TestNewErrorTool(t *testing.T) {
	tool := agentstest.NewErrorTool("flaky", "upstream timeout")
	out, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "upstream timeout") {
		t.Errorf("err = %v, want to contain %q", err, "upstream timeout")
	}
	if out != "" {
		t.Errorf("Execute() out = %q, want empty", out)
	}
}

func TestStubTool_SatisfiesAgentsToolInterface(t *testing.T) {
	// Compile-time assertion already exists in stub.go; this exercise
	// confirms a StubTool value flows through agents.AsLLMTool (the
	// downstream consumer most likely to break).
	var tool agents.Tool = agentstest.NewStubTool("trans", "out")
	llmTool := agents.AsLLMTool(tool)
	if llmTool.Name != "trans" {
		t.Errorf("AsLLMTool name = %q, want trans", llmTool.Name)
	}
}

func TestRecordingTool_DelegatesAndRecords(t *testing.T) {
	rec := agentstest.NewRecordingTool(agentstest.NewStubTool("upper", "OK"))

	if rec.Name() != "upper" {
		t.Errorf("Name() = %q, want upper (delegated)", rec.Name())
	}

	out1, err := rec.Execute(context.Background(), json.RawMessage(`{"q":"first"}`))
	if err != nil || out1 != "OK" {
		t.Fatalf("Execute 1 = (%q, %v)", out1, err)
	}
	out2, err := rec.Execute(context.Background(), json.RawMessage(`{"q":"second"}`))
	if err != nil || out2 != "OK" {
		t.Fatalf("Execute 2 = (%q, %v)", out2, err)
	}

	calls := rec.Calls()
	if len(calls) != 2 {
		t.Fatalf("Calls() len = %d, want 2", len(calls))
	}
	if !strings.Contains(string(calls[0].Args), `"first"`) {
		t.Errorf("Calls[0].Args = %s, want to contain \"first\"", calls[0].Args)
	}
	if !strings.Contains(string(calls[1].Args), `"second"`) {
		t.Errorf("Calls[1].Args = %s, want to contain \"second\"", calls[1].Args)
	}
	if calls[0].Out != "OK" || calls[1].Out != "OK" {
		t.Errorf("Calls outputs = (%q, %q), want both OK", calls[0].Out, calls[1].Out)
	}
}

func TestRecordingTool_PropagatesError(t *testing.T) {
	rec := agentstest.NewRecordingTool(agentstest.NewErrorTool("flaky", "boom"))

	_, err := rec.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Execute err = %v, want to contain \"boom\"", err)
	}
	calls := rec.Calls()
	if len(calls) != 1 {
		t.Fatalf("Calls() len = %d, want 1", len(calls))
	}
	if calls[0].Err == nil || !strings.Contains(calls[0].Err.Error(), "boom") {
		t.Errorf("Calls[0].Err = %v, want to contain \"boom\"", calls[0].Err)
	}
}

func TestRecordingTool_Reset(t *testing.T) {
	rec := agentstest.NewRecordingTool(agentstest.NewStubTool("t", "x"))
	_, _ = rec.Execute(context.Background(), nil)
	_, _ = rec.Execute(context.Background(), nil)
	if got := len(rec.Calls()); got != 2 {
		t.Fatalf("len(Calls()) before reset = %d, want 2", got)
	}
	rec.Reset()
	if got := len(rec.Calls()); got != 0 {
		t.Fatalf("len(Calls()) after reset = %d, want 0", got)
	}
}

func TestRecordingTool_ConcurrentSafe(t *testing.T) {
	rec := agentstest.NewRecordingTool(agentstest.NewStubTool("t", "x"))

	const n = 64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = rec.Execute(context.Background(), json.RawMessage(`{}`))
		}()
	}
	wg.Wait()

	if got := len(rec.Calls()); got != n {
		t.Errorf("len(Calls()) = %d, want %d", got, n)
	}
}

func TestRecordingTool_CallsReturnsSnapshot(t *testing.T) {
	rec := agentstest.NewRecordingTool(agentstest.NewStubTool("t", "x"))
	_, _ = rec.Execute(context.Background(), json.RawMessage(`{}`))

	snap := rec.Calls()
	if len(snap) != 1 {
		t.Fatalf("len(snap) = %d, want 1", len(snap))
	}
	snap[0].Out = "MUTATED"

	// Mutating the snapshot must not affect future Calls() returns.
	if got := rec.Calls()[0].Out; got != "x" {
		t.Errorf("after mutating snapshot, internal Calls()[0].Out = %q, want %q", got, "x")
	}
}
