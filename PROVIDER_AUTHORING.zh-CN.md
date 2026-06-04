[English](./PROVIDER_AUTHORING.md) | [简体中文](./PROVIDER_AUTHORING.zh-CN.md)

# Provider Author Guide

**Version:** v0.1（Phase 1 - Generate-only 契约）  
**Applies to:** `github.com/costa92/llm-agent` v0.3.0+

> **这是什么：** 一个 Go provider adapter 要声称与 `llm.ChatModel` 一致所必须满足的契约。
> v0.1 仅覆盖 Generate（同步）。流式、工具调用、嵌入，以及 OTel 专属指引会在
> 后续 milestone phase 中加入。

## 1. Audience and Scope

本文档面向围绕某个 LLM 提供方 API 构建 Go adapter、并将其暴露为 `llm.ChatModel` 的作者。

规范的 Phase 1 示例位于兄弟仓：

- `github.com/costa92/llm-agent-providers/openai`
- `github.com/costa92/llm-agent-providers/anthropic`
- `github.com/costa92/llm-agent-providers/ollama`
- `github.com/costa92/llm-agent-providers/deepseek`
- `github.com/costa92/llm-agent-providers/minimax`

最初的 Phase 1 示例只有 Generate。当前的 provider 仓库还在 `deepseek` 和 `minimax` 中包含了后续 phase 的 stream/tool 覆盖。新 adapter 应在结构上镜像最接近的协议家族。

Phase 1 范围：

- 单次 `Generate`
- 稳定的 `Info()`
- 类型化的提供方错误
- 经由 `internal/contract` 的共享一致性

v0.1 不在范围内：

- 流式
- 原生工具调用
- 嵌入
- 结构化输出
- 超出 `reported` / `unknown` 的三态成本记录
- 重试状态机
- OTel 装饰器

## 2. Contract

每个提供方都实现 `llm.ChatModel`：

```go
type ChatModel interface {
	Generate(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (StreamReader, error)
	Info() ProviderInfo
}
```

来源：`llm/chatmodel.go`。

Phase 1 边界：`Stream` 存在于接口上，但 provider adapter 在 Phase 2 之前可以返回一个未实现错误。该方法现在就在场，从而类型形状在走通骨架期间保持稳定。

重要的支撑类型：

- `llm.Request`
- `llm.Response`
- `llm.Message`
- `llm.Usage`
- `llm.ProviderInfo`
- `llm.Capabilities`
- `llm.AuthError`
- `llm.RateLimitError`
- `llm.InvalidRequestError`
- `llm.TransientError`

所有实现都必须可安全并发使用。对同一个值的并发 `Generate` 和 `Stream` 调用是契约的一部分。

## 3. Generate Contract

`Generate(ctx, req)` 必须满足以下行为：

| Aspect | Required behavior |
|---|---|
| `req.Messages` | 保留 user/assistant/tool 的轮次顺序，并映射到提供方的线上格式 |
| `req.SystemPrompt` | 提升到提供方放置 system-prompt 的位置 |
| `req.MaxOutputTokens > 0` | 透传到提供方的 max-tokens 字段 |
| `req.Temperature != nil` | 在提供方支持时透传 |
| `Response.Provider` | 规范的提供方名称，例如 `openai`、`anthropic`、`ollama` |
| `Response.Model` | 在构造时绑定的模型 |
| `Response.Usage.Source` | 当存在 token 计数时为 `llm.UsageReported`；否则为 `llm.UsageUnknown` |
| `Response.FinishReason` | 归一化到现有的 finish-reason 常量 |
| Errors | 包装进第 5 节的类型化分类法，同时把 SDK 错误保留在 `Wrapped` 中 |

来自三个 Phase 1 adapter 的提供方专属注意事项：

- OpenAI 和 Ollama 从 `req.SystemPrompt` 派生一条 system-role 消息。
- Anthropic 把 `req.SystemPrompt` 提升到顶层的 `system` 字段。
- Phase 1 adapter 不尊重工具；`Capabilities.Tools` 保持 `false`。
- 如果提供方对某个响应不报告 token 用量，Phase 1 中不要猜测。返回 `Usage.Source = llm.UsageUnknown`。

## 4. Constructor Pattern

Phase 1 使用构造时模型绑定 + 函数式选项。

必需的构造器形状：

```go
func New(opts ...Option) (*X, error)
```

必需的选项模式：

```go
type Option func(*config)
```

规范的期望：

| Option | Purpose | Notes |
|---|---|---|
| `WithModel(string)` | 绑定模型 | 必需；`New` 在为空时必须失败 |
| `WithAPIKey(string)` | 覆盖环境变量回退 | 无 key 的提供方不需要 |
| `WithBaseURL(string)` | 指向自定义端点 | 用于测试、代理、本地 host |
| `WithHTTPClient(*http.Client)` | 注入自定义传输 | 重试包装器、追踪、模拟 |
| `WithTimeout(time.Duration)` | 默认请求超时 | 与每次调用的 `ctx` 区分 |

规范草图：

```go
type Option func(*config)

func WithModel(m string) Option   { return func(c *config) { c.model = m } }
func WithAPIKey(k string) Option  { return func(c *config) { c.apiKey = k } }
func WithBaseURL(u string) Option { return func(c *config) { c.baseURL = u } }

func New(opts ...Option) (*OpenAI, error) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.model == "" {
		return nil, errors.New("openai: WithModel is required")
	}
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAI{/* ... */}, nil
}
```

这与兄弟仓中 provider adapter 使用的构造模式相同。当传输面需要时，诸如 `WithRegion(...)` 的提供方专属选项可以扩展该基础模式。

## 5. Error Taxonomy

每个 adapter 都应把传输或 SDK 失败包装进 `llm/errors.go` 中的某个类型化错误，并把原始 SDK 错误保留在 `Wrapped` 中，从而调用方仍可对提供方专属的错误类型使用 `errors.As`。

推荐的 HTTP-status -> typed-error 映射表：

| HTTP / Cause | Typed error |
|---|---|
| 401, 403 | `*llm.AuthError` |
| 429 | `*llm.RateLimitError` |
| 其他 4xx（400、404、422 等） | `*llm.InvalidRequestError` |
| 5xx | `*llm.TransientError` |
| 网络 I/O、EOF、连接重置 | `*llm.TransientError` |
| `context.DeadlineExceeded` | `*llm.TransientError` |
| `context.Canceled` | 原样返回；不要包装 |

额外指引：

- 在每个类型化错误上填充 `Provider`。
- 当提供方暴露时，在 `RateLimitError` 上填充 `RetryAfter`。
- 当提供方给出一个稳定的、机器可读的配额判别符时，在 `RateLimitError` 上填充 `Reason`。
- 在 `Wrapped` 中保留提供方细节；不要把它字符串化后丢弃。

检测示例：

```go
var authErr *llm.AuthError
if errors.As(err, &authErr) {
	// credentials or permission failure
}
```

## 6. Conformance Test Pattern

Phase 1 的 provider adapter 应针对兄弟仓中的共享一致性测试套件来自我验证：

- `github.com/costa92/llm-agent-providers/internal/contract`

模式是：

1. 为顺利路径和错误情形创建提供方专属的 fixture JSON 文件。
2. 添加一个返回你的 `llm.ChatModel` 的 adapter factory。
3. 复用 `LoadFixture(...)` 和 `AssertGenerate(...)`。
4. 把提供方映射逻辑保持在小巧的 `map.go` 或 `errors.go` 辅助文件里，从而 fixture 矩阵保持可读。
5. 如果提供方实现了 `llm.Embedder`，就用嵌入 fixture 扩展同一个共享套件，而非创建第二个测试框架。

对于诸如 Ollama 的仅本地提供方，Phase 1 也允许一个带构建标签的实时测试。每夜工作流在 PR CI 之外运行它，从而真实容器漂移不会阻塞正常开发。

## 7. Embedder Guidance

Phase 4 为支持嵌入的提供方加入 `llm.Embedder` 契约。

必需行为是：

- `Embed(ctx, []string)` 在返回的向量中保留输入顺序。
- `EmbedDimensions()` 在已知时报告所绑定模型的向量宽度。
- `llm.Usage` 通过与 chat 相同的共享 usage 类型填充。
- `Info().Capabilities.Embeddings` 对所绑定模型如实反映。

对于**不**支持嵌入的提供方：

- 不要伪造一个降级实现
- 返回 `Capabilities.Embeddings=false`
- 当某个包装器或更高层辅助选择浮现一个嵌入调用路径时，通过 `llm.ErrCapabilityNotSupported` 语义暴露这一缺口

v0.3 中的 Anthropic 是规范的「有文档记录的缺口」示例。

## 8. Phase 1 Boundary

本指南刻意保持狭窄。一个提供方只能为 Generate-only 行为声称 v0.1 一致性。

尚未完成：

- 没有流式契约指引
- 没有 `StreamEvent` 验证规则
- 没有原生 `ToolCaller` 指引
- 没有结构化输出指引
- 没有估算的 token 计量
- 没有重试状态机
- 没有 OTel 包装器或 semconv 规则

不要在 Phase 1 adapter 中预先发明这些行为。先匹配当前契约，然后在定义该行为的那个 milestone phase 中扩展。

## 9. Cross-References

核心仓：

- `llm/chatmodel.go`
- `llm/types.go`
- `llm/info.go`
- `llm/errors.go`
- `llm/scripted.go`
- `llm/chat_only_mock.go`

规范的兄弟仓示例：

- `github.com/costa92/llm-agent-providers/openai`
- `github.com/costa92/llm-agent-providers/anthropic`
- `github.com/costa92/llm-agent-providers/ollama`
- `github.com/costa92/llm-agent-providers/deepseek`
- `github.com/costa92/llm-agent-providers/minimax`
- `github.com/costa92/llm-agent-providers/internal/contract`

版本说明：

- 本指南的 v0.1 对应 v0.3 路线图的 Phase 1。
- v0.2 在 Phase 2 落地后加入了流式指引。
- v0.3 在 Phase 3 和 4 落地后加入工具和嵌入指引。
