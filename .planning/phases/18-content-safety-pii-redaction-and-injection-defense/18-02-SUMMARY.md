---
phase: 18-content-safety-pii-redaction-and-injection-defense
plan: 02
type: execute
status: complete
completed: 2026-05-17
repo: llm-agent-rag
requirements: [RAG-SEC-02]
---

# Summary: 18-02 injection scanner + sanitize

## Objective

Deliver RAG-SEC-02 — a `guard` injection scanner and a fail-safe sanitize
step wired into `Ask` so suspicious retrieved content is neutralized or
dropped before prompt assembly.

## Delivered

- `guard/inject.go` (new):
  - `InjectionVerdict{Suspicious, Patterns}`, `InjectionScanner` interface.
  - `InjectionPattern{Name, Pattern}`; `PatternScanner{Patterns}` with an
    exported (caller-configurable) `Patterns` slice; `Scan` reports every
    matched pattern name.
  - `NewPatternScanner()` — built-in case-insensitive patterns:
    `instruction_override`, `disregard_above`, `role_override`,
    `prompt_exfiltration`.
  - `SanitizeMode` (`Neutralize` = zero value, `Drop`).
  - `NeutralizeText(text)` — wraps content in explicit untrusted-data
    markers so a model treats it strictly as data.
- `rag.Options.InjectionScanner` + `rag.Options.SanitizeMode`;
  `System.injectionScanner`/`sanitizeMode` set in `New` (nil scanner = off).
- `rag.InjectionFinding{ChunkID, Patterns, Action}` and
  `Diagnostics.InjectionFindings`.
- `rag/inject.go` `sanitizeHits` — scans each packed hit; a suspicious hit
  is dropped or has its content replaced with `guard.NeutralizeText(...)`
  per `SanitizeMode`. `Ask` runs it after packing and before `tpl.Render`;
  non-suspicious hits and the nil-scanner path are unaffected.

## Files

- `guard/inject.go`, `guard/inject_test.go` — new.
- `rag/options.go` — `Options.InjectionScanner` + `SanitizeMode`.
- `rag/system.go` — `System` fields + `New` wiring; `Diagnostics.InjectionFindings`.
- `rag/inject.go` — new: `InjectionFinding` + `sanitizeHits`.
- `rag/ask.go` — sanitize step before `tpl.Render`; `InjectionFindings`
  on `Diagnostics`.
- `rag/inject_test.go` — new: neutralize, drop, and no-scanner tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./guard ./rag -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 17 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Deviation

The plan named the string wrapper `Neutralize` — but `SanitizeMode` already
has a `Neutralize` constant, so the two collided in package `guard` (caught
at build). The function was renamed **`NeutralizeText`**; the `SanitizeMode`
constant keeps the name `Neutralize`. Behavior is unchanged.

## Notes

- Injection defense is opt-in: a nil `Options.InjectionScanner` leaves
  `Ask` behaving exactly as before. `guard.NewPatternScanner()` is the
  ready default; `SanitizeMode` zero value is `Neutralize`.
- Both modes are fail-safe: neutralized content is wrapped as inert data so
  it cannot act as a live instruction; `Drop` removes it entirely. Default
  `Neutralize` preserves retrieval recall.
- `Answer.Hits` reflects what actually reached the prompt — neutralized
  content for flagged-and-kept chunks, dropped chunks absent.
- Detection is regex-based and best-effort (known patterns, not novel/
  obfuscated attacks) — the `Patterns` slice is the extension point. A
  model-based classifier is a later milestone.
- No new module dependency — `regexp`/`strings` are stdlib.

## Phase 18 status

Both slices complete. RAG-SEC-01 (18-01 `guard` package + PII redaction at
ingest) and RAG-SEC-02 (18-02 injection scanner + sanitize before prompt
assembly) are delivered. `llm-agent-rag` gained no new dependency.
