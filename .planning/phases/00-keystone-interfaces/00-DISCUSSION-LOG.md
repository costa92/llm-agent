# Phase 0: Multi-repo infra + keystone interfaces - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-10
**Phase:** 0-keystone-interfaces
**Areas discussed:** Package path, Capabilities shape, Mock organization, Sister repo creation timing

---

## Package Path for New Capability Interfaces

| Option | Description | Selected |
|--------|-------------|----------|
| Reboot `llm/` | New types in `llm/`, old `Client` renamed to `LegacyClient`. Same import path. v0.x BC break is allowed. | ✓ |
| `llm/v2` subpath | New `package v2` in `llm/v2/`. Strict isolation but unusual within a single Go module. | |
| Fresh name `model/` or `chat/` | Clean break with new namespace. Old `llm/` deprecated as a whole package. Cleanest but maximum churn. | |

**User's choice:** Reboot `llm/` (D-01)
**Notes:** Project is small (only `client.go` + `doc.go` in `llm/`). v0.x policy explicitly allows breaks. Rebooting keeps import path stable; users `sed`-migrate type names. `LegacyClient` lives next to new types until v0.4 removal.

---

## ProviderInfo / Capabilities Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Embedded struct field | `info.Capabilities.Tools` boolean field. JSON-serializable, godoc-friendly, OTel-attribute-friendly. | ✓ |
| Methods on ProviderInfo | `info.SupportsTools()`. Cleaner intent, but harder to JSON-serialize and harder to encode three-state ("unknown"). | |
| Bitmask | `type Capabilities uint32`. Compact but opaque in test failures and OTel attributes. | |

**User's choice:** Embedded struct field (D-02)
**Notes:** Type assertion stays as the COMPILE-TIME signal (Eino's `BindTools` lesson — small interfaces let agents check `if tc, ok := model.(ToolCaller); ok { ... }`). `Capabilities` is the RUNTIME signal because capabilities are per-(provider × model), not per-provider — an `*Ollama` bound to llama2 type-asserts as `ToolCaller` the same as one bound to llama3, but only the latter actually supports tools. Pitfall 6 is the binding source.

---

## Mock Organization

| Option | Description | Selected |
|--------|-------------|----------|
| Full ScriptedLLM v2 + small per-capability mocks | One full impl as the default; `ChatOnlyMock` etc. for fallback testing. | ✓ |
| Single full ScriptedLLM v2 only | Always-full impl; degradation tests rely on configuration flags. | |
| Multiple independent mocks (compose-only) | `ChatModelMock` / `ToolCallerMock` / `EmbedderMock` as separate structs; tests compose them. Most flexible but most boilerplate. | |

**User's choice:** Full ScriptedLLM v2 + small per-capability mocks (D-03)
**Notes:** ScriptedLLM is promoted out of `_test.go` (sister repos' conformance suites need to import it). `ChatOnlyMock` exists specifically to verify Phase 3's capability-degradation paths (e.g., ReAct's scratchpad fallback when `model.(ToolCaller)` fails).

---

## Sister Repo Creation Timing

| Option | Description | Selected |
|--------|-------------|----------|
| Phase 0 creates + pushes all 3 GitHub repos | Skeleton (`go.mod` + LICENSE + OWNERS + README + CI) on day 1. | ✓ |
| Phase 0 local go.mod only; Phase 1 push | Local-only modules in sibling directories; push GitHub when first PR is needed. | |
| Phase 0 design-only; physical repos in Phase 1 | All sister-repo-related work pushed to Phase 1; ROADMAP would need amendment. | |

**User's choice:** Phase 0 creates + pushes all 3 GitHub repos (D-04)
**Notes:** Umbrella CI (INFRA-05) physically requires the 3 GitHub repos to exist (it clones them). Discovering this in Phase 1 means redoing Phase 0. Cost is ~1-2 hours of repo creation; benefit is unblocked rest of v0.3 work.

---

## Claude's Discretion

- **Migration guide depth (CORE-09):** target 1 worked example (Simple paradigm — it's the docs example) + a generic type-rename mapping table. Other 4 paradigms covered by the table; per-paradigm worked examples deferred until users ask.
- **`StreamEvent` exact field naming:** locked directionally (Kind enum + per-tool-call Index + delta fields), but `Args` vs `Arguments` vs `ArgsDelta` is researcher/planner discretion — match Anthropic SDK's `partial_json` and OpenAI SDK's `function.arguments` ergonomics where possible.
- **Sister repo READMEs (INFRA-06):** Claude drafts; user reviews at PR time.

## Deferred Ideas

- **Provider Author Guide v0.1** — Phase 1 scope (CORE-11). Phase 0 provides the *types* the guide will document; prose waits until at least one adapter exists.
- **Agent paradigm refactor (CORE-10)** — Phase 3 scope. Phase 0 designs interfaces with the 5 paradigms in mind but does NOT touch `simple.go` / `react.go` / etc.
- **OTel decorator implementation** — Phase 5 scope. Phase 0 ensures interfaces COMPOSE under wrapping but does NOT write the decorator.
- **Conformance test harness** — Phase 1 scope. Phase 0 promotes ScriptedLLM v2 out of `_test.go` so the harness can import it.
