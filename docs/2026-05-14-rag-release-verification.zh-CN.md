[English](./2026-05-14-rag-release-verification.md) | [简体中文](./2026-05-14-rag-release-verification.zh-CN.md)

# RAG Release Verification

Date: 2026-05-14
Project: `github.com/costa92/llm-agent`
Related module: `github.com/costa92/llm-agent-rag`
Related tag: `v0.1.0`

## Scope

本文档记录在以下操作之后进行的验证：

- 发布 `github.com/costa92/llm-agent-rag`
- 打 tag `v0.1.0`
- 把 `github.com/costa92/llm-agent` 从本地 `replace` 切换为真实的远程 module 依赖
- 移除 vendored 的 `third_party/llm-agent-rag` 副本

## Release State

已验证状态：

- 独立仓库已推送：
  - `git@github.com:costa92/llm-agent-rag.git`
- 独立 tag 已推送：
  - `v0.1.0`
- 主仓库依赖已切换为：
  - `require github.com/costa92/llm-agent-rag v0.1.0`

## Verification Performed

### 1. Main repo full test pass

命令：

```bash
cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off GOCACHE=/tmp/go-build go test ./...
```

结果：

- PASS

覆盖的包包括：

- `rag`
- `memory`
- `context`
- `bench`
- `comm`
- `orchestrate`
- `rl`

### 2. Standalone SDK full test pass

命令：

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./...
```

结果：

- PASS

覆盖的包包括：

- `advanced`
- `embed`
- `examples`
- `ingest`
- `prompt`
- `rag`
- `store`

### 3. External consumer smoke test for `llm-agent-rag`

在以下位置创建了一个全新的临时 module：

- `/tmp/rag-smoke`

最小程序行为：

- 导入 `github.com/costa92/llm-agent-rag`
- 构造 `rag.System`
- 导入一个文档
- 检索一个命中
- 打印检索到的文档 ID

依赖解析：

- 解析到远程 module：
  - `github.com/costa92/llm-agent-rag v0.1.0`

命令：

```bash
cd /tmp/rag-smoke
GOCACHE=/tmp/go-build go mod tidy
GOWORK=off GOCACHE=/tmp/go-build go run .
```

观察到的输出：

```text
doc1
```

结果：

- PASS

### 4. External consumer smoke test for `llm-agent`

在以下位置创建了一个全新的临时 module：

- `/tmp/agent-smoke`

最小程序行为：

- 导入 `github.com/costa92/llm-agent`
- 构建一个脚本化 LLM
- 构建一个 `FunctionCallAgent`
- 调用 calculator builtin
- 打印答案

依赖解析：

- 解析到远程 module：
  - `github.com/costa92/llm-agent v0.4.0`

命令：

```bash
cd /tmp/agent-smoke
GOCACHE=/tmp/go-build go mod tidy
GOWORK=off GOCACHE=/tmp/go-build go run .
```

观察到的输出：

```text
calculator: 96
```

结果：

- PASS

## Notes

### `go.work` interference

第一次运行临时 smoke module 的尝试失败了，因为本地工作区的 `go.work` 干扰了独立 module 的执行。

这通过以下方式运行得以解决：

```bash
GOWORK=off
```

这是预期的，并不表示 module 或发布缺陷。

### Adapter test boundary

`llm-agent-rag` 中的 `adapter/llmagent` 路径仍然刻意位于默认的独立验证路径之外，因为它需要一个对 `github.com/costa92/llm-agent` 的临时本地依赖。

这是核心 SDK 一个刻意为之的可发布性边界。

## Conclusion

发布状态已验证：

- `llm-agent-rag v0.1.0` 可作为一个独立 module 被获取和运行
- `llm-agent` 在切换到远程 SDK 依赖后正确工作
- 移除 `third_party/llm-agent-rag` 没有引入回归
