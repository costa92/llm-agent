[English](./2026-05-14-rag-release-verification.md) | [简体中文](./2026-05-14-rag-release-verification.zh-CN.md)

# RAG Release Verification

Date: 2026-05-14
Project: `github.com/costa92/llm-agent`
Related module: `github.com/costa92/llm-agent-rag`
Related tag: `v0.1.0`

## Scope

This document records the verification performed after:

- publishing `github.com/costa92/llm-agent-rag`
- tagging `v0.1.0`
- switching `github.com/costa92/llm-agent` from a local `replace` to the
  real remote module dependency
- removing the vendored `third_party/llm-agent-rag` copy

## Release State

Verified state:

- standalone repo pushed:
  - `git@github.com:costa92/llm-agent-rag.git`
- standalone tag pushed:
  - `v0.1.0`
- main repo dependency switched to:
  - `require github.com/costa92/llm-agent-rag v0.1.0`

## Verification Performed

### 1. Main repo full test pass

Command:

```bash
cd /home/hellotalk/code/go/src/github.com/costa92/llm-agent
GOWORK=off GOCACHE=/tmp/go-build go test ./...
```

Result:

- PASS

Covered packages include:

- `rag`
- `memory`
- `context`
- `bench`
- `comm`
- `orchestrate`
- `rl`

### 2. Standalone SDK full test pass

Command:

```bash
cd /tmp/llm-agent-rag
GOWORK=off GOCACHE=/tmp/go-build go test ./...
```

Result:

- PASS

Covered packages include:

- `advanced`
- `embed`
- `examples`
- `ingest`
- `prompt`
- `rag`
- `store`

### 3. External consumer smoke test for `llm-agent-rag`

A fresh temporary module was created under:

- `/tmp/rag-smoke`

Minimal program behavior:

- imports `github.com/costa92/llm-agent-rag`
- constructs `rag.System`
- imports one document
- retrieves one hit
- prints the retrieved document ID

Dependency resolution:

- resolved remote module:
  - `github.com/costa92/llm-agent-rag v0.1.0`

Commands:

```bash
cd /tmp/rag-smoke
GOCACHE=/tmp/go-build go mod tidy
GOWORK=off GOCACHE=/tmp/go-build go run .
```

Observed output:

```text
doc1
```

Result:

- PASS

### 4. External consumer smoke test for `llm-agent`

A fresh temporary module was created under:

- `/tmp/agent-smoke`

Minimal program behavior:

- imports `github.com/costa92/llm-agent`
- builds a scripted LLM
- builds a `FunctionCallAgent`
- invokes the calculator builtin
- prints the answer

Dependency resolution:

- resolved remote module:
  - `github.com/costa92/llm-agent v0.4.0`

Commands:

```bash
cd /tmp/agent-smoke
GOCACHE=/tmp/go-build go mod tidy
GOWORK=off GOCACHE=/tmp/go-build go run .
```

Observed output:

```text
calculator: 96
```

Result:

- PASS

## Notes

### `go.work` interference

The first attempt to run the temporary smoke modules failed because the local
workspace `go.work` interfered with standalone module execution.

This was resolved by running with:

```bash
GOWORK=off
```

This is expected and does not indicate a module or release defect.

### Adapter test boundary

The `adapter/llmagent` path in `llm-agent-rag` remains intentionally outside
the default standalone verification path because it requires a temporary local
dependency on `github.com/costa92/llm-agent`.

This is an intentional publishability boundary for the core SDK.

## Conclusion

The release state is verified:

- `llm-agent-rag v0.1.0` is fetchable and runnable as a standalone module
- `llm-agent` works correctly after switching to the remote SDK dependency
- removing `third_party/llm-agent-rag` did not introduce regressions
