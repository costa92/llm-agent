# PR Governance 02: 治理规则

## 设计目标

目标只有两条：

1. 如果 PR 是 `costa92` 自己提交的，在 CI 通过后应自动合并。
2. 如果 PR 是别人提交的，必须经过 `costa92` 审核后才能合并。

硬约束：

- 不引入第二个 GitHub 账号或额外 bot 账号
- 不要求维护单独的审批 token
- 不执行 PR 分支里的不可信代码来做治理判断
- 方案要适配这 4 个项目的协作关系，并首先稳定落到 3 个下游 sister repos

## 为什么不用 GitHub 内建 required review

GitHub 的 branch protection 对 required approval 的判断不区分 author policy。它只知道“要不要 approval”，不知道“谁提交的 PR”。

因此：

- owner 自己的 PR 也会被卡在 `REVIEW_REQUIRED`
- 规则无法表达“owner PR 放行、external PR 仍需审核”

结论是：GitHub 内建 required review 不能直接表达这次想要的规则，必须把“作者是谁”这层逻辑移到自定义检查里。

## 最终规则

最终采用的是“自定义治理检查 + auto-merge”的两段式方案。

### 决策树

```text
PR opened
  -> run go
  -> run governance

governance:
  if author == costa92:
    PASS
    -> enable auto-merge
  else:
    request review from costa92
    if costa92 approved current head:
      PASS
    else:
      FAIL
```

## 规则拆解

### 1. `go`

每个仓库继续保留原有 `go` 检查：

- `go mod tidy` drift check
- `go vet`
- `go build`
- `go test`

这个检查负责代码正确性，不负责人审策略。

### 2. `governance`

在 3 个 sister repos 新增：

- `.github/workflows/pr-governance.yml`

这个 workflow 的职责不是跑代码，而是判断“这个 PR 在治理层面是否允许合并”。

它监听：

- `pull_request_target`
- `pull_request_review`

核心规则：

- 如果 PR 作者是 `costa92`，`governance` 直接通过
- 如果 PR 作者不是 `costa92`，workflow 会自动 request review 给 `costa92`
- 对 non-owner PR，只有当 `costa92` 对当前 head commit 给出 `APPROVED`，`governance` 才通过

### 3. owner auto-merge

同一个 workflow 中还有第二个 job：

- 如果作者是 `costa92`
- `governance` 已通过
- 则调用 `gh pr merge --auto --merge --delete-branch`

这里有一个容易忽略的约束：

- workflow permissions 必须同时包含 `contents: write` 和 `pull-requests: write`
- 如果只有 `pull-requests: write`，`gh pr merge --auto` 会返回 `Resource not accessible by integration`
- 这个步骤不能用 `|| true` 吞掉失败，否则检查会显示成功，但 PR 不会真的进入 auto-merge

这样 owner PR 在 `go` 和 `governance` 都绿时，会自动进入合并流程。

### 4. merge gate

最终 `main` 分支不再要求 GitHub 内建 approval，而是要求两个检查：

- `go`
- `governance`

也就是说，merge gate 从：

- “必须有 1 个 GitHub approving review”

切成了：

- “必须通过代码检查”
- “必须通过治理检查”

## 如何使用

### 场景 1：在 `llm-agent` 做核心能力变更

典型动作：

1. 在 `llm-agent` 修改 API 或实现
2. 发布一个新 tag 或稳定提交
3. 在下游仓库更新依赖
4. 分别发起 PR

这套规则的作用是：

- 你自己开的下游依赖升级 PR，不会再被 required review 卡死
- 下游仓库仍然保留正常 CI 和治理门槛

### 场景 2：在 `llm-agent-providers`、`llm-agent-otel` 或 `llm-agent-customer-support` 做日常维护

如果 PR 作者是 `costa92`：

- 正常提交 PR
- 等 `go` 和 `governance`
- `governance` 会自动通过
- 系统会自动开启 auto-merge

如果 PR 作者不是 `costa92`：

- 正常提交 PR
- 系统会自动 request review 给 `costa92`
- 只有当 `costa92` 审批当前 head 后，`governance` 才通过

### 场景 3：跨仓库联动修改

例如一次核心 API 调整，可能同时触发：

1. `llm-agent` 改接口
2. `llm-agent-providers` 跟进 provider adapter
3. `llm-agent-otel` 跟进 wrapper
4. `llm-agent-customer-support` 跟进 reference app

统一治理规则的价值，就是让这类依赖链式变更保持可推进，而不是在多个仓库同时被 owner review gate 卡住。

## 优势

### 1. 规则和真实协作关系对齐

4 个项目本来就存在明显的上下游关系，这套规则让 PR 治理和这种关系保持一致。

### 2. owner PR 不再被平台原语反向阻塞

如果平台原生 required review 不能表达 owner self-PR 场景，就应该换一层表达，而不是继续让 owner PR 永远卡住。

### 3. 下游仓库依然保留人工审核门槛

这不是“全仓库自动合并”。它只是放行 owner 自己的 PR，把外部贡献保留在人工审核之下。

### 4. 无需额外 bot / 第二身份

这对小型多仓库项目很关键。规则越依赖额外身份，运维成本越高，长期越容易失效。

### 5. 适合跨仓库发布链路

当一次核心变更要穿过 provider、otel 和 reference app 时，统一治理规则能减少发布收尾摩擦。

## 确定性

这里的“确定性”指的是：相同输入下，规则能给出清晰、可预期、可验证的结果。

这套规则的确定性主要来自 4 点：

1. merge gate 是显式的：`go` 和 `governance`
2. author policy 是显式的：owner PR 和 non-owner PR 分流
3. review 校验是显式的：必须针对 current head
4. 行为结果是显式的：通过、阻塞、自动 merge，都能从检查状态直接看出来

相比“靠约定记住谁的 PR 该怎么处理”，这种确定性更适合多仓库协作，因为它不会随着上下文切换而失真。
