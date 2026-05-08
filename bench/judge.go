package bench

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/costa92/llm-agent/pkg/fanout"
	"github.com/costa92/llm-agent/llm"
)

// JudgeRubric is one scoring dimension. Scale defines the int range
// 1..Scale (Scale=5 → 1-5 scoring).
type JudgeRubric struct {
	Name        string
	Description string
	Scale       int
}

// JudgeOptions configures Judge.Evaluate.
type JudgeOptions struct {
	Concurrency    int
	Timeout        time.Duration // per-item timeout; default 30s
	PromptTemplate string        // override default; %s order: question, answer, ref-line, scale, scale, rubric-list
}

// Judge wraps an LLM client into a multi-rubric evaluator.
type Judge struct {
	llm     llm.Client
	rubrics []JudgeRubric
	opts    JudgeOptions
}

// NewJudge constructs a Judge.
func NewJudge(client llm.Client, rubrics []JudgeRubric, opts JudgeOptions) *Judge {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.PromptTemplate == "" {
		opts.PromptTemplate = judgePromptDefault
	}
	return &Judge{llm: client, rubrics: rubrics, opts: opts}
}

// JudgeItem is one (question, answer[, reference]) input row.
type JudgeItem struct {
	SampleID  string
	Question  string
	Answer    string
	Reference string // optional
}

// JudgeScore is the LLM's per-item verdict.
type JudgeScore struct {
	SampleID  string
	Scores    map[string]int
	AvgScore  float64
	Reasoning string
}

// ErrJudgeNoLLM is returned by Evaluate when the Judge has no LLM.
var ErrJudgeNoLLM = stderrors.New("bench: judge requires non-nil llm.Client")

// Evaluate scores every item under bounded concurrency.
func (j *Judge) Evaluate(ctx context.Context, items []JudgeItem) ([]JudgeScore, error) {
	if j.llm == nil {
		return nil, ErrJudgeNoLLM
	}
	if len(items) == 0 {
		return nil, nil
	}

	tasks := make([]fanout.Task[JudgeScore], len(items))
	for i, item := range items {
		item := item
		tasks[i] = func(ctx context.Context) (JudgeScore, error) {
			return j.scoreOne(ctx, item), nil
		}
	}

	results, err := fanout.Run(ctx, j.opts.Concurrency, tasks)
	if err != nil {
		return nil, err
	}

	scores := make([]JudgeScore, len(items))
	for i, r := range results {
		scores[i] = r.Value
	}
	return scores, nil
}

// scoreOne runs one judging round under the per-item timeout.
func (j *Judge) scoreOne(ctx context.Context, item JudgeItem) JudgeScore {
	runCtx, cancel := context.WithTimeout(ctx, j.opts.Timeout)
	defer cancel()

	scale := 5
	if len(j.rubrics) > 0 {
		scale = j.rubrics[0].Scale
	}
	if scale <= 0 {
		scale = 5
	}
	rubricList := strings.Builder{}
	for _, r := range j.rubrics {
		fmt.Fprintf(&rubricList, "- %s: %s\n", r.Name, r.Description)
	}
	refLine := ""
	if item.Reference != "" {
		refLine = "Reference: " + item.Reference + "\n"
	}
	prompt := fmt.Sprintf(j.opts.PromptTemplate, item.Question, item.Answer, refLine, scale, scale, rubricList.String())

	resp, err := j.llm.Generate(runCtx, llm.GenerateRequest{Prompt: prompt})
	if err != nil {
		return JudgeScore{SampleID: item.SampleID, Reasoning: "judge llm error: " + err.Error()}
	}
	return parseJudgeOutput(item.SampleID, resp.Text)
}

// parseJudgeOutput extracts the JSON payload from the LLM response,
// tolerating ```json ... ``` fences. Returns a JudgeScore with
// best-effort fields populated.
func parseJudgeOutput(id, text string) JudgeScore {
	out := JudgeScore{SampleID: id, Scores: map[string]int{}}
	body := stripFences(text)
	type raw struct {
		Scores    map[string]int `json:"scores"`
		Reasoning string         `json:"reasoning"`
	}
	var r raw
	if err := json.Unmarshal([]byte(body), &r); err == nil {
		out.Scores = r.Scores
		out.Reasoning = r.Reasoning
	} else {
		out.Reasoning = "parse error: " + err.Error() + "; raw=" + truncate(text, 200)
	}
	if len(out.Scores) > 0 {
		var sum float64
		for _, v := range out.Scores {
			sum += float64(v)
		}
		out.AvgScore = sum / float64(len(out.Scores))
	}
	return out
}

// stripFences removes ```json ... ``` wrappers (LLMs love them).
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.Index(s, "\n"); nl >= 0 {
		s = s[nl+1:]
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// JudgeMetrics aggregates JudgeScores into a scoresheet.
type JudgeMetrics struct {
	Total         int
	AverageScore  float64
	PassRate      float64 // AvgScore ≥ scale*0.7
	ExcellentRate float64 // AvgScore ≥ scale*0.9
	PerRubric     map[string]float64
}

// ComputeJudgeMetrics aggregates scores. scale is the rubric scale
// (used to compute PassRate / ExcellentRate thresholds).
func ComputeJudgeMetrics(scores []JudgeScore, scale int) JudgeMetrics {
	if len(scores) == 0 || scale <= 0 {
		return JudgeMetrics{PerRubric: map[string]float64{}}
	}
	out := JudgeMetrics{Total: len(scores), PerRubric: map[string]float64{}}
	rubricSum := make(map[string]float64)
	rubricCount := make(map[string]int)
	var avgSum float64
	pass := 0
	excellent := 0
	passT := float64(scale) * 0.7
	excelT := float64(scale) * 0.9
	for _, s := range scores {
		avgSum += s.AvgScore
		if s.AvgScore >= passT {
			pass++
		}
		if s.AvgScore >= excelT {
			excellent++
		}
		for k, v := range s.Scores {
			rubricSum[k] += float64(v)
			rubricCount[k]++
		}
	}
	out.AverageScore = avgSum / float64(out.Total)
	out.PassRate = float64(pass) / float64(out.Total)
	out.ExcellentRate = float64(excellent) / float64(out.Total)
	for k, sum := range rubricSum {
		out.PerRubric[k] = sum / float64(rubricCount[k])
	}
	return out
}

const judgePromptDefault = `You are an expert judge. Score the answer to the question below using each rubric.

Question: %s
Answer: %s
%s
Rubrics (each scored 1-%d, %d=excellent):
%s
Output strict JSON only — no prose, no markdown fences:
{
  "scores": {"<rubric_name>": <int>, ...},
  "reasoning": "<one paragraph>"
}`
