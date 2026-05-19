---
phase: 29-documentation-completeness-and-compatibility-policy
plan: 02
type: execute
status: complete
completed: 2026-05-19
repo: llm-agent-rag
requirements: [RAG-API-03]
---

# Summary: 29-02 — exported-symbol doc sweep + repo-wide gofmt freeze hygiene

## Objective

Sweep every `llm-agent-rag` package and fill every exported-symbol
documentation gap, so every exported type, function, method, `Err*` value,
and exported const carries a name-prefixed doc comment (Go convention).
Plus the folded-in freeze-hygiene task: run `gofmt -w .` across the whole
repo so it is `gofmt`-clean at the v1.0 freeze. Comment-only and
whitespace-only — the exported surface was frozen by Phase 28 (KS-2);
this slice documents it, it does not change it. RAG-API-03.

## Delivered

### 1. Exported-symbol doc sweep

Every package in `docs/api-audit-v1.0.md`'s inventory was walked with
`go doc ./<pkg>` and each exported symbol expanded. **135 top-level
exported declarations** (types, funcs, methods, `Err*` values, the
`ingest` metadata-key const block) and **~440 exported struct fields and
interface methods** that previously rendered bare now carry a
name-prefixed doc comment. Field comments match the `rag/options.go`
quality bar (the `GlobalOptions`/`DriftOptions` exemplars).

Doc comments added, package by package:

- **advanced** — `ErrModelRequired`.
- **embed** — `Vector`, `Embedder` (+methods), `HashEmbedder` (+`Dim`,
  `Dimension`, `Embed`), `NewHashEmbedder`, `CosineSimilarity`.
- **generate** — `Message`, `Request`, `Response`, `Usage` (all fields),
  `Model` (+`Generate`).
- **prompt** — `Template` (+`Render`), `RenderContext` (fields),
  `DefaultQATemplate` (+fields, `Render`).
- **obs** — `StageTiming`, `CallCounts`, `TokenUsage`, `Metrics` fields.
- **store** — `Filter`, `Query` (fields), `Store` (+7 methods),
  `LexicalSearcher.LexicalSearch`, `ErrNotFound`, `ErrDimensionMismatch`,
  `StoredChunk`/`Hit`/`Stats` (fields), `InMemoryStore` (+`NewInMemoryStore`
  and its 8 exported methods).
- **pack** — `TokenCounter` (+`Count`), `SimpleCounter` (+`Count`),
  `Request`/`Trace`/`Result` (fields), `Packer` (+`Pack`),
  `GreedyTokenPacker` (+`Counter`, `Pack`).
- **tree** — `Node`/`DocumentTree` (fields), `Build`, `BuildStored`,
  `Find`, `Sections`, `Leaves`.
- **ingest** — the 7-const `Metadata*Key` block (block comment + each
  const name-prefixed), `Splitter` (+`Split`), `CharSplitter`/
  `MarkdownSplitter` (+fields, constructors, `Split`), `Source`/
  `SourceFunc`/`StreamingSource` (+methods), `StaticSource`, `Collect`,
  `ErrNilSource`, `ErrNilSplitter`, `Document`/`Chunk`/`ImportResult`
  (fields), `ImportOptions`/`Importer` (+fields), `NewImporter`,
  `Importer.Import`, `ImportFrom`.
- **guard** — `Redaction`/`RedactResult`/`Rule`/`PIIRedactor` fields,
  `Redactor.Redact`, `InjectionVerdict`/`InjectionPattern`/
  `PatternScanner` fields, `InjectionScanner.Scan`.
- **rerank** — `Request`/`Trace`/`RerankScore` (fields), `Reranker`
  (+`Rerank`), `NoopReranker`/`HeuristicReranker` (+`Rerank`),
  `ScoringModel.Score`, `ModelReranker` (+`Model` field, `Rerank`).
- **retrieve** — `Request` (21 fields), `PreprocessResult`,
  `RouteCandidate`, `RoutePolicyTrace`, `TrajectoryStep`,
  `SectionPlannerDecision`, `Trace`, `FusionAttribution` (all fields);
  `SectionPlanner`/`QueryPreprocessor`/`QueryEmbedder`/`Retriever` (+
  methods); `GapAwareSectionPlanner`/`NoopPreprocessor`/
  `LLMExpansionPreprocessor`/`VariantRetriever`/`DenseRetriever`/
  `LexicalRetriever`/`StructureRetriever`/`HybridRetriever` (+fields,
  `Retrieve`/`Plan`/`Process`); `BM25Params`; `ErrBaseRetrieverRequired`;
  `QueryDecomposer.Decompose`, `EntityLinker.Link`, `PathRanker.RankPaths`;
  `HopAttribution`/`LLMDecomposer`/`MultiHopRetriever`/`GraphTrace`
  fields.
- **graph** — `Entity`/`Relation`/`Graph`/`Subgraph`/`Community`/
  `CommunityReport`/`RankedPath` (all fields), `EntityExtractor.Extract`,
  `CommunityDetector.Detect`, `CommunitySummarizer.Summarize`,
  `EntityResolver.Resolve`, `PathRanker.RankPaths`,
  `LLMCommunitySummarizer.Model`, `LLMEntityExtractor.Model`.
- **eval** — `Example`/`Dataset`/`Metrics`/`ExampleResult`/
  `RetrievalResult`/`RetrievalEvaluator` fields, `Retriever.Retrieve`;
  `Asker.Ask`, `GenerationMetrics`/`TriadExampleResult`/`TriadResult`/
  `TriadEvaluator` fields; `JudgeRequest`/`Judgement`/`LLMJudge` fields,
  `Judge.Judge`; `DriftAsker.AskDrift`, `DriftExampleResult`/
  `DriftEvalResult` fields; `GlobalAsker.AskGlobal`,
  `GlobalExampleResult`/`GlobalEvalResult` fields; `GraphABResult` fields.
- **agentic** — `ErrAskerRequired`/`ErrJudgeRequired`/
  `ErrReformulatorRequired` (per-var, inside the existing group),
  `QueryReformulator.Reformulate`, `LLMReformulator.Model`,
  `Attempt`/`Result`/`CorrectiveAsker` fields.
- **rag** — `Answer`/`Citation`/`Diagnostics`/`Trace` (all fields),
  `System` type, `New`, `Remove`, `Stats`, `Model`, `Ask`, `Import`,
  `ImportFrom`, `Retrieve`; the 5 bare `Err*` sentinels in `errors.go`;
  `ImportTrace`/`Observer` fields, `InjectionFinding` fields;
  `SearchOptions`/`AskOptions`/`Options` (types + all bare fields).
- **adapter/llmagent** (build-tagged) — `ModelAdapter` (+`Inner` field,
  `Generate`), `AsTool`.

Symbols already carrying name-prefixed comments before this slice
(`postgres.*`, `eval.RetrievalEvaluator`/`WriteJSONL`/`LoadJSONL`,
`graph.CommunityContentHash`/`NormalizeName`/`Canonicalize`,
`guard.NeutralizeText`/`NewPIIRedactor`/`NewPatternScanner`,
`obs.*` funcs, `rerank.HTTPScoringModel`, `rag.AskGlobal`/`AskDrift`/
`PrewarmCommunityReports`, `feedback.*`, the package comments from 29-01,
etc.) were left untouched.

### 2. Freeze hygiene — repo-wide `gofmt -w .`

Before this slice, `gofmt -l .` listed 10 non-compliant files. The pure
pre-existing `v0.6.0` debt (files 29-02 did **not** otherwise edit):

- `adapter/llmagent/model_test.go`, `adapter/llmagent/tool_test.go` —
  import ordering within a group (`corellm` sorted before the
  `llm-agent-rag` import).
- `postgres/community.go` — struct-field alignment in a `var (...)` block.
- `rag/community_test.go` — struct-literal field alignment.
- `rag/drift.go` — one trailing blank line at EOF.

`gofmt -d` was inspected on each before the pass to confirm every hunk is
pure whitespace (no token, no symbol, no behavior change) — confirmed.
`gofmt -w .` was then run across the whole repo. It also re-aligned the
comment columns of this slice's own doc-comment edits (expected — trailing
field comments shift the alignment column). Post-pass `gofmt -l .` is
**empty** — the repository is `gofmt`-clean at the v1.0 freeze.

## Files

58 `.go` files modified in `/tmp/llm-agent-rag` (819 insertions, 482
deletions — the deletions are gofmt re-alignment, not removed code).
Doc-edit files span every importable package plus `adapter/llmagent`;
the 5 pre-existing-debt files above changed by whitespace only.

`go.mod` / `go.sum` unchanged.

## Verification

All `<verify>` commands run with `GOWORK=off GOCACHE=/tmp/go-build`:

- **`go build ./...`** — `BUILD OK`, no errors.
- **`go vet ./...`** — `VET OK`, no errors.
- **`go test ./... -count=1`** — every package `ok`, no `FAIL`. 3 packages
  report `[no test files]` (root `ragkit`, `generate`, `store/storetest`),
  unchanged.
- **`gofmt -l . | grep -v vendor`** — **empty**. Repo is `gofmt`-clean.
- **Spot-checks** — 6 previously-bare symbols across 6 packages now render
  a name-prefixed doc comment under `go doc`:
  - `go doc ./rag System` → "System is the top-level RAG pipeline …"
  - `go doc ./retrieve Retriever` → "Retriever fetches candidate chunks …"
  - `go doc ./store Store` → "Store is the core storage seam …"
  - `go doc ./embed Embedder` → "Embedder turns text into a Vector …"
  - `go doc ./generate Model` → "Model generates text from a Request …"
  - `go doc ./pack TokenCounter` → "TokenCounter estimates the token cost …"
- **`go test ./contract/... -count=1`** — `ok` — the contract gate passes,
  confirming no exported symbol was renamed, re-signed, added, or removed.
- **`git diff --stat go.mod go.sum`** — empty. No new module dependency.
- **Core-facade smoke** (from core repo `llm-agent`): `GOWORK=off go vet
  ./rag/...` → `VET OK`; `GOWORK=off go test ./rag/...` → `ok
  github.com/costa92/llm-agent/rag`. The core repo's `rag/` facade still
  compiles and passes against the documented `llm-agent-rag` surface.

A repeat run of the doc-gap scan (every non-test `.go` file, every
exported top-level decl, every exported struct field / interface method)
reports **0 remaining gaps on the exported surface**. The only items the
scanner still flags are exported fields/methods on **unexported** types
(`rag.globalPartial`, `llmagent.ragToolArgs`, `postgres.scanRow`,
`eval.wireJudgement`, `rerank.httpRerank*`, and the `rag.countingEmbedder`/
`countingModel` instrumentation decorators) — these are not part of the
v1.0 exported API and are correctly out of scope per the audit
("every exported symbol of every importable package").

## Notes / deviations

- **No deviations.** The plan executed exactly as written: a comment-only
  doc sweep plus a whitespace-only `gofmt -w .` freeze-hygiene pass.
- **No `/v2` naming notes.** Documenting the surface symbol-by-symbol
  surfaced no genuine naming or design wart that would warrant a `/v2`
  note. The one known asymmetry — `AskGlobal`/`AskDrift` vs.
  `GlobalOptions`/`DriftOptions` — was already reviewed and **ratified
  as-is** in Phase 28's `docs/api-audit-v1.0.md` ("Ratified naming
  decisions" §1); it is a deliberate, recorded decision, not a wart, and
  needs no `/v2`. The frozen surface is documented as-is.
- **No Go API symbol changed** — no rename, no signature change, no symbol
  added or removed. The `contract` gate and the core-facade smoke both
  pass, confirming the cross-repo surface is byte-stable.
- **`gofmt` pass is whitespace-only** — verified via `gofmt -d` on every
  pre-existing-debt file before the pass: import-group ordering,
  struct-field/var-block alignment, one trailing blank line. No token
  changed. The pass also re-flowed this slice's own trailing field
  comments to the gofmt alignment column.
- **No new module dependency** — `git diff --stat go.mod go.sum` empty.
- All `go` commands ran with `GOWORK=off GOCACHE=/tmp/go-build` per the
  verify block.
- **No git write commands were run.** The 58 modified `.go` files are left
  uncommitted in the working tree for the operator to commit separately,
  alongside the untouched Phase-28 changes and slice 29-01's 13 package
  comments.
- Out of scope, as planned: `docs/compatibility.md` and the README status
  line (slice 29-03); package-level comments (done in 29-01).

## Self-Check: PASSED

- All 58 modified files present in the working tree (`git diff --stat`
  lists each); the doc-edit files verified to carry name-prefixed comments
  via the repeat doc-gap scan (0 exported-surface gaps) and the 6 `go doc`
  spot-checks.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1`, and
  `go test ./contract/...` all green.
- `gofmt -l .` empty — repo is `gofmt`-clean.
- Core-facade smoke (`go vet ./rag/...` + `go test ./rag/...` from the
  core repo) green.
- `git diff --stat go.mod go.sum` empty — no new dependency.
- No commits made — per operator instruction, changes left uncommitted for
  a separate commit.
