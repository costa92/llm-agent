---
phase: 27-drift-search
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-GRAPH4-04]
---

# Summary: 27-02 DRIFT-search evaluation harness — Triad scoring over AskDrift

## Objective

Add a DRIFT-search evaluation harness to `eval` — run whole-corpus questions
through `rag.System.AskDrift` and score each answer with the RAG-Triad `Judge`.
DRIFT, like global search, synthesizes an answer with no gold chunk set, so the
harness measures only the two generation-side legs of the Triad — groundedness
and answer relevance — with no chunk recall@k. Completes RAG-GRAPH4-04.

## Delivered

- `eval.DriftAsker` interface — `AskDrift(ctx, question string, opts
  rag.DriftOptions) (rag.Answer, error)`. `*rag.System` satisfies it via the
  27-01 `AskDrift`. It is the DRIFT counterpart of `GlobalAsker` and `Asker` —
  a separate seam because DRIFT, like global search, has no gold chunk set and
  is scored only on its generation side.
- `eval.DriftEvalResult struct` — the DRIFT scoreboard: `MeanGroundedness`,
  `MeanAnswerRelevance`, `Examples int`, and a `PerExample []DriftExampleResult`
  detail slice. Carries NO chunk recall@k / precision@k, mirroring
  `GlobalEvalResult`. A `Summary()` method renders the compact scoreboard.
- `eval.DriftExampleResult struct` — the per-example detail: the `Example`, the
  synthesized `Answer` text, the primer's `PrimerCommunityIDs`, the local
  `Rounds` actually run, and the `Judgement`.
- `eval.DriftEvaluator struct { Asker DriftAsker; Judge Judge; MaxCommunities
  int; Rounds int }` with `Run(ctx, dataset Dataset) (DriftEvalResult, error)`:
  - errors if `Asker` or `Judge` is nil — the same nil-guard as
    `GlobalEvaluator.Run`.
  - per `Example`: calls `AskDrift` with `DriftOptions{Namespace: ex.Namespace,
    MaxCommunities: e.MaxCommunities, Rounds: e.Rounds}`; builds the judge
    context from `Answer.Diagnostics.Drift.ConsultedReports` via the existing
    `reportContext` helper (one `"Title: Summary"` passage per primer report);
    calls `Judge.Judge` for groundedness + answer relevance; accumulates means.
  - the gold chunk/doc fields of `Example` (`GoldDocIDs`, `GoldChunkIDs`) are
    unused — documented on the `DriftEvaluator` doc comment. DRIFT search has
    no chunk-recall notion; `RunGraphAB` / `Evaluator` measure that for the
    local path, `DriftEvaluator` does not.
- `eval/drift_test.go` — `TestDriftEvalGate`, the DRIFT-search CI gate
  mirroring `TestGlobalEvalGate`: a scripted `driftEvalModel` (keys replies
  off `req.SystemPrompt` to drive all four DRIFT generation kinds — community
  summarization, the primer map step, each local follow-up round, and the
  synthesis) and a scripted `driftRecordingJudge` (records every
  `JudgeRequest`, scores groundedness by answer/context word-overlap). Builds
  an in-memory store + four-triangle graph + a provenance chunk per entity +
  detected communities, runs a 3-example dataset through
  `DriftEvaluator.Run`, and asserts: every example is scored, the means land
  in `[0,1]`, `MeanAnswerRelevance == 1.0`, `MeanGroundedness > 0` (proof the
  consulted-report context reached the judge), the judge saw one
  context-bearing request per example, and each `PerExample` records non-empty
  `PrimerCommunityIDs` and `Rounds >= 1`. `TestDriftEvaluatorRequiresAskerAndJudge`
  verifies the nil-`Asker` / nil-`Judge` guard.

## Files

- `eval/drift.go` — created; `DriftAsker`, `DriftEvalResult`,
  `DriftExampleResult`, `DriftEvaluator` + `Run` + `Summary`.
- `eval/drift_test.go` — created; `TestDriftEvalGate`,
  `TestDriftEvaluatorRequiresAskerAndJudge`, the scripted model + judge.

Exactly the plan's `files_modified` list — no extra files.

## Verification

All `<verify>` commands run, all green:

- `GOWORK=off GOCACHE=/tmp/go-build go build ./...` — BUILD OK
- `GOWORK=off GOCACHE=/tmp/go-build go vet ./...` — VET OK
- `GOWORK=off GOCACHE=/tmp/go-build go test ./eval ./rag/... -count=1` —
  `ok github.com/costa92/llm-agent-rag/eval`,
  `ok github.com/costa92/llm-agent-rag/rag`
- `GOWORK=off GOCACHE=/tmp/go-build go test ./... -count=1` — all packages
  `ok`, no FAIL
- `git diff --stat go.mod go.sum` — empty (no new module dependency)
- core facade (from the core repo `llm-agent`): `GOWORK=off go vet ./rag/...
  && go test ./rag/...` — `ok github.com/costa92/llm-agent/rag`

## Deviations from plan

None. The `files_modified` list is matched exactly — `eval/drift.go` and
`eval/drift_test.go`, no extra file.

## Notes

- `DriftEvaluator` reuses `eval/global.go`'s `reportContext` helper unchanged
  to render `[]graph.CommunityReport` into the judge's context passages — the
  grounding context is the primer's `ConsultedReports`, exactly as the plan
  and `27-RESEARCH.md` Decision 4 specify, and the same way `GlobalEvaluator`
  reads `Diagnostics.Global.ConsultedReports`. No chunk-recall metric is
  computed.
- The scripted `driftEvalModel` keys off `req.SystemPrompt`. It must answer
  four prompt families because `AskDrift` drives four generation kinds: the
  community summarizer (`"You summarize one community..."`, lazy report build),
  the primer map step (`"You are answering a whole-corpus question using ONE
  community summary..."`), each local follow-up round (`"You are answering a
  question using a slice of a knowledge graph..."`), and the synthesis (`"You
  are answering a question using DRIFT search."`). The local-round reply emits
  `"Follow-up: none"` so the bounded loop terminates after exactly one round —
  the gate stays deterministic. No live calls anywhere.
- The test graph and store fixture mirror the 27-01 `rag/drift_test.go`
  fixture: four dense triangles joined by weak bridges (so `LouvainDetector`
  builds a multi-level hierarchy), with a provenance `chunk-<entityID>` per
  entity so the local loop has passages to pack. `embed.Vector` is `[]float32`,
  so the zero vector is `make([]float32, 32)`.
- No new module dependency — `eval/drift.go` imports only the already-present
  `rag` package and stdlib (`context`, `errors`, `fmt`, `strings`); the test
  imports only already-present module packages (`eval`, `generate`, `graph`,
  `rag`, `store`) and stdlib. `go.mod` / `go.sum` diff is empty.
- The `/tmp/llm-agent-rag` worktree carries uncommitted code from prior
  Phase 26-27 slices (including the 27-01 `AskDrift` implementation) — left
  untouched per the plan; only this slice's two files were added. No git
  write command was run — all changes are left uncommitted.
- Out of scope, per plan: the deterministic DRIFT worked example and the
  `docs/graphrag.md` DRIFT section — 27-03.

## Self-Check: PASSED

- `eval/drift.go` — FOUND (created: `DriftAsker`, `DriftEvalResult`,
  `DriftExampleResult`, `DriftEvaluator` + `Run` + `Summary`)
- `eval/drift_test.go` — FOUND (created: `TestDriftEvalGate`,
  `TestDriftEvaluatorRequiresAskerAndJudge`)
- All `<verify>` commands green; `go.mod` / `go.sum` diff empty.
