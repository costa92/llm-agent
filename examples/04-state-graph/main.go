// Demo 04: StateGraph — branching workflow with shared typed state.
//
// orchestrate.StateGraph models supervisor / specialist patterns: named
// nodes, unconditional edges, conditional edges (route by inspecting state),
// and a per-run step cap. Pure stdlib; no DSL.
//
// This demo is a customer-service triage:
//
//	classify ──▶ auto-reply ─────▶ end       (eligible self-service)
//	         ─▶ request-more-info ─▶ classify (loop until enough info)
//	         ─▶ handover-human ───▶ end       (escalate)
//
// Run:
//
//	cd examples/04-state-graph && go run .
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/costa92/llm-agent/orchestrate"
)

type triageState struct {
	Question         string
	Tries            int
	NeedMoreInfo     bool
	NeedsHuman       bool
	Reply            string
	CollectedDetails []string
}

func main() {
	g := orchestrate.NewStateGraph[triageState]()

	g.AddNode("classify", func(_ context.Context, s triageState) (triageState, error) {
		s.Tries++
		s.NeedMoreInfo = false
		q := strings.ToLower(s.Question)
		switch {
		case strings.Contains(q, "chargeback"), strings.Contains(q, "fraud"):
			s.NeedsHuman = true
		case strings.Contains(q, "order"):
			// classified — auto-reply takes it from here
		case s.Tries < 2:
			// First pass without an order ID: try to collect more info.
			s.NeedMoreInfo = true
		default:
			// Asked once and still no actionable signal — escalate.
			s.NeedsHuman = true
		}
		return s, nil
	})
	g.AddNode("request-more-info", func(_ context.Context, s triageState) (triageState, error) {
		s.CollectedDetails = append(s.CollectedDetails, "order-id provided")
		s.NeedMoreInfo = false
		return s, nil
	})
	g.AddNode("auto-reply", func(_ context.Context, s triageState) (triageState, error) {
		s.Reply = "Your request is eligible for the standard self-service refund flow."
		return s, nil
	})
	g.AddNode("handover-human", func(_ context.Context, s triageState) (triageState, error) {
		s.Reply = "Escalating to a human agent for manual review."
		return s, nil
	})

	g.AddEdge(orchestrate.NodeStart, "classify")
	g.AddConditionalEdge("classify", func(s triageState) string {
		switch {
		case s.NeedsHuman:
			return "handover-human"
		case s.NeedMoreInfo:
			return "request-more-info"
		default:
			return "auto-reply"
		}
	})
	g.AddConditionalEdge("request-more-info", func(s triageState) string {
		if len(s.CollectedDetails) > 0 {
			return "classify"
		}
		return "request-more-info"
	})
	g.AddEdge("auto-reply", orchestrate.NodeEnd)
	g.AddEdge("handover-human", orchestrate.NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		log.Fatalf("compile: %v", err)
	}

	cases := []string{
		"I need a refund for my order 123",
		"Help me with this chargeback",
		"Hi, how are you?",
	}

	for _, q := range cases {
		out, err := cg.Run(context.Background(), triageState{Question: q}, orchestrate.WithMaxSteps(8))
		if err != nil {
			log.Fatalf("run: %v", err)
		}
		fmt.Printf("Q: %s\n", q)
		fmt.Printf("A: %s\n\n", out.Reply)
	}
}
