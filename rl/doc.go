// Package rl is the EVALUATION layer of Agentic Reinforcement Learning.
//
// What this package gives you:
//
//   - Dataset / Sample loaders (JSONL + GSM8K)
//   - Trajectory / Episode types reusing pkg/llm/agents.Step
//   - Reward interface + 4 built-ins (Accuracy / LengthPenalty /
//     StepBonus / Composite)
//   - Evaluator with bounded concurrency + Pass@K + per-sample timeout
//   - Pure metric functions: ComputeAccuracy / ComputePassAtK /
//     ComputeFormatCorrectness / ComputeNumericError
//   - TrainerProxy interface (default: UnsupportedTrainer fails fast)
//
// What this package DOES NOT do (and intentionally never will in Go):
//
//   - SFT / GRPO / PPO / LoRA / any gradient-based training.
//     Training requires Python TRL+transformers+torch. Go bridges
//     to Python via a TrainerProxy implementation (out of scope).
//
// # Recommended workflow
//
// Train in Python (TRL → produces a HF checkpoint), serve via
// vllm/TGI/sglang as OpenAI-compatible. Then point pkg/llm at the
// served endpoint and use this package's Evaluator to benchmark.
//
// # Portability
//
// rl inherits the agents/pkg/llm portability contract — no internal/*,
// no project pkg/*, no business vocabulary.
package rl
