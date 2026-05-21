---
phase: 19-agentic-retrieval-decomposition-and-self-correction
plan: 02
type: execute
status: complete
completed: 2026-05-17
repo: llm-agent-rag
requirements: [RAG-AGENT-02]
---

> Archived planning record only.
> Do not use as current development guide.
> Current development follows live code and `llm-agent-rag` current docs.


# Summary: 19-02 CorrectiveAsker (self-correcting retrieval loop)

## Objective

Deliver RAG-AGENT-02 — a self-correcting retrieval loop. A new `agentic`
package's `CorrectiveAsker` runs the retrieve+generate pipeline, judges the
answer's groundedness with the Phase 16 `eval.Judge` signal, and — when
grounding is low — reformulates the query and retries under a bounded cap.

## Delivered

- `agentic` package (new — imports `eval`, `rag`, `generate`; linear, no
  cycle):
  - `QueryReformulator` interface; `LLMReformulator{Model}` — prompts a
    `generate.Model` for a better retrieval query; nil model or empty
    response → the original question unchanged.
  - `Attempt{Query, Groundedness, AnswerRelevance}`,
    `Result{Answer, Attempts, Corrected}`.
  - `CorrectiveAsker{Asker, Judge, Reformulator, MinGrounding, MaxRetries}`.
  - `AskWithCorrection` — the loop: call `Asker.Ask`, judge groundedness
    (always against the **original** question), and if grounding is below
    `MinGrounding` (default 0.5) reformulate and retry up to `MaxRetries`
    (default 2). At most `MaxRetries+1` `Asker.Ask` calls; returns the best
    attempt by groundedness. Nil `Asker`/`Judge`/`Reformulator` → the
    matching `Err*Required`.
  - `Ask` — wraps `AskWithCorrection`, returns just the answer; satisfies
    `eval.Asker` (compile-time `var _ eval.Asker = CorrectiveAsker{}`), so
    a `CorrectiveAsker` composes inside an `eval.TriadEvaluator`.

## Files

- `agentic/correct.go` — new package.
- `agentic/correct_test.go` — new: correction-succeeds, retry-cap-respected,
  no-correction-needed, nil-dependency errors, and `LLMReformulator`
  (scripted model + nil model) tests.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off go build ./...` — BUILD OK
- `GOWORK=off go vet ./...` — VET OK
- `GOWORK=off go test ./agentic ./eval -count=1` — ok
- `GOWORK=off go test ./... -count=1` — all 18 packages ok
- `git diff --stat go.mod go.sum` — empty (no new dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — ok

## Notes

- The retry cap is hard-bounded — `TestCorrectiveAskerRespectsRetryCap`
  verifies exactly `MaxRetries+1` `Asker.Ask` calls and that the best
  (not the last) attempt is returned.
- `CorrectiveAsker` lives in `agentic`, not `rag`: it needs `eval.Judge`,
  and `rag` cannot import `eval` (`eval` imports `rag`). The correction
  trace is the `agentic.Result` returned by `AskWithCorrection` — it is not
  surfaced on `rag.Diagnostics` (which `agentic` cannot reach into).
- All three collaborators are abstract seams — tested with deterministic
  stubs; `LLMReformulator` with a scripted model. The project's mock
  discipline.
- No new module dependency — `generate.Model`/`eval` seams + stdlib only.

## Phase 19 status

Both slices complete. RAG-AGENT-01 (19-01 `MultiHopRetriever` + query
decomposition) and RAG-AGENT-02 (19-02 `CorrectiveAsker` self-correcting
loop) are delivered. `llm-agent-rag` gained no new dependency. Phase 19 is
the final v0.6 phase — all six phases (14-19) are now complete.
