package agents

import (
	"context"
	"errors"
	"testing"
)

func TestSimpleAgent_Run_TransparentlyForwards(t *testing.T) {
	llmMock := newScriptedLLM(textResp("hello world"))
	a := NewSimpleAgent(llmMock, SimpleOptions{SystemPrompt: "you are helpful"})

	res, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "hello world" {
		t.Errorf("Answer = %q", res.Answer)
	}
	if res.Usage.LLMCalls != 1 {
		t.Errorf("LLMCalls = %d, want 1", res.Usage.LLMCalls)
	}
	if len(res.Trace) != 1 || res.Trace[0].Kind != StepFinal {
		t.Errorf("Trace = %+v", res.Trace)
	}
}

func TestSimpleAgent_Run_EmptyInput(t *testing.T) {
	a := NewSimpleAgent(newScriptedLLM(), SimpleOptions{})
	_, err := a.Run(context.Background(), "")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v, want ErrEmptyInput", err)
	}
}

func TestSimpleAgent_RunStream_DeliversFinal(t *testing.T) {
	llmMock := newScriptedLLM(textResp("answer"))
	a := NewSimpleAgent(llmMock, SimpleOptions{})
	ch, err := a.RunStream(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	var events []StepEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) < 2 {
		t.Fatalf("events = %d", len(events))
	}
	last := events[len(events)-1]
	if !last.Done || last.Final == nil || last.Final.Answer != "answer" {
		t.Errorf("last event = %+v", last)
	}
}

func TestSimpleAgent_RunStream_CtxCancelClosesChannel(t *testing.T) {
	llmMock := newScriptedLLM(textResp("ok"))
	a := NewSimpleAgent(llmMock, SimpleOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := a.RunStream(ctx, "x")
	cancel()
	for range ch { // drain — channel must close, not deadlock
	}
}

func TestSimpleAgent_OnStep_Invoked(t *testing.T) {
	llmMock := newScriptedLLM(textResp("hi"))
	var got []Step
	a := NewSimpleAgent(llmMock, SimpleOptions{
		OnStep: func(s Step) { got = append(got, s) },
	})
	_, err := a.Run(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != StepFinal {
		t.Errorf("got = %v", got)
	}
}

func TestSimpleAgent_Name(t *testing.T) {
	a := NewSimpleAgent(newScriptedLLM(), SimpleOptions{Name: "custom"})
	if a.Name() != "custom" {
		t.Errorf("Name = %q", a.Name())
	}
	def := NewSimpleAgent(newScriptedLLM(), SimpleOptions{})
	if def.Name() != "simple" {
		t.Errorf("default Name = %q", def.Name())
	}
}
