[English](./SUPERVISOR.md) | [简体中文](./SUPERVISOR.zh-CN.md)

# Supervisor

`Supervisor` 是 `orchestrate` 中的迭代式多智能体编排原语。它运行一个 planner/worker 循环：planner 决定下一次 worker dispatch，worker 执行，planner 观察结果，如此循环往复，直到 planner 完成或触达 `MaxRounds`。

## When to use it

当一个 agent 必须跨多个回合协调专家，且下一步取决于先前的 worker 输出时，使用 `Supervisor`。

典型适用：

- 带后续检查的客服分诊
- reviewer/revise 循环
- 需要增量 dispatch 的调研工作流
- 嵌套编排，其中一个监督者自身又成为一个 worker

当 planner 能一次性拆分工作、worker 并行运行、且单个 aggregator 合并结果时，改用 `FanOutFanIn`。

## Core model

`SupervisorOptions` 有五个必需部分：

- `Planner` 决定接下来做什么。
- `Workers` 是以 worker 名称为 key 的 dispatch 表。
- `MaxRounds` 限制 planner/worker 循环。
- `ParseDispatch` 把 planner 的文本转成一个 `Dispatch`。
- `BuildAggregate` 把收集到的 `WorkerResult` 值转成最终答案。

每个回合遵循以下流程：

1. planner 收到任务以及先前的 worker 结果。
2. `ParseDispatch` 把 planner 答案转换成一个 `Dispatch` 或 `nil`。
3. 被选中的 worker 在 `Dispatch.Input` 上运行。
4. 结果被存储，并在下一回合返回给 planner。
5. `BuildAggregate` 产生最终答案。

`MaxRounds` 是一个优雅的停止，而非错误。当触达上限时，监督者返回到目前为止收集到的结果。

## Minimal example

```go
sup := orchestrate.NewSupervisor("demo", orchestrate.SupervisorOptions{
    Planner: plannerAgent,
    Workers: map[string]agents.Agent{
        "alpha": alphaAgent,
        "beta":  betaAgent,
    },
    MaxRounds: 3,
    ParseDispatch: func(answer string) (*orchestrate.Dispatch, error) {
        // parse planner text into worker name + input
    },
    BuildAggregate: func(results []orchestrate.WorkerResult) (string, error) {
        // merge all worker outputs into the final answer
    },
})
```

## Example project

三个确定性演示见 [`examples/08-supervisor`](../examples/08-supervisor)：

- 基础的 planner + workers 循环
- 用 `budget.WithBudget` 的预算传播
- 在 `StateGraph` 内的嵌套使用

用以下命令运行：

```bash
cd examples && go run ./08-supervisor
```

## Related primitives

- `Pipeline` 用于固定交接
- `FanOutFanIn` 用于一次性并行分解
- `StateGraph` 用于更底层的类型化控制流
