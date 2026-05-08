package rl

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/costa92/llm-agent"
)

// --- ParseGSM8K -----------------------------------------------------------

func TestParseGSM8K_HappyPath(t *testing.T) {
	line := []byte(`{"question":"What is 2+2?","answer":"Adding... #### 4"}`)
	s, err := ParseGSM8K(line)
	if err != nil {
		t.Fatalf("ParseGSM8K: %v", err)
	}
	if s.Prompt != "What is 2+2?" {
		t.Errorf("Prompt = %q", s.Prompt)
	}
	if s.GroundTruth != "4" {
		t.Errorf("GroundTruth = %q, want 4", s.GroundTruth)
	}
	if s.ID == "" {
		t.Error("ID empty")
	}
}

func TestParseGSM8K_StripsCommasInGroundTruth(t *testing.T) {
	line := []byte(`{"question":"q","answer":"work #### 1,234"}`)
	s, _ := ParseGSM8K(line)
	if s.GroundTruth != "1234" {
		t.Errorf("GroundTruth = %q, want 1234 (commas stripped)", s.GroundTruth)
	}
}

func TestParseGSM8K_RejectsMissingMarker(t *testing.T) {
	line := []byte(`{"question":"q","answer":"no marker here"}`)
	if _, err := ParseGSM8K(line); err == nil {
		t.Error("expected error for missing #### marker")
	}
}

func TestParseGSM8K_RejectsBadJSON(t *testing.T) {
	if _, err := ParseGSM8K([]byte(`not json`)); err == nil {
		t.Error("expected error for bad JSON")
	}
}

// --- JSONLDataset ---------------------------------------------------------

func writeTempJSONL(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestJSONLDataset_IterAndLen(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"question":"q1","answer":"#### 1"}`,
		`{"question":"q2","answer":"#### 2"}`,
		`{"question":"q3","answer":"#### 3"}`,
	})
	ds, err := NewJSONLDataset("test", path, ParseGSM8K)
	if err != nil {
		t.Fatalf("NewJSONLDataset: %v", err)
	}
	if ds.Len() != 3 {
		t.Errorf("Len = %d, want 3", ds.Len())
	}
	got := []Sample{}
	for s := range ds.Iter(context.Background()) {
		got = append(got, s)
	}
	if len(got) != 3 {
		t.Errorf("got %d samples, want 3", len(got))
	}
}

func TestJSONLDataset_SkipsBadLines(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"question":"q1","answer":"#### 1"}`,
		`malformed`,
		`{"question":"q2","answer":"#### 2"}`,
		``, // blank
	})
	ds, _ := NewJSONLDataset("test", path, ParseGSM8K)
	got := 0
	for range ds.Iter(context.Background()) {
		got++
	}
	if got != 2 {
		t.Errorf("got %d samples, want 2 (1 malformed + 1 blank dropped)", got)
	}
}

func TestJSONLDataset_RejectsMissingFile(t *testing.T) {
	_, err := NewJSONLDataset("nope", "/nonexistent/file.jsonl", ParseGSM8K)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestJSONLDataset_RejectsNilParser(t *testing.T) {
	path := writeTempJSONL(t, []string{`{}`})
	_, err := NewJSONLDataset("x", path, nil)
	if err == nil {
		t.Error("expected error for nil parser")
	}
}

// --- Reward functions -----------------------------------------------------

func TestAccuracyReward_Match(t *testing.T) {
	r := NewAccuracyReward()
	score, _ := r.Score(context.Background(),
		Trajectory{Answer: "Final Answer: 42"},
		Sample{GroundTruth: "42"},
	)
	if score != 1 {
		t.Errorf("score = %f, want 1", score)
	}
}

func TestAccuracyReward_Mismatch(t *testing.T) {
	r := NewAccuracyReward()
	score, _ := r.Score(context.Background(),
		Trajectory{Answer: "Final Answer: 41"},
		Sample{GroundTruth: "42"},
	)
	if score != 0 {
		t.Errorf("score = %f, want 0", score)
	}
}

func TestAccuracyReward_NormalizesCommas(t *testing.T) {
	r := NewAccuracyReward()
	score, _ := r.Score(context.Background(),
		Trajectory{Answer: "Final Answer: 1,234"},
		Sample{GroundTruth: "1234"},
	)
	if score != 1 {
		t.Errorf("score = %f, want 1 (commas should normalize)", score)
	}
}

func TestLengthPenalty_OverLimit(t *testing.T) {
	r := NewLengthPenalty(10, 0.1)
	score, _ := r.Score(context.Background(),
		Trajectory{Answer: strings.Repeat("a", 30)}, Sample{},
	)
	if score >= 0 {
		t.Errorf("score = %f, want negative", score)
	}
}

func TestLengthPenalty_UnderLimit(t *testing.T) {
	r := NewLengthPenalty(100, 0.1)
	score, _ := r.Score(context.Background(),
		Trajectory{Answer: "short"}, Sample{},
	)
	if score != 0 {
		t.Errorf("score = %f, want 0 (under limit)", score)
	}
}

func TestStepBonus_CountsThoughts(t *testing.T) {
	r := NewStepBonus(0.5)
	traj := Trajectory{Steps: []agents.Step{
		{Kind: agents.StepThought},
		{Kind: agents.StepAction},
		{Kind: agents.StepThought},
		{Kind: agents.StepFinal},
	}}
	score, _ := r.Score(context.Background(), traj, Sample{})
	if score != 1.0 { // 2 thoughts × 0.5
		t.Errorf("score = %f, want 1.0", score)
	}
}

func TestComposite_Sums(t *testing.T) {
	r := NewComposite(
		NewAccuracyReward(),       // +1 (matches GroundTruth=42)
		NewLengthPenalty(10, 0.1), // -2 (excess=20, weight=0.1)
		NewStepBonus(0.25),        // +0.5 (2 thoughts)
	)
	// Build answer so that extracted-final-answer is "42" but length is 30:
	// padding goes BEFORE the marker so "Final Answer: 42" stays at the end.
	traj := Trajectory{
		Answer: strings.Repeat("x", 14) + "Final Answer: 42", // 30 chars total
		Steps:  []agents.Step{{Kind: agents.StepThought}, {Kind: agents.StepThought}},
	}
	sample := Sample{GroundTruth: "42"}
	score, _ := r.Score(context.Background(), traj, sample)
	// 1 + (-2) + 0.5 = -0.5
	if score < -0.6 || score > -0.4 {
		t.Errorf("composite score = %f, want ≈ -0.5", score)
	}
}

// --- metrics --------------------------------------------------------------

func TestComputeAccuracy_Mixed(t *testing.T) {
	samples := []Sample{
		{ID: "s1", GroundTruth: "1"},
		{ID: "s2", GroundTruth: "2"},
		{ID: "s3", GroundTruth: "3"},
	}
	trajs := []Trajectory{
		{SampleID: "s1", Answer: "Final Answer: 1"}, // hit
		{SampleID: "s2", Answer: "Final Answer: 2"}, // hit
		{SampleID: "s3", Answer: "Final Answer: X"}, // miss
	}
	if got := ComputeAccuracy(trajs, samples); got < 0.66 || got > 0.67 {
		t.Errorf("accuracy = %f, want 0.667", got)
	}
}

func TestComputePassAtK_MultipleSamplesPerTask(t *testing.T) {
	samples := []Sample{{ID: "s1", GroundTruth: "42"}, {ID: "s2", GroundTruth: "100"}}
	trajsByID := map[string][]Trajectory{
		"s1": {
			{SampleID: "s1", Answer: "Final Answer: wrong"},
			{SampleID: "s1", Answer: "Final Answer: 42"}, // pass at K=2
		},
		"s2": {
			{SampleID: "s2", Answer: "Final Answer: 99"},
			{SampleID: "s2", Answer: "Final Answer: 101"}, // both wrong
		},
	}
	if got := ComputePassAtK(trajsByID, samples, 2); got != 0.5 {
		t.Errorf("pass@2 = %f, want 0.5", got)
	}
}

func TestComputeFormatCorrectness(t *testing.T) {
	v := func(s string) bool { return strings.Contains(s, "Final Answer:") }
	trajs := []Trajectory{
		{Answer: "Final Answer: 42"},
		{Answer: "blah blah"},
		{Answer: "Final Answer: 3"},
	}
	if got := ComputeFormatCorrectness(trajs, v); got < 0.66 || got > 0.67 {
		t.Errorf("format = %f, want ~0.67", got)
	}
}

func TestComputeNumericError(t *testing.T) {
	samples := []Sample{
		{ID: "s1", GroundTruth: "10"},
		{ID: "s2", GroundTruth: "20"},
	}
	trajs := []Trajectory{
		{SampleID: "s1", Answer: "Final Answer: 12"}, // |12-10| = 2
		{SampleID: "s2", Answer: "Final Answer: 25"}, // |25-20| = 5
	}
	mean, p50, p95 := ComputeNumericError(trajs, samples)
	if mean != 3.5 {
		t.Errorf("mean = %f, want 3.5", mean)
	}
	if p50 != 5 || p95 != 5 {
		t.Errorf("p50/p95 = %f/%f", p50, p95)
	}
}

// --- Evaluator ------------------------------------------------------------

// stubAgent returns a fixed answer + records call count.
type stubAgent struct {
	name      string
	transform func(input string) string
	calls     atomic.Int32
}

func (a *stubAgent) Name() string { return a.name }
func (a *stubAgent) Run(_ context.Context, input string) (agents.Result, error) {
	a.calls.Add(1)
	out := input
	if a.transform != nil {
		out = a.transform(input)
	}
	return agents.Result{
		Answer: out,
		Trace:  []agents.Step{{Kind: agents.StepFinal, Content: out}},
	}, nil
}
func (a *stubAgent) RunStream(_ context.Context, _ string) (<-chan agents.StepEvent, error) {
	return nil, errors.New("not impl")
}

func TestEvaluator_Run(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"question":"What is 2+2?","answer":"#### 4"}`,
		`{"question":"What is 3+3?","answer":"#### 6"}`,
	})
	ds, _ := NewJSONLDataset("test", path, ParseGSM8K)

	agent := &stubAgent{
		name:      "stub",
		transform: func(input string) string { return "Final Answer: 4" }, // always says 4
	}
	ev := NewEvaluator(agent, NewAccuracyReward(), EvaluatorOptions{Concurrency: 2})

	metrics, trajs, err := ev.Run(context.Background(), ds)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if metrics.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", metrics.SampleCount)
	}
	if metrics.Accuracy != 0.5 {
		t.Errorf("Accuracy = %f, want 0.5 (1 of 2 right)", metrics.Accuracy)
	}
	if len(trajs) != 2 {
		t.Errorf("trajs count = %d, want 2", len(trajs))
	}
	if agent.calls.Load() != 2 {
		t.Errorf("agent call count = %d, want 2", agent.calls.Load())
	}
}

func TestEvaluator_PassAtK(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"question":"What is 2+2?","answer":"#### 4"}`,
	})
	ds, _ := NewJSONLDataset("t", path, ParseGSM8K)

	// Agent flips between right and wrong on alternating calls.
	agent := &stubAgent{name: "flaky"}
	count := atomic.Int32{}
	agent.transform = func(_ string) string {
		c := count.Add(1)
		if c%2 == 0 {
			return "Final Answer: 4"
		}
		return "Final Answer: 3"
	}
	ev := NewEvaluator(agent, NewAccuracyReward(), EvaluatorOptions{K: 3})
	metrics, _, _ := ev.Run(context.Background(), ds)
	if metrics.PassAtK != 1.0 {
		t.Errorf("Pass@3 = %f, want 1.0 (1 of 3 attempts hits)", metrics.PassAtK)
	}
}

func TestEvaluator_DefaultsApplied(t *testing.T) {
	ev := NewEvaluator(&stubAgent{}, NewAccuracyReward(), EvaluatorOptions{})
	if ev.opts.Concurrency != 1 || ev.opts.K != 1 || ev.opts.Timeout == 0 {
		t.Errorf("defaults not applied: %+v", ev.opts)
	}
}

// --- TrainerProxy ---------------------------------------------------------

func TestUnsupportedTrainer_Errors(t *testing.T) {
	tr := NewUnsupportedTrainer()
	_, err := tr.Train(context.Background(), TrainConfig{})
	if !errors.Is(err, ErrTrainingNotSupported) {
		t.Errorf("Train err = %v, want ErrTrainingNotSupported", err)
	}
	_, err = tr.LoadModel("anything")
	if !errors.Is(err, ErrTrainingNotSupported) {
		t.Errorf("LoadModel err = %v, want ErrTrainingNotSupported", err)
	}
}
