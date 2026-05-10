# Phase 0 Validation

**Phase:** 00-keystone-interfaces
**Generated:** 2026-05-10
**Source:** lifted from 00-RESEARCH.md §"Validation Architecture"

> This document satisfies the `workflow.nyquist_validation: true` gate
> (config.json). It records test framework + per-requirement test mapping
> + sampling rates + Wave 0 gaps for Phase 0. Plans 00-01a, 00-01b, 00-02,
> 00-03, 00-04, 00-05 implement the gates listed below.

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.26) |
| Config file | none — `go test ./...` is the canonical invocation |
| Quick run command (this repo, this package only) | `go test ./llm/... -run . -count=1` |
| Quick run command (whole repo) | `go test ./... -count=1` |
| Full suite command | `go vet ./... && go build ./... && go test ./... && (cd examples && go vet ./... && go build ./...)` |

## Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CORE-01 | `ChatModel` interface compiles + has Generate/Stream/Info | type-level | `go vet ./llm/...` (compiles is enough) + `var _ llm.ChatModel = (*llm.ScriptedLLM)(nil)` in `llm/scripted.go` | ❌ Wave 0 |
| CORE-02 | `ToolCaller` defines `WithTools` returning new value (immutable) | unit | `go test ./llm/ -run TestToolCallerImmutable` | ❌ Wave 0 |
| CORE-03 | `Embedder.Embed` returns vectors + Usage + error | unit | `go test ./llm/ -run TestScriptedLLM_Embed` | ❌ Wave 0 |
| CORE-04 | `StructuredOutputs.WithSchema` returns `ChatModel` | unit | `go test ./llm/ -run TestStructuredOutputs_WithSchema` | ❌ Wave 0 |
| CORE-05 | `StreamEvent.Kind` enum has 6 variants; `ToolCallDelta.Index` int | type-level + unit | `go test ./llm/ -run TestStreamEventKindCount` (asserts iota max value) + `go test ./llm/ -run TestToolCallDelta_Index` | ❌ Wave 0 |
| CORE-06 | `ProviderInfo` has Provider/Model/Capabilities; `Capabilities` has 4 bool fields | unit | `go test ./llm/ -run TestProviderInfo_Shape` (uses reflect to assert fields) | ❌ Wave 0 |
| CORE-07 | `ScriptedLLM` satisfies all 4 capability interfaces | type-level | compile-time `var _` assertions in `llm/scripted.go` | ❌ Wave 0 |
| CORE-07 | `ChatOnlyMock` satisfies ChatModel ONLY (not the others) | runtime | `go test ./llm/ -run TestChatOnlyMockExcludesCapabilities` | ❌ Wave 0 |
| CORE-08 | `LegacyClient` is callable; `Client` alias resolves to it | type-level | compile-time `var _ llm.Client = (llm.LegacyClient)(nil)` + symmetric reverse | ❌ Wave 0 |
| CORE-08 | `// Deprecated:` godoc comment on `LegacyClient` and companion types | manual / linter | `go vet ./...` + grep check `grep -c "// Deprecated:" llm/legacy.go` ≥ 5 | partial (grep is automated) |
| CORE-09 | Migration guide exists with at least 1 worked example | manual | `test -f docs/migration-v0.2-to-v0.3.md && grep -c "## Worked example" docs/migration-v0.2-to-v0.3.md` ≥ 1 | ❌ Wave 0 |
| INFRA-01 | 4 modules exist (this repo + 3 sisters) | smoke | `gh repo view costa92/llm-agent-providers && gh repo view costa92/llm-agent-otel && gh repo view costa92/llm-agent-customer-support` (manual; CI cannot test repo creation) | n/a — manual verification |
| INFRA-02 | `go.work` is gitignored in all 4 repos | smoke | `git check-ignore go.work` returns 0 in each repo (in CI) | ❌ Wave 0 |
| INFRA-02 | CI runs with GOWORK=off | smoke | grep `GOWORK: off` in `.github/workflows/test.yml` of each repo | ❌ Wave 0 |
| INFRA-03 | `scripts/workspace.sh` writes a sibling-aware go.work | shell test | execute the script in a tmpdir with mock sibling dirs and assert `go.work` written | ❌ Wave 0 |
| INFRA-04 | release-precheck rejects `replace` directives | integration | push a `release/test` branch with a replace directive; assert workflow fails | ❌ Wave 0 (manual integration test at phase exit) |
| INFRA-05 | Umbrella CI builds all 4 repos | integration | open a PR to llm-agent that intentionally breaks a public type used by sisters; assert umbrella job fails | ❌ Wave 0 (manual integration test at phase exit) |
| INFRA-06 | Each sister-repo README documents iteration pattern | manual | grep "go.work" in each README; grep "replace" + "escape hatch" | ❌ Wave 0 |
| INFRA-07 | Versioning policy documented in all 4 repos | manual | grep version table in each README; CHANGELOG `### Breaking` template present | ❌ Wave 0 |
| Pitfall 22 | `go doc ./...` baseline snapshot captured at phase exit | smoke | `test -s docs/api-snapshot.txt && git ls-files docs/api-snapshot.txt | wc -l` returns 1 | ❌ Wave 0 (added to plan 00-05 final task) |

## Sampling Rate

- **Per task commit:** `go test ./llm/... -count=1` (sub-second).
- **Per wave merge:** `go vet ./... && go build ./... && go test ./... && (cd examples && go vet ./... && go build ./...)`.
- **Phase gate (`/gsd-verify-work`):**
  1. Full suite green in core repo.
  2. `go build ./...` green in each of 3 sister repos when cloned individually with `GOWORK=off`.
  3. Umbrella workflow run on a fresh PR shows green (4-repo build).
  4. Manual: push a `release/test` branch with a `replace` directive; verify release-precheck workflow fails. Delete the test branch.
  5. `go doc ./...` snapshot captured to `docs/api-snapshot.txt` (Pitfall 22 baseline; rolling per Q2 RESOLVED).
  6. Tag `v0.3.0-pre.1` on core; sister-repo `go.mod` `require` lines validated.

## Wave 0 Gaps

- [ ] `llm/llm_test.go` — interface satisfaction tests (CORE-01..07).
- [ ] `llm/scripted.go` — compile-time `var _ llm.ChatModel = (*ScriptedLLM)(nil)` etc.
- [ ] `llm/chat_only_mock.go` — same assertions in negative form.
- [ ] No new test framework needed; Go stdlib `testing` is the existing choice.
- [ ] `scripts/test-release-precheck.sh` (optional) — local script to verify release-precheck workflow against a tmp branch.
- [ ] `docs/api-snapshot.txt` — Pitfall 22 baseline; created by plan 00-05's final task at phase exit.

*(All listed items are net-new in Phase 0; existing test infrastructure covers everything once those files land.)*

## Plan → Validation Map

| Plan | Tests Created | Phase Gate Step |
|------|--------------|----------------|
| 00-01a | `llm/legacy.go` deprecation grep + var-_ alias asserts (compile-time) | full suite green (core) |
| 00-01b | `llm/llm_test.go` 8 tests + `scriptedllm_test.go` shim | full suite green (core) |
| 00-02 | (markdown only — grep checks per task) | n/a (docs) |
| 00-03 | sister-repo gh-API smoke + first-CI-fired check | sister repos exist + first CI fired |
| 00-04 | core .gitignore git check-ignore + workspace.sh `bash -n` + GOWORK env grep | full suite green (core) |
| 00-05 | umbrella.yml + release-precheck.yml YAML lint + smoke-test PR + `release/test-replace-ban` | umbrella green on PR; release-precheck red on replace; api-snapshot.txt captured |

---
*Validation generated: 2026-05-10 by gsd-planner (revision iteration 1)*
*Source of truth: `00-RESEARCH.md` §"Validation Architecture"*
