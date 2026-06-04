[English](./2026-05-13-standalone-rag-sdk-design.md) | [简体中文](./2026-05-13-standalone-rag-sdk-design.zh-CN.md)

# Standalone RAG SDK Design

Date: 2026-05-13
Planning repo: `github.com/costa92/llm-agent`
Target repo: `github.com/costa92/llm-agent-rag`
Status: proposed

## Goal

设计一个独立的 Go RAG SDK，它之后可以住在自己的仓库里，并支持三个一等工作流：

- 抽象的数据导入
- 抽象的数据检索
- 使用自定义 LLM 和自定义提示词模板的答案生成

该 SDK 在其核心包中绝不可依赖 `llm-agent` 核心类型。任何 `llm-agent` 集成都必须住在 adapter 之后。

## Scope

### In scope

- 一个纯 Go 的 SDK 布局
- 抽象的导入流水线
- 抽象的检索流水线
- 可插拔的文本生成接口
- 可插拔的提示词模板接口
- 一个默认切分器
- 一个默认嵌入器
- 一个默认 store：
  - `InMemoryStore`
- 一个默认 QA 提示词模板
- 一个面向 `llm-agent` 的 adapter 边界
- 从当前 `rag/` 包的迁移映射

### Out of scope for v0.1

- HTTP 服务层
- CLI 工具
- 内置的生产向量后端，如 pgvector、Qdrant 或 Milvus
- 核心中的 MQE / HyDE
- 重排器
- 超出一个最小 `map[string]any` 的 filter DSL 设计
- 异步摄入编排
- 分布式索引任务

## Design Choice

构建一个带小核心和显式 adapter 包的独立 SDK。

选定的形状：

- 核心包拥有 import、split、embed、store、retrieve、prompt 和 ask
- 核心包定义它们自己的 generator 接口
- `llm-agent` 集成被隔离在 `adapter/llmagent` 中

为什么：

- 让独立 SDK 在 `llm-agent` 生态之外仍可复用
- 保留 `rag` 中已存在的良好接缝：
  - `Chunker`
  - `Embedder`
  - `VectorStore`
- 避免把 `llm.ChatModel` 或 `agents.Tool` 泄漏进核心公共 API

## Recommended Repository Layout

```text
llm-agent-rag/
├── go.mod
├── README.md
├── doc.go
├── ingest/
│   ├── source.go
│   ├── splitter.go
│   ├── types.go
│   └── import.go
├── embed/
│   ├── embedder.go
│   ├── hash.go
│   └── vector.go
├── store/
│   ├── types.go
│   ├── store.go
│   └── inmemory.go
├── prompt/
│   ├── template.go
│   ├── default.go
│   └── types.go
├── generate/
│   ├── model.go
│   └── types.go
├── rag/
│   ├── options.go
│   ├── system.go
│   ├── import.go
│   ├── retrieve.go
│   ├── ask.go
│   └── errors.go
└── adapter/
    └── llmagent/
        ├── model.go
        └── tool.go
```

## Core Concepts

### 1. Import

该 SDK 不可假设数据来自单个原始字符串。

核心导入抽象：

```go
type Document struct {
    ID       string
    Title    string
    Content  string
    Metadata map[string]any
}

type Source interface {
    Next(ctx context.Context) (Document, error)
}
```

v0.1 支持的导入形状：

- `Import(ctx, []Document, opts)`
- `ImportFrom(ctx, Source, opts)`

导入流水线：

1. 读取文档
2. 切分成文本块
3. 派生稳定的文本块 ID
4. 嵌入每个文本块
5. upsert 进 store

### 2. Split

切块仍是一个一等接缝：

```go
type Splitter interface {
    Split(doc Document, maxChars int) []Chunk
}
```

默认实现：

- `CharSplitter`

要求：

- 稳定的文本块顺序
- 确定性输出
- 重叠支持
- 文本块元数据富化：
  - 来源文档 ID
  - 文本块索引
  - 文本块总数

### 3. Embed

嵌入保持与提供方无关：

```go
type Vector []float32

type Embedder interface {
    Embed(ctx context.Context, text string) (Vector, error)
    Dimension() int
}
```

默认实现：

- `HashEmbedder`

理由：

- 让 v0.1 零依赖且在测试中可运行
- 让下游用户换入 OpenAI、Ollama、Voyage 或自定义嵌入器

### 4. Store

v0.1 中该 SDK 只附带一个内置 store：

- `InMemoryStore`

Store 抽象：

```go
type StoredChunk struct {
    ID        string
    Namespace string
    DocID     string
    Title     string
    Content   string
    Vector    Vector
    Metadata  map[string]any
}

type Query struct {
    Namespace string
    Vector    Vector
    TopK      int
    Filters   map[string]any
}

type Hit struct {
    Chunk StoredChunk
    Score float64
}

type Store interface {
    Upsert(ctx context.Context, chunks []StoredChunk) error
    Search(ctx context.Context, q Query) ([]Hit, error)
    Get(ctx context.Context, id string) (StoredChunk, error)
    Remove(ctx context.Context, id string) error
    Stats(ctx context.Context, namespace string) (Stats, error)
}
```

约束：

- store 维度必须固定
- 维度不匹配必须显式失败
- 命名空间从第一天起就在 API 中受支持
- v0.1 中 filter 可保持最小

### 5. Retrieve

原始检索必须在没有任何 LLM 的情况下仍可使用。

核心调用形状：

```go
type SearchOptions struct {
    TopK      int
    Namespace string
    Filters   map[string]any
}

func (s *System) Retrieve(ctx context.Context, query string, opts SearchOptions) ([]store.Hit, error)
```

行为：

- 嵌入查询
- 执行向量搜索
- 降序排序命中
- 返回检索结果而不生成

这使得不想要答案合成的应用也能独立使用检索。

### 6. Generate

核心 SDK 不可依赖 `llm.ChatModel`。

SDK 本地的生成接口：

```go
type Message struct {
    Role    string
    Content string
}

type Request struct {
    SystemPrompt string
    Messages     []Message
    Metadata     map[string]any
}

type Response struct {
    Text string
}

type Model interface {
    Generate(ctx context.Context, req Request) (Response, error)
}
```

这刻意保持很小，因为当前的 `rag` 只需要提示词进、文本出的单次生成。

### 7. Prompt Templates

提示词构造必须是一个一等接缝，而非 `Ask` 中硬编码的字符串拼接。

```go
type RenderContext struct {
    Question  string
    Namespace string
    Hits      []store.Hit
    Metadata  map[string]any
}

type Template interface {
    Render(ctx context.Context, rc RenderContext) (generate.Request, error)
}
```

默认实现：

- `DefaultQATemplate`

要求：

- 可定制的 system prompt
- 可定制的答案指令
- 确定性的文本块渲染顺序
- 可选的引用指令

### 8. Ask

答案生成坐落在检索和模板化之上：

```go
type AskOptions struct {
    Search   SearchOptions
    Template prompt.Template
    Metadata map[string]any
}

type Answer struct {
    Text   string
    Hits   []store.Hit
    Prompt generate.Request
}

func (s *System) Ask(ctx context.Context, question string, opts AskOptions) (Answer, error)
```

行为：

1. 检索命中
2. 通过选定的模板渲染提示词
3. 调用配置的 generator
4. 返回答案文本加上支撑性的检索上下文

## Stable Identity Model

当前的 `chunk_N` ID 对一个独立 SDK 来说不够。

v0.1 要求：

- 每个被导入的来源文档都有一个稳定的来源 ID
- 文本块 ID 派生自：
  - 来源 ID
  - 文本块索引

推荐的形状：

- `doc-123#chunk-0`
- `doc-123#chunk-1`

为什么：

- 确定性的重新导入
- 安全删除
- 溯源追踪
- 更易去重

## Error Behavior

独立 SDK 需要它自己狭窄的错误集：

- `ErrEmptyQuery`
- `ErrModelRequired`
- `ErrDimensionMismatch`
- `ErrChunkerRequired`，仅当构造器允许 nil 而无默认值时
- `ErrNotFound`

指引：

- 让错误保持类型化或基于哨兵
- 在被包装的错误中包含流水线阶段：
  - import
  - split
  - embed
  - upsert
  - search
  - generate

## Adapter Boundary

一切依赖 `llm-agent` 的东西都移到一个 adapter 包之后。

示例：

- `llm.ChatModel -> generate.Model`
- `rag.System -> agents.Tool`

这保留了：

- 独立 SDK 的纯净
- 现有的 `llm-agent` 工效
- 未来在 `llm-agent` 之外的提供方复用

## Migration Mapping From Current `rag/`

原样移动：

- `rag/chunk.go`
- `rag/embedder.go`
- `rag/store.go`
- `rag/chunk_test.go`
- `rag/embedder_test.go`
- `rag/store_test.go`

重构进核心：

- `rag/rag.go`
  - 用 SDK 本地的 `generate.Model` 替换 `llm.ChatModel`
- `rag/advanced.go`
  - 如果之后保留，让它依赖 SDK 本地的生成
- `rag/doc.go`
  - 移除 `llm-agent` 可移植性措辞

移入 adapter 层：

- `rag/tool.go`
- `rag/tool_test.go`
- `rag/llm_embedder_test.go` 桥接模式

## Testing Strategy

### Core SDK tests

- 切分器确定性
- hash 嵌入器维度和余弦行为
- 内存 store 的 CRUD 和排名
- 导入流水线顺利路径
- 检索顺利路径
- 用假 generator 的 ask 顺利路径
- 提示词模板渲染
- 维度不匹配失败
- 稳定文本块 ID 生成

### Adapter tests

- `llm.ChatModel` adapter 的 request/response 映射
- `agents.Tool` 包装行为

## v0.1 Success Criteria

当独立 SDK 能做到以下时即为成功：

1. 通过一个抽象 source 导入多个文档
2. 用默认实现切分并嵌入它们
3. 把它们索引进一个内存 store
4. 为一个查询检索排名后的命中
5. 使用用户提供的 model adapter 生成答案
6. 让调用方在不修改检索代码的情况下换用提示词模板
7. 通过一个 adapter 而非直接的核心导入集成回 `llm-agent`
