package rl

import (
	"context"
	"strings"

	"github.com/costa92/llm-agent"
)

// Reward scores a Trajectory against its Sample. Stateless (the
// scoring doesn't update gradients — that's training, out of scope).
type Reward interface {
	Score(ctx context.Context, traj Trajectory, sample Sample) (float64, error)
	Name() string
}

// --- AccuracyReward --------------------------------------------------------

// NewAccuracyReward returns a Reward that gives 1.0 when the extracted
// numeric answer in traj.Answer matches sample.GroundTruth, else 0.0.
// Match strips commas, whitespace, and trailing periods.
func NewAccuracyReward() Reward { return accuracyReward{} }

type accuracyReward struct{}

func (accuracyReward) Name() string { return "accuracy" }

func (accuracyReward) Score(_ context.Context, traj Trajectory, sample Sample) (float64, error) {
	if traj.Err != nil {
		return 0, nil
	}
	a := normalizeAnswer(extractFinalAnswer(traj.Answer))
	g := normalizeAnswer(sample.GroundTruth)
	if a == "" || g == "" {
		return 0, nil
	}
	if a == g {
		return 1, nil
	}
	return 0, nil
}

// extractFinalAnswer pulls the answer after "Final Answer:" /
// "answer:" markers (case-insensitive). Falls back to the last line.
func extractFinalAnswer(s string) string {
	lower := strings.ToLower(s)
	for _, marker := range []string{"final answer:", "answer:", "####"} {
		if idx := strings.LastIndex(lower, marker); idx >= 0 {
			return strings.TrimSpace(s[idx+len(marker):])
		}
	}
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func normalizeAnswer(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// --- LengthPenalty ---------------------------------------------------------

// NewLengthPenalty returns a Reward that subtracts weight × max(0,
// len(answer)-maxLen). Discourages verbose answers.
func NewLengthPenalty(maxLen int, weight float64) Reward {
	return lengthPenalty{maxLen: maxLen, weight: weight}
}

type lengthPenalty struct {
	maxLen int
	weight float64
}

func (lengthPenalty) Name() string { return "length_penalty" }

func (l lengthPenalty) Score(_ context.Context, traj Trajectory, _ Sample) (float64, error) {
	excess := len([]rune(traj.Answer)) - l.maxLen
	if excess <= 0 {
		return 0, nil
	}
	return -l.weight * float64(excess), nil
}

// --- StepBonus -------------------------------------------------------------

// NewStepBonus returns a Reward that adds bonus × #thought-steps.
// Encourages multi-step reasoning chains.
func NewStepBonus(bonus float64) Reward {
	return stepBonus{bonus: bonus}
}

type stepBonus struct{ bonus float64 }

func (stepBonus) Name() string { return "step_bonus" }

func (s stepBonus) Score(_ context.Context, traj Trajectory, _ Sample) (float64, error) {
	thoughtSteps := 0
	for _, st := range traj.Steps {
		if st.Kind == agents.StepThought {
			thoughtSteps++
		}
	}
	return s.bonus * float64(thoughtSteps), nil
}

// --- Composite -------------------------------------------------------------

// NewComposite sums multiple Rewards. First-error short-circuits.
func NewComposite(rewards ...Reward) Reward {
	return composite{rs: rewards}
}

type composite struct{ rs []Reward }

func (composite) Name() string { return "composite" }

func (c composite) Score(ctx context.Context, traj Trajectory, sample Sample) (float64, error) {
	var sum float64
	for _, r := range c.rs {
		s, err := r.Score(ctx, traj, sample)
		if err != nil {
			return 0, err
		}
		sum += s
	}
	return sum, nil
}
