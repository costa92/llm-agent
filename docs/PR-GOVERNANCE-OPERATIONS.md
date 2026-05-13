# PR Governance 03: 落地与运维

## 为什么用 `pull_request_target`

治理 workflow 的目标是：

- 读 PR 元数据
- 读 review 状态
- 请求 reviewer
- 开启 auto-merge

它不需要 checkout PR 分支，也不应该执行 PR 代码。

因此 `pull_request_target` 更适合这个场景，因为：

- 它运行的是 base branch 上的 workflow 定义
- 不需要信任 PR 分支内容
- 可以安全拿到写权限去做 review routing 和 merge orchestration

要让 owner auto-merge 真正生效，workflow permissions 至少要包含：

- `contents: write`
- `pull-requests: write`

如果缺少 `contents: write`，`gh pr merge --auto` 会在 GitHub Actions 日志里报：

- `GraphQL: Resource not accessible by integration (enablePullRequestAutoMerge)`

## 迁移时序

这次不是在一个全新的仓库里启用规则，而是在已有 PR 已经打开、已经被旧规则卡住的状态下迁移，所以必须按顺序切换。

```text
旧状态:
  branch protection = required review
  现有 owner PR = 卡在 REVIEW_REQUIRED

第 1 步:
  把 pr-governance workflow 推到现有 PR 分支

第 2 步:
  临时移除 required_pull_request_reviews

第 3 步:
  合并现有 owner PR
  -> governance workflow 进入 main

第 4 步:
  把 main 的 required status checks 改成:
    - go
    - governance

最终状态:
  owner PR 自动合并
  non-owner PR 必须经 costa92 审核
```

## 仓库落地矩阵

### 角色矩阵

| Repo | 角色 | 与其他项目关系 | 当前治理状态 |
|---|---|---|---|
| `llm-agent` | core framework | 上游核心契约源 | 本文档组作为规则说明与后续扩展基线 |
| `llm-agent-providers` | provider layer | 消费 `llm-agent` API | 已切到 `go + governance` |
| `llm-agent-otel` | observability layer | 消费 `llm-agent` API | 已切到 `go + governance` |
| `llm-agent-customer-support` | reference application | 组合前 3 个项目 | 已切到 `go + governance` |

### 实际落地矩阵

| Repo | Workflow | Required checks | Auto-merge | 目标行为 |
|---|---|---|---|---|
| `llm-agent-providers` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | owner PR 自动合并，external PR 需 owner 审核 |
| `llm-agent-otel` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | owner PR 自动合并，external PR 需 owner 审核 |
| `llm-agent-customer-support` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | owner PR 自动合并，external PR 需 owner 审核 |

## Workflow 结构

### Job 1: `governance`

输入：

- PR author
- PR current head SHA
- `costa92` 的 review 状态

输出：

- PASS: owner PR，或 non-owner PR 已被 `costa92` 对当前 head 审批
- FAIL: non-owner PR 尚未满足 owner 审批条件

它还承担 reviewer routing：

- 如果作者不是 `costa92`
- 自动 request review 给 `costa92`

### Job 2: owner auto-merge

触发条件：

- `governance` 已通过
- PR 作者是 `costa92`
- PR 不是 draft

执行动作：

- 调用 `gh pr merge --auto --merge --delete-branch`
- 失败必须直接让 job 变红，不能用 `|| true` 吞掉

## 失败模式

### 1. `go` 绿了，但 `governance` 红了

通常说明：

- PR 不是 `costa92` 提交的
- 但 `costa92` 还没审批当前 head

这不是故障，而是预期行为。

### 2. `governance` 没出现

通常说明：

- workflow 还没在 `main` 上落地
- branch protection 已经要求 `governance`
- 或 workflow 文件名 / 触发条件被改坏了

这是最需要优先检查的一类问题，因为它会直接把 PR 卡在 `Expected` 状态。

### 3. owner PR 没有自动合并

排查顺序：

1. `allow_auto_merge` 是否仍为 `true`
2. PR 是否是 draft
3. `governance` 是否通过
4. `go` 是否通过
5. workflow 是否成功执行了 `gh pr merge --auto`
6. workflow permissions 是否同时包含 `contents: write` 和 `pull-requests: write`
7. 日志里是否出现 `enablePullRequestAutoMerge` 权限错误

### 4. non-owner PR 已经审过，但 `governance` 仍然失败

最常见原因：

- 审批针对的是旧 commit
- 审批之后作者又 push 了新 commit

这时不是系统错判，而是 current-head 校验在起作用。重新对最新 head 审批一次即可。

## 安全边界

核心原则：

- `go` 检查负责执行代码
- `governance` 检查负责治理判断

`governance` 不应该：

- checkout PR 代码
- 执行来自 PR 分支的脚本
- 读取 PR 提交里新增的任意自动化逻辑

它应该只做：

- 读 PR metadata
- 读 review metadata
- 请求 reviewer
- 开启 auto-merge

`pull_request_target` 自带更高权限，所以这条边界必须保持清晰。

## 运维检查清单

1. 仓库级 `allow_auto_merge` 仍为 `true`
2. `main` 的 required status checks 仍为 `go` 和 `governance`
3. `.github/workflows/pr-governance.yml` 仍在默认分支
4. owner PR 能在 `go` 通过后自动进入 merge
5. non-owner PR 会自动 request review 给 `costa92`
6. non-owner PR 在未审批 current head 时，`governance` 保持失败
7. non-owner PR 在审批 current head 后，`governance` 变绿

## 真实变更记录

### 合并记录

以下 3 个 sister repo PR 已在 2026-05-13 合并：

| Repo | PR | Merged at (UTC) | Merge commit |
|---|---|---|---|
| `llm-agent-providers` | `#1` | `2026-05-13T04:14:10Z` | `f24c5d665b07ad0c003d517b31c3bf715c99b738` |
| `llm-agent-otel` | `#1` | `2026-05-13T04:14:10Z` | `b64f082d3e1bd3db596c0ab76c8cea89cd99f2cd` |
| `llm-agent-customer-support` | `#2` | `2026-05-13T04:14:10Z` | `03385b77fccad5db1c1e8e2063d8e0ee6a62f1cd` |

### 最终 protection 快照

| Repo | `allow_auto_merge` | Required checks | Required review gate |
|---|---|---|---|
| `llm-agent-providers` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-otel` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-customer-support` | `true` | `go`, `governance` | 已移除 |

## 回滚思路

最小回滚路径：

1. 先把 branch protection 的 required checks 从 `go + governance` 改回旧规则
2. 再决定是否恢复 GitHub 内建 required review
3. 最后再移除或停用 `pr-governance.yml`

不要反过来做。因为如果先删 workflow，再保留 `governance` 为 required check，就会把 PR 卡在永久 `Expected` 状态。
