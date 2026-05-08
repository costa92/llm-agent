package orchestrate

import "strings"

// Message is one turn in a multi-Agent conversation. Used by RoundRobinChat,
// RolePlay, and any Termination implementation that inspects history.
type Message struct {
	Speaker string
	Content string
}

// Termination decides when a multi-Agent loop should stop. Implementations
// receive the full history-so-far and return true to stop.
type Termination interface {
	ShouldStop(history []Message) bool
}

// MaxTurns stops after the history reaches n messages.
func MaxTurns(n int) Termination { return maxTurns(n) }

type maxTurns int

func (m maxTurns) ShouldStop(history []Message) bool { return len(history) >= int(m) }

// TextMatch stops when any message's content contains the marker
// (case-insensitive). Useful for "<TASK_DONE>" sentinels.
func TextMatch(marker string) Termination { return textMatch(strings.ToLower(marker)) }

type textMatch string

func (t textMatch) ShouldStop(history []Message) bool {
	if t == "" {
		return false
	}
	for _, m := range history {
		if strings.Contains(strings.ToLower(m.Content), string(t)) {
			return true
		}
	}
	return false
}

// And combines terminations: stops only when ALL of them want to stop.
// Empty And never stops.
func And(ts ...Termination) Termination { return and(ts) }

type and []Termination

func (a and) ShouldStop(history []Message) bool {
	if len(a) == 0 {
		return false
	}
	for _, t := range a {
		if !t.ShouldStop(history) {
			return false
		}
	}
	return true
}

// Or combines terminations: stops when ANY of them wants to stop.
// Empty Or never stops.
func Or(ts ...Termination) Termination { return or(ts) }

type or []Termination

func (o or) ShouldStop(history []Message) bool {
	for _, t := range o {
		if t.ShouldStop(history) {
			return true
		}
	}
	return false
}
