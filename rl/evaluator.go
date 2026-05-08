package rl

import (
	"context"
	"sync"
	"time"

	"github.com/costa92/llm-agent"
)

// EvaluatorOptions configures Evaluator.Run.
type EvaluatorOptions struct {
	Concurrency int           // parallel sample workers; default 1
	K           int           // Pass@K sampling; default 1
	Timeout     time.Duration // per-sample timeout; default 60s
}

// Evaluator orchestrates Agent×Sample evaluation. One Evaluator per
// (agent, reward, opts) trio — reuse Run for multiple datasets.
type Evaluator struct {
	agent  agents.Agent
	reward Reward
	opts   EvaluatorOptions
}

// NewEvaluator constructs an Evaluator. Agent + Reward must be non-nil.
func NewEvaluator(agent agents.Agent, reward Reward, opts EvaluatorOptions) *Evaluator {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.K <= 0 {
		opts.K = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	return &Evaluator{agent: agent, reward: reward, opts: opts}
}

// Run iterates the dataset, runs the agent K times per sample (bounded
// by Concurrency), scores each trajectory, and returns aggregated
// Metrics + the full trajectory list (in arbitrary order — sort by
// SampleID for deterministic display).
func (e *Evaluator) Run(ctx context.Context, ds Dataset) (Metrics, []Trajectory, error) {
	// Materialize sample list — eval needs Len() for metrics anyway.
	samples := make([]Sample, 0, ds.Len())
	for s := range ds.Iter(ctx) {
		samples = append(samples, s)
	}
	if len(samples) == 0 {
		return Metrics{}, nil, nil
	}

	type job struct{ sample Sample }
	type out struct{ traj Trajectory }

	jobs := make(chan job, len(samples)*e.opts.K)
	results := make(chan out, len(samples)*e.opts.K)

	var wg sync.WaitGroup
	for i := 0; i < e.opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				traj := e.runOne(ctx, j.sample)
				results <- out{traj: traj}
			}
		}()
	}

	go func() {
		for _, s := range samples {
			for k := 0; k < e.opts.K; k++ {
				jobs <- job{sample: s}
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	all := make([]Trajectory, 0, len(samples)*e.opts.K)
	byID := make(map[string][]Trajectory, len(samples))
	for r := range results {
		all = append(all, r.traj)
		byID[r.traj.SampleID] = append(byID[r.traj.SampleID], r.traj)
	}

	metrics := computeMetrics(all, byID, samples, e.opts.K)
	return metrics, all, nil
}

// runOne executes the agent on one sample under per-sample timeout
// and computes the Reward. Errors are captured on the Trajectory
// (not returned) so partial-failure runs still produce metrics.
func (e *Evaluator) runOne(ctx context.Context, s Sample) Trajectory {
	runCtx, cancel := context.WithTimeout(ctx, e.opts.Timeout)
	defer cancel()
	start := time.Now()
	res, err := e.agent.Run(runCtx, s.Prompt)
	traj := Trajectory{
		SampleID: s.ID,
		Prompt:   s.Prompt,
		Steps:    res.Trace,
		Answer:   res.Answer,
		Latency:  time.Since(start).Seconds(),
		Err:      err,
	}
	if err == nil {
		score, scoreErr := e.reward.Score(runCtx, traj, s)
		if scoreErr == nil {
			traj.Reward = score
		} else {
			traj.Err = scoreErr
		}
	}
	return traj
}

func computeMetrics(all []Trajectory, byID map[string][]Trajectory, samples []Sample, k int) Metrics {
	m := Metrics{SampleCount: len(samples)}
	if len(all) == 0 {
		return m
	}
	m.Accuracy = ComputeAccuracy(all, samples)
	if k > 1 {
		m.PassAtK = ComputePassAtK(byID, samples, k)
	} else {
		m.PassAtK = m.Accuracy
	}

	var lenSum, stepSum, rewardSum float64
	for _, t := range all {
		lenSum += float64(len([]rune(t.Answer)))
		stepSum += float64(len(t.Steps))
		rewardSum += t.Reward
	}
	m.AverageLength = lenSum / float64(len(all))
	m.AverageSteps = stepSum / float64(len(all))
	m.AverageReward = rewardSum / float64(len(all))
	return m
}
