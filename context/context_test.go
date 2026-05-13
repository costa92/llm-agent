package context

import (
	stdctx "context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/costa92/llm-agent/llm"
	"github.com/costa92/llm-agent/memory"
	"github.com/costa92/llm-agent/rag"
)

// --- token counter ---------------------------------------------------------

func TestSimpleCounter_AsciiWords(t *testing.T) {
	c := SimpleCounter{}
	if got := c.Count("hello world go modules"); got < 4 || got > 6 {
		t.Errorf("4 ASCII words → got %d, want ~5 (4×1.3=5.2)", got)
	}
}

func TestSimpleCounter_CJKChars(t *testing.T) {
	c := SimpleCounter{}
	if got := c.Count("你好世界"); got != 4 {
		t.Errorf("4 Han chars → got %d, want 4", got)
	}
}

func TestSimpleCounter_Mixed(t *testing.T) {
	c := SimpleCounter{}
	// "hello 你好" → 1 ASCII word (1.3) + 2 CJK chars (2) ≈ 3
	if got := c.Count("hello 你好"); got < 3 || got > 4 {
		t.Errorf("mixed count = %d, want 3-4", got)
	}
}

func TestSimpleCounter_Empty(t *testing.T) {
	c := SimpleCounter{}
	if c.Count("") != 0 {
		t.Error("empty input should give 0")
	}
}

// --- packet / config -------------------------------------------------------

func TestConfig_Defaults(t *testing.T) {
	c := Config{}.applyDefaults()
	if c.MaxTokens != 3000 {
		t.Errorf("MaxTokens = %d, want 3000", c.MaxTokens)
	}
	if c.ReserveRatio != 0.2 {
		t.Errorf("ReserveRatio = %f, want 0.2", c.ReserveRatio)
	}
	if c.MinRelevance != 0.1 {
		t.Errorf("MinRelevance = %f, want 0.1", c.MinRelevance)
	}
	if c.RecencyWeight != 0.3 || c.RelevanceWeight != 0.7 {
		t.Errorf("Weights = %f/%f, want 0.3/0.7", c.RecencyWeight, c.RelevanceWeight)
	}
}

// --- relevance + select ----------------------------------------------------

func TestJaccardRelevance(t *testing.T) {
	q := "go modules dependency"
	cases := []struct {
		name    string
		content string
		wantPos bool
	}{
		{"all match", "go modules dependency management", true},
		{"some match", "rust ownership", false},
		{"empty content", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score := jaccardRelevance(stdctx.Background(), q, Packet{Content: tc.content})
			if tc.wantPos && score == 0 {
				t.Errorf("expected positive score, got 0")
			}
			if !tc.wantPos && score > 0.2 {
				t.Errorf("expected near-zero, got %f", score)
			}
		})
	}
}

func TestSelect_GreedyFillsBudget(t *testing.T) {
	cfg := Config{MaxTokens: 30, ReserveRatio: 0.1, MinRelevance: 0, RecencyWeight: 0.3, RelevanceWeight: 0.7}.applyDefaults()
	now := time.Now()
	packets := []Packet{
		{Content: "go", Source: SourceMemory, Timestamp: now, TokenCount: 10},
		{Content: "modules", Source: SourceMemory, Timestamp: now, TokenCount: 10},
		{Content: "deps", Source: SourceMemory, Timestamp: now, TokenCount: 10},
	}
	score := func(_ stdctx.Context, _ string, _ Packet) float64 { return 0.5 }
	kept, dropped := selectPackets(stdctx.Background(), "x", packets, cfg, score)
	// Budget = 30 × (1-0.1) = 27 → 2 packets fit (each 10), 1 dropped.
	if len(kept) != 2 {
		t.Errorf("kept %d, want 2 (budget=27, items=10 each)", len(kept))
	}
	if dropped != 1 {
		t.Errorf("dropped %d, want 1", dropped)
	}
}

func TestSelect_SystemAlwaysKept(t *testing.T) {
	cfg := Config{MaxTokens: 100, ReserveRatio: 0.2}.applyDefaults()
	packets := []Packet{
		{Content: "system policy", Source: SourceSystem, TokenCount: 5},
		{Content: "low rel", Source: SourceMemory, TokenCount: 5, Timestamp: time.Now()},
	}
	score := func(_ stdctx.Context, _ string, p Packet) float64 {
		if p.Source == SourceMemory {
			return 0.01 // below MinRelevance default 0.1
		}
		return 0
	}
	kept, _ := selectPackets(stdctx.Background(), "x", packets, cfg, score)
	hasSystem := false
	for _, p := range kept {
		if p.Source == SourceSystem {
			hasSystem = true
		}
	}
	if !hasSystem {
		t.Error("system packet was dropped; it should always survive")
	}
}

func TestSelect_FiltersBelowMinRelevance(t *testing.T) {
	cfg := Config{MaxTokens: 100, MinRelevance: 0.5}.applyDefaults()
	packets := []Packet{
		{Content: "low", Source: SourceMemory, TokenCount: 5, Timestamp: time.Now()},
		{Content: "high", Source: SourceMemory, TokenCount: 5, Timestamp: time.Now()},
	}
	score := func(_ stdctx.Context, _ string, p Packet) float64 {
		if p.Content == "high" {
			return 0.9
		}
		return 0.1
	}
	kept, _ := selectPackets(stdctx.Background(), "x", packets, cfg, score)
	if len(kept) != 1 || kept[0].Content != "high" {
		t.Errorf("kept %v, want only 'high'", kept)
	}
}

// --- structure -------------------------------------------------------------

func TestStructure_AllSectionsRendered(t *testing.T) {
	now := time.Now()
	packets := []Packet{
		{Content: "be helpful", Source: SourceSystem, Timestamp: now},
		{Content: "go modules manage deps", Source: SourceRAG, Timestamp: now},
		{Content: "user prefers concise answers", Source: SourceMemory, Timestamp: now},
		{Content: "user: hi", Source: SourceConversation, Timestamp: now},
	}
	out := structurePackets("how do go modules work?", packets)
	for _, want := range []string{"[Role & Policies]", "[Task]", "[Evidence]", "[Context]", "[History]", "go modules"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestStructure_SkipsEmptySections(t *testing.T) {
	out := structurePackets("query only", nil)
	if !strings.Contains(out, "[Task]") {
		t.Error("Task section missing")
	}
	if strings.Contains(out, "[Role & Policies]") || strings.Contains(out, "[Evidence]") {
		t.Errorf("empty sections should be skipped:\n%s", out)
	}
}

// --- compress --------------------------------------------------------------

type fakeLLM struct {
	resp string
	err  error
}

func (f *fakeLLM) Generate(_ stdctx.Context, _ llm.Request) (llm.Response, error) {
	if f.err != nil {
		return llm.Response{}, f.err
	}
	return llm.Response{Text: f.resp}, nil
}

func (f *fakeLLM) Stream(_ stdctx.Context, _ llm.Request) (llm.StreamReader, error) {
	return nil, errors.New("nope")
}

func (f *fakeLLM) Info() llm.ProviderInfo { return llm.ProviderInfo{} }

func TestCompress_NoOpUnderBudget(t *testing.T) {
	cfg := Config{MaxTokens: 100, EnableCompress: true}.applyDefaults()
	in := "short prompt"
	out := compress(stdctx.Background(), in, SimpleCounter{}, cfg, nil)
	if out != in {
		t.Errorf("compress changed under-budget prompt: %q → %q", in, out)
	}
}

func TestCompress_HardTruncateWithoutLLM(t *testing.T) {
	cfg := Config{MaxTokens: 5, EnableCompress: false}.applyDefaults()
	in := strings.Repeat("foo bar ", 50)
	out := compress(stdctx.Background(), in, SimpleCounter{}, cfg, nil)
	if !strings.Contains(out, "[truncated]") {
		t.Errorf("expected [truncated] marker, got: %q", out)
	}
	c := SimpleCounter{}
	if c.Count(out) > 20 {
		t.Errorf("output token count %d > expected ~5+truncated", c.Count(out))
	}
}

func TestCompress_LLMSummarizesWhenEnabled(t *testing.T) {
	cfg := Config{MaxTokens: 5, EnableCompress: true}.applyDefaults()
	llmStub := &fakeLLM{resp: "tiny"}
	in := strings.Repeat("foo bar ", 50)
	out := compress(stdctx.Background(), in, SimpleCounter{}, cfg, llmStub)
	if out != "tiny" {
		t.Errorf("expected LLM summary 'tiny', got %q", out)
	}
}

func TestCompress_LLMFailFallsBackToTruncate(t *testing.T) {
	cfg := Config{MaxTokens: 5, EnableCompress: true}.applyDefaults()
	llmStub := &fakeLLM{err: errors.New("boom")}
	in := strings.Repeat("foo bar ", 50)
	out := compress(stdctx.Background(), in, SimpleCounter{}, cfg, llmStub)
	if !strings.Contains(out, "[truncated]") {
		t.Errorf("expected fallback truncation, got: %q", out)
	}
}

// --- builder integration ---------------------------------------------------

func TestBuilder_FullPipelineRendersPrompt(t *testing.T) {
	b := New(Config{MaxTokens: 1000})
	out := b.Build(BuildInput{
		UserQuery:    "how do go modules work?",
		SystemPrompt: "You are a helpful assistant.",
		MemoryHits: []memory.SearchResult{
			{Item: memory.MemoryItem{Content: "user prefers Go", CreatedAt: time.Now()}, Score: 0.8},
		},
		RAGHits: []rag.SearchHit{
			{Doc: rag.Document{ID: "doc-1", Content: "go modules are go's dependency manager"}, Score: 0.9},
		},
	})
	if !strings.Contains(out.Prompt, "[Task]") || !strings.Contains(out.Prompt, "go modules") {
		t.Errorf("prompt missing essentials:\n%s", out.Prompt)
	}
	if out.UsedTokens == 0 {
		t.Error("UsedTokens not computed")
	}
	if len(out.Selected) == 0 {
		t.Error("Selected packets empty")
	}
}

func TestBuilder_DropsLowRelevance(t *testing.T) {
	b := New(Config{MaxTokens: 1000, MinRelevance: 0.3})
	out := b.Build(BuildInput{
		UserQuery: "rocket science",
		MemoryHits: []memory.SearchResult{
			{Item: memory.MemoryItem{Content: "go modules history", CreatedAt: time.Now()}, Score: 0.1},
		},
	})
	if strings.Contains(out.Prompt, "go modules history") {
		t.Errorf("low-relevance memory leaked into prompt:\n%s", out.Prompt)
	}
	if out.DroppedCount == 0 {
		t.Error("DroppedCount = 0; expected at least 1")
	}
}

func TestBuilder_WithEmbedderUsesCosine(t *testing.T) {
	b := New(Config{MaxTokens: 1000}, WithEmbedder(rag.NewHashEmbedder(64)))
	out := b.Build(BuildInput{
		UserQuery: "go modules dependency",
		MemoryHits: []memory.SearchResult{
			{Item: memory.MemoryItem{Content: "go modules dependency management", CreatedAt: time.Now()}, Score: 0.5},
		},
	})
	if !strings.Contains(out.Prompt, "go modules dependency management") {
		t.Errorf("embedder-relevance kept too few items:\n%s", out.Prompt)
	}
}

func TestBuilder_WithLLMCompressesOversize(t *testing.T) {
	llmStub := &fakeLLM{resp: "tight summary"}
	b := New(Config{MaxTokens: 5, EnableCompress: true}, WithLLM(llmStub))
	out := b.Build(BuildInput{
		UserQuery:    strings.Repeat("foo bar baz ", 50),
		SystemPrompt: strings.Repeat("scaffolding ", 20),
	})
	if out.Prompt != "tight summary" {
		t.Errorf("expected LLM summary, got: %q", out.Prompt)
	}
}
