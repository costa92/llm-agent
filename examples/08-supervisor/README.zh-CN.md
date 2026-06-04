[English](./README.md) | [简体中文](./README.zh-CN.md)

# 08 - Supervisor

`orchestrate.Supervisor` 的确定性演示。

`Supervisor` 是面向 planner/worker 工作流的迭代式编排原语。当 planner 需要在选择下一次 dispatch 前检视上一个 worker 的输出时，它是最佳选择。

## Run

```sh
cd examples && go run ./08-supervisor
```

## Demos

- `Basic` 展示一个 planner 在两个回合中协调两个 worker。
- `Budget` 展示预算在 planner 与 worker 调用间的传播。
- `Compose` 展示将 `Supervisor` 用作一个 `StateGraph` 节点。

## API shape

```go
sup := orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
    Planner:        plannerAgent,
    Workers:        map[string]agents.Agent{"alpha": alphaAgent},
    MaxRounds:      3,
    ParseDispatch:  parseDispatch,
    BuildAggregate: aggregateResults,
})
```

## Notes

- `MaxRounds` 是监督者循环的上限。
- `Budget.MaxCalls` 是 planner 加 worker 的 LLM 调用总数上限。
- 规范的 worker 组合栈是 `policy.Wrap(otelmodel.Wrap(provider))`；本示例并不导入 `otelmodel`。
- 触达 `MaxRounds` 时是优雅的：监督者会聚合到目前为止收集到的结果。
