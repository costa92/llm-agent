// Package bench layers benchmark + judging primitives on top of
// pkg/llm/agents/rl. Phase 7 of the agents framework.
//
// What's here:
//
//   - BFCL (function calling): mini fixture, BFCLEvaluator, simplified
//     parseSimpleCall + matchFunctionCalls (no Python AST dependency).
//   - GAIA (general assistant): mini fixture, GAIAEvaluator, Quasi-
//     Exact-Match scoring (number normalization + article/punct strip).
//   - Judge (LLM-as-Judge): rubric-driven scoring + ComputeJudgeMetrics.
//   - Win Rate (pairwise): WinRateEvaluator with SwapEval (default on)
//     to fight position bias; ComputeWinRate aggregator.
//   - Reporter: Markdown + ASCII bar + JSON renderer; one-shot helpers
//     for each evaluator (RenderBFCLReport / RenderGAIAReport / etc.).
//
// What's NOT here (deliberate cuts per spec §11):
//
//   - Real BFCL / GAIA dataset downloaders (size + auth costs)
//   - BFCL official CLI integration / GAIA leaderboard submission
//   - Python AST matching for BFCL
//   - Gradio annotation UI
//   - Engineering perf metrics (latency / throughput / cost) — separate
//     concern, would need a perf/ subpackage
//
// All evaluators reuse rl.Dataset / rl.Trajectory / rl.EvaluatorOptions
// — DRY with Phase 6.
//
// # Portability
//
// bench inherits the agents/pkg/llm portability contract.
package bench
