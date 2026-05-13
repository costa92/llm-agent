# PR Governance 01: 四个项目的关系

## 四个项目

这套治理规则不是围绕一个仓库孤立设计的，而是围绕 4 个关联项目建立的：

- `llm-agent`
- `llm-agent-providers`
- `llm-agent-otel`
- `llm-agent-customer-support`

## 各自角色

### `llm-agent`

核心仓库，负责：

- 定义稳定的 Go API 面
- 提供 agent、llm、memory、rag、orchestrate 等基础能力
- 作为其他 3 个仓库的依赖上游

它是整个多仓库体系里的核心契约源。

### `llm-agent-providers`

provider adapters 仓库，负责把核心能力接到具体模型服务上，例如：

- OpenAI
- Anthropic
- Ollama
- DeepSeek
- MiniMax

它消费 `llm-agent` 暴露的能力接口，并把这些接口绑定到真实 provider 行为。

### `llm-agent-otel`

observability wrappers 仓库，负责给核心运行时增加：

- tracing
- metrics
- slog bridge
- OTLP exporter wiring

它依赖 `llm-agent` 的抽象层，但关注点是运行时可观测性。

### `llm-agent-customer-support`

参考应用仓库，用来把前三个仓库组合成一个真实可运行的 demo / reference service：

- 依赖 `llm-agent` 提供基础能力
- 依赖 `llm-agent-providers` 提供模型接入
- 依赖 `llm-agent-otel` 提供可观测性

它是体系里最靠近真实业务应用的一层。

## 总体关系

可以先粗略理解成一条从“核心能力”到“真实应用”的链路：

```text
llm-agent
  -> llm-agent-providers
  -> llm-agent-otel
  -> llm-agent-customer-support
```

更准确的表达是“一个核心 + 三个下游”的扇出结构：

```text
                +----------------------+
                |      llm-agent       |
                | core contracts / API |
                +----------+-----------+
                           |
          +----------------+----------------+
          |                                 |
  +-------v---------+               +-------v--------+
  | llm-agent-      |               | llm-agent-     |
  | providers       |               | otel           |
  | provider layer  |               | observability  |
  +-------+---------+               +-------+--------+
          |                                 |
          +----------------+----------------+
                           |
                +----------v----------------+
                | llm-agent-customer-       |
                | support                   |
                | reference application     |
                +---------------------------+
```

## 为什么这会影响 PR 治理

也正因为它们是关联项目，PR 治理不能只看某一个仓库的局部体验，而必须考虑整个多仓库发布链路的一致性。

典型场景是：

1. `llm-agent` 核心 API 变更
2. `llm-agent-providers` 跟进 provider adapter
3. `llm-agent-otel` 跟进 observability wrapper
4. `llm-agent-customer-support` 跟进 reference app

如果每个仓库都沿用“必须有一个单独 approval”的僵硬规则，owner 自己的收尾 PR 会在多个仓库同时卡住。统一治理规则的价值，就是让这类依赖链式变更保持可推进。

## 这套设计要对齐什么

PR 治理最终要对齐的是 4 个项目之间的真实协作关系：

- 核心仓库继续做核心 API 演进
- 下游仓库继续按依赖链消费核心版本
- 各仓库的 PR 治理规则保持一致

也就是说，我们不是在做“某个仓库的自动合并技巧”，而是在为一个多仓库系统建立统一的变更入口规则。
