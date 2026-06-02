package agents_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/costa92/llm-agent"
	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent/orchestrate"
)

// newScriptedMultiAgent is a local test helper that creates a SimpleAgent
// backed by a deterministic scripted LLM that returns the given responses in order.
func newScriptedMultiAgent(name, systemPrompt string, responses ...string) agents.Agent {
	resps := make([]llm.Response, len(responses))
	for i, text := range responses {
		resps[i] = llm.Response{
			Text:         text,
			FinishReason: llm.FinishReasonStop,
			Provider:     "scripted",
			Usage:        llm.Usage{Source: llm.UsageReported},
		}
	}
	client := &multiAgentScriptedLLM{resps: resps}
	return agents.NewSimpleAgent(client, agents.SimpleOptions{
		Name:         name,
		SystemPrompt: systemPrompt,
	})
}

// multiAgentScriptedLLM is a minimal llm.ChatModel stub for multi-agent examples.
type multiAgentScriptedLLM struct {
	calls int
	resps []llm.Response
}

func (s *multiAgentScriptedLLM) Generate(_ context.Context, _ llm.Request) (llm.Response, error) {
	if s.calls >= len(s.resps) {
		return llm.Response{}, fmt.Errorf("scripted LLM: script exhausted")
	}
	r := s.resps[s.calls]
	s.calls++
	return r, nil
}

func (s *multiAgentScriptedLLM) Stream(ctx context.Context, req llm.Request) (llm.StreamReader, error) {
	resp, err := s.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	return llm.NewScriptedLLM(llm.WithResponses(resp)).Stream(ctx, llm.Request{})
}

func (s *multiAgentScriptedLLM) Info() llm.ProviderInfo {
	return llm.ProviderInfo{
		Provider: "scripted",
		Model:    "example",
	}
}

// Example_pipeline demonstrates using orchestrate.Pipeline to chain two agents:
// a "researcher" agent that produces a brief finding, and a "summarizer" agent
// that condenses it. Pipeline threads the first agent's answer into the second
// agent's input automatically.
//
// In production, each agent would use a real llm.ChatModel with a domain-specific
// system prompt. The scripted stubs here produce deterministic output for testing.
func Example_pipeline() {
	// Researcher: scripted to return a one-line finding.
	researcher := newScriptedMultiAgent(
		"researcher",
		"You are a technical researcher. Summarize in one sentence.",
		"Finding: Go modules support nested modules via sub-path tags.",
	)

	// Summarizer: scripted to return a concise summary of whatever it receives.
	summarizer := newScriptedMultiAgent(
		"summarizer",
		"Condense the finding into a single summary sentence.",
		"Summary: nested module tags enable independent versioning.",
	)

	// Pipeline chains researcher → summarizer. Second agent receives first agent's Answer.
	pipeline := orchestrate.NewPipeline("research-summarize",
		orchestrate.Step{Name: "research", Agent: researcher},
		orchestrate.Step{Name: "summarize", Agent: summarizer},
	)

	result, err := pipeline.Run(context.Background(), "How do Go nested modules work?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Summary: nested module tags enable independent versioning.
}

// Example_fanOutFanIn demonstrates using orchestrate.FanOutFanIn for a
// planner -> parallel workers -> aggregator workflow.
func Example_fanOutFanIn() {
	planner := newScriptedMultiAgent(
		"planner",
		"Break the task into newline-separated subtasks.",
		"Research the API surface\nReview the test coverage",
	)

	workerA := newScriptedMultiAgent(
		"worker-a",
		"Handle the assigned subtask.",
		"API notes: the public surface is stable.",
	)
	workerB := newScriptedMultiAgent(
		"worker-b",
		"Handle the assigned subtask.",
		"Test notes: edge cases need more coverage.",
	)

	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Combine worker outputs into one summary.",
		"Summary: stable API, but test edge cases still need coverage.",
	)

	flow := orchestrate.NewFanOutFanIn(
		"plan-workers-aggregate",
		orchestrate.FanOutFanInOptions{
			Planner: planner,
			Workers: map[string]agents.Agent{
				"researcher": workerA,
				"reviewer":   workerB,
			},
			Aggregator: aggregator,
			ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
				lines := strings.Split(plan.Answer, "\n")
				tasks := make([]orchestrate.PlannedTask, 0, len(lines))
				for i, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					tasks = append(tasks, orchestrate.PlannedTask{
						Name:  fmt.Sprintf("task-%d", i+1),
						Input: line,
					})
				}
				return tasks, nil
			},
		},
	)

	result, err := flow.Run(context.Background(), "Assess release readiness.")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Summary: stable API, but test edge cases still need coverage.
}

// Example_fanOutFanIn_supportTriage demonstrates a support-oriented
// planner -> parallel specialists -> answer workflow.
func Example_fanOutFanIn_supportTriage() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the support issue into specialist checks.",
		"Check refund policy\nCheck fraud / abuse risk",
	)

	policyAgent := newScriptedMultiAgent(
		"policy-agent",
		"Answer only from policy context.",
		"Policy: orders cancelled within 24h are eligible for a full refund.",
	)
	riskAgent := newScriptedMultiAgent(
		"risk-agent",
		"Assess operational risk.",
		"Risk: no fraud indicators found; safe to proceed with normal refund flow.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Merge specialist findings into a support reply.",
		"Reply: the order is eligible for a full refund, and there is no extra risk review required.",
	)

	flow := orchestrate.NewFanOutFanIn("support-triage", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"policy": policyAgent,
			"risk":   riskAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				worker := "policy"
				if i == 1 {
					worker = "risk"
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("check-%d", i+1),
					Worker: worker,
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "The customer cancelled an order after 2 hours and asks for a refund.")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Reply: the order is eligible for a full refund, and there is no extra risk review required.
}

// Example_fanOutFanIn_incidentAnalysis demonstrates using orchestrate.FanOutFanIn
// to split an incident into independent analysis tracks and merge the findings.
func Example_fanOutFanIn_incidentAnalysis() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the incident into independent analysis tracks.",
		"Build timeline\nAssess blast radius\nDraft mitigation",
	)

	timelineAgent := newScriptedMultiAgent(
		"timeline-agent",
		"Reconstruct the incident timeline.",
		"Timeline: deploy at 10:02, error spike at 10:04, rollback at 10:08.",
	)
	radiusAgent := newScriptedMultiAgent(
		"radius-agent",
		"Assess impact scope.",
		"Blast radius: checkout traffic and payment retries were affected.",
	)
	mitigationAgent := newScriptedMultiAgent(
		"mitigation-agent",
		"Draft immediate mitigation actions.",
		"Mitigation: keep rollback active, monitor error rate, and notify support.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Merge the incident tracks into a concise incident summary.",
		"Summary: rollback was effective; checkout and payment retries were impacted; mitigation is in place.",
	)

	flow := orchestrate.NewFanOutFanIn("incident-analysis", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"timeline":   timelineAgent,
			"radius":     radiusAgent,
			"mitigation": mitigationAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				worker := ""
				switch i {
				case 0:
					worker = "timeline"
				case 1:
					worker = "radius"
				default:
					worker = "mitigation"
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("incident-%d", i+1),
					Worker: worker,
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "Investigate the checkout outage.")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Summary: rollback was effective; checkout and payment retries were impacted; mitigation is in place.
}

// Example_fanOutFanIn_researchBrief demonstrates a research summarization flow
// with one planner, multiple specialists, and a single synthesized output.
func Example_fanOutFanIn_researchBrief() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the research task into specialist sub-questions.",
		"Summarize API changes\nSummarize test changes",
	)

	apiAgent := newScriptedMultiAgent(
		"api-agent",
		"Summarize public API changes.",
		"API changes: new fan-out orchestration and explicit usage aggregation.",
	)
	testAgent := newScriptedMultiAgent(
		"test-agent",
		"Summarize test changes.",
		"Test changes: new examples and graph coverage were added.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Turn specialist notes into one research brief.",
		"Brief: the orchestration layer now has fan-out support, usage aggregation, and broader scenario coverage.",
	)

	flow := orchestrate.NewFanOutFanIn("research-brief", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"api":  apiAgent,
			"test": testAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				worker := "api"
				if i == 1 {
					worker = "test"
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("brief-%d", i+1),
					Worker: worker,
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "What changed in the orchestration layer?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Brief: the orchestration layer now has fan-out support, usage aggregation, and broader scenario coverage.
}

// Example_fanOutFanIn_returnPolicy demonstrates a customer support workflow
// that splits a refund case into eligibility, logistics, and messaging checks.
func Example_fanOutFanIn_returnPolicy() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the refund case into support checks.",
		"Check eligibility\nCheck return logistics\nCheck customer message",
	)

	eligibilityAgent := newScriptedMultiAgent(
		"eligibility-agent",
		"Check refund eligibility against policy.",
		"Eligibility: the order is within the 14-day return window.",
	)
	logisticsAgent := newScriptedMultiAgent(
		"logistics-agent",
		"Check reverse logistics readiness.",
		"Logistics: a prepaid return label can be issued immediately.",
	)
	messageAgent := newScriptedMultiAgent(
		"message-agent",
		"Draft a clear customer-facing message.",
		"Message: please print the return label and send the item back for a full refund.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Merge support checks into one after-sales reply.",
		"Reply: the return is approved, the label is ready, and the customer can ship the item back today.",
	)

	flow := orchestrate.NewFanOutFanIn("return-policy", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"eligibility": eligibilityAgent,
			"logistics":   logisticsAgent,
			"message":     messageAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			workers := []string{"eligibility", "logistics", "message"}
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("return-%d", i+1),
					Worker: workers[i],
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "The customer wants to return a jacket bought 7 days ago.")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Reply: the return is approved, the label is ready, and the customer can ship the item back today.
}

// Example_fanOutFanIn_fraudReview demonstrates a risk workflow that splits a
// transaction into independent checks and returns a manual-review decision.
func Example_fanOutFanIn_fraudReview() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the transaction review into risk checks.",
		"Check transaction pattern\nCheck account history\nCheck policy rule",
	)

	patternAgent := newScriptedMultiAgent(
		"pattern-agent",
		"Summarize transaction anomalies.",
		"Pattern: three failed payment attempts were followed by one success.",
	)
	historyAgent := newScriptedMultiAgent(
		"history-agent",
		"Summarize account history risk signals.",
		"History: the account is new and the shipping address does not match billing.",
	)
	policyAgent := newScriptedMultiAgent(
		"policy-agent",
		"State the policy outcome.",
		"Policy: this combination requires a manual review hold.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Merge the risk checks into one fraud decision.",
		"Decision: hold the transaction for manual review.",
	)

	flow := orchestrate.NewFanOutFanIn("fraud-review", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"pattern": patternAgent,
			"history": historyAgent,
			"policy":  policyAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			workers := []string{"pattern", "history", "policy"}
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("risk-%d", i+1),
					Worker: workers[i],
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "Review transaction TX-44321.")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Decision: hold the transaction for manual review.
}

// Example_fanOutFanIn_knowledgeLookup demonstrates a knowledge-base style
// retrieval workflow that searches docs, release notes, and FAQ in parallel.
func Example_fanOutFanIn_knowledgeLookup() {
	planner := newScriptedMultiAgent(
		"planner",
		"Split the lookup into independent knowledge sources.",
		"Search docs\nCheck release notes\nCheck FAQ",
	)

	docsAgent := newScriptedMultiAgent(
		"docs-agent",
		"Return the canonical documentation answer.",
		"Docs: token reset uses the account settings page.",
	)
	releaseAgent := newScriptedMultiAgent(
		"release-agent",
		"Return the relevant release note.",
		"Release notes: token reset was added in v2.4.",
	)
	faqAgent := newScriptedMultiAgent(
		"faq-agent",
		"Return the FAQ answer.",
		"FAQ: users can send a reset link from the account settings page.",
	)
	aggregator := newScriptedMultiAgent(
		"aggregator",
		"Merge the knowledge sources into one answer.",
		"Answer: use the account settings page to reset the token; the feature was added in v2.4.",
	)

	flow := orchestrate.NewFanOutFanIn("knowledge-lookup", orchestrate.FanOutFanInOptions{
		Planner: planner,
		Workers: map[string]agents.Agent{
			"docs":  docsAgent,
			"notes": releaseAgent,
			"faq":   faqAgent,
		},
		Aggregator: aggregator,
		ParsePlan: func(plan agents.Result) ([]orchestrate.PlannedTask, error) {
			lines := strings.Split(plan.Answer, "\n")
			workers := []string{"docs", "notes", "faq"}
			tasks := make([]orchestrate.PlannedTask, 0, len(lines))
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				tasks = append(tasks, orchestrate.PlannedTask{
					Name:   fmt.Sprintf("knowledge-%d", i+1),
					Worker: workers[i],
					Input:  line,
				})
			}
			return tasks, nil
		},
	})

	result, err := flow.Run(context.Background(), "How do users reset a token?")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(result.FinalAnswer)
	// Output:
	// Answer: use the account settings page to reset the token; the feature was added in v2.4.
}
