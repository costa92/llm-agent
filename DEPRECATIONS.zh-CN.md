[English](./DEPRECATIONS.md) | [简体中文](./DEPRECATIONS.zh-CN.md)

# Deprecations

本文件是**什么被弃用、何时消失**的唯一事实来源。本仓库中每一个带 `// Deprecated:` godoc 注释的符号都必须出现在此处，并带一个目标移除版本。v0.3 → v0.4 周期的 Phase 7 会审计这份列表，并移除每一行其 `Removed In` 与在途 tag 匹配的条目。

> 本文件存在的原因：v0.3 研究包的 Pitfall 15（`deprecation kept forever`，弃用被永久保留）。
> `// Deprecated:` 注释会在 IDE 悬浮提示中浮现，但在发布打 tag 时是不可见的 —— 一个独立文件迫使每次 minor 切版前都做一次清扫。

## Active deprecations

*(none)*

## Removed (historical)

- `llm.Client`（interface）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.ChatModel`。
- `llm.LegacyClient`（interface）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.ChatModel`。
- `llm.GenerateRequest`（struct）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.Request`。
- `llm.GenerateResponse`（struct）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.Response`。
- `llm.StreamChunk`（struct）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.StreamEvent`。
- `llm.StreamUsage`（struct）—— 在 `v0.3.0` 中弃用，在 Phase 7 的 `v0.4` 切版中移除；改用 `llm.Usage`。

## Adding new deprecations

当你为任何公共符号添加 `// Deprecated:` 注释时：

1. 在上面的 **Active deprecations** 表中加一行，带上精确的符号路径、引入弃用的版本、目标移除版本，以及一个迁移链接。
2. 在符号上使用以下精确的 godoc 格式，以便 `gopls` 和 `staticcheck` 能告警：
   ```
   // Deprecated: <use what instead>. Will be removed in vX.Y.Z. See <doc link>.
   ```
3. `Removed In` 列**必须**指向一个真实的、已规划的发布。模糊的目标（"future"、"TBD"）被禁止 —— 它们正是 Pitfall 15 的编码。

## Removal procedure

当一个 tag 匹配某个 `Removed In` 值时：

1. 确认 `git grep -n '<symbol>' -- ':!DEPRECATIONS.md' ':!CHANGELOG.md' ':!docs/'` 返回零个内部使用者。
2. 删除该符号声明**以及**它的 godoc 注释。
3. 把该行从 **Active deprecations** 移到 **Removed (historical)**，并加上实际的移除提交 SHA。
4. 为该发布在 `CHANGELOG.md` 中加一条 `### Breaking` 条目。
5. 在协调一致的 tag 中提升兄弟仓的 `require github.com/costa92/llm-agent` 行。

---

Last updated: 2026-05-10（v0.3 milestone 的 Phase 0 —— 首个条目）。
