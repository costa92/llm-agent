---
phase: 18-content-safety-pii-redaction-and-injection-defense
plan: 01
type: execute
status: complete
completed: 2026-05-17
repo: llm-agent-rag
requirements: [RAG-SEC-01]
---

# Summary: 18-01 guard package + PII redaction

## Objective

Deliver RAG-SEC-01 — a new `guard` package with a `Redactor` seam and a
configurable rule-based `PIIRedactor`, wired into `Import` so PII is
redacted before chunking and embedding.

## Delivered

- `guard` package (new, leaf — imports only `regexp`):
  - `Redaction{Kind, Count}`, `RedactResult{Text, Redactions}`,
    `Redactor` interface.
  - `Rule{Kind, Pattern, Placeholder}`; `PIIRedactor{Rules []Rule}` with an
    exported (caller-configurable) `Rules` slice; `Redact` applies each
    rule in order, replacing matches and tallying per-kind counts.
  - `NewPIIRedactor()` — built-in rules for SSN, credit-card, phone, IPv4,
    and email, each collapsing to a `[REDACTED:<KIND>]` placeholder. Rules
    are ordered specific-before-broad (SSN before credit-card/phone) so a
    broad numeric rule cannot consume a specific match first.
- `rag.Options.Redactor guard.Redactor`; `System.redactor` set in `New`
  (nil = off — opt-in, backward-compatible).
- `Import` redacts `doc.Content` before `splitter.Split`, so chunks,
  vectors, and the store never see raw PII; per-kind counts are accumulated
  across documents and surfaced (kind-sorted, deterministic) as
  `Redactions` on `ingest.ImportResult` and `rag.ImportTrace`.

## Files

- `guard/redact.go`, `guard/redact_test.go` — new package + tests.
- `rag/options.go` — `guard` import; `Options.Redactor`.
- `rag/system.go` — `guard` import; `System.redactor`; wired in `New`.
- `rag/import.go` — `guard`/`sort` imports; redact-before-split;
  `redactionSummary` helper; `Redactions` on result + trace.
- `rag/observer.go` — `ImportTrace.Redactions`.
- `ingest/types.go` — `guard` import; `ImportResult.Redactions`.
- `rag/redact_test.go` — new: import-redacts-PII and no-redactor tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./guard ./rag ./ingest -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 17 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Notes

- Redaction is opt-in: a nil `Options.Redactor` leaves content verbatim, so
  existing callers and tests are unaffected. `guard.NewPIIRedactor()` is
  the ready default.
- Scope is `doc.Content` (the requirement's "ingested content"). `Title`/
  metadata redaction is a noted extension, not in v0.6.
- Detection is regex-based and best-effort — it catches the built-in
  entity patterns; the `Rules` slice is the extension point.
- No new module dependency — `regexp` is stdlib.
