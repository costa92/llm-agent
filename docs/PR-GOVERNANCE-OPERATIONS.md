# PR Governance 03: 落地与运维

## 为什么用 `pull_request_target`

治理 workflow 的目标是：

- 读 PR 元数据
- 读 review 状态
- 请求 reviewer
- 开启 auto-merge
- 在 merge 确认后删除同仓库 head branch

它不需要 checkout PR 分支，也不应该执行 PR 代码。

因此 `pull_request_target` 更适合这个场景，因为：

- 它运行的是 default branch 上的 workflow 定义
- 不需要信任 PR 分支内容
- 可以安全拿到写权限去做 review routing、merge orchestration 和 branch cleanup

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
  -> governance workflow 进入默认分支

第 4 步:
  把默认分支的 required status checks 改成:
    - go
    - governance

最终状态:
  owner PR 自动合并
  non-owner PR 必须经 costa92 审核
  owner 同仓库分支在 merge 后被 workflow 显式删除
```

## 仓库落地矩阵

### 角色矩阵

| Repo | 角色 | 与其他项目关系 | 当前治理状态 |
|---|---|---|---|
| `llm-agent` | core framework | 上游核心契约源 | 已切到 `go + governance` |
| `llm-agent-rag` | standalone RAG SDK | 独立版本的 RAG 固定点 | 已切到 `go + governance` |
| `llm-agent-flow` | flow IR / executor | 依赖 `llm-agent` 的执行层 | 已切到 `go + governance` |
| `llm-agent-providers` | provider layer | 消费 `llm-agent` API | 已切到 `go + governance` |
| `llm-agent-otel` | observability layer | 消费 `llm-agent`、`rag`、`flow` API | 已切到 `go + governance` |
| `llm-agent-customer-support` | reference application | 组合整个生态 | 已切到 `go + governance` |

### 实际落地矩阵

| Repo | Default branch | Workflow | Required checks | Auto-merge | `deleteBranchOnMerge` | 目标行为 |
|---|---|---|---|---|---|---|
| `llm-agent` | `main` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |
| `llm-agent-rag` | `master` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |
| `llm-agent-flow` | `main` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |
| `llm-agent-providers` | `main` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |
| `llm-agent-otel` | `main` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |
| `llm-agent-customer-support` | `main` | `.github/workflows/pr-governance.yml` | `go`, `governance` | 开启 | 开启 | owner PR 自动合并并删分支，external PR 需 owner 审核 |

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

### Job 2: `auto-merge-owner`

触发条件：

- `governance` 已通过
- PR 作者是 `costa92`
- PR 不是 draft

执行动作：

1. 查询 `autoMergeRequest`
2. 如未开启，则调用 `gh pr merge --auto --merge`
3. 轮询 PR 是否已经真正进入 `MERGED`
4. 如已 merged，则删除同仓库 head branch

最终稳定版本刻意没有把主删除路径写成：

- `gh pr merge --auto --merge --delete-branch`
- 单独的 `delete-merged-branch.yml`

因为这两种路径在这次实际治理链路里都不够稳定，尤其是 owner PR 由 `github.token` 驱动 auto-merge 时，下游 cleanup workflow 不是可靠触发源。

### `auto-merge-owner` 幂等性

owner PR 的治理 workflow 不只会在 `opened` / `synchronize` 时触发，也会在 review 事件上再次运行。

因此 `auto-merge-owner` 不能假设自己只会执行一次。最终稳定版本必须满足：

- `auto-merge-owner` job 单独声明：
  - `contents: write`
  - `pull-requests: write`
- 在调用 `gh pr merge --auto --merge` 之前，先查询 `autoMergeRequest`
- 如果 auto-merge 已经开启，则不再重复发起 enable 请求

否则会出现一个非常隐蔽的假失败：

- 第一次运行已经成功开启 auto-merge
- 后续事件再次触发同一 job
- workflow 重新执行 `gh pr merge --auto`
- GitHub CLI 返回非零，导致必需检查链路变红

这不是权限错误，而是**幂等性错误**。

## 为什么不依赖独立 cleanup workflow

这次实际排障里，先后测试过两类独立 cleanup 方案：

- `pull_request_target` 上的 `closed` 事件
- default branch `push` 之后再删 merged branch

它们的问题不是语法错误，而是**触发链路不可靠**：

- owner PR 是由 `pr-governance.yml` 使用 `github.token` 开启 auto-merge
- merge 发生后，下游 cleanup workflow 并不是稳定的最终触发源
- 同样的 repo 设置在一次验证里可见，在另一次 auto-merge 链路里又可能不触发到预期步骤

因此最终方案收敛为：

- 仓库设置里保留 `deleteBranchOnMerge = true` 作为安全网
- 真正依赖的删除逻辑内嵌在 `pr-governance.yml`
- 也就是“谁发起 owner auto-merge，谁负责在 merge 确认后删同仓库分支”

## bootstrap 限制

`pull_request_target` 的一个关键限制是：

- PR 运行时使用的是 **default branch** 上的 workflow 定义
- 不是 PR head 分支里的 workflow 定义

这意味着：

- 你不能依赖“这个 PR 自己带的新 workflow”来修复“这个 PR 自己当前正在跑的旧 workflow”
- 修治理 workflow 的 PR 在落到默认分支之前，仍然会继续执行默认分支上的旧版本

这次真实落地里就出现了这个 bootstrap 场景：

1. `llm-agent-otel` 和 `llm-agent-customer-support` 的 owner PR 已经打开
2. PR 分支里推入了修复后的 workflow
3. 但 `pull_request_target` 仍执行默认分支上的旧 workflow
4. 旧 workflow 只有 `contents: read`，日志继续报：
   `GraphQL: Resource not accessible by integration (enablePullRequestAutoMerge)`
5. 只能先手动为当下 owner PR 开启或完成合并
6. 等修复进入默认分支后，后续 owner PR 才会自动受益

运维上要接受这个事实：**bootstrap PR 可能需要一次人工托底**。

## 失败模式

### 1. `go` 绿了，但 `governance` 红了

通常说明：

- PR 不是 `costa92` 提交的
- 但 `costa92` 还没审批当前 head

这不是故障，而是预期行为。

### 2. `governance` 没出现

通常说明：

- workflow 还没在默认分支上落地
- branch protection 已经要求 `governance`
- 或 workflow 文件名 / 触发条件被改坏了

这是最需要优先检查的一类问题，因为它会直接把 PR 卡在 `Expected` 状态。

### 3. owner PR 没有自动合并

排查顺序：

1. `allow_auto_merge` 是否仍为 `true`
2. PR 是否是 draft
3. `governance` 是否通过
4. `go` 是否通过
5. workflow 是否成功执行了 `gh pr merge --auto --merge`
6. workflow permissions 是否同时包含 `contents: write` 和 `pull-requests: write`
7. 日志里是否出现 `enablePullRequestAutoMerge` 权限错误

### 4. owner PR 合并了，但分支没删掉

排查顺序：

1. head branch 是否来自同一个仓库，而不是 fork
2. head branch 是否误等于默认分支
3. `auto-merge-owner` 是否已经等到 `state == MERGED && mergedAt != ""`
4. 日志里 `gh api -X DELETE` 是否真正执行
5. 仓库级 `deleteBranchOnMerge` 是否仍保持开启

如果第 3 步之前 workflow 就结束，说明这次 merge 完成时间超出了轮询窗口；当前推荐仍是保留 repo 级自动删分支作为安全网。

### 5. non-owner PR 已经审过，但 `governance` 仍然失败

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
- 在 merge 确认后删除同仓库 head branch

`pull_request_target` 自带更高权限，所以这条边界必须保持清晰。

## 运维检查清单

1. 仓库级 `allow_auto_merge` 仍为 `true`
2. 仓库级 `deleteBranchOnMerge` 仍为 `true`
3. 默认分支的 required status checks 仍为 `go` 和 `governance`
4. `.github/workflows/pr-governance.yml` 仍在默认分支
5. owner PR 能在 `go` 通过后自动进入 merge
6. owner PR 在 merge 完成后会删除同仓库分支
7. non-owner PR 会自动 request review 给 `costa92`
8. non-owner PR 在未审批 current head 时，`governance` 保持失败
9. non-owner PR 在审批 current head 后，`governance` 变绿

## 真实变更记录

### 2026-05-13 初次治理落地

以下 3 个 sister repo PR 已在 2026-05-13 合并：

| Repo | PR | Merged at (UTC) | Merge commit |
|---|---|---|---|
| `llm-agent-providers` | `#1` | `2026-05-13T04:14:10Z` | `f24c5d665b07ad0c003d517b31c3bf715c99b738` |
| `llm-agent-otel` | `#1` | `2026-05-13T04:14:10Z` | `b64f082d3e1bd3db596c0ab76c8cea89cd99f2cd` |
| `llm-agent-customer-support` | `#2` | `2026-05-13T04:14:10Z` | `03385b77fccad5db1c1e8e2063d8e0ee6a62f1cd` |

### 2026-05-13 bootstrap 修复记录

治理规则初次落地后，又补了一个 owner auto-merge 幂等性修复，避免 review 事件重跑时把 required check 误打红。最终进入各仓库默认分支的 PR 是：

| Repo | PR | Merged at (UTC) | 作用 |
|---|---|---|---|
| `llm-agent-providers` | `#5` | `2026-05-13T08:39:54Z` | owner auto-merge 幂等化 + job 级写权限 |
| `llm-agent-otel` | `#3` | `2026-05-13T08:29:35Z` | bootstrap owner PR 手工托底后进入主线 |
| `llm-agent-customer-support` | `#4` | `2026-05-13T08:29:35Z` | bootstrap owner PR 手工托底后进入主线 |

### 2026-05-22 分支删除路径定稿

在这一天，针对“owner PR 自动合并后没有稳定删分支”的问题，先后验证了多个方案，并最终收敛到把删除逻辑内嵌回 `pr-governance.yml`。

关键事实：

- 单独 cleanup workflow 方案经过测试，但不再作为主方案依赖
- 最终验证样本是 `llm-agent-providers` PR `#15`
- merged 时间为 `2026-05-22T02:13:35Z`
- 分支名为 `test/final-pr-governance-delete-branch`
- merge 后再次查询 GitHub API，返回 `404 Branch not found`

这说明最终链路已经验证为：

1. `pr-governance.yml` 开启 owner auto-merge
2. 同一个 workflow 等待 merged 状态可见
3. 同一个 workflow 删除同仓库 head branch

### 2026-05-22 全仓推广

这次最终版本随后补齐到另外 3 个仓库：

| Repo | Commit | 说明 |
|---|---|---|
| `llm-agent` | `90e264d` | 新增统一 `pr-governance.yml` |
| `llm-agent-rag` | `23b2f14` | 新增统一 `pr-governance.yml` |
| `llm-agent-flow` | `1ef1feb` | 新增统一 `pr-governance.yml` |

### 最终 protection 快照

| Repo | `allow_auto_merge` | `deleteBranchOnMerge` | Required checks | Required review gate |
|---|---|---|---|---|
| `llm-agent` | `true` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-rag` | `true` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-flow` | `true` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-providers` | `true` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-otel` | `true` | `true` | `go`, `governance` | 已移除 |
| `llm-agent-customer-support` | `true` | `true` | `go`, `governance` | 已移除 |

## 回滚思路

最小回滚路径：

1. 先把 branch protection 的 required checks 从 `go + governance` 改回旧规则
2. 再决定是否恢复 GitHub 内建 required review
3. 最后再移除或停用 `pr-governance.yml`

不要反过来做。因为如果先删 workflow，再保留 `governance` 为 required check，就会把 PR 卡在永久 `Expected` 状态。
