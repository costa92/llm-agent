package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// researchState mirrors the spec §12.6 example.
type researchState struct {
	Query     string
	Plan      []string
	Summaries []string
	Final     string
}

func TestStateGraph_LinearPath(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("plan", func(_ context.Context, s researchState) (researchState, error) {
		s.Plan = []string{"task1", "task2"}
		return s, nil
	})
	g.AddNode("report", func(_ context.Context, s researchState) (researchState, error) {
		s.Final = "Report on " + s.Query + ": " + strings.Join(s.Plan, ", ")
		return s, nil
	})
	g.SetEntry("plan").AddEdge("plan", "report").AddEdge("report", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	final, err := cg.Run(context.Background(), researchState{Query: "AI"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Final != "Report on AI: task1, task2" {
		t.Errorf("Final = %q", final.Final)
	}
}

func TestStateGraph_ConditionalBranching(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("plan", func(_ context.Context, s researchState) (researchState, error) {
		s.Plan = []string{"a"}
		return s, nil
	})
	g.AddNode("summarize", func(_ context.Context, s researchState) (researchState, error) {
		s.Summaries = append(s.Summaries, "sum-"+s.Plan[len(s.Summaries)])
		return s, nil
	})
	g.AddNode("report", func(_ context.Context, s researchState) (researchState, error) {
		s.Final = strings.Join(s.Summaries, "; ")
		return s, nil
	})
	g.SetEntry("plan").
		AddEdge("plan", "summarize").
		AddConditionalEdge("summarize", func(s researchState) string {
			if len(s.Summaries) >= len(s.Plan) {
				return "report"
			}
			return "summarize" // loop until all done
		}).
		AddEdge("report", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	final, err := cg.Run(context.Background(), researchState{Query: "x"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Final != "sum-a" {
		t.Errorf("Final = %q, want sum-a", final.Final)
	}
}

func TestStateGraph_StartEdgeSetsEntry(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("plan", func(_ context.Context, s researchState) (researchState, error) {
		s.Plan = []string{"from-start"}
		return s, nil
	})
	g.AddEdge(NodeStart, "plan").AddEdge("plan", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	final, err := cg.Run(context.Background(), researchState{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.Join(final.Plan, ","); got != "from-start" {
		t.Fatalf("Plan = %q, want from-start", got)
	}
}

func TestStateGraph_ConditionalEdgeTakesPrecedence(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("route", func(_ context.Context, s researchState) (researchState, error) {
		return s, nil
	})
	g.AddNode("conditional", func(_ context.Context, s researchState) (researchState, error) {
		s.Final = "conditional"
		return s, nil
	})
	g.AddNode("fallback", func(_ context.Context, s researchState) (researchState, error) {
		s.Final = "fallback"
		return s, nil
	})
	g.SetEntry("route").
		AddEdge("route", "fallback").
		AddConditionalEdge("route", func(_ researchState) string { return "conditional" }).
		AddEdge("conditional", NodeEnd).
		AddEdge("fallback", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	final, err := cg.Run(context.Background(), researchState{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Final != "conditional" {
		t.Fatalf("Final = %q, want conditional", final.Final)
	}
}

func TestStateGraph_CompileSnapshotIsImmutable(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("plan", func(_ context.Context, s researchState) (researchState, error) {
		s.Plan = append(s.Plan, "compiled")
		return s, nil
	})
	g.SetEntry("plan").AddEdge("plan", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	g.AddNode("plan", func(_ context.Context, s researchState) (researchState, error) {
		s.Plan = append(s.Plan, "mutated")
		return s, nil
	})
	g.SetEntry("ghost").AddEdge("plan", "ghost")

	final, err := cg.Run(context.Background(), researchState{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.Join(final.Plan, ","); got != "compiled" {
		t.Fatalf("Plan = %q, want compiled", got)
	}
}

func TestStateGraph_MaxStepsBreaksInfiniteLoop(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	g.SetEntry("a").AddConditionalEdge("a", func(_ researchState) string { return "a" }) // self-loop forever

	cg, _ := g.Compile()
	_, err := cg.Run(context.Background(), researchState{}, WithMaxSteps(5))
	if !errors.Is(err, ErrGraphMaxSteps) {
		t.Errorf("expected ErrGraphMaxSteps, got %v", err)
	}
}

func TestStateGraph_NodeErrorPropagates(t *testing.T) {
	want := errors.New("boom")
	g := NewStateGraph[researchState]()
	g.AddNode("bad", func(_ context.Context, s researchState) (researchState, error) {
		return s, want
	})
	g.SetEntry("bad").AddEdge("bad", NodeEnd)

	cg, _ := g.Compile()
	_, err := cg.Run(context.Background(), researchState{})
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped boom, got %v", err)
	}
}

func TestStateGraph_Compile_RejectsNoEntry(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	_, err := g.Compile()
	if err == nil || !strings.Contains(err.Error(), "entry") {
		t.Errorf("expected entry error, got %v", err)
	}
}

func TestStateGraph_Compile_RejectsUnknownEdgeTarget(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	g.SetEntry("a").AddEdge("a", "nonexistent")
	_, err := g.Compile()
	if err == nil || !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected unknown-target error, got %v", err)
	}
}

func TestStateGraph_Compile_RejectsEntryNotNode(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.SetEntry("ghost")
	_, err := g.Compile()
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected entry-not-node error, got %v", err)
	}
}

func TestStateGraph_ReservedNodeNamesPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on reserved node name")
		}
	}()
	g := NewStateGraph[researchState]()
	g.AddNode(NodeEnd, func(_ context.Context, s researchState) (researchState, error) { return s, nil })
}

func TestStateGraph_RouteToUnknownNodeErrorsAtRunTime(t *testing.T) {
	// ConditionFunc names aren't validated until Run.
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	g.SetEntry("a").AddConditionalEdge("a", func(_ researchState) string { return "ghost" })

	cg, _ := g.Compile()
	_, err := cg.Run(context.Background(), researchState{})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected unknown-routed-to error, got %v", err)
	}
}

func TestStateGraph_MissingOutgoingEdgeErrors(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	g.SetEntry("a") // no edge from a

	cg, _ := g.Compile()
	_, err := cg.Run(context.Background(), researchState{})
	if err == nil || !strings.Contains(err.Error(), "no outgoing edge") {
		t.Errorf("expected no-outgoing-edge error, got %v", err)
	}
}

func TestStateGraph_ContextCancel(t *testing.T) {
	g := NewStateGraph[researchState]()
	g.AddNode("a", func(_ context.Context, s researchState) (researchState, error) { return s, nil })
	g.SetEntry("a").AddConditionalEdge("a", func(_ researchState) string { return "a" })

	cg, _ := g.Compile()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	_, err := cg.Run(ctx, researchState{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func ExampleStateGraph_loop() {
	type reviewState struct {
		Tasks   []string
		Done    []string
		Summary string
	}

	g := NewStateGraph[reviewState]()
	g.AddNode("dispatch", func(_ context.Context, s reviewState) (reviewState, error) {
		next := s.Tasks[len(s.Done)]
		s.Done = append(s.Done, "reviewed:"+next)
		return s, nil
	})
	g.AddNode("finalize", func(_ context.Context, s reviewState) (reviewState, error) {
		s.Summary = strings.Join(s.Done, ", ")
		return s, nil
	})
	g.AddEdge(NodeStart, "dispatch").
		AddConditionalEdge("dispatch", func(s reviewState) string {
			if len(s.Done) >= len(s.Tasks) {
				return "finalize"
			}
			return "dispatch"
		}).
		AddEdge("finalize", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), reviewState{Tasks: []string{"planner", "writer"}})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Summary)
	// Output:
	// reviewed:planner, reviewed:writer
}

func ExampleStateGraph_supportEscalation() {
	type triageState struct {
		Question         string
		NeedMoreInfo     bool
		NeedsHuman       bool
		Reply            string
		CollectedDetails []string
	}

	g := NewStateGraph[triageState]()
	g.AddNode("classify", func(_ context.Context, s triageState) (triageState, error) {
		if strings.Contains(strings.ToLower(s.Question), "chargeback") {
			s.NeedsHuman = true
			return s, nil
		}
		if !strings.Contains(strings.ToLower(s.Question), "order") {
			s.NeedMoreInfo = true
		}
		return s, nil
	})
	g.AddNode("request-more-info", func(_ context.Context, s triageState) (triageState, error) {
		s.CollectedDetails = append(s.CollectedDetails, "order-id provided")
		s.NeedMoreInfo = false
		s.Reply = "Please share your order ID so I can continue."
		return s, nil
	})
	g.AddNode("auto-reply", func(_ context.Context, s triageState) (triageState, error) {
		s.Reply = "Your request is eligible for the standard self-service refund flow."
		return s, nil
	})
	g.AddNode("handover-human", func(_ context.Context, s triageState) (triageState, error) {
		s.Reply = "I’m escalating this case to a human agent for manual review."
		return s, nil
	})

	g.AddEdge(NodeStart, "classify")
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
			return "auto-reply"
		}
		return "request-more-info"
	})
	g.AddEdge("auto-reply", NodeEnd)
	g.AddEdge("handover-human", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), triageState{
		Question: "I need a refund for my order 123",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Reply)
	// Output:
	// Your request is eligible for the standard self-service refund flow.
}

func ExampleStateGraph_ticketDispatch() {
	type dispatchState struct {
		TicketID string
		Priority string
		Route    string
		Log      []string
		Closed   bool
	}

	g := NewStateGraph[dispatchState]()
	g.AddNode("classify", func(_ context.Context, s dispatchState) (dispatchState, error) {
		s.Log = append(s.Log, "classify:"+s.TicketID)
		if strings.Contains(strings.ToLower(s.Priority), "p0") {
			s.Route = "human"
		} else {
			s.Route = "auto"
		}
		return s, nil
	})
	g.AddNode("auto-route", func(_ context.Context, s dispatchState) (dispatchState, error) {
		s.Log = append(s.Log, "auto-route")
		s.Closed = true
		return s, nil
	})
	g.AddNode("human-route", func(_ context.Context, s dispatchState) (dispatchState, error) {
		s.Log = append(s.Log, "human-route")
		s.Closed = true
		return s, nil
	})

	g.AddEdge(NodeStart, "classify")
	g.AddConditionalEdge("classify", func(s dispatchState) string {
		if s.Route == "human" {
			return "human-route"
		}
		return "auto-route"
	})
	g.AddEdge("auto-route", NodeEnd)
	g.AddEdge("human-route", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), dispatchState{
		TicketID: "T-1024",
		Priority: "p1",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(strings.Join(out.Log, " -> "))
	// Output:
	// classify:T-1024 -> auto-route
}

func ExampleStateGraph_complianceReview() {
	type complianceState struct {
		Draft      string
		Violations []string
		Approved   bool
		History    []string
	}

	g := NewStateGraph[complianceState]()
	g.AddNode("review", func(_ context.Context, s complianceState) (complianceState, error) {
		s.History = append(s.History, "review")
		if strings.Contains(s.Draft, "PII") {
			s.Violations = []string{"remove PII"}
			return s, nil
		}
		s.Violations = nil
		return s, nil
	})
	g.AddNode("revise", func(_ context.Context, s complianceState) (complianceState, error) {
		s.History = append(s.History, "revise")
		s.Draft = strings.ReplaceAll(s.Draft, "PII", "masked data")
		return s, nil
	})
	g.AddNode("approve", func(_ context.Context, s complianceState) (complianceState, error) {
		s.History = append(s.History, "approve")
		s.Approved = true
		return s, nil
	})

	g.AddEdge(NodeStart, "review")
	g.AddConditionalEdge("review", func(s complianceState) string {
		if len(s.Violations) > 0 {
			return "revise"
		}
		return "approve"
	})
	g.AddConditionalEdge("revise", func(s complianceState) string {
		if strings.Contains(s.Draft, "PII") {
			return "review"
		}
		return "review"
	})
	g.AddEdge("approve", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), complianceState{
		Draft: "Customer complaint contains PII and needs masking.",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Approved, strings.Join(out.History, " -> "))
	// Output:
	// true review -> revise -> review -> approve
}

func ExampleStateGraph_warrantyEscalation() {
	type warrantyState struct {
		TicketID     string
		MonthsOwned  int
		Product      string
		NeedsHuman   bool
		NeedsRepair  bool
		Resolution   string
		RouteHistory []string
	}

	g := NewStateGraph[warrantyState]()
	g.AddNode("classify", func(_ context.Context, s warrantyState) (warrantyState, error) {
		s.RouteHistory = append(s.RouteHistory, "classify")
		if s.MonthsOwned > 12 {
			s.NeedsHuman = true
			return s, nil
		}
		s.NeedsRepair = true
		return s, nil
	})
	g.AddNode("repair-intake", func(_ context.Context, s warrantyState) (warrantyState, error) {
		s.RouteHistory = append(s.RouteHistory, "repair-intake")
		s.Resolution = "Create a repair ticket and ask the customer to send photos."
		return s, nil
	})
	g.AddNode("human-escalation", func(_ context.Context, s warrantyState) (warrantyState, error) {
		s.RouteHistory = append(s.RouteHistory, "human-escalation")
		s.Resolution = "Escalate to a human for out-of-warranty review."
		return s, nil
	})

	g.AddEdge(NodeStart, "classify")
	g.AddConditionalEdge("classify", func(s warrantyState) string {
		if s.NeedsHuman {
			return "human-escalation"
		}
		return "repair-intake"
	})
	g.AddEdge("repair-intake", NodeEnd)
	g.AddEdge("human-escalation", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), warrantyState{
		TicketID:    "W-7788",
		MonthsOwned: 6,
		Product:     "headphones",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Resolution)
	// Output:
	// Create a repair ticket and ask the customer to send photos.
}

func ExampleStateGraph_fraudHold() {
	type fraudState struct {
		TransactionID string
		RiskScore     int
		AddressMatch  bool
		NeedHold      bool
		Decision      string
		History       []string
	}

	g := NewStateGraph[fraudState]()
	g.AddNode("screen", func(_ context.Context, s fraudState) (fraudState, error) {
		s.History = append(s.History, "screen")
		if s.RiskScore >= 80 || !s.AddressMatch {
			s.NeedHold = true
		}
		return s, nil
	})
	g.AddNode("manual-review", func(_ context.Context, s fraudState) (fraudState, error) {
		s.History = append(s.History, "manual-review")
		s.Decision = "Hold transaction for manual review."
		return s, nil
	})
	g.AddNode("approve", func(_ context.Context, s fraudState) (fraudState, error) {
		s.History = append(s.History, "approve")
		s.Decision = "Approve transaction."
		return s, nil
	})

	g.AddEdge(NodeStart, "screen")
	g.AddConditionalEdge("screen", func(s fraudState) string {
		if s.NeedHold {
			return "manual-review"
		}
		return "approve"
	})
	g.AddEdge("manual-review", NodeEnd)
	g.AddEdge("approve", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), fraudState{
		TransactionID: "TX-44321",
		RiskScore:     92,
		AddressMatch:  false,
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Decision)
	// Output:
	// Hold transaction for manual review.
}

func ExampleStateGraph_knowledgeLookup() {
	type kbState struct {
		Query      string
		Product    string
		NeedInfo   bool
		Evidence   []string
		Answer     string
		Transcript []string
	}

	g := NewStateGraph[kbState]()
	g.AddNode("triage", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "triage")
		if s.Product == "" {
			s.NeedInfo = true
		}
		return s, nil
	})
	g.AddNode("collect-context", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "collect-context")
		s.Product = "mobile app"
		s.NeedInfo = false
		s.Evidence = append(s.Evidence, "context: product is mobile app")
		return s, nil
	})
	g.AddNode("search-docs", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "search-docs")
		s.Evidence = append(s.Evidence, "docs: token reset uses account settings")
		return s, nil
	})
	g.AddNode("answer", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "answer")
		s.Answer = "Use the account settings page in the mobile app to reset the token."
		return s, nil
	})

	g.AddEdge(NodeStart, "triage")
	g.AddConditionalEdge("triage", func(s kbState) string {
		if s.NeedInfo {
			return "collect-context"
		}
		return "search-docs"
	})
	g.AddEdge("collect-context", "search-docs")
	g.AddEdge("search-docs", "answer")
	g.AddEdge("answer", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), kbState{
		Query: "How do users reset a token?",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Answer)
	// Output:
	// Use the account settings page in the mobile app to reset the token.
}

func ExampleStateGraph_knowledgeLookup_clarifySearchAnswer() {
	type kbState struct {
		Query      string
		Product    string
		Version    string
		NeedInfo   bool
		Evidence   []string
		Answer     string
		Transcript []string
	}

	g := NewStateGraph[kbState]()
	g.AddNode("triage", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "triage")
		if s.Product == "" || s.Version == "" {
			s.NeedInfo = true
		}
		return s, nil
	})
	g.AddNode("request-more-info", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "request-more-info")
		if s.Product == "" {
			s.Product = "mobile app"
		}
		if s.Version == "" {
			s.Version = "v2.4"
		}
		s.NeedInfo = false
		return s, nil
	})
	g.AddNode("search-docs", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "search-docs")
		s.Evidence = append(s.Evidence, "docs: token reset uses account settings")
		s.Evidence = append(s.Evidence, "release notes: reset support added in v2.4")
		return s, nil
	})
	g.AddNode("answer", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "answer")
		s.Answer = "Use account settings in the mobile app to reset the token in v2.4."
		return s, nil
	})

	g.AddEdge(NodeStart, "triage")
	g.AddConditionalEdge("triage", func(s kbState) string {
		if s.NeedInfo {
			return "request-more-info"
		}
		return "search-docs"
	})
	g.AddConditionalEdge("request-more-info", func(s kbState) string {
		if s.Product != "" && s.Version != "" {
			return "search-docs"
		}
		return "request-more-info"
	})
	g.AddEdge("search-docs", "answer")
	g.AddEdge("answer", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), kbState{
		Query: "How do users reset a token?",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Answer)
	// Output:
	// Use account settings in the mobile app to reset the token in v2.4.
}

func ExampleStateGraph_humanReviewReseek() {
	type kbState struct {
		Query        string
		AccountID    string
		Version      string
		NeedHuman    bool
		NeedMoreInfo bool
		NeedReseek   bool
		SearchCount  int
		Evidence     []string
		Answer       string
		Transcript   []string
	}

	g := NewStateGraph[kbState]()
	g.AddNode("triage", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "triage")
		if strings.Contains(strings.ToLower(s.Query), "chargeback") {
			s.NeedHuman = true
		}
		if s.AccountID == "" || s.Version == "" {
			s.NeedMoreInfo = true
		}
		return s, nil
	})
	g.AddNode("human-review", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "human-review")
		if s.SearchCount == 0 {
			s.NeedMoreInfo = true
			s.NeedReseek = true
			s.NeedHuman = false
			s.Transcript = append(s.Transcript, "human-review: collect missing context")
			return s, nil
		}
		s.NeedReseek = false
		s.NeedHuman = false
		s.Transcript = append(s.Transcript, "human-review: approve after re-search")
		return s, nil
	})
	g.AddNode("request-more-info", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "request-more-info")
		if s.AccountID == "" {
			s.AccountID = "AC-7788"
		}
		if s.Version == "" {
			s.Version = "v2.4"
		}
		s.NeedMoreInfo = false
		return s, nil
	})
	g.AddNode("search-docs", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "search-docs")
		s.SearchCount++
		s.Evidence = append(s.Evidence, "docs: token reset uses account settings")
		if s.Version == "v2.4" {
			s.Evidence = append(s.Evidence, "release notes: reset support added in v2.4")
		}
		return s, nil
	})
	g.AddNode("answer", func(_ context.Context, s kbState) (kbState, error) {
		s.Transcript = append(s.Transcript, "answer")
		s.Answer = "After human review and a second search, use account settings in v2.4 to reset the token."
		return s, nil
	})

	g.AddEdge(NodeStart, "triage")
	g.AddConditionalEdge("triage", func(s kbState) string {
		switch {
		case s.NeedHuman:
			return "human-review"
		case s.NeedMoreInfo:
			return "request-more-info"
		default:
			return "search-docs"
		}
	})
	g.AddConditionalEdge("human-review", func(s kbState) string {
		switch {
		case s.NeedMoreInfo:
			return "request-more-info"
		case s.NeedReseek:
			return "search-docs"
		default:
			return "answer"
		}
	})
	g.AddConditionalEdge("request-more-info", func(s kbState) string {
		if s.NeedReseek {
			return "search-docs"
		}
		return "search-docs"
	})
	g.AddConditionalEdge("search-docs", func(s kbState) string {
		if s.NeedReseek && s.SearchCount == 1 {
			return "human-review"
		}
		return "answer"
	})
	g.AddEdge("answer", NodeEnd)

	cg, err := g.Compile()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	out, err := cg.Run(context.Background(), kbState{
		Query: "Chargeback policy for token reset?",
	})
	if err != nil {
		fmt.Println("run:", err)
		return
	}
	fmt.Println(out.Answer)
	// Output:
	// After human review and a second search, use account settings in v2.4 to reset the token.
}
