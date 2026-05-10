---
phase: 00-keystone-interfaces
plan: 02
subsystem: docs
tags:
  - migration
  - deprecation
  - changelog
  - documentation
dependency_graph:
  requires:
    - plan 00-01a (all 6 deprecated symbols in llm/legacy.go)
    - plan 00-01b (ScriptedLLM v2 — used in worked example)
  provides:
    - docs/migration-v0.2-to-v0.3.md (CORE-09: migration guide with Simple paradigm worked example)
    - DEPRECATIONS.md (Pitfall-15 enforcement: single source of truth, symbol → removal version)
    - CHANGELOG.md [Unreleased] section (INFRA-07: versioning policy across 4 repos)
  affects:
    - future callers planning v0.2→v0.3 migration
    - Phase 7 auditors who grep DEPRECATIONS.md before v0.4 cut
    - sister-repo authors who need the version-track mapping
tech_stack:
  added: []
  patterns:
    - Keep a Changelog format (## [Unreleased] / ### Added / ### Deprecated)
    - 4-column DEPRECATIONS.md table (Symbol | Deprecated In | Removed In | Migration)
    - relative cross-links: docs/ → ../DEPRECATIONS.md; DEPRECATIONS.md → docs/migration-*
key_files:
  created:
    - docs/migration-v0.2-to-v0.3.md  # 207 LOC — Quick reference table + Simple paradigm worked example (3 variants) + capability detection + streaming + when-to-migrate + unchanged types
    - DEPRECATIONS.md                  # 56 LOC  — 4-column table of all 7 deprecated symbols + removal procedure + adding-new-deprecations howto
  modified:
    - CHANGELOG.md                     # +83 LOC — [Unreleased] block with ### Added (16 new symbols) + ### Deprecated (7 symbols) + ### Versioning policy (INFRA-07)
decisions:
  - "Migration guide scoped to Simple paradigm only per CONTEXT.md Claude's Discretion; other paradigms covered by Quick reference table"
  - "agents.scriptedLLM shim listed in DEPRECATIONS.md with Phase 3 target (not v0.4.0) — different lifecycle from the public llm.* symbols"
  - "Worked example uses 3 variants (v0.2 / v0.3 transitional / v0.3 idiomatic) instead of diff blocks — clearer for self-contained small examples per RESEARCH.md"
  - "CHANGELOG [Unreleased] inserted above existing v0.1.0 entry (which appears before v0.2.0 in the file) — zero deletions, only additions"
metrics:
  duration: ~20min
  completed: 2026-05-10
  tasks_completed: 3
  files_created: 2
  files_modified: 1
---

# Phase 0 Plan 02: Migration Guide + DEPRECATIONS.md + CHANGELOG [Unreleased] Summary

**One-liner:** Human-facing documentation closing the deprecation loop — `docs/migration-v0.2-to-v0.3.md` with Simple-paradigm worked example + type-rename table, `DEPRECATIONS.md` as single-source-of-truth for symbol→v0.4.0 removal, and `CHANGELOG.md` [Unreleased] ratifying INFRA-07 multi-repo versioning policy.

## Tasks Completed

| # | Name | Commit | Key Files |
|---|------|--------|-----------|
| 1 | Create docs/migration-v0.2-to-v0.3.md (worked Simple example + generic type-rename mapping table) | 5196ae2 | docs/migration-v0.2-to-v0.3.md (NEW, 207 LOC) |
| 2 | Create DEPRECATIONS.md at repo root (single source of truth for symbol → removal version) | 928db7c | DEPRECATIONS.md (NEW, 56 LOC) |
| 3 | Add ## [Unreleased] section to CHANGELOG.md (Added + Deprecated + Versioning subsections; INFRA-07) | efd417d | CHANGELOG.md (MODIFIED, +83 LOC) |

## Verification Results

- `go vet ./...` — PASS (docs-only plan; Go surface unchanged)
- `go test ./... -count=1` — PASS (15 packages, all green)
- `grep -q '## [Unreleased]' CHANGELOG.md` — PASS
- `grep -q '## [v0.2.0]' CHANGELOG.md && grep -q '## [v0.1.0]' CHANGELOG.md` — PASS (historical entries preserved)
- `git diff --stat CHANGELOG.md` — 83 insertions, 0 deletions (verified before commit)
- Cross-link validation:
  - `docs/migration-v0.2-to-v0.3.md` links to `../DEPRECATIONS.md` (relative, resolves to repo root)
  - `DEPRECATIONS.md` links to `docs/migration-v0.2-to-v0.3.md` (relative, resolves correctly)
  - `CHANGELOG.md` links to `docs/migration-v0.2-to-v0.3.md` and `DEPRECATIONS.md` (root-relative, correct)

## File Inventory

| File | LOC | Provides |
|------|-----|---------|
| docs/migration-v0.2-to-v0.3.md | 207 | Quick reference table (13 rows covering all renamed + unchanged types); Worked example Simple paradigm (3 variants: v0.2 / v0.3 transitional / v0.3 idiomatic); Capability detection forward-looking section; Streaming section; When to migrate timeline; Notes on unchanged types (Tool/Message/FinishReason/ToolCall) |
| DEPRECATIONS.md | 56 | 4-column table: 6 deprecated llm.* symbols + agents.scriptedLLM shim; Removal procedure (Phase 7 workflow); Adding new deprecations howto (exact godoc format) |
| CHANGELOG.md | 173 (+83 new) | [Unreleased]: ### Added (16 exported symbols documented), ### Deprecated (7 symbols with v0.4.0/Phase-3 targets), ### Versioning policy (INFRA-07: 4-repo table + BC policy + replace ban + GOWORK=off) |

## Cross-Link Validation

| From | To | Path | Status |
|------|----|------|--------|
| `docs/migration-v0.2-to-v0.3.md` | `DEPRECATIONS.md` | `../DEPRECATIONS.md` | Resolves (docs/ → parent = repo root) |
| `DEPRECATIONS.md` | `docs/migration-v0.2-to-v0.3.md` | `docs/migration-v0.2-to-v0.3.md` | Resolves (from repo root) |
| `CHANGELOG.md` | `docs/migration-v0.2-to-v0.3.md` | `docs/migration-v0.2-to-v0.3.md` | Resolves (from repo root) |
| `CHANGELOG.md` | `DEPRECATIONS.md` | `DEPRECATIONS.md` | Resolves (same directory) |
| `llm/legacy.go` (// Deprecated: comments) | `docs/migration-v0.2-to-v0.3.md` | text `docs/migration-v0.2-to-v0.3.md` | Already present (plan 00-01a Task 1) |

## Requirement Satisfaction

- **CORE-09 (migration guide with worked example):** Satisfied by `docs/migration-v0.2-to-v0.3.md`.
  - Simple paradigm worked example: 3 variants (v0.2 / v0.3 transitional / v0.3 idiomatic)
  - Quick reference table: 13 rows covering all 6 deprecated symbols + unchanged types
- **INFRA-07 (versioning policy across 4 repos):** Satisfied by `CHANGELOG.md ### Versioning policy`.
  - 4-repo table (llm-agent v0.3.x + 3 sister repos at v0.1.x)
  - BC policy, replace ban, GOWORK=off invariant documented

## Process Note (for future contributors)

Any `// Deprecated:` comment added to a public symbol in this repo MUST be accompanied
by a new row in `DEPRECATIONS.md`. The exact godoc format is:
```
// Deprecated: Use <replacement> instead. <Symbol> will be removed in vX.Y.Z. See docs/migration-v0.2-to-v0.3.md.
```
Vague removal targets ("TBD", "future") are forbidden — Pitfall 15 enforcement.

## Deviations from Plan

None — plan executed exactly as written.

The one minor note: the CHANGELOG.md file's existing entries were ordered `v0.1.0` then
`v0.2.0` (v0.1.0 appeared first in the file, which is atypical for Keep a Changelog order
where newest appears first). The [Unreleased] section was inserted above the existing
`v0.1.0` heading, maintaining the file's existing structure without reordering historical
entries. Verified: 83 insertions, 0 deletions.

## Known Stubs

None. All three files contain substantive content with no placeholder text.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes. All three
files are markdown documentation only. No threat flags beyond those in the plan's STRIDE
register (T-00-02-01 through T-00-02-05, all accepted or mitigated by the verify steps).

## Self-Check: PASSED

Files verified:
- FOUND: docs/migration-v0.2-to-v0.3.md
- FOUND: DEPRECATIONS.md
- FOUND: CHANGELOG.md (with ## [Unreleased] and preserved ## [v0.1.0] + ## [v0.2.0])

Commits verified:
- FOUND: 5196ae2 (Task 1 — migration guide)
- FOUND: 928db7c (Task 2 — DEPRECATIONS.md)
- FOUND: efd417d (Task 3 — CHANGELOG [Unreleased])
