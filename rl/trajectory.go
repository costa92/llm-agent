package rl

import "github.com/costa92/llm-agent"

// Trajectory captures one Agent's run on one Sample: the prompt, the
// step-by-step trace, the final answer, and the reward assigned.
type Trajectory struct {
	SampleID string
	Prompt   string
	Steps    []agents.Step
	Answer   string
	Reward   float64
	Latency  float64 // seconds (set by Evaluator)
	Err      error   // non-nil if the agent or reward failed
}

// Episode is K trajectories from a single Sample (Pass@K sampling).
// AggReward is whatever aggregation the caller wants — Evaluator
// fills it via mean(rewards).
type Episode struct {
	SampleID     string
	Trajectories []Trajectory
	AggReward    float64
}
