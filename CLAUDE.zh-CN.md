[English](./CLAUDE.md) | [简体中文](./CLAUDE.zh-CN.md)

# Project guide for AI assistants

本项目使用 **GSD（Get Shit Done）**进行 milestone 规划。`.planning/` 目录是「在建什么、为什么建」的唯一事实来源。

## Read first (in this order)

1. `.planning/PROJECT.md` —— 本项目「是什么」、核心价值、需求、约束、关键决策
2. `.planning/STATE.md` —— 我们当前在哪里（当前 phase、plan、近期活动）
3. `.planning/ROADMAP.md` —— phase 计划（8 个 phase，多仓伞形）
4. `.planning/REQUIREMENTS.md` —— 65 项 v1 需求 + 到各 phase 的可追溯性
5. `.planning/research/SUMMARY.md` —— 来自研究的横切；K1–K7 基石决策就住在这里
6. `.planning/config.json` —— 工作流开关（YOLO 模式、标准粒度、并行化开启、所有门禁开启）

当人类下达指令时，先查这些文件再去探索代码库。多数「我该做什么？」的问题在那里已有答案。

## Project at a glance

- **Repo：** `github.com/costa92/llm-agent` —— 一个仅标准库的 Go LLM agents 框架（v0.2.0 → v0.3.0）。
- **Milestone（v0.3）：** 加入真实的 provider adapter（OpenAI/Anthropic/Ollama/DeepSeek/MiniMax）、OpenTelemetry 可观测性，以及一个可部署的客服参考服务 —— 全部放在**兄弟仓**里，从而核心保持仅标准库。
- **4 仓伞形：** `llm-agent`（核心，本仓）、`llm-agent-providers`、`llm-agent-otel`、`llm-agent-customer-support`。
- **节奏：** 单人、副业项目、无截止日期。**质量 > 速度。**

## Hard rules

1. **核心仓（`llm-agent`）保持仅标准库。** 没有 `go.sum`，`go.mod` 中没有非标准库依赖。永远如此。如果某个特性需要依赖，它就进兄弟仓。
2. **v0.3 中不引入 K8s。** Helm/K8s 清单按 `PITFALLS.md` 的 Pitfall 16 不在范围内。不要加它们；把任何加它们的请求标记为针对 milestone 的范围蔓延。
3. **已 tag 的发布分支中不出现 `replace` 指令。** `replace` 只是本地开发的逃生舱。CI 门禁强制此项（INFRA-04）。
4. **`go.work` 在每个仓里都被 `.gitignore`。** CI 用 `GOWORK=off` 运行。
5. **能力是按（provider × model）划分，而非按 provider。** 一个 provider 实例在构造时绑定一个模型（`openai.New(openai.WithModel("gpt-4o"))`）；`Info()` 反映的是**那个**模型的能力。（基石 K2。）
6. **流事件是一个类型化联合，而非最小公分母 chunk。** `StreamEvent.Kind` 枚举 + 稳定的每个工具调用的 `Index` 字段。（基石 K1。）
7. **OTel 以装饰器包装器的方式附加，绝不用钩子。** `otelmodel.Wrap(inner) ChatModel`。（基石 K3。）
8. **Refsvc 从第一天起就有硬上限 + `DISABLE_LLM=1` panic 开关。** 不是后续补丁。（基石 K7。）

## GSD slash commands you can invoke

这些是面向用户的 slash 命令，驱动规划生命周期。用户运行它们；你协助。

- `/gsd-plan-phase <N>` —— 为 phase N 创建详细计划
- `/gsd-execute-phase <N>` —— 执行 phase N 中所有计划（波次式并行化）
- `/gsd-discuss-phase <N>` —— 在规划一个 phase 前收集上下文
- `/gsd-progress` —— 情境检查；接下来做什么？
- `/gsd-transition` —— 从一个 phase 移动到下一个（更新 PROJECT.md、REQUIREMENTS、STATE）
- `/gsd-debug` —— 带持久状态的系统化调试
- `/gsd-code-review` —— 评审某个 phase 期间改动的源文件

## When the user asks for code

- 信任仓库中现有的模式。5 种 agent 范式（Simple/ReAct/Reflection/PlanSolve/FunctionCall）和 `README.md` 中的包布局都已验证；除非当前 phase 明确要求，否则不要重构。
- 在改动 `llm/`、`agents/` 或 `orchestrate/` 中的公共类型之前，查看 `.planning/PROJECT.md` 的「Validated」—— 那些能力是锁定的。
- `ScriptedLLM`（在 `scriptedllm_test.go` 中）是规范的模拟。`/examples/` 中的示例使用它 —— 让新示例也保持确定性。
- 测试通过 `go vet ./... && go test ./...` 运行。按设计没有 `go.sum`。

## When the user asks "what's next?"

最快的答案是 `cat .planning/STATE.md` —— 它指明当前 phase + plan 和最近一次提交。如果他们想开始下一个具体步骤，那就是：

```
/gsd-plan-phase 0     # currently — Multi-repo infra + llm/v2 keystone interfaces
```

Phase 0 必须在 Phase 1（走通骨架的开始）之前完成。

## Files you should NOT touch without explicit ask

- `LICENSE`、`OWNERS`、`CHANGELOG.md`（仅在版本提升时更新）
- `.github/workflows/test.yml`（Phase 0 会重构它；在那之前别动）
- `go.mod`（这里的任何改动都必须证明仍在仅标准库范围内）

## When in doubt

问。规划工件很详细（PROJECT.md 本身约 150 行，ROADMAP.md 是 8 个 phase × 每个约 30 行，研究包约 13k 行代码）。多数「我该不该 X？」的问题在那里都有答案 —— 先查比来回问更快。
