# Multi-Repo PR Governance Overview

这组文档记录 `llm-agent` 及其 3 个关联仓库的统一 PR 治理设计。目标不是描述某个单独仓库的 GitHub 设置，而是说明一个多仓库系统如何把变更入口规则统一起来。

## 文档结构

- [01. 项目关系](./PR-GOVERNANCE-PROJECTS.md)
- [02. 治理规则](./PR-GOVERNANCE-RULES.md)
- [03. 落地与运维](./PR-GOVERNANCE-OPERATIONS.md)

## TL;DR

- GitHub 内建 required approving review 无法表达“owner 自己的 PR 不需要 review，其他人的 PR 需要 owner review”。
- 所以 merge gate 被改成两个 required status checks：`go` 和 `governance`。
- `go` 负责代码正确性，`governance` 负责 author-sensitive 审核策略。
- owner PR 会自动通过 `governance` 并开启 auto-merge。
- non-owner PR 会自动 request review 给 `costa92`，且只有 `costa92` 对当前 head 审批后，`governance` 才通过。
- 这套规则不是单仓库技巧，而是围绕 4 个关联项目建立的一致化多仓库治理策略。

## 问题定义

我们希望同时满足两条规则：

1. `costa92` 自己提交的 PR，在 CI 通过后应自动合并。
2. 其他人提交的 PR，必须经过 `costa92` 审核后才能合并。

GitHub 内建的 required approving review 无法直接表达这条 author-sensitive 规则，所以真正的设计重点不在于“怎么自动点 merge”，而在于“怎么把治理规则从平台原语里抽离出来，变成一个显式、可验证的检查”。

## 最终结论

最终采用的是“代码检查 + 治理检查”分层：

- `go` 负责代码正确性
- `governance` 负责 author-sensitive 合并策略

也就是说，merge gate 从：

- “必须有一个 GitHub approving review”

改成了：

- “必须通过 `go`”
- “必须通过 `governance`”

## 这组文档适合谁

- 维护 owner 主导的小型仓库
- 需要让 owner PR 自动合并、外部 PR 人工审核
- 有 1 个核心仓库和多个下游仓库的项目
- 不想为了审批问题引入 bot、GitHub App 或第二身份 token

## 阅读顺序

如果你想快速理解：

1. 先看 [项目关系](./PR-GOVERNANCE-PROJECTS.md)
2. 再看 [治理规则](./PR-GOVERNANCE-RULES.md)
3. 最后看 [落地与运维](./PR-GOVERNANCE-OPERATIONS.md)

## 一句话总结

这次设计的本质是：把“必须有一个 approval”改成“必须通过一个能理解作者身份的治理检查”。
