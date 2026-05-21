package agentstest

import (
	"context"
	"encoding/json"
	"errors"

	agents "github.com/costa92/llm-agent"
)

// defaultSchema is the empty-but-valid object schema used when a
// StubTool leaves SchemaValue unset.
var defaultSchema = json.RawMessage(`{"type":"object"}`)

// StubTool is a configurable agents.Tool for tests. Zero value is
// usable: Name()="stub", Description()="", Schema()={"type":"object"},
// Execute returns ("ok", nil).
//
// Set ExecuteFn to drive Execute from a closure (for input-dependent
// behavior). When ExecuteFn is nil, Execute returns OutputValue (or
// "ok" if that is also empty).
//
// StubTool is a value type — copies are independent. Use a pointer
// receiver only if you embed it in a larger struct and need stable
// identity.
type StubTool struct {
	NameValue        string
	DescriptionValue string
	SchemaValue      json.RawMessage
	ExecuteFn        func(ctx context.Context, args json.RawMessage) (string, error)
	OutputValue      string
}

// Name implements agents.Tool.
func (s StubTool) Name() string {
	if s.NameValue == "" {
		return "stub"
	}
	return s.NameValue
}

// Description implements agents.Tool.
func (s StubTool) Description() string { return s.DescriptionValue }

// Schema implements agents.Tool.
func (s StubTool) Schema() json.RawMessage {
	if len(s.SchemaValue) == 0 {
		return defaultSchema
	}
	return s.SchemaValue
}

// Execute implements agents.Tool.
func (s StubTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if s.ExecuteFn != nil {
		return s.ExecuteFn(ctx, args)
	}
	if s.OutputValue != "" {
		return s.OutputValue, nil
	}
	return "ok", nil
}

// Compile-time assertion: StubTool satisfies agents.Tool.
var _ agents.Tool = StubTool{}

// NewStubTool is a convenience constructor for the common case of a
// stub that returns a fixed string from Execute.
//
//	tool := agentstest.NewStubTool("lookup", "row found")
func NewStubTool(name, output string) agents.Tool {
	return StubTool{NameValue: name, OutputValue: output}
}

// NewErrorTool returns a Tool whose Execute always fails with the
// given message. Useful for testing tool-error paths in agent loops.
//
//	tool := agentstest.NewErrorTool("flaky", "upstream timeout")
//	// agent.Run(...) -> error contains "upstream timeout"
func NewErrorTool(name, errMsg string) agents.Tool {
	return StubTool{
		NameValue: name,
		ExecuteFn: func(_ context.Context, _ json.RawMessage) (string, error) {
			return "", errors.New(errMsg)
		},
	}
}
