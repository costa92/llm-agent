# Changelog

`github.com/costa92/llm-agent` 的所有重要变更 ——
一个独立的 Go LLM agents 框架 module。

<!-- Keep a Changelog format: https://keepachangelog.com/en/1.1.0/ -->
<!-- Semver: https://semver.org/ -->
<!-- Sections per release: Added | Changed | Deprecated | Removed | Fixed | Security | Breaking -->
<!-- 0.x BC policy: minor/patch within a 0.x line are BC-compatible; 0.x→0.y (y>x) may break -->
<!-- Breaking changes: include "### Breaking" section + migration notes in the release entry -->

## [Unreleased]

## [v0.7.0] - 2026-05-25

合并发布：在单个 tag 中既关闭了 v1.2「核心能力深化」milestone（`CC-1`/`CC-2`/`CC-3`/`CC-4`），又提前带来了 v1.3 的 memory 工作。此前已交付的 v0.6.0（Phase 35 CC-1 budget），以及仅增量的补丁 tag v0.6.1（Phase 36 CC-2 policy）和 v0.6.2（Phase 37 CC-3 orchestrate.Supervisor）共同交付了 v1.2 的能力面；这里的 v0.7.0 是**显式的 milestone 关闭 tag**，外加提前带来的 v1.3 memory 特性套件。合并的运维决策于 2026-05-25 作出；诚实的重新定调记录见 `.planning/v1.2-MILESTONE-AUDIT.md`。全程保持 KC-5（仅增量，无 `/v2`）—— `memory/` 包是严格增量的：对 `Memory` 接口、`MemoryItem` 字段、`Manager` 方法签名，或任何 v1.x 测试 fixture 都零改动。

### Added

- `memory`：叠加在现有 `MemoryItem.Metadata` map 之上的 ChatGPT 风格 profile 元数据辅助。对 `MemoryItem` 结构体字段或 `Memory` 接口都没有改动 —— 所有状态都存在一个保留的、以 `_` 为前缀的 key 命名空间下（`_source`、`_category`、`_pinned`、`_disabled`；`_scope` 为未来的 PR 保留）。
  - 类型：`Source`（带 `SourceUserSaved`、`SourceAgentInferred`、`SourceSystem`、`SourceUnknown`）；`Category`（带 `CategoryUser`、`CategoryFeedback`、`CategoryProject`、`CategoryReference`）。
  - 构造器：`NewSavedMemory(content, cat)`（Importance=0.9、Pinned=true、Source=SourceUserSaved）；`NewInferredMemory(content, cat, confidence)`（confidence 被钳制到 [0,1] → Importance、Source=SourceAgentInferred）。
  - 访问器：`GetSource` / `SetSource`、`GetCategory` / `SetCategory`、`IsPinned` / `SetPinned`、`IsDisabled` / `SetDisabled`。getter 对 nil / 缺失 / 类型不匹配的元数据是零值安全的；setter 在 `Metadata` 为 nil 时会初始化它。
- `memory`：`WorkingOptions.SavedBoost`、`EpisodicOptions.SavedBoost`、`SemanticOptions.SavedBoost` —— 当条目 `IsPinned` 或 `GetSource(it) == SourceUserSaved` 时，在 `Search` 时应用的乘性分数因子。零值（或任何非正值）是严格的空操作，保留 v0.7 之前的评分。
- `memory`：`Scope{User, Project, Session}` 加上 `WithScope` / `ScopeFrom` ctx 辅助 —— 一个三轴分区描述符，在 Add 时被打入 `Metadata["_scope"]`，并在 Get / Search / SearchAll / Update / Remove 时读取。零值 `Scope{}` 是一个匹配每个条目的通配符，因此从不调用 `WithScope` 的现有调用方看不到行为改变。
- `memory`：`ScopedManager` —— `*Manager` 之上的装饰器，镜像 9 个公共 Manager 方法。Add 打入 ctx 作用域；Get / Search / SearchAll 按它过滤；Update / Remove 在跨作用域访问时返回 `ErrNotFound`（避免跨作用域泄漏 ID 的存在）。通过 `NewScopedManager(inner)` 构造；`Inner()` 暴露底层的 `*Manager`。
  - **v0.7 限制：** `ScopedManager` 上的 `Consolidate`、`Forget` 和 `StatsAll` **不**尊重作用域 —— 它们透传到内层 Manager，并对所有已存储条目操作，不论作用域。这些操作绕过 `Memory` 抽象以直接访问底层 store；作用域感知的变体推迟到未来版本。
- `memory`：`ErrManagerRequired` —— 由 `NewScopedManager(nil)` 返回的哨兵。
- `memory`：`Lister` 接口 + `ListFilter` + `ListPage`。`Lister` 是**可选**的 —— `Memory` 接口**不**嵌入它，保留仅增量契约。三个捆绑的 Memory 类型（`*WorkingMemory`、`*EpisodicMemory`、`*SemanticMemory`）都实现它。`List` 返回的条目按 `(CreatedAt DESC, ID ASC)` 确定性排序。`ListPage.NextCursor` 是一个不透明的 base64(JSON{after_created_at, after_id}) blob —— 调用方原样把它传回以获取下一页；流结束通过一个空 cursor 信号。`ListFilter` 在 `Scope`（通配符轴）、`Source`、`Category`、`Tags`（任意之一）、`PinnedOnly`、`IncludeDisabled`、`MinImportance` 上施加约束。
- `memory`：`Manager.ListAll(ctx, filter, pageSize, cursors)` —— 跨三种 kind 扇出 `List`。`cursors` 是按 kind 的（`map[Kind]string`）；缺失条目意味着该 kind「从头开始」。被禁用的 kind 会从结果 map 中静默省略（镜像 `SearchAll`）。
- `memory`：`ScopedManager.ListAll` —— 相同的扇出，在 `filter.Scope` 之上应用 ctx 作用域。非零的 ctx 作用域**覆盖** `filter.Scope`（ctx 作用域是信任边界）；零值 ctx 作用域原样尊重 `filter.Scope`。
- `memory`：`WithSanitizer(inner, chain...) Memory` —— 隐私钩子装饰器。该链仅在 `Add` 上从左到右运行。每个 `Sanitizer` 返回 `(newItem, keep, err)`：`keep=false` 短路该链且 `Add` 返回 `ErrRejectedByPolicy`；非 nil 的 `err` 原样传播；否则下一阶段收到 `newItem`。读路径（Get/Search/Update/Remove/Stats）完全绕过该链。`SanitizerFunc` 适配一个普通函数。`WithSanitizer(inner)`（空链）原样返回 `inner` —— 无分配、无行为改变。
  - **v0.7 限制：** `WithSanitizer` 返回一个 `Memory` 接口值，而非 `*WorkingMemory` / `*EpisodicMemory` / `*SemanticMemory`，因此它不能直接用作 `ManagerOptions` 字段。想要 Sanitizer + Manager 扇出的调用方必须在更高层组合（例如在 `Manager.Add` 之前运行 sanitizer），或在 Tool 接口面应用它。直接嵌入 `ManagerOptions` 推迟到未来版本。
- `memory`：`ErrRejectedByPolicy` —— 当链中某个 Sanitizer 返回 `keep=false` 时由 `Add` 返回的哨兵。
- `memory`：新的 `AsTool` 动作：`list`、`pin`、`unpin`、`disable`、`enable`。schema 获得四个可选的顶层字段（`filter`、`page_size`、`cursor`、`cursors`），`action` 枚举增加这五个新值。所有现有的 action 枚举条目和字段名都未改变 —— v0.7 之前的调用方看不到行为改变。
- `memory`：持久化层。`Snapshot{Version, Kind, Items}` + `SnapshotItem{Item, Vector}` 构成一个可 JSON 序列化的转储，它内联缓存的嵌入，从而接收方复用向量而非对恢复的内容重新嵌入。`SnapshotVersion = 1` 是当前的 schema 版本；导入时遇到未知版本返回 `ErrSnapshotVersionMismatch`。kind 不匹配（例如把一个 `KindEpisodic` 快照导入到一个 `*WorkingMemory`）返回 `ErrSnapshotKindMismatch`。
- `memory`：`Exporter` 和 `Importer` 可选能力接口。`*WorkingMemory`、`*EpisodicMemory`、`*SemanticMemory` 都实现两者。`Export(ctx) (Snapshot, error)` 按 `(CreatedAt ASC, ID ASC)` 顺序发出条目，从而 JSON 字节跨运行稳定。`Import(ctx, snap, mode)` 返回一个带按模式的 `Loaded` / `Skipped` / `Replaced` 计数器的 `ImportReport`。
- `memory`：`ImportMode` 枚举：`ImportReplace` 先擦除目标再加载每个快照条目；`ImportMerge` 仅添加未见过的 ID（冲突使 `Skipped` 计数增加）；`ImportUpsert` 添加未见过的并覆盖已有的（冲突使 `Replaced` 计数增加）。
- `memory`：`SnapshotStore` 可插拔持久化接口（`Save / Load / Delete / List`）。`FilesystemStore` 是仅标准库的默认实现 —— 每个 `(key, kind)` 元组一个 JSON 文件，经过净化的文件名（`[a-zA-Z0-9_-]` 之外的每个字符都变成 `_`）防止路径遍历，经由 `os.CreateTemp` + `os.Rename` 进行原子写入。`FilesystemStore.LoadKind(ctx, key, kind)` 是 `Manager.ImportAll` 使用的类型化变体。
- `memory`：`RestoreWorking` / `RestoreEpisodic` / `RestoreSemantic` 构造器。每个都构建具体的 Memory 类型**并**立即以 `ImportReplace` 模式导入所提供的快照。Embedder 仍然必需（用于后续的 `Add` / `Search`），但恢复的条目复用其内联向量 —— 无重新嵌入。
- `memory`：`ManagerOptions.SnapshotStore`（可选）。设置后，`Manager.ExportAll(ctx, persistKey)` 把每个活动 kind 的快照写入 store，`Manager.ImportAll(ctx, nil, persistKey, mode)` 把它们读回。带内联 `snaps` map 的 `ImportAll` 完全绕过 store（snaps 胜出）。`persistKey != ""` 而无 `SnapshotStore` 返回 `ErrSnapshotStoreNotConfigured`。
- `memory`：新的哨兵错误：`ErrSnapshotVersionMismatch`、`ErrSnapshotKindMismatch`、`ErrSnapshotStoreNotConfigured`。
- `memory`：新的 `AsTool` 动作：`export`、`import`。schema 获得两个可选的顶层字段（`snapshot_key`、`import_mode`）；`action` 枚举增加这两个新值。`export` 包装 `Manager.ExportAll`；`import` 包装 `Manager.ImportAll`，并在省略 `import_mode` 时默认为 `ImportMerge`（最安全）。
- `memory`：持久化层保持**仅标准库** —— 新的导入限于 `encoding/json`、`os`、`io`、`path/filepath`、`sort`、`strings`、`errors`、`fmt`、`context`。核心中无第三方存储依赖；下游 store 经由 `SnapshotStore` 接口插入。

### Changed

- `memory`：跨所有三种 memory 类型的 `Search` 现在跳过被标记为 `IsDisabled(it) == true` 的条目。被禁用的条目仍留在存储中（Get / Stats / Forget 仍能看到它们）；它们只是从查询结果中被隐藏。
- `memory`：`Manager.Forget` 策略（`ForgetByImportance`、`ForgetByAge`、`ForgetByCapacity`）现在跳过被标记为 `IsPinned(it) == true` 的条目。被钉住的条目从候选集中排除；在 `ForgetByCapacity` 下它们既不计入 `Keep` 也不被驱逐。

## [v0.6.2] - 2026-05-23

捆绑发布：在 `StateGraph[S]` 之上引入一个仅标准库的 `orchestrate.Supervisor` 门面以用于 planner/worker 协调，并关闭来自 v1.3 K1 闭合波次的三个正确性修复 —— Phase 2 Gap B（按 K1 进行 AccumulateStream 的 Index-keyed 合并）、P1-4（RunStream 取消时发出终结的 Done 事件），以及 P1-3（a2a server worker 的 DELETE 取消）。所有签名未变；按已记录的 K1 和取消契约，可观测行为严格地更正确。针对 `v0.6.1` 编译的调用方针对 `v0.6.2` 编译无需改动（保留 KC-5）。

### Added

- 新的 `orchestrate.Supervisor` 接口面 —— `NewSupervisor`、`SupervisorOptions`、`Dispatch`、`WorkerResult`、`DispatchParser`、`Aggregator`、`Run`、`RunStream`，以及用于验证/dispatch 错误的哨兵家族。
- 新的 `orchestrate/supervisor.go` 实现，外加针对预算传播、policy 组合，以及与 `StateGraph[S]` 的运行时组合的确定性测试。
- 新的 `examples/08-supervisor/` 演示 —— 基础协调、预算门控，以及与图的组合冒烟测试。

### Fixed

- `llm.AccumulateStream` 现在按 `ToolCallDelta.Index`（按 K1 契约稳定的、每个工具调用的 key）合并流式工具调用增量，而非按 `ID`。此前的 ID-keyed 实现会静默丢弃那些 `ID` 字段为空的 `EventToolCallArgsDelta` chunk —— 这正是标准 OpenAI / Anthropic / Ollama 的线上形状，其中 `ID` 仅在 `EventToolCallStart` 事件上被填充。另一个症状：两个在不同 `Index` 上具有相同 `ID` / `Name` 的并行工具调用（Ollama 的 `ID==Name` 回退）会塌缩成一条。函数签名未变。未导出的 `appendToolCallDelta` 辅助被移除；其逻辑被内联进 `AccumulateStream`，使用新的 Index-keyed map 和一个确定性的 first-Start 排序。（Phase 2 Gap B，关闭 `llm/stream.go` 处 K1「生产级累加器」的免责声明。）
- `runStreamFromBlocking` 和 `Supervisor.RunStream` 在 `ctx` 于运行中途被取消时不再静默关闭 `StepEvent` 通道。两者现在在关闭前都发出一个终结的 `StepEvent{Done: true, Err: ctx.Err()}`，从而 SSE 处理器和任何 `for ev := range ch` 消费方都能区分干净的结束与中途取消。终结事件的优先级是 `err > ctx.Err > Final`，以保证即使 `runFn` 与取消竞速也恰好有一个 Done 事件。（P1-4）
- `comm/a2a` 服务端的 worker goroutine 现在经由 `DELETE /tasks/{id}` 取消，而非不可杀死。该 worker 此前以 `context.Background()` 运行；它现在使用一个按任务的 `WithCancel`，其 cancel funcval 存在于 Task 上，并由 DELETE 处理器调用。取消复用 `TaskFailed` 并带 `Error="canceled by DELETE"`，以避免增加一个新的枚举状态（在 `TaskState` 上 switch 的客户端保持穷尽）。（P1-3）

### Compatibility

- `llm.AccumulateStream` 签名未变（`func(StreamReader) (Response, error)`）。按 K1 契约可观测行为严格地更正确：没有生产调用方依赖此前的损坏行为（现有生产路径喂入纯文本流;没有工具流式路径针对一个在 Start 上 `ID` 非空、在后续 ArgsDelta chunk 上为空的提供方消费 `AccumulateStream`）。
- 保留仅标准库不变量（无新的第三方导入）。

## [v0.6.1] - 2026-05-21

增量发布：引入一个仅标准库的 `policy` 子包 —— 一个保留能力的 `llm.ChatModel` 装饰器，它在请求、响应和流边界运行类型化的 `Gate` 事件。对任何现有包都没有行为改变；如果消费方不需要 policy 强制，可以停留在 `v0.6.0`。严格增量：针对 `v0.6.0` 编译的调用方针对 `v0.6.1` 编译无需改动（原样遵守 KC-5 —— `llm/`、各范式文件、`agent_chatmodel.go`、`memory/`、`orchestrate/`、`go.mod`、`go.sum` 都与 Phase 36 之前的状态逐字节相同）。

### Added

- 新的 `policy` 子包 —— 保留能力的 `llm.ChatModel` 装饰器。镜像 `otelmodel.Wrap` 的形状（KC-3），带 8 个包装器的类型 switch 树 + 21 个编译期接口断言，从而 `ToolCaller` / `Embedder` / `StructuredOutputs` 能力在 wrap 中被保留。
  - `policy.Wrap(model, gates...)` —— 便利的入口点。
  - `policy.WrapConfig(model, Config{Gates: ..., OnDecision: f})` —— 带可选审计回调（同步、nil 安全、panic 恢复）的结构化入口。
  - `policy.Gate` 接口 + `policy.Event` 结构体 + 带 5 种 kind 的 `policy.EventKind` 枚举（`PreGenerate` / `PostGenerate` / `PreStream` / `StreamDelta` / `PostStream`）。
  - `policy.Decision` 结构体 + 带 4 个动作的 `policy.DecisionAction` 枚举（`Allow` / `Block` / `Redact` / `Replace`）。
  - `policy.ErrBlocked` 哨兵 + `policy.BlockedError` 富错误对（`errors.Is` 总括 + `errors.As` 细节，带嵌入的 `Decision` 副本）。
- 三个内置门控（全部仅标准库）：
  - `policy.NewPIIRedactor()` —— 脱敏 email / phone / IPv4 模式（US-locale 的 ssn / credit_card 推迟到未来的 `NewUSLocalePIIRedactor` 增量）。`WithStreamRedaction` 选择开启逐增量扫描（按 Q4 默认关闭 —— 逐增量正则昂贵，且跨增量 PII 按设计可能泄漏）。
  - `policy.NewInjectionScanner()` —— 对 4 个规范的提示词注入模式进行调用前拦截（按 KS-5 从 `llm-agent-rag/guard` 复制而来，非导入）。
  - `policy.NewMaxInputLen(n int)` —— 当 prompt 大小超过 n 字节时调用前拦截（Q3 —— 字节是面向提供方 HTTP 预算的有效上限；未来的 `MaxInputLenRunes` 是一个 v1.3 增量候选）。
- 与 `otelmodel.Wrap` 的组合在 `examples/07-policy/README.md` 中有文档记录 —— 规范的 v1.2+ 栈是 `policy.Wrap(otelmodel.Wrap(provider), ...)`：最外层在被观测前拒绝，中间观测，最内层调用。该示例的 main.go 刻意**不**导入 otel 兄弟仓（决策 G —— 当 `llm-agent-otel` 版本提升以匹配核心 `v0.6.x` 时，兄弟仓示例在 v1.3 中交付）。
- 新的 `examples/07-policy/` —— 三个由 `llm.ScriptedLLM` 驱动的确定性演示（`demoPIIRedaction`、`demoInjectionBlock`、`demoMaxInputLen`），端到端地证明每个门控的决策动作。用 `cd examples && go run ./07-policy` 运行。

## [v0.6.0] - 2026-05-21

增量发布：引入一个仅标准库的共享测试辅助子包。对任何现有包都没有行为改变；如果消费方不需要这些新辅助，可以停留在 `v0.5.1`。

### Added

- 新的 `agentstest` 子包 —— 面向 `agents.Tool` 的仅标准库共享测试辅助。提供 `StubTool` / `NewStubTool` / `NewErrorTool` 用于构建假工具，以及 `RecordingTool`（一个记录每次 Execute 调用的线程安全装饰器）。意在供兄弟仓的 `*_test.go` 消费（类比 `net/http` 的 `net/http/httptest`）；避免此前每个仓库本地重新打桩 `agents.Tool` 的模式。桥接说明见 `agentstest/doc.go`（使用 `flow.FromAgentTool` 适配到更窄的 `flow.Tool` 接口）。

## [v0.5.1] - 2026-05-20

### Changed

- 把 `llm-agent-rag` 版本提升到 `v1.0.1`（回边刷新，无公共 API 改变）。

## [v0.5.0] - 2026-05-21

`v0.4` 之后的 RAG 兼容性维护，外加 Phase-31 与独立 SDK 冻结的 `v1.0` API 的对齐。核心保持仅标准库；公共 `rag` 门面 API 未变。

### Changed

- 把 `github.com/costa92/llm-agent-rag` 从 `v0.1.4` 版本提升到 `v1.0.0`（独立 SDK 冻结的 `v1.0` API）。
- 为 `v1.0.0` store 契约修复了核心 `rag/` 兼容性门面 —— `storeAdapter` 现在经由一个真实的 list 路由（`*InMemoryStore.ListDocuments` + 一个可选的 `lister` 接口 + 一个 id-index 回退）枚举文档，而非一个 `nil` 向量的相似度搜索（后者被 `v1.0.0` 更严格的 `store.InMemoryStore.Search` 拒绝）。
- 把 `github.com/costa92/llm-agent-rag` 从 `v0.1.2` 版本提升到 `v0.1.4`（`v0.4` 之后的中间维护提升，被上面的 `v1.0.0` 提升取代）。
- 把核心 `rag/` 兼容性门面与独立的检索 policy 路径对齐：
  - MQE / HyDE 现在委托给独立的检索编排
  - `Ask(...)` 委托给独立的 rerank + 上下文打包流程
  - 面向工具的 `enable_rerank` 现在被打通到独立的 ask/search
- 为独立契约对等扩展了核心门面的内部 store adapter：
  - `List(...)`
  - `RemoveByFilter(...)`

### Breaking

- 从 `llm/` 中移除了已弃用的 v0.2 兼容性符号：
  - `llm.Client`
  - `llm.LegacyClient`
  - `llm.GenerateRequest`
  - `llm.GenerateResponse`
  - `llm.StreamChunk`
  - `llm.StreamUsage`
- 任何仍针对被移除接口面编译的下游必须迁移到：
  - `llm.ChatModel`
  - `llm.Request`
  - `llm.Response`
  - `llm.StreamReader`
  - `llm.StreamEvent`

### Changed

- 核心运行时包现在仅依赖 `llm.ChatModel`：
  - `rag`
  - `context`
  - `bench`
  - `rl`
- 仓库示例、测试辅助和快速开始文档现在只展示当前的 `ChatModel` API。
- `rag` 已被对外拆分为 `github.com/costa92/llm-agent-rag`。主仓库的 `rag/` 现在充当一个兼容性门面，同时本仓库依赖：
  - `github.com/costa92/llm-agent-rag v0.1.0`

### Removed

- 删除了 `llm/legacy.go`。
- 移除了为证明 `Client`/`LegacyClient` 往返而存在的仅别名测试。

### Added

- `llm/` 中新的能力感知接口：
  - `llm.ChatModel` —— 基础契约（`Generate` + `Stream` + `Info`）
  - `llm.ToolCaller` —— 原生函数调用的能力（`WithTools`，不可变）
  - `llm.Embedder` —— 向量嵌入的能力（**不**嵌入 `ChatModel`）
  - `llm.StructuredOutputs` —— JSON-schema 约束生成的能力
- `llm/stream.go` 中的类型化流式联合：
  - `llm.StreamReader` —— 迭代器风格的流式（`Next` + `Close`）
  - `llm.StreamEvent` + `llm.StreamEventKind` —— 带 `EventTextDelta`、`EventToolCallStart`、`EventToolCallArgsDelta`、`EventToolCallEnd`、`EventThinkingDelta`、`EventDone` 的类型化联合
  - `llm.ToolCallDelta` —— 带稳定 `Index` 字段的每个工具调用流式状态
  - `llm.AccumulateStream` —— 为想要一个扁平 `Response` 的消费方提供的便利
- 每个（provider x model）的身份：
  - `llm.ProviderInfo` —— 由 `Info()` 返回的绑定 provider+model 身份
  - `llm.Capabilities` —— 可 JSON 序列化的特性结构体（`Tools`、`Embeddings`、`StructuredOutputs`、`PromptCaching` 布尔字段）
- 新的 chat 层 request/response 类型：
  - `llm.Request`（替代 `GenerateRequest`）
  - `llm.Response`（替代 `GenerateResponse`）
  - `llm.Vector`（`[]float32`）
  - `llm.Usage` + `llm.UsageSource`（`Reported` / `Estimated` / `Unknown`），用于 K4 三态成本追踪
- 新的 mock（从 `_test.go` 提升而来）：
  - `llm.ScriptedLLM` —— 带函数式选项（`WithProvider`、`WithModel`、`WithCapabilities`、`WithResponses`、`WithEmbedDimensions`）的全能力确定性 mock；辅助 `TextResponse`、`ToolCallResponse`
  - `llm.ChatOnlyMock` —— 仅 `ChatModel` 的 mock，用于能力降级测试
- 哨兵错误：
  - `llm.ErrCapabilityNotSupported` —— 用 `fmt.Errorf("...: %w", ...)` 包装
  - `llm.ErrScriptExhausted` —— 当脚本耗尽时由 `ScriptedLLM` 发出
- 位于 `docs/migration-v0.2-to-v0.3.md` 的迁移指南，带一个已演练的 Simple 范式示例 + 一个通用的类型重命名映射表。
- 仓库根的 `DEPRECATIONS.md` —— 符号 → 目标移除版本的唯一事实来源；Phase 7 在 v0.4 切版前审计此文件（Pitfall 15）。

### Deprecated

以下符号为 v0.3.x 源兼容性而保留，但将**在 v0.4.0 中移除**。迁移步骤见 [`docs/migration-v0.2-to-v0.3.md`](docs/migration-v0.2-to-v0.3.md)；完整表格见 [`DEPRECATIONS.md`](DEPRECATIONS.md)。

- `llm.Client`（interface）—— 现在是 `llm.LegacyClient` 的别名。改用 `llm.ChatModel`。
- `llm.LegacyClient`（interface）—— 从 `llm.Client` 重命名而来。改用 `llm.ChatModel`。
- `llm.GenerateRequest`（struct）—— 改用 `llm.Request`。
- `llm.GenerateResponse`（struct）—— 改用 `llm.Response`。
- `llm.StreamChunk`（struct）—— 改用 `llm.StreamEvent`（类型化联合）。
- `llm.StreamUsage`（struct）—— 改用 `llm.Usage`（带 `Source` 字段）。
- `agents.scriptedLLM`（根包测试辅助）—— 改用 `llm.NewScriptedLLM`。在 agent 范式迁移到 `llm.ChatModel` 后于 Phase 3（约 v0.3.3）移除。

### Versioning policy (INFRA-07)

**跨 4 个仓库的版本策略：** `llm-agent` v0.3.x 核心；兄弟仓从 v0.1.x 起步；每个仓库的 CHANGELOG 带 `### Breaking`。完整的 BC 矩阵见 README §Versioning。

v0.3 milestone 覆盖一个 4 仓伞形：

| Repo | v0.3 track | Notes |
|---|---|---|
| `github.com/costa92/llm-agent`（本仓） | `v0.3.x` | 仅标准库核心。预发布 tag `v0.3.0-pre.1` 在 Phase 0 结束时切出；最终的 `v0.3.0` 在 Phase 6 之后。 |
| `github.com/costa92/llm-agent-providers` | `v0.1.x` | 兄弟仓在 Phase 0 创建；首批内容在 Phase 1 落地。 |
| `github.com/costa92/llm-agent-otel` | `v0.1.x` | 兄弟仓在 Phase 0 创建；首批内容在 Phase 5 落地。 |
| `github.com/costa92/llm-agent-customer-support` | `v0.1.x` | 兄弟仓在 Phase 0 创建；首批内容在 Phase 6 落地。 |

- 0.x BC 策略按仓库适用：一条 0.x 线内的 minor/patch 是 BC 兼容的；0.x→0.y（y>x）可能破坏。每个仓库在其 CHANGELOG 中用 `### Breaking` 小节声明破坏性变更。
- 兄弟仓按 Phase / 兄弟仓发布锚定 `require github.com/costa92/llm-agent vX.Y.Z`；在 v0.4 切版（Phase 7）期间协调一致地打 tag。
- `replace` 指令在任何匹配 `release/**` 的分支上被**禁止** —— 由每个仓库的 `release-precheck` CI 工作流强制。
- `go.work` 在每个仓库里都被 `.gitignore`；CI 用 `GOWORK=off` 运行。

## [v0.1.0] — 2026-04-28

初次 module 发布。框架于 2026-04-27 到 2026-04-28 之间作为 9 个 phase 在父仓库中实现；本次发布把它抽取进自己的 Go module，从而外部用户能 `go get` 它，而无需拉入 AICS 主 module 的传递依赖（Kratos、GORM、Redis 等）。

### Added

- 独立 Go module：`github.com/costa92/llm-agent`
- 新的 `agents/llm` 子包，拥有 LLM 契约：`Client`、`GenerateRequest`、`GenerateResponse`、`Message`、`Tool`、`ToolCall`、`StreamChunk`、`StreamUsage`、`FinishReason` + 6 个常量
- 暴露 12 个包：`agents`（根）、`agents/llm`、`agents/builtin`、`agents/memory`、`agents/rag`、`agents/context`、`agents/comm`、`agents/comm/mcp`、`agents/comm/a2a`、`agents/comm/anp`、`agents/orchestrate`、`agents/rl`、`agents/bench`
- 仅标准库 —— 零第三方 Go 依赖

### Notes

- v0.1.0 是 **学習 / 原型** —— API 在 0.x 各版本之间可能破坏。稳定性承诺请等待 **v1.0**。
- 源设计 spec：`docs/superpowers/specs/2026-04-27-pkg-llm-agents-design.md`（在父 AICS 仓库中）。
- 抽取设计 spec：`docs/superpowers/specs/2026-04-28-pkg-llm-agents-module-extraction-design.md`（在父 AICS 仓库中）。

## [v0.2.0] — 2026-05-08

独立仓库抽取。框架在父 AICS 单体仓库（`github.com/costa92/aics-core/pkg/llm/agents`）内经由 Phase R 开发；本次发布把它提升进自己的 GitHub 仓库和 Go module，从而导入路径不再嵌套。

### Changed (Breaking)

- **Module path** —— `github.com/costa92/aics-core/pkg/llm/agents` → `github.com/costa92/llm-agent`。子包路径遵循相同的扁平化（例如 `.../aics-core/pkg/llm/agents/llm` → `.../llm-agent/llm`）。调用方必须相应更新 import 语句。

### Added

- `pkg/fanout` —— 并发任务执行器（`fanout.Task[T]` / `fanout.Run` / `WithFailFast`），从 aics-core 复制；此前是一个传递依赖，现在是一个一等子包。
- `internal/testenv` —— 仅测试的 HTTP listen 辅助，从 aics-core 复制以保持本 module 零第三方。

### Fixed

- 更新了 `doc.go` 的可移植性契约 —— 去掉对旧单体仓库的 `internal/*` 和 `pkg/*` 约束的引用；现在读作一个独立的仅标准库 module。

### Migration

```diff
- import "github.com/costa92/aics-core/pkg/llm/agents"
+ import "github.com/costa92/llm-agent"

- import "github.com/costa92/aics-core/pkg/llm/agents/llm"
+ import "github.com/costa92/llm-agent/llm"
```

```bash
go get github.com/costa92/llm-agent@v0.2.0
```

---

### Versioning

本仓库上的 tag 是扁平的（`vX.Y.Z`）—— 没有 `<module-subpath>/` 前缀。0.x 线：minor/patch 是 BC；0.x → 0.y（y > x）可能破坏。
