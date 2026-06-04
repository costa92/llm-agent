[English](./2026-05-13-standalone-rag-sdk-implementation-plan.md) | [简体中文](./2026-05-13-standalone-rag-sdk-implementation-plan.zh-CN.md)

# Standalone RAG SDK Implementation Plan

> 仅为归档记录。
> 本实现计划是历史性的项目记录，绝不可作为当前的开发指南。
> 当前开发必须遵循 `github.com/costa92/llm-agent-rag` 中的活代码与现行文档。

> **致 agentic worker：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 来逐任务实现本计划。各步骤使用 checkbox（`- [ ]`）语法以便追踪。

**Goal：** 设计并搭建一个独立的 Go RAG SDK，带有抽象导入、检索、自定义 LLM 生成和自定义提示词模板接缝，同时把 `llm-agent` 集成隔离在 adapter 之后。

**Architecture：** 本实现把可复用的 RAG 原语抽取进一个新的独立 module 形状。核心包使用 SDK 本地的接口拥有 import、split、embed、store、retrieve 和 ask 编排。`llm-agent` 集成点移到 adapter 包，从而独立核心不导入 `llm-agent` 类型。

**Tech Stack：** Go 1.26、仅标准库默认、以现有 `rag` 包为迁移源、面向 `llm-agent` 的 adapter 接缝

---

## File Map

### Planning / design artifacts

- Create: `docs/2026-05-13-standalone-rag-sdk-design.md`
- Create: `docs/2026-05-13-standalone-rag-sdk-implementation-plan.md`

### Future standalone SDK module

- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/go.mod`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/doc.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/source.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/embedder.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/vector.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/store.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/template.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/options.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/retrieve.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/ask.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/errors.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool.go`

### Existing source files to reference during extraction

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/chunk.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/embedder.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/store.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/advanced.go`
- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`

## Task 1: Initialize standalone SDK module and public package skeleton

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/go.mod`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/doc.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/vector.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/types.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/errors.go`

- [ ] **Step 1: Create the failing package-compilation test scaffold**

添加最小测试：

```go
package rag_test

import "testing"

func TestModulePackagesCompile(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify the new module is not yet wired**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./...`
预期：FAIL，缺少文件或缺少包符号。

- [ ] **Step 3: Create the module and public type skeleton**

定义：

- `ingest.Document`
- `embed.Vector`
- `store.StoredChunk`、`store.Hit`、`store.Stats`
- `generate.Message`、`generate.Request`、`generate.Response`
- `prompt.RenderContext`
- `rag/errors.go` 中的哨兵错误

- [ ] **Step 4: Re-run package compilation**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./...`
预期：PASS，或仅在后续刻意缺失的接口上失败。

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: scaffold standalone rag sdk module"
```

## Task 2: Implement import abstractions and default splitter

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/source.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/splitter_test.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/ingest/import_test.go`

- [ ] **Step 1: Write failing splitter tests**

覆盖：

- 空输入
- 单个文本块
- 多文本块段落切分
- 重叠行为
- 确定性输出

- [ ] **Step 2: Run splitter tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./ingest -run 'Test(CharSplitter|StableChunkIDs)' -count=1`
预期：FAIL，`CharSplitter` 未定义或行为错误。

- [ ] **Step 3: Port and adapt the current chunker**

实现基于：

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/chunk.go`

调整它以在 `ingest.Document` 上操作，并发出带稳定 ID 的 `ingest.Chunk`，例如：

- `<docID>#chunk-0`
- `<docID>#chunk-1`

- [ ] **Step 4: Re-run splitter tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./ingest -run 'Test(CharSplitter|StableChunkIDs)' -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add ingest
git commit -m "feat: add import abstractions and default splitter"
```

## Task 3: Implement embed abstractions and hash embedder

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/embedder.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/embed/hash_test.go`

- [ ] **Step 1: Write failing embedder tests**

覆盖：

- 默认维度回退
- 确定性向量
- 余弦相似度边界
- 零向量归一化的边界情况

- [ ] **Step 2: Run embed tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./embed -count=1`
预期：FAIL，`HashEmbedder` 或辅助未定义。

- [ ] **Step 3: Port the current hash embedder**

实现基于：

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/embedder.go`

- [ ] **Step 4: Re-run embed tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./embed -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add embed
git commit -m "feat: add sdk embedder abstractions"
```

## Task 4: Implement store abstractions and in-memory backend

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/store.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/store/inmemory_test.go`

- [ ] **Step 1: Write failing store tests**

覆盖：

- upsert/get/remove
- topK 排名
- 维度不匹配
- 命名空间隔离
- stats 报告

- [ ] **Step 2: Run store tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./store -count=1`
预期：FAIL，`Store` 或 `InMemoryStore` 未定义。

- [ ] **Step 3: Port and adapt the in-memory store**

实现基于：

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/store.go`

调整：

- 存储 `Namespace`
- 保留显式的维度检查
- 接受批量 `Upsert`

- [ ] **Step 4: Re-run store tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./store -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add store
git commit -m "feat: add in-memory vector store"
```

## Task 5: Implement generation and prompt-template seams

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/generate/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/template.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/prompt/default_test.go`

- [ ] **Step 1: Write failing prompt-template tests**

覆盖：

- 渲染后的请求包含 system prompt
- 渲染后的请求按稳定顺序包含检索到的文本块
- 渲染后的请求包含问题文本
- 可配置的引用指令

- [ ] **Step 2: Run prompt tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./prompt -count=1`
预期：FAIL，`Template` 或 `DefaultQATemplate` 未定义。

- [ ] **Step 3: Implement the prompt seams**

添加：

- `generate.Model`
- `prompt.Template`
- `prompt.DefaultQATemplate`

要求：

- 确定性渲染
- 不依赖 `llm-agent`

- [ ] **Step 4: Re-run prompt tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./prompt -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add generate prompt
git commit -m "feat: add generation and prompt template seams"
```

## Task 6: Implement core RAG system import, retrieve, and ask flows

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/options.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/import.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/retrieve.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/ask.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/rag/system_test.go`

- [ ] **Step 1: Write failing system tests**

覆盖：

- 带显式文档切片的 `Import`
- 经由 source 迭代器的 `ImportFrom`
- `Retrieve` 顺利路径
- 用假 generator 的 `Ask` 顺利路径
- `ErrEmptyQuery`
- `ErrModelRequired`

- [ ] **Step 2: Run system tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./rag -count=1`
预期：FAIL，`System` 未定义或流水线行为错误。

- [ ] **Step 3: Port the current orchestration logic**

实现基于：

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/rag.go`

必需改动：

- 用 SDK 本地的 `generate.Model` 替换 `llm.ChatModel`
- 把编排拆分成 `Import`、`Retrieve` 和 `Ask`
- 使用来自 `ingest` 的稳定文本块 ID
- 返回 `Answer{Text, Hits, Prompt}`

- [ ] **Step 4: Re-run system tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./rag -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add rag
git commit -m "feat: add standalone rag system"
```

## Task 7: Add `llm-agent` adapter package

**Files:**
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model.go`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/model_test.go`
- Test: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/adapter/llmagent/tool_test.go`

- [ ] **Step 1: Write failing adapter tests**

覆盖：

- `llm.ChatModel` 到 `generate.Model` 的请求映射
- 响应文本透传
- `AsTool` 动作覆盖：
  - add_text
  - search
  - ask
  - remove
  - stats

- [ ] **Step 2: Run adapter tests to verify they fail**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./adapter/llmagent -count=1`
预期：FAIL，adapter 类型未定义。

- [ ] **Step 3: Move `tool.go` logic into the adapter**

实现基于：

- `/home/hellotalk/code/go/src/github.com/costa92/llm-agent/rag/tool.go`

必需改动：

- 依赖新的独立 SDK 类型
- 把 `llm-agent` 导入保持在 adapter 包本地

- [ ] **Step 4: Re-run adapter tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./adapter/llmagent -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add adapter/llmagent
git commit -m "feat: add llm-agent rag adapters"
```

## Task 8: Write README and usage examples

**Files:**
- Modify: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/README.md`
- Create: `/home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag/examples/basic_import_and_ask_test.go`

- [ ] **Step 1: Write failing example test**

添加一个示例，它：

- 用 `HashEmbedder` 构建一个 system
- 导入两个文档
- 检索命中
- 用一个假 generator 提一个问题

- [ ] **Step 2: Run the example test to verify it fails**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -run Example -count=1`
预期：FAIL，直到 README/示例类型与实现匹配。

- [ ] **Step 3: Document the public API**

README 小节：

- 项目目标
- 导入流程
- 检索流程
- 自定义 generator
- 自定义提示词模板
- `llm-agent` adapter 说明
- v0.1 限制

- [ ] **Step 4: Re-run example tests**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -run Example -count=1`
预期：PASS

- [ ] **Step 5: Commit**

```bash
git add README.md examples
git commit -m "docs: add standalone rag sdk usage guide"
```

## Task 9: Final verification and release-readiness check

**Files:**
- Review: entire module

- [ ] **Step 1: Run the full test suite**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && GOCACHE=/tmp/go-build go test ./... -count=1`
预期：PASS

- [ ] **Step 2: Verify core packages do not import `llm-agent`**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && rg -n 'github.com/costa92/llm-agent' ingest embed store prompt generate rag`
预期：无匹配

- [ ] **Step 3: Verify adapter package is the only `llm-agent` integration seam**

Run: `cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent-rag && rg -n 'github.com/costa92/llm-agent' adapter`
预期：仅在 `adapter/llmagent` 下有匹配

- [ ] **Step 4: Commit final cleanup if needed**

```bash
git add .
git commit -m "chore: finalize standalone rag sdk extraction"
```

## Spec Coverage Check

本计划覆盖：

- 经由 `Import` 和 `ImportFrom` 的抽象导入
- 经由 `Retrieve` 的抽象检索
- 经由 `generate.Model` 的自定义 LLM
- 经由 `prompt.Template` 的自定义提示词模板
- 内置的 `InMemoryStore`
- 面向 `llm-agent` 的 adapter 隔离

刻意从 spec 中推迟：

- 生产向量后端
- 核心中的 MQE / HyDE
- HTTP 服务
- CLI

## Execution Handoff

计划完成并保存到 `docs/2026-05-13-standalone-rag-sdk-implementation-plan.md`。两个执行选项：

**1. Subagent-Driven（推荐）** —— 我为每个任务派遣一个全新的 subagent，在任务之间评审，快速迭代

**2. Inline Execution** —— 在本会话中用 executing-plans 执行任务，带检查点的批量执行

选哪种方式？
