[English](./README.md) | [简体中文](./README.zh-CN.md)

# 06 — Budget / cancellation context

一个确定性、无网络的演示，将 `budget.WithBudget` 接入一个 `SimpleAgent`。它演练由阻塞点 `agent_chatmodel.go::generateFromPrompt` 强制执行的三个预算维度（`MaxCalls`、`MaxTokens`、`MaxWall`）。覆盖需求 CC-1（基石 K7 的本地表亲）。

Run：

```bash
cd examples && go run ./06-budget
```

`examples/` 是它自己的 Go module（`replace … => ../`），所以该演示会即时拾取对框架的本地改动。

## The Budget struct

```go
type Budget struct {
    MaxTokens int           // cap on cumulative resp.Usage.TotalTokens. 0 = no cap.
    MaxCalls  int           // cap on attempts (Q2: counts attempts, not successes). 0 = no cap.
    MaxWall   time.Duration // cap on wall-clock; WithBudget derives context.WithDeadline.
    MaxCost   float64       // cap on accumulated cost. v1.2: NOT charged (CostMapper deferred to v1.3).
}
```

## How enforcement happens

每一种 agent 范式（Simple / ReAct / Reflection / PlanSolve / FunctionCall）都汇聚到 `agent_chatmodel.go::generateFromPrompt`。该辅助函数围绕底层的 `model.Generate` 做两次 `Tracker.Charge` 调用：

1. **Pre-call** `Charge(Usage{Calls: 1})` —— 在网络请求之前触发。当 `MaxCalls` 将被超出时返回 `ErrCallsExceeded` 并**短路掉 LLM 调用**。按照 Q2，阻塞点对尝试计数：一次被拒绝的扣减仍消耗上限中的一次调用。
2. **Post-call** `Charge(Usage{Tokens: resp.Usage.TotalTokens})` —— 在一次成功的 Generate 之后触发。当 `MaxTokens` 将被超出时，阻塞点同时返回有效响应**和**哨兵错误（`ErrTokensExceeded`）。该演示通过统计底层 model.Generate 被调用的次数来证明这一点。

当没有附加 `Tracker` 时（`budget.From(ctx)` 返回 `(nil, false)`），两个分支都是空操作 —— 这是承载性的向后兼容保证。

## Wall-clock + cancellation

当 `MaxWall > 0` 时，`WithBudget` 会自动组合一个 `context.WithDeadline`。调用方**不**需要再包装自己的 deadline：

```go
ctx, _ := budget.WithBudget(context.Background(), budget.Budget{MaxWall: 50*time.Millisecond})
// ctx carries both the Tracker AND a deadline at now+50ms.
```

该演示用一个 `slowLLM` 垫片包装脚本化模型，它在尊重 `ctx.Done()` 的同时睡眠 200 ms。deadline 会在响应返回之前触发，阻塞点直接抛出原始的 `context.DeadlineExceeded`（wall-clock 不引入新的错误面 —— 决策 4）。

## What's deferred to v1.3

- `budget.Wrap(inner) llm.ChatModel` 装饰器，让非 agent 的调用方（rag、context-compress、eval 套件）无需经过 `generateFromPrompt` 即可选择开启。
- `Estimator` 接口 —— 调用前的 token 拒绝（`MaxTokens` 变为使用 `Estimator.EstimateTokens(req)` 的调用前检查）。
- `CostMapper` / 成本表 —— 定价 × tokens → `Usage.Cost`，从而 `MaxCost` 真正可被扣减。

## Stdlib-only

`budget` 包仅依赖 `context`、`errors`、`fmt`、`sync`、`sync/atomic`、`time` —— 核心仓库的仅标准库不变量得以保持。
