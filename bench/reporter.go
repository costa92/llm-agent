package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/costa92/llm-agent/rl"
)

// Reporter writes Markdown-flavored reports + embedded JSON to an
// io.Writer. Sections are independent — call any combination.
type Reporter struct {
	title  string
	output io.Writer
}

// NewReporter constructs a Reporter. title becomes the top H1 heading
// when WriteHeader is called.
func NewReporter(title string, output io.Writer) *Reporter {
	return &Reporter{title: title, output: output}
}

// WriteHeader writes the report's H1 + a blank line.
func (r *Reporter) WriteHeader() {
	fmt.Fprintf(r.output, "# %s\n\n", r.title)
}

// WriteOverview writes a "## Overview" section as a 2-column table.
func (r *Reporter) WriteOverview(fields map[string]string) {
	fmt.Fprintf(r.output, "## Overview\n\n")
	fmt.Fprintf(r.output, "| Field | Value |\n|---|---|\n")
	for _, k := range sortedStringKeys(fields) {
		fmt.Fprintf(r.output, "| %s | %s |\n", k, fields[k])
	}
	fmt.Fprintln(r.output)
}

// WriteAccuracyBar writes one ASCII progress bar:
//
//	label: ████████░░░░░░░ 50.00%
func (r *Reporter) WriteAccuracyBar(label string, ratio float64, width int) {
	if width <= 0 {
		width = 20
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Fprintf(r.output, "%s: %s %.2f%%\n", label, bar, ratio*100)
}

// WriteTable writes a Markdown table.
func (r *Reporter) WriteTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	fmt.Fprintf(r.output, "| %s |\n", strings.Join(headers, " | "))
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	fmt.Fprintf(r.output, "| %s |\n", strings.Join(sep, " | "))
	for _, row := range rows {
		fmt.Fprintf(r.output, "| %s |\n", strings.Join(row, " | "))
	}
	fmt.Fprintln(r.output)
}

// WriteSection writes "## <heading>\n\n<body>\n\n".
func (r *Reporter) WriteSection(heading, body string) {
	fmt.Fprintf(r.output, "## %s\n\n%s\n\n", heading, body)
}

// WriteJSON writes the value as a fenced JSON code block.
func (r *Reporter) WriteJSON(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("reporter: marshal %s: %w", name, err)
	}
	fmt.Fprintf(r.output, "## %s (JSON)\n\n```json\n%s\n```\n\n", name, b)
	return nil
}

// ExportJSON writes the value as JSON to w (no markdown wrapper).
func ExportJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// --- one-shot renderers ---------------------------------------------------

// RenderBFCLReport writes a full BFCL evaluation report.
func RenderBFCLReport(w io.Writer, agentName string, m BFCLMetrics, trajs []rl.Trajectory) {
	r := NewReporter("BFCL Evaluation Report", w)
	r.WriteHeader()
	r.WriteOverview(map[string]string{
		"agent":     agentName,
		"samples":   fmt.Sprintf("%d", m.SampleCount),
		"overall":   fmt.Sprintf("%.2f%%", m.OverallAccuracy*100),
		"weighted":  fmt.Sprintf("%.2f%%", m.WeightedAccuracy*100),
	})
	r.WriteAccuracyBar("Overall accuracy", m.OverallAccuracy, 30)
	fmt.Fprintln(w)
	if len(m.PerCategory) > 0 {
		rows := make([][]string, 0, len(m.PerCategory))
		for _, k := range sortedStringKeys(m.PerCategory) {
			rows = append(rows, []string{k, fmt.Sprintf("%.2f%%", m.PerCategory[k]*100)})
		}
		r.WriteTable([]string{"Category", "Accuracy"}, rows)
	}
	if len(trajs) > 0 {
		r.WriteSection("Failed samples", failedSamplesSection(trajs))
	}
}

// RenderGAIAReport writes a full GAIA evaluation report.
func RenderGAIAReport(w io.Writer, agentName string, m GAIAMetrics, trajs []rl.Trajectory) {
	r := NewReporter("GAIA Evaluation Report", w)
	r.WriteHeader()
	r.WriteOverview(map[string]string{
		"agent":   agentName,
		"samples": fmt.Sprintf("%d", m.SampleCount),
		"em_rate": fmt.Sprintf("%.2f%%", m.ExactMatchRate*100),
	})
	r.WriteAccuracyBar("Exact match rate", m.ExactMatchRate, 30)
	fmt.Fprintln(w)
	if len(m.PerLevel) > 0 {
		rows := make([][]string, 0, len(m.PerLevel))
		for _, lvl := range sortedIntKeys(m.PerLevel) {
			rows = append(rows, []string{fmt.Sprintf("Level %d", lvl), fmt.Sprintf("%.2f%%", m.PerLevel[lvl]*100)})
		}
		r.WriteTable([]string{"Level", "Accuracy"}, rows)
	}
	r.WriteSection("Drop rates", fmt.Sprintf("L1→L2: %.2f%%, L2→L3: %.2f%%", m.DropRate12*100, m.DropRate23*100))
}

// RenderJudgeReport writes a Judge scoring report.
func RenderJudgeReport(w io.Writer, scope string, m JudgeMetrics, scores []JudgeScore) {
	r := NewReporter("LLM-as-Judge Report — "+scope, w)
	r.WriteHeader()
	r.WriteOverview(map[string]string{
		"items":          fmt.Sprintf("%d", m.Total),
		"avg_score":      fmt.Sprintf("%.2f", m.AverageScore),
		"pass_rate":      fmt.Sprintf("%.2f%%", m.PassRate*100),
		"excellent_rate": fmt.Sprintf("%.2f%%", m.ExcellentRate*100),
	})
	if len(m.PerRubric) > 0 {
		rows := make([][]string, 0, len(m.PerRubric))
		for _, k := range sortedStringKeys(m.PerRubric) {
			rows = append(rows, []string{k, fmt.Sprintf("%.2f", m.PerRubric[k])})
		}
		r.WriteTable([]string{"Rubric", "Avg Score"}, rows)
	}
	if len(scores) > 0 {
		_ = r.WriteJSON("Per-item scores", scores)
	}
}

// RenderWinRateReport writes a head-to-head report.
func RenderWinRateReport(w io.Writer, scope string, m WinRateMetrics) {
	r := NewReporter("Win Rate Report — "+scope, w)
	r.WriteHeader()
	r.WriteOverview(map[string]string{
		"total":     fmt.Sprintf("%d", m.Total),
		"win_a":     fmt.Sprintf("%d (%.2f%%)", m.WinA, m.WinRateA*100),
		"win_b":     fmt.Sprintf("%d (%.2f%%)", m.WinB, m.WinRateB*100),
		"tie":       fmt.Sprintf("%d (%.2f%%)", m.Tie, m.TieRate*100),
	})
	r.WriteAccuracyBar("Win A", m.WinRateA, 30)
	r.WriteAccuracyBar("Win B", m.WinRateB, 30)
	r.WriteAccuracyBar("Tie  ", m.TieRate, 30)
}

// failedSamplesSection lists samples whose Reward<1 with a brief excerpt.
func failedSamplesSection(trajs []rl.Trajectory) string {
	var b strings.Builder
	for _, t := range trajs {
		if t.Reward >= 1 {
			continue
		}
		fmt.Fprintf(&b, "- **%s**: answer=%q\n", t.SampleID, truncate(t.Answer, 80))
	}
	if b.Len() == 0 {
		return "All samples passed."
	}
	return b.String()
}

func sortedStringKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedIntKeys[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
