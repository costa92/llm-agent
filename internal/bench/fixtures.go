package bench

// fixtures.go — built-in mini BFCL / GAIA / AIME-style samples for
// learning and testing. Real benchmark data lives elsewhere — these
// fixtures cover concept demonstration, not statistical validity.

// MiniBFCLSamples returns 5 hand-crafted BFCL-style function-call
// samples covering: simple, multiple, parallel, irrelevance.
func MiniBFCLSamples() []BFCLSample {
	return []BFCLSample{
		{
			Sample: rlSample("bfcl_1", "Get the weather in Beijing in Celsius",
				`get_weather(city='Beijing', units='c')`),
			Category: "simple",
			ExpectedCalls: []FunctionCall{
				{Name: "get_weather", Args: map[string]any{"city": "Beijing", "units": "c"}},
			},
		},
		{
			Sample: rlSample("bfcl_2", "Calculate 25 * 4 and convert to hex",
				`calculate(expression='25*4')`),
			Category: "multiple",
			ExpectedCalls: []FunctionCall{
				{Name: "calculate", Args: map[string]any{"expression": "25*4"}},
			},
		},
		{
			Sample: rlSample("bfcl_3", "Search for 'go modules' and get current time",
				`search(query='go modules'), get_time()`),
			Category: "parallel",
			ExpectedCalls: []FunctionCall{
				{Name: "search", Args: map[string]any{"query": "go modules"}},
				{Name: "get_time", Args: map[string]any{}},
			},
		},
		{
			Sample: rlSample("bfcl_4", "Tell me a joke",
				``),
			Category:      "irrelevance",
			ExpectedCalls: nil, // model should NOT call any function
		},
		{
			Sample: rlSample("bfcl_5", "Send email to alice@example.com with subject 'hi'",
				`send_email(to='alice@example.com', subject='hi')`),
			Category: "simple",
			ExpectedCalls: []FunctionCall{
				{Name: "send_email", Args: map[string]any{"to": "alice@example.com", "subject": "hi"}},
			},
		},
	}
}

// MiniGAIASamples returns 3 hand-crafted GAIA-style samples — one per
// difficulty level. Question / GroundTruth match GAIA's exact-match
// scoring style.
func MiniGAIASamples() []GAIASample {
	return []GAIASample{
		{
			Sample: rlSample("gaia_1", "What is the capital of France?", "Paris"),
			Level:  1,
		},
		{
			Sample: rlSample("gaia_2", "How many planets in our solar system are gas giants? List them.",
				"4: Jupiter, Saturn, Uranus, Neptune"),
			Level: 2,
		},
		{
			Sample: rlSample("gaia_3",
				"Compute the product of (number of months in a year) × (number of weeks in a month, rounded). Show the formula.",
				"48"),
			Level: 3,
		},
	}
}

// MiniAIMESamples returns 3 AIME-style competition math problems —
// fixture for Judge-style subjective scoring (LLM judges the quality of
// the *reasoning*, not just final answer).
func MiniAIMESamples() []AIMESample {
	return []AIMESample{
		{
			ID:       "aime_1",
			Question: "Find the sum of all integer solutions of x^2 - 5x + 6 = 0.",
			Reference: "Factor as (x-2)(x-3)=0; solutions x=2 and x=3; sum is 5.",
		},
		{
			ID:       "aime_2",
			Question: "How many positive divisors does 60 have?",
			Reference: "60 = 2^2 × 3 × 5. Divisor count = (2+1)(1+1)(1+1) = 12.",
		},
		{
			ID:       "aime_3",
			Question: "If f(x) = 2x + 3, what is f(f(2))?",
			Reference: "f(2) = 7; f(7) = 17.",
		},
	}
}

// AIMESample is a competition-math style sample for Judge evaluation.
// No GroundTruth field — Judge scores based on Reference solution
// + LLM-rendered rubrics.
type AIMESample struct {
	ID        string
	Question  string
	Reference string
}
