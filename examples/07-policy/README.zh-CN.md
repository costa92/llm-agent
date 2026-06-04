[English](./README.md) | [简体中文](./README.zh-CN.md)

# Demo 07: Policy / safety middleware

演示来自 `policy` 包（Phase 36，CC-2）的三个内置门控：PII 脱敏、提示词注入扫描，以及最大输入长度强制。

## Run

```sh
cd examples
go run ./07-policy
```

确定性；无网络、无提供方 —— 按 `CLAUDE.md` 使用规范的 `scriptedllm` 模拟。

## What the demos show

| Demo | Gate | Action | What you should see |
|------|------|--------|---------------------|
| demoPIIRedaction   | `policy.NewPIIRedactor()`      | `Replace` (pre-call) | LLM 收到的是你提示词的脱敏版本；响应仍正常产生 |
| demoInjectionBlock | `policy.NewInjectionScanner()` | `Block` (pre-call)   | `errors.Is(err, policy.ErrBlocked) == true`；LLM 永不被触达（计数器 == 0） |
| demoMaxInputLen    | `policy.NewMaxInputLen(4096)`  | `Block` (pre-call)   | 超大输入在任何网络往返之前即被拒绝 |

## Canonical setup

```go
wrapped := policy.Wrap(model,
    policy.NewPIIRedactor(),
    policy.NewInjectionScanner(),
    policy.NewMaxInputLen(4096),
)
agent := agents.NewSimpleAgent(wrapped, agents.SimpleOptions{Name: "my-agent"})
```

三个门控按顺序注册；第一个非 `Allow` 的决策胜出；`Block` 会短路；`Replace` 会改写请求，后续门控看到的是改写后的版本。

## Composition with OpenTelemetry observability

在生产中你通常会把 `policy` 叠在来自兄弟仓 `llm-agent-otel` 的 `otelmodel.Wrap` 之上：

```go
// requires github.com/costa92/llm-agent-otel/otelmodel
// (lives in a sister repo; NOT imported by this example)
wrapped := policy.Wrap(otelmodel.Wrap(provider),
    policy.NewPIIRedactor(),
    policy.NewInjectionScanner(),
    policy.NewMaxInputLen(4096),
)
```

最外层 `policy` 在被观测前拒绝；中间层 `otelmodel` 进行观测；最内层 provider 发起调用。`policy/integration_test.go` 中的组合测试（`.planning/phases/36-policy-safety-middleware/36-RESEARCH.md` 的决策 G）验证了这一栈在**不**把 `otelmodel` 导入核心的情况下也能工作 —— 当 `llm-agent-otel` 在 v1.3 中版本提升以匹配核心 v0.6.x 时，会有一个兄弟仓示例演示真实世界的栈。

## Audit log

`policy.WrapConfig(model, policy.Config{Gates: ..., OnDecision: f})` 接受一个可选的 `OnDecision func(policy.Decision)` 回调，它对每一个非 `Allow` 的决策同步触发。用它来接入你的日志器（`slog`、`zerolog` 等）。

```go
cfg := policy.Config{
    Gates: []policy.Gate{policy.NewPIIRedactor(), policy.NewInjectionScanner()},
    OnDecision: func(d policy.Decision) {
        slog.Info("policy decision", "gate", d.Gate, "reason", d.Reason, "action", d.Action)
    },
}
wrapped := policy.WrapConfig(model, cfg)
```

完整的接口面以及已批准的 Q1-Q5 决策见 `policy/doc.go`。
