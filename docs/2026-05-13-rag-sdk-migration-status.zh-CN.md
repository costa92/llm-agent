[English](./2026-05-13-rag-sdk-migration-status.md) | [简体中文](./2026-05-13-rag-sdk-migration-status.zh-CN.md)

# RAG SDK Migration Status

> 仅为归档记录。
> 本文档是历史性的迁移背景，绝不可作为开发指南或实现的事实来源。
> 当前开发必须遵循 `github.com/costa92/llm-agent-rag` 中的活代码与现行文档。

Date: 2026-05-13
Project: `github.com/costa92/llm-agent`
Historical in-repo staging path: `third_party/llm-agent-rag`
Status: external release complete, main repo switched to remote module

## Summary

`llm-agent` 中原本的 `rag/` 包已被拆分为两层：

- `github.com/costa92/llm-agent-rag`
  - 独立的 SDK 核心
  - 与提供方无关的 import / retrieve / ask 编排
  - `advanced/` LLM 辅助的检索辅助
  - 可选的 `adapter/llmagent` 桥接
- `rag/`
  - 面向现有 `llm-agent` 调用方的兼容性门面
  - 保留历史公共 API 和测试
  - 把大部分实现委托给独立 SDK

## What Moved

以下职责现在主要住在嵌入的 SDK 中：

- 文本块切分
- hash 嵌入
- 内存向量存储
- 导入编排
- 检索编排
- 默认 QA 的提示词渲染
- LLM 辅助的查询扩展和 HyDE 提示词逻辑
- 可选的 `llm-agent` adapter 代码

主仓库的 `rag/` 包现在主要提供：

- 兼容性类型和错误值
- 兼容性方法签名
- 面向遗留调用方的 adapter/转换胶水
- 现有测试和下游包所期望的工具门面行为

## Current Compatibility Shape

诸如 `memory/` 和 `context/` 的现有包仍然导入：

- `rag.Embedder`
- `rag.NewHashEmbedder`
- `rag.CosineSimilarity`
- `rag.Document`
- `rag.SearchHit`
- `rag.NewInMemoryStore`
- `rag.RAGSystem`
- `rag.AsTool`

这些入口点继续工作，但它们不再是实现逻辑的事实来源。

## Completed Milestones

已完成：

- 创建了设计和实现规划文档
- 搭建了独立 SDK 脚手架并发布为 `github.com/costa92/llm-agent-rag`
- 把 `rag.RAGSystem` 转换为 SDK 之上的兼容性门面
- 把 `MQE` / `HyDE` 提示词逻辑移入 SDK `advanced/`
- 通过门面打通工具级命名空间支持
- 把 `rag/chunk.go`、`rag/embedder.go` 和 `rag/store.go` 转换为 SDK 支撑的兼容性包装器
- 扩展 `adapter/llmagent/tool.go` 以支持：
  - `namespace`
  - `enable_mqe`
  - `enable_hyde`
  - `mqe_count`
- 把独立 SDK 推送到 GitHub 并打 tag：
  - `v0.1.0`
- 主仓库从本地 `replace` 切换为：
  - `require github.com/costa92/llm-agent-rag v0.1.0`

## Known Boundaries

### 1. Main repo now uses the remote module

vendored 的暂存副本已被移除。主仓库的解析现在走已发布的 module 版本：

- `github.com/costa92/llm-agent-rag v0.1.0`

### 2. `adapter/llmagent` is dev-only in the standalone module

核心 SDK 刻意不在其可发布的 `go.mod` 中保留对 `github.com/costa92/llm-agent` 的硬依赖。

这意味着：

- 默认的 SDK 测试在没有 `llm-agent` 的情况下通过
- 打 tag 的 adapter 测试需要一个临时的本地 `require` / `replace`

这是刻意为之，并保留了独立核心边界。

### 3. Main-repo `rag/tool.go` is still the default public entry point

虽然 SDK adapter 现在在工具路径上已达到近乎特性对等，主仓库仍从兼容层暴露 `rag.AsTool`。

目前这是可接受的，因为：

- 现有调用方无需改变
- 默认的仓库测试保持简单
- 门面保持历史行为稳定

## Recommended Next Step

下一个高价值步骤是对外化，而非更多的仓内重构。

从这里推荐的顺序：

1. 继续把 `github.com/costa92/llm-agent-rag` 作为实现的事实来源发布
2. 决定 `rag/` 是保留为一个永久的兼容性包，还是进入弃用
3. 如果弃用，为以下情形发布显式的迁移指引：
   - 直接的 SDK 导入
   - 工具调用方
   - 依赖 `rag` 兼容性类型的下游测试

## Verification Snapshot

迁移期间已验证：

- 主仓库：
  - `GOWORK=off GOCACHE=/tmp/go-build go test ./...`
- 嵌入的 SDK 核心：
  - `GOWORK=off GOCACHE=/tmp/go-build go test ./...`

`adapter/llmagent` 的打 tag 测试在添加一个临时的本地依赖之前，被刻意排除在默认的独立验证路径之外。
