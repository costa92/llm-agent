---
phase: 00-keystone-interfaces
plan: 01a
subsystem: llm
tags:
  - llm
  - interfaces
  - capability-negotiation
  - go-stdlib-only
dependency_graph:
  requires: []
  provides:
    - llm.ChatModel
    - llm.ToolCaller
    - llm.Embedder
    - llm.StructuredOutputs
    - llm.StreamReader
    - llm.StreamEvent
    - llm.ProviderInfo
    - llm.Capabilities
    - llm.LegacyClient
    - llm.Client (alias)
    - llm.Request
    - llm.Response
    - llm.Message
    - llm.Tool
    - llm.ToolCall (with ID)
    - llm.Vector
    - llm.Usage
    - llm.UsageSource
    - llm.ErrCapabilityNotSupported
    - llm.ErrScriptExhausted
  affects:
    - all agent paradigm callers (via Client=LegacyClient alias — no source change needed)
    - plan 00-01b (builds ScriptedLLM v2 + ChatOnlyMock + doc.go + tests on top of this surface)
tech_stack:
  added: []
  patterns:
    - stdlib-only imports (context, encoding/json, errors)
    - iterator-style StreamReader (Next + Close) vs channel-based
    - immutable capability constructors (WithTools, WithSchema return new values)
    - legacyFinishReason private type + public FinishReason alias (bridges v0.2/v0.3)
    - D-02: Capabilities as embedded struct field (not methods, not bitmask)
key_files:
  created:
    - llm/chatmodel.go      # 21 LOC — ChatModel interface (Generate + Stream + Info)
    - llm/capabilities.go   # 42 LOC — ToolCaller, Embedder, StructuredOutputs
    - llm/info.go           # 30 LOC — ProviderInfo + Capabilities struct (D-02)
    - llm/stream.go         # 145 LOC — StreamReader + StreamEventKind + StreamEvent + ToolCallDelta + AccumulateStream
    - llm/errors.go         # 21 LOC — ErrCapabilityNotSupported + ErrScriptExhausted sentinels
    - llm/types.go          # 105 LOC — Request, Response, Message, Tool, ToolCall (ID added), Vector, Usage, UsageSource, FinishReason alias
  modified:
    - llm/legacy.go         # 85 LOC — renamed from client.go; LegacyClient + Client alias + deprecated types
decisions:
  - "D-01 ratified: reboot llm/ in-place; import path unchanged; LegacyClient rename + Client alias"
  - "D-02 ratified: Capabilities embedded struct (not methods/bitmask); 4 bool fields with snake_case JSON tags"
  - "Q1 resolved: Tool/Message/ToolCall shared between LegacyClient and ChatModel surfaces in same package"
  - "legacyFinishReason private type + FinishReason alias approach bridges v0.2 constants to v0.3 type system"
  - "isEOF string comparison indirection in stream.go avoids importing io at this layer (Phase 2 accumulator imports io directly)"
  - "StreamEventKind uses uint8+iota (performance, 6 known variants) vs string enum (existing repo preference) — justified by Phase 2 hot path"
metrics:
  duration: ~30min
  completed: 2026-05-10
  tasks_completed: 3
  files_created: 6
  files_modified: 1
---

# Phase 0 Plan 01a: llm/ Keystone Interface Contract Summary

**One-liner:** JWT-style capability negotiation surface locked in `llm/` — ChatModel+ToolCaller+Embedder+StructuredOutputs+StreamEvent typed union with stable Index + per-(provider×model) ProviderInfo struct — stdlib-only, zero new deps, all v0.2 callers compile unchanged via `Client = LegacyClient` alias.

## Tasks Completed

| # | Name | Commit | Key Files |
|---|------|--------|-----------|
| 1 | Rename client.go → legacy.go; LegacyClient + Client alias + Deprecated godoc | 7abbeb9 | llm/legacy.go (git mv from client.go) |
| 2 | Create ChatModel + capability interfaces + ProviderInfo + StreamEvent + sentinel errors | 33e0d06 | llm/chatmodel.go, capabilities.go, info.go, stream.go, errors.go |
| 3 | Create llm/types.go (shared types + FinishReason alias) | 560aab9 | llm/types.go |

## Verification Results

- `go vet ./...` — PASS
- `go build ./...` — PASS (all existing agent paradigms compile via Client=LegacyClient alias)
- `cd examples && go vet ./... && go build ./...` — PASS (examples/ module compiles unchanged)
- `go test ./...` — PASS (15 packages, all green)
- `grep -c '^require' go.mod` — 0 (stdlib-only invariant intact)
- `grep -c '^// Deprecated:' llm/legacy.go` — 6 (>= 5 required; covers LegacyClient, Client alias, GenerateRequest, GenerateResponse, StreamChunk, StreamUsage)

## File Inventory

| File | LOC | Provides |
|------|-----|---------|
| llm/legacy.go | 85 | LegacyClient interface + Client alias + GenerateRequest/Response/StreamChunk/StreamUsage (all Deprecated) + legacyFinishReason private type + FinishReason constants |
| llm/chatmodel.go | 21 | ChatModel interface (Generate + Stream + Info) |
| llm/capabilities.go | 42 | ToolCaller (ChatModel + WithTools immutable), Embedder, StructuredOutputs (ChatModel + WithSchema immutable) |
| llm/info.go | 30 | ProviderInfo struct + Capabilities struct (D-02: 4 bool fields, snake_case JSON tags) |
| llm/stream.go | 145 | StreamReader interface, StreamEventKind enum (6 variants), StreamEvent typed union, ToolCallDelta (Index+ID+Name+ArgsDelta), AccumulateStream helper |
| llm/errors.go | 21 | ErrCapabilityNotSupported, ErrScriptExhausted sentinel errors |
| llm/types.go | 105 | Request, Response, Message, Tool, ToolCall (ID added), Vector, Usage, UsageSource (+3 consts), FinishReason alias |
| **Total** | **449** | (467 inc. doc.go which was not modified) |

## go.mod Confirmation

`go.mod` was NOT modified. Content after this plan:
```
module github.com/costa92/llm-agent

go 1.26.0
```
No `require` block. Stdlib-only invariant intact.

## Deprecation Coverage

All 6 `// Deprecated:` comments in `llm/legacy.go` use the exact format:
```
// Deprecated: Use llm.X instead. Y will be removed in v0.4.0. See docs/migration-v0.2-to-v0.3.md.
```
Covered symbols: LegacyClient, Client (alias), GenerateRequest, GenerateResponse, StreamChunk, StreamUsage.

## Deviations from Plan

None — plan executed exactly as written.

The one minor implementation note: `StreamEventKind` uses `uint8 + iota` (as specified in the plan's RESEARCH.md §"Concrete Go Type Definitions") rather than the existing repo preference for string-typed enums (noted in PATTERNS.md). The plan explicitly specifies `uint8` for the hot-path performance argument (Phase 2 streaming adapters will dispatch on Kind in tight loops). This is consistent with the plan specification.

## Known Stubs

None. This plan is purely contract/interface — no data-flow implementation that could produce stub rendering issues.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries were introduced. All 7 files are pure type declarations in the `llm` package. The alias chain `Client = LegacyClient` is the only new linkage that touches existing callers — verified by `go build ./...`.

## What Comes Next (Plan 00-01b)

- ScriptedLLM v2 (full-capability mock — `llm/scripted.go`)
- ChatOnlyMock (`llm/chat_only_mock.go`)
- Compile-time interface satisfaction assertions (`var _ ChatModel = (*ScriptedLLM)(nil)`)
- `llm/doc.go` replacement with capability-negotiation guide
- `llm/llm_test.go` with ProviderInfo JSON round-trip + sentinel error wrapping tests

## Self-Check: PASSED

Files verified:
- FOUND: llm/legacy.go
- FOUND: llm/chatmodel.go
- FOUND: llm/capabilities.go
- FOUND: llm/info.go
- FOUND: llm/stream.go
- FOUND: llm/errors.go
- FOUND: llm/types.go

Commits verified:
- FOUND: 7abbeb9 (Task 1 — legacy.go rename)
- FOUND: 33e0d06 (Task 2 — capability interfaces)
- FOUND: 560aab9 (Task 3 — types.go)
