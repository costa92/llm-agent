package orchestrate

import "testing"

func TestMaxTurns(t *testing.T) {
	term := MaxTurns(3)
	if term.ShouldStop(nil) {
		t.Errorf("empty history should not stop")
	}
	if term.ShouldStop([]Message{{}, {}}) {
		t.Errorf("2 messages should not stop (need 3)")
	}
	if !term.ShouldStop([]Message{{}, {}, {}}) {
		t.Errorf("3 messages should stop")
	}
	if !term.ShouldStop([]Message{{}, {}, {}, {}}) {
		t.Errorf("4 messages should stop")
	}
}

func TestTextMatch(t *testing.T) {
	term := TextMatch("<TASK_DONE>")
	if term.ShouldStop(nil) {
		t.Errorf("empty history should not stop")
	}
	if term.ShouldStop([]Message{{Content: "hello world"}}) {
		t.Errorf("no marker should not stop")
	}
	if !term.ShouldStop([]Message{{Content: "Result: <TASK_DONE>"}}) {
		t.Errorf("exact-case marker should stop")
	}
	if !term.ShouldStop([]Message{{Content: "Result: <task_done>"}}) {
		t.Errorf("case-insensitive marker should stop")
	}
	if !term.ShouldStop([]Message{{Content: "first"}, {Content: "<task_done> here"}}) {
		t.Errorf("any message containing marker should stop")
	}
}

func TestTextMatch_EmptyMarkerNeverStops(t *testing.T) {
	term := TextMatch("")
	if term.ShouldStop([]Message{{Content: "anything"}}) {
		t.Errorf("empty marker should never stop")
	}
}

func TestAnd_AllMustWantStop(t *testing.T) {
	term := And(MaxTurns(2), TextMatch("done"))
	// 2 messages without marker → MaxTurns wants stop, TextMatch doesn't → no stop
	if term.ShouldStop([]Message{{}, {}}) {
		t.Errorf("And: not all want stop yet")
	}
	// 2 messages WITH marker → both want stop → stop
	if !term.ShouldStop([]Message{{}, {Content: "done"}}) {
		t.Errorf("And: all want stop, should stop")
	}
}

func TestAnd_EmptyNeverStops(t *testing.T) {
	if And().ShouldStop([]Message{{}, {}, {}}) {
		t.Error("empty And should never stop")
	}
}

func TestOr_AnyTriggersStop(t *testing.T) {
	term := Or(MaxTurns(10), TextMatch("done"))
	// 1 message with marker → TextMatch stops → stop
	if !term.ShouldStop([]Message{{Content: "done"}}) {
		t.Errorf("Or: any wants stop, should stop")
	}
	// 5 messages without marker, < MaxTurns → neither stops → no stop
	if term.ShouldStop([]Message{{}, {}, {}, {}, {}}) {
		t.Errorf("Or: none wants stop")
	}
}

func TestOr_EmptyNeverStops(t *testing.T) {
	if Or().ShouldStop([]Message{{}}) {
		t.Error("empty Or should never stop")
	}
}
