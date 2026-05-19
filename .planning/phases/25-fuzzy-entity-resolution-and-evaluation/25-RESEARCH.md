# Phase 25 Research: Fuzzy entity resolution and evaluation

**Researched:** 2026-05-19
**Phase:** 25 — fuzzy entity resolution & evaluation (final v0.8 phase)
**Requirements:** RAG-GRAPH3-05, RAG-GRAPH3-06
**Repos:** `llm-agent-rag`
**Upstream:** `.planning/research/v0.8-graphrag-tier3-SUMMARY.md` §6 (fuzzy
resolution) and §7 (eval); `24-RESEARCH.md`; Phase 23-24 code at HEAD.

## Current state (codebase scan, `/tmp/llm-agent-rag` after Phase 24)

- `graph.Canonicalize(entities, relations) Graph` — exact-match
  `(NormalizeName, type)` merge; assigns stable IDs `type:normname`;
  resolves relation endpoints (names) to entity IDs, **dropping a relation
  whose endpoint matches no entity**. Its doc comment already says "fuzzy
  resolution is a v0.8 item".
- `graph` is a leaf package — currently stdlib + `generate` seam.
- `embed.Embedder` — `Embed(ctx, text) (Vector, error)`, `Dimension()`;
  `embed.CosineSimilarity(a, b Vector) float64` in `embed/hash.go`. `embed`
  is itself a stdlib-only leaf package — `graph` importing it adds no cycle.
- `rag.Import` (`rag/import.go`) — accumulates `graphEnts`/`graphRels`, then
  `graph.Canonicalize(graphEnts, graphRels)`.
- `eval` — `Dataset`/`Example`, `Evaluator` (retrieval `Metrics`),
  `TriadEvaluator{Asker, Judge, Options}` over an `Asker`
  (`Ask(ctx, q, rag.AskOptions)`), `LLMJudge` →
  `Judgement{Groundedness, AnswerRelevance}`. `RunGraphAB` is the v0.7
  graph-recall A/B.
- `rag.System.AskGlobal(ctx, q, GlobalOptions) (Answer, error)` (Phase 24);
  `Diagnostics.Global` carries `CommunityIDs`, `MapScores`, call counts.

## Decision 1 — `EntityResolver` seam rewrites entities AND relations

v0.7's `Canonicalize` exact-merges `(name, type)`; "Acme" and "Acme Corp"
stay two nodes. The fuzzy resolver is a **pre-pass before `Canonicalize`**
(KG3-8) — `Canonicalize` and its tests stay untouched.

The research §6 sketched `Resolve(ctx, []Entity) ([]Entity, error)`, but
that is insufficient: relations carry endpoint **names**, and `Canonicalize`
drops a relation whose endpoint name matches no entity. If the resolver
rewrites entity "Acme" → "Acme Corp" without rewriting a relation that still
says "Acme", that relation is silently dropped. **The seam must rewrite both
consistently:**

```go
package graph

// EntityResolver rewrites near-duplicate entity names to a shared canonical
// surface form before Canonicalize runs. It rewrites relation endpoints to
// match so no relation is orphaned.
type EntityResolver interface {
    Resolve(ctx context.Context, entities []Entity, relations []Relation) ([]Entity, []Relation, error)
}
// NoopEntityResolver{} — the default; returns its input unchanged (v0.7 behavior)
// EmbeddingEntityResolver{Embedder embed.Embedder; Threshold float64}
```

## Decision 2 — `EmbeddingEntityResolver` is deterministic and conservative

- Embed each entity's `Name` (optionally `Name + ": " + Description`) via
  the `embed.Embedder`.
- Cluster by cosine similarity `>= Threshold`, **restricted to same `Type`**
  (never merge a person with an org).
- Within a cluster, pick the canonical surface form deterministically — the
  **longest** `Name`, tie-broken lexically (lowest) — and rewrite every
  member entity's `Name` and every matching relation endpoint to it.
- Determinism (KG3-6): sorted iteration over entities, a fixed clustering
  order (single-link by sorted pair scan), no randomness. Unit-tested
  against a **scripted embedder** returning fixed vectors.
- Conservative: same-type-only, a high default `Threshold`, opt-in.
  False-positive risk ("Apple" the company vs the fruit) is documented.

`graph` gains an `embed` import — its package doc comment ("stdlib + the
generate seam") is updated to name the `embed` seam too.

## Decision 3 — `Import` wiring, `NoopEntityResolver` default

`rag.Options.EntityResolver` carried as `System.entityResolver`; default
`NoopEntityResolver{}`. In `rag.Import`, before
`graph.Canonicalize(graphEnts, graphRels)`:
`graphEnts, graphRels, err = s.entityResolver.Resolve(ctx, graphEnts, graphRels)`.
With the `Noop` default the import is byte-identical to v0.7/Phase-24
behavior — opt-in discipline, mirroring v0.7's KG-4.

## Decision 4 — global-search eval is the Triad/Judge path, not recall@k

`RunGraphAB` measures chunk recall@k — meaningless for global search, which
synthesizes an answer with no gold chunk set (research §7). v0.8 adds a
**separate** global-search harness in `eval`:

```go
package eval

type GlobalAsker interface {
    AskGlobal(ctx context.Context, question string, opts rag.GlobalOptions) (rag.Answer, error)
}
// GlobalEvaluator{GlobalAsker, Judge, Options} — runs whole-corpus
// questions through AskGlobal, scores each answer with the Judge.
```

The judge's grounding context is the **community reports the answer
consulted**. To make that available to the evaluator without store
plumbing, `Diagnostics.Global` gains a `ConsultedReports
[]graph.CommunityReport` field, populated by `AskGlobal`. `GlobalEvaluator`
reads it from the `Answer` and passes the reports' summaries as the judge
context — global-search groundedness becomes "is the answer grounded in the
community reports it read", answer-relevance is question-vs-answer.
`GlobalEvalResult` carries mean groundedness / answer-relevance + per
example. A CI gate test uses a scripted model + scripted judge.

## Slice breakdown

- **25-01** — `graph.EntityResolver` seam + `NoopEntityResolver` +
  `EmbeddingEntityResolver` (cosine clustering over `embed.Embedder`,
  same-type-only, deterministic); wired as an opt-in pre-pass before
  `Canonicalize` in `Import`; scripted-embedder tests. (RAG-GRAPH3-05)
- **25-02** — `eval` global-search harness: `GlobalAsker` +
  `GlobalEvaluator` + `GlobalEvalResult` over the Triad/`LLMJudge` path;
  `Diagnostics.Global.ConsultedReports`; a scripted-model CI gate +
  deterministic eval example. (RAG-GRAPH3-06)
- **25-03** — `docs/graphrag.md` updated for Tier-3: communities, global
  search, the lazy-vs-eager tradeoff, the fuzzy-resolution false-positive
  caveat, and the v0.9 deferral list (DRIFT, incremental community
  maintenance, path-ranking). (RAG-GRAPH3-06)

## Risks / notes

- A relation rewritten so both endpoints collapse to the *same* entity (a
  self-loop) — `Canonicalize` already merges relations by
  `(Source, Relation, Target)`; a self-loop is harmless but worth a test.
- `EmbeddingEntityResolver` false positives are real — ship conservative
  (high threshold), opt-in, documented (25-03).
- 25-02 depends on 25-01? No — independent; but keep sequential for a clean
  green chain. 25-03 is docs, depends on 25-01+02 being done.
- No new module dependency — `embed` is already in the module; the eval
  harness reuses `LLMJudge`.
