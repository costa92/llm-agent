package agents

import (
	"errors"
	"fmt"
	"testing"
)

// Compile-time interface conformance check for all 5 Agents.
var (
	_ Agent = (*SimpleAgent)(nil)
	_ Agent = (*ReActAgent)(nil)
	_ Agent = (*ReflectionAgent)(nil)
	_ Agent = (*PlanAndSolveAgent)(nil)
	_ Agent = (*FunctionCallAgent)(nil)
)

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	cases := []error{
		ErrMaxStepsExceeded,
		ErrToolNotFound,
		ErrToolAlreadyRegistered,
		ErrPlanningFailed,
		ErrParseToolCall,
		ErrEmptyInput,
	}
	for _, sentinel := range cases {
		wrapped := fmt.Errorf("wrap: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is should match %v through wrap", sentinel)
		}
	}
}

func TestStepKind_Constants(t *testing.T) {
	want := map[StepKind]string{
		StepThought:     "thought",
		StepAction:      "action",
		StepObservation: "observation",
		StepReflection:  "reflection",
		StepPlan:        "plan",
		StepFinal:       "final",
	}
	for k, s := range want {
		if string(k) != s {
			t.Errorf("StepKind %q != %q", string(k), s)
		}
	}
}
