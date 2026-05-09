# Stack Research

**Domain:** Go LLM agent framework — provider adapters + OpenTelemetry observability + reference deployable service (sister repos to a stdlib-only core)
**Researched:** 2026-05-10
**Confidence:** HIGH on provider SDKs, OTel SDK shape, multi-repo workflow, deployment, and testing. MEDIUM on `gen_ai.*` semconv (still in **Development** as of 2026-05-10 — must be opt-in, not assumed stable).

---

## Executive Decisions (TL;DR)

1. **OpenAI adapter** — depend on `github.com/openai/openai-go/v3` (v3.35.0, 2026-05-07). Official, fully typed, supports streaming + tool calls + embeddings cleanly.
2. **Anthropic adapter** — depend on `github.com/anthropics/anthropic-sdk-go` (v1.41.0, 2026-05-06). v1+ is stable; first-class streaming + tool use; **no embeddings endpoint** (Anthropic doesn't offer one — return `ErrNotSupported` from the `Embedder` capability).
3. **Ollama adapter** — depend on `github.com/ollama/ollama/api` (v0.23.2, 2026-05-07). Official typed client used by the CLI itself. Pin minor versions; treat as `v0.x` (still pre-1.0, expect occasional API churn).
4. **OTel adapter** — `go.opentelemetry.io/otel` `v1.43.0` (stable API/SDK), exporters `otlptracegrpc/http` `v1.43.0`, metrics `v0.65.0`, logs `v0.19.0`. Default to **OTLP/HTTP (port 4318)** for the reference service (per spec `SHOULD` rule); offer gRPC as an opt-in.
5. **`gen_ai.*` semconv** — still **Development** in 2026-05-10. Use behind `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`. Pin to `semconv/v1.41.0` for non-genai attributes (service.name etc.). **Do not** advertise the otel sister repo as "stable semconv" until OTel promotes it.
6. **Multi-repo layout** — 4 separate repos, 4 separate `go.mod` files. Local dev via gitignored `go.work`. Cross-repo iteration via tagged releases (`replace` only as a doc'd escape hatch). CI never sees `go.work`.
7. **Deployment** — `docker compose` based on the **`grafana/otel-lgtm`** all-in-one image (Loki + Tempo + Prometheus + OTel Collector + Grafana in one container) for the reference service; Ollama as separate service with named volume. Helm: Ollama as `StatefulSet` with `volumeClaimTemplates`, Go service as `Deployment` (off-the-shelf chart `otwld/ollama-helm` cited as reference).
8. **Testing** — wire-format conformance via `httptest.Server` per provider + recorded fixtures committed under `testdata/`. Ollama-live nightly via `testcontainers-go/modules/ollama` (already an upstream module).

---

## Recommended Stack

### Core Technologies (per sister repo)

| Repo | Technology | Version (2026-05-10) | Purpose | Why Recommended |
|------|-----------|----------------------|---------|-----------------|
| `llm-agent-providers` | `github.com/openai/openai-go/v3` | v3.35.0 (2026-05-07) | OpenAI SDK | Official, typed, exposes Chat Completions + Responses API + Embeddings + tool calls + a stream `Accumulator` that handles delta merging. Replaces the unofficial `sashabaranov/go-openai` for new code. |
| `llm-agent-providers` | `github.com/anthropics/anthropic-sdk-go` | v1.41.0 (2026-05-06) | Anthropic SDK | Official, post-v1.0 (stable per upstream policy), `Message.Accumulate(event)` for streaming, comprehensive `BetaToolRunner`/`NewToolRunnerStreaming` helpers for tool loops. |
| `llm-agent-providers` | `github.com/ollama/ollama/api` | v0.23.2 (2026-05-07) | Ollama SDK | Official sub-module of `ollama/ollama` used by the CLI. Streams via callback (`GenerateResponseFunc`/`ChatResponseFunc`); supports tool calling and `Embed` / `Embeddings` endpoints. Acceptable substitute for hand-rolled HTTP. |
| `llm-agent-providers` | `net/http`, `encoding/json` | stdlib (Go 1.26) | Wire-format contract tests | Use stdlib `httptest.Server` to record/replay provider wire-format fixtures so PR CI never makes real LLM calls. |
| `llm-agent-otel` | `go.opentelemetry.io/otel` | v1.43.0 | OTel API/SDK | The de-facto standard. Stable API. Use `WithBatcher` + tuned `WithMaxExportBatchSize`/`WithMaxQueueSize` for production. |
| `llm-agent-otel` | `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | v1.43.0 | Trace exporter (default) | Per OTLP spec: HTTP/protobuf is the recommended default. Easier to firewall, simpler ops, port 4318. |
| `llm-agent-otel` | `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.43.0 | Trace exporter (opt-in) | gRPC for high-throughput / lower-overhead use. Port 4317. Requires `google.golang.org/grpc`. |
| `llm-agent-otel` | `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetric{http,grpc}` | v0.65.0 | Metrics exporter | Token usage, latency, cost counters. v0.x — metrics SDK API is stable, the exporter package version trails. |
| `llm-agent-otel` | `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlplog{http,grpc}` | v0.19.0 | Logs exporter | Bridge `log/slog` → OTel logs. Lower version because logs SDK promoted later. |
| `llm-agent-otel` | `go.opentelemetry.io/otel/semconv/v1.41.0` | v1.41.0 | Semantic conventions | Latest stable semconv release (2024-04-28 per upstream releases page). For non-genai attrs (service.name, service.version, http.*). |
| `llm-agent-otel` | `gen_ai.*` semconv | Development (opt-in) | LLM-specific attributes | Manually emit `gen_ai.operation.name`, `gen_ai.request.model`, `gen_ai.usage.input_tokens` etc. Behind `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental` (or your own opt-in flag) — not yet stable as of 2026-05-10. |
| `llm-agent-customer-support` | `net/http` (chi-style) or `github.com/go-chi/chi/v5` | stdlib or chi v5 | HTTP server | Service is small enough for stdlib `http.ServeMux` (Go 1.22+ patterns: `mux.HandleFunc("POST /chat", ...)`). chi only if middleware composition becomes painful. |
| `llm-agent-customer-support` | `log/slog` | stdlib (Go 1.26) | Structured logging | Bridge to OTel logs via the otel sister-repo adapter. No `zerolog`/`zap` dep. |
| `llm-agent-customer-support` | `grafana/otel-lgtm` (image) | latest | Local observability backend | Single container ships OTel Collector + Tempo + Loki + Prometheus + Grafana. One-line compose entry; demos work in 30 seconds. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/testcontainers/testcontainers-go` + `.../modules/ollama` | v0.33.x+ | Ollama-live integration tests | Nightly job in `llm-agent-providers` and `llm-agent-customer-support`. Pulls `ollama:0.x` image, primes a tiny model (`tinyllama` or `qwen3:0.6b`), runs the test, tears down. |
| `github.com/google/go-cmp/cmp` | v0.6.x+ | Wire-format diffing in contract tests | Compare recorded provider response shape vs adapter-emitted normalized form. Avoid `reflect.DeepEqual` for JSON. |
| `gopkg.in/yaml.v3` | v3.0.x | OTel Collector config + service config | Only if the reference service needs YAML config beyond env vars. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go.work` (Go 1.26) | Local cross-repo iteration | **Always gitignored.** Created on demand by a `make workspace` target that detects sibling clones. |
| `goreleaser` | Multi-repo tag/release | Optional for v0.3 — manual `git tag` is fine for the first cycle. |
| `golangci-lint` | Lint sister repos (NOT the core) | Core stays stdlib-only and has no extra tooling. Sister repos can adopt standard lint set (`govet`, `staticcheck`, `errcheck`). |
| `docker compose` v2 | Local bring-up | `compose.yaml` (not legacy `docker-compose.yml`). |
| GitHub Actions `services:` | CI containers | For Ollama-live nightly, prefer `testcontainers-go` over services block — it self-contains the lifecycle and keeps PRs and nightlies symmetric. |

---

## Installation

### Provider sister repo

```bash
# llm-agent-providers
go mod init github.com/costa92/llm-agent-providers

go get github.com/costa92/llm-agent@v0.3.0
go get github.com/openai/openai-go/v3@latest
go get github.com/anthropics/anthropic-sdk-go@latest
go get github.com/ollama/ollama/api@latest

# Test deps
go get github.com/google/go-cmp/cmp
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/ollama
```

### OTel sister repo

```bash
# llm-agent-otel
go mod init github.com/costa92/llm-agent-otel

go get github.com/costa92/llm-agent@v0.3.0
go get go.opentelemetry.io/otel@v1.43.0
go get go.opentelemetry.io/otel/sdk@v1.43.0
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@v1.43.0
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.43.0
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp@v0.65.0
go get go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp@v0.19.0
go get go.opentelemetry.io/otel/semconv/v1.41.0
```

### Reference service sister repo

```bash
# llm-agent-customer-support
go mod init github.com/costa92/llm-agent-customer-support

go get github.com/costa92/llm-agent@v0.3.0
go get github.com/costa92/llm-agent-providers@v0.1.0
go get github.com/costa92/llm-agent-otel@v0.1.0
# stdlib http.ServeMux is enough; chi is optional
```

---

## Multi-Repo Layout (concrete)

Four repos, four `go.mod` files. **Sister repos depend on tagged versions of `llm-agent` core**; `go.work` is local-only.

```
~/code/
├── llm-agent/                       # core, stdlib-only
│   └── go.mod  (module github.com/costa92/llm-agent)
├── llm-agent-providers/             # OpenAI / Anthropic / Ollama
│   └── go.mod  (require github.com/costa92/llm-agent v0.3.0)
├── llm-agent-otel/                  # OTel adapter
│   └── go.mod  (require github.com/costa92/llm-agent v0.3.0)
└── llm-agent-customer-support/      # reference service
    └── go.mod  (require
                   github.com/costa92/llm-agent          v0.3.0
                   github.com/costa92/llm-agent-providers v0.1.0
                   github.com/costa92/llm-agent-otel      v0.1.0
                 )
```

### Local development pattern

`go.work` lives in the **parent directory**, alongside the four clones, and is **gitignored everywhere**:

```
# ~/code/go.work  (gitignored — never committed to any of the 4 repos)
go 1.26.0

use (
    ./llm-agent
    ./llm-agent-providers
    ./llm-agent-otel
    ./llm-agent-customer-support
)
```

A `make workspace` (or `scripts/workspace.sh`) target in each sister repo detects siblings and writes the file into `..`:

```bash
# scripts/workspace.sh — every sister repo ships this
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
[ -f go.work ] && { echo "go.work exists"; exit 0; }
USES=()
for d in llm-agent llm-agent-providers llm-agent-otel llm-agent-customer-support; do
  [ -d "$d" ] && USES+=("./$d")
done
go work init "${USES[@]}"
```

### Versioning strategy

| Repo | Initial tag | Bump policy |
|------|-------------|-------------|
| `llm-agent` | v0.3.0 (this milestone) | `0.x` minor for BC-compatible; `0.x major` for breaking; v1.0 gated on adoption |
| `llm-agent-providers` | v0.1.0 | Bump major (`0.1 → 0.2`) when shape of provider adapter ABI changes |
| `llm-agent-otel` | v0.1.0 | Bump major when `gen_ai.*` semconv stabilizes upstream and we cut over |
| `llm-agent-customer-support` | v0.1.0 | Tags pinned to specific provider/otel combinations; this repo is allowed to break freely |

**Cross-repo dep rule:** sister repos pin `llm-agent` to an **exact tag** (not `latest`). Bumping the core requires a deliberate sister-repo PR. This is what keeps the core's BC contract honest.

### `replace` directive — only as documented escape hatch

Per existing `llm-agent` README pattern: a `replace` line in a sister repo's `go.mod` pointing at a local path is documented as the way to iterate on a not-yet-tagged core change. **Never committed to main.** Sample doc snippet for sister repo READMEs:

```
# Iterating against an unreleased llm-agent core:
go mod edit -replace=github.com/costa92/llm-agent=../llm-agent
# revert before tagging:
go mod edit -dropreplace=github.com/costa92/llm-agent
```

---

## Provider SDK Feature Parity

Verified against current SDK versions (2026-05-10):

| Capability | OpenAI v3.35.0 | Anthropic v1.41.0 | Ollama v0.23.2 |
|-----------|----------------|--------------------|-----------------|
| Generate (one-shot) | `client.Chat.Completions.New(...)` and `client.Responses.New(...)` (Responses API recommended) | `client.Messages.New(...)` | `client.Generate(...)` / `client.Chat(...)` |
| Stream | `client.Chat.Completions.NewStreaming(...)` returns iterable; `Accumulator` reassembles | `client.Messages.NewStreaming(...)` + `Message.Accumulate(event)` helper | Callback-based: pass `GenerateResponseFunc` / `ChatResponseFunc` — different shape from the others |
| Native tool calls | `openai.ChatCompletionFunctionTool(...)` + `Choices[0].Message.ToolCalls` | First-class `Tools` param + `BetaToolRunner` / `NewToolRunnerStreaming` for full loops | `Tool` / `Tools` types in `ChatRequest`; per-model support varies |
| Embeddings | `client.Embeddings.New(...)` — yes | **NO** (Anthropic does not ship an embeddings API; recommend Voyage AI or another provider — out of scope for v0.3, return `ErrNotSupported` from the `Embedder` capability) | `client.Embed(...)` (preferred, supports dim truncation) and `client.Embeddings(...)` (single-prompt legacy) |
| Wire-format stability | Stable v3 | Stable v1+ | **Pre-v1** — pin minor; expect API churn |
| Auth | `OPENAI_API_KEY` env (default) | `ANTHROPIC_API_KEY` env (default) | None for local; env `OLLAMA_HOST` |

**Implications for the `ToolCaller` + `Embedder` capability interfaces in `llm/v2`:**
- Streaming is a `<-chan` in `llm-agent` core but **callback-based in Ollama** — the Ollama adapter must own the goroutine that bridges callback → channel and lifecycle (close on context cancel).
- `Embedder` should be a **separate optional interface** (not part of `Client`); Anthropic genuinely cannot satisfy it.
- Tool argument JSON shape varies enough between providers that adapters must own normalization to/from the framework's `llm.ToolCall.Arguments json.RawMessage`.

---

## OpenTelemetry — Agent Loop Spanning

### Recommended span tree for ReAct (one Run)

```
[parent: invoke_agent {agent.name}]                       INTERNAL
  ├─ [child: chat openai/gpt-5]                           CLIENT  iteration 1 LLM call
  │     attrs: gen_ai.operation.name=chat,
  │            gen_ai.request.model, gen_ai.response.model,
  │            gen_ai.usage.input_tokens, .output_tokens,
  │            gen_ai.response.finish_reasons
  ├─ [child: execute_tool {tool.name}]                    INTERNAL  iteration 1 tool exec
  │     attrs: gen_ai.tool.name, gen_ai.tool.call.arguments
  ├─ [child: chat openai/gpt-5]                           CLIENT  iteration 2 LLM call
  └─ [child: execute_tool {tool.name}]                    INTERNAL  iteration 2 tool exec
```

### Recommended span tree for multi-agent (one orchestrate run)

```
[parent: invoke_workflow {workflow.name}]                 INTERNAL
  ├─ [invoke_agent planner]                               INTERNAL
  │     └─ chat anthropic/claude-sonnet-4-5
  ├─ [invoke_agent worker-1]                              INTERNAL
  │     ├─ chat openai/gpt-5
  │     └─ execute_tool calculator
  └─ [invoke_agent aggregator]                            INTERNAL
        └─ chat openai/gpt-5
```

### Recommended metrics

| Metric | Type | Unit | Attributes |
|--------|------|------|-----------|
| `gen_ai.client.token.usage` | histogram | `{token}` | `gen_ai.operation.name`, `gen_ai.request.model`, `gen_ai.token.type` (input/output) |
| `gen_ai.client.operation.duration` | histogram | `s` | same + `error.type` if failed |
| `agent.iterations` (custom) | histogram | `{iteration}` | `agent.paradigm` (react/plan-solve/...), `agent.name` |
| `agent.tool.invocations` (custom) | counter | `{call}` | `gen_ai.tool.name` |

**Important:** the `gen_ai.*` namespace is `Development`. Emit it behind a flag **or** publish a one-line caveat in the otel-sister-repo README that pinning `llm-agent-otel` to a major version tracks one upstream semconv generation; bumping major when upstream stabilizes.

---

## Reference Service — `docker compose` shape

`compose.yaml` for `llm-agent-customer-support` — minimal, demo-ready, ~50 lines:

```yaml
services:
  app:
    build: .
    environment:
      - LLM_PROVIDER=ollama       # or openai|anthropic
      - OLLAMA_HOST=http://ollama:11434
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://lgtm:4318
      - OTEL_SERVICE_NAME=customer-support
      - OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental
    ports:
      - "8080:8080"
    depends_on:
      - ollama
      - lgtm

  ollama:
    image: ollama/ollama:0.23
    volumes:
      - ollama-models:/root/.ollama
    ports:
      - "11434:11434"
    # `docker compose exec ollama ollama pull qwen3:0.6b` after up

  lgtm:
    image: grafana/otel-lgtm:latest
    ports:
      - "3000:3000"   # Grafana UI
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP

volumes:
  ollama-models:
```

**Why `grafana/otel-lgtm` over a hand-rolled stack:** one image, one config-free entry, full Loki+Tempo+Prometheus+Grafana with pre-wired datasources. Replacing it with a 5-service compose (`otel-collector` + `tempo` + `loki` + `prometheus` + `grafana`) is the production path, but doubles compose complexity for the demo.

### Helm / K8s shape (optional)

| Component | Workload | Volume | Service |
|-----------|----------|--------|---------|
| `customer-support` (Go) | `Deployment` (3 replicas) | none (stateless) | `ClusterIP` (or `Ingress`) |
| `ollama` | `StatefulSet` (1 replica; `volumeClaimTemplates`) | `PVC` 50–100Gi for `/root/.ollama/models` | headless + ClusterIP |
| `otel-collector` | `Deployment` (DaemonSet for node-local; `Deployment` is fine for the demo) | none | ClusterIP `:4317`/`:4318` |
| Grafana / Tempo / Prometheus / Loki | upstream charts | their own PVCs | ClusterIP |

**Reference Helm chart for Ollama:** [`otwld/ollama-helm`](https://github.com/otwld/ollama-helm) — supports GPU (NVIDIA / AMD / DRA), `volumeClaimTemplates`, model preloading at startup. **Do not** vendor it; document it as the recommended off-the-shelf chart.

---

## Testing Strategy

### Wire-format conformance tests (PR CI, mock-only)

For each provider, an `httptest.Server` returns recorded fixtures (real responses captured once and committed to `testdata/`). Adapter is pointed at the test server's URL and the test asserts adapter output matches the framework's `llm.GenerateResponse` / `StreamChunk` shape.

```go
// providers/openai/streaming_test.go (sketch)
func TestStreaming_ToolCallInterleave(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Assert request shape (model, tools, messages)
        w.Header().Set("Content-Type", "text/event-stream")
        // Stream recorded SSE chunks from testdata/openai-tool-stream.sse
        replaySSE(t, w, "testdata/openai-tool-stream.sse")
    }))
    defer ts.Close()

    adapter := openai.New(openai.WithBaseURL(ts.URL), openai.WithAPIKey("test"))
    // exercise GenerateStream, assert tool-call delta merging matches expectation
}
```

**Fixture capture script** lives in each provider package (`testdata/capture/`) and is run manually with a real key — never in CI.

### Ollama-live nightly tests

`testcontainers-go/modules/ollama` already exists upstream. Pattern:

```go
// providers/ollama/integration_test.go
//go:build integration

func TestOllama_GenerateLive(t *testing.T) {
    ctx := context.Background()
    c, err := ollama.Run(ctx, "ollama/ollama:0.23")
    require.NoError(t, err)
    t.Cleanup(func() { _ = c.Terminate(ctx) })
    host, _ := c.ConnectionString(ctx)
    // pull a tiny model: `c.Exec(ctx, []string{"ollama", "pull", "qwen3:0.6b"})`
    adapter := myollama.New(myollama.WithHost(host))
    resp, err := adapter.Generate(ctx, llm.GenerateRequest{Prompt: "hello"})
    // assert Provider == "ollama", Text != ""
}
```

GitHub Actions: nightly cron, `runs-on: ubuntu-latest` (Docker pre-installed), tests guarded by `-tags=integration` so PRs never run them.

**Cost / time guardrails:**
- `qwen3:0.6b` or `tinyllama` to keep model pull <500MB and inference <5s.
- Cache `~/.ollama/models` between runs via `actions/cache@v4` keyed on test-fixture hash.

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `github.com/openai/openai-go/v3` | `github.com/sashabaranov/go-openai` | If you need a model OpenAI hasn't shipped to v3 yet, or want minimal-deps; otherwise the official SDK is tracked daily by OpenAI engineers and gets new endpoints first. **Not recommended for new code in 2026.** |
| `github.com/ollama/ollama/api` | Hand-roll over `net/http` | If you want a tiny adapter (~150 LOC) and only need `Generate` + `Chat` + `Embed`. Hand-rolled keeps the dep graph cleaner but loses streaming-callback ergonomics. **Acceptable per project constraints; the official SDK is recommended for full feature support.** |
| OTLP/HTTP default | OTLP/gRPC default | High-throughput services (>1k req/s) where serialization overhead matters; environments where gRPC is already a dep |
| `grafana/otel-lgtm` (one container) | Separate `otel-collector` + Tempo + Loki + Prometheus + Grafana | Production deployments. The single-container image is for demo / dev only. |
| `testcontainers-go/modules/ollama` | GitHub Actions `services:` block | If your test only needs a running Ollama and not lifecycle hooks; service blocks are simpler in YAML but harder to use from `go test`. |
| `go.work` (gitignored) | `replace` directives in committed `go.mod` | Never. `replace` in committed code breaks downstream `go get`. Document `replace` only as a transient escape hatch. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `github.com/sashabaranov/go-openai` for new adapters | Unofficial, tracks behind OpenAI's spec, doesn't get realtime/Responses API features first; v3 of the official SDK now has full coverage | `github.com/openai/openai-go/v3` |
| `langchaingo` as a foundation | Heavy dep tree; overlaps significantly with this framework; cross-framework interop is explicitly Out of Scope | This framework's own `Agent` + `llm.Client` interfaces |
| `github.com/jackc/pgx` or any DB driver | Not needed for v0.3 — the customer-support reference uses RAG with the existing `InMemoryStore`. Adding a real vector DB is a separate milestone | `agents/rag.InMemoryStore` (or document a reader-supplied store) |
| `github.com/spf13/viper` for service config | Adds a heavyweight config dep for what's basically env vars + maybe one YAML | stdlib `flag` + env (`os.Getenv`); add `gopkg.in/yaml.v3` only if YAML is unavoidable |
| `github.com/sirupsen/logrus` or `github.com/uber-go/zap` | Go 1.26 has `log/slog` in stdlib; OTel has a slog bridge | `log/slog` + the OTel logs bridge in `llm-agent-otel` |
| Committing `go.work` to any repo | Breaks `go get` for downstream callers; makes CI reproduce-impossible | Gitignore it. Use a `make workspace` script to generate it locally. |
| `gen_ai.*` semconv as if stable | It's still **Development** as of 2026-05-10. Treating it as stable will require rename pain when it promotes | Emit behind `OTEL_SEMCONV_STABILITY_OPT_IN`; bump `llm-agent-otel` major when upstream stabilizes |
| OTLP default port assumption | `:4317` (gRPC) and `:4318` (HTTP) are different defaults; users will misconfigure | Always document both; default the SDK to HTTP/`:4318` per spec recommendation |
| Adding embeddings to Anthropic adapter | Anthropic doesn't ship an embeddings API | Return `ErrNotSupported` from the `Embedder` capability and document that users wanting embeddings via Anthropic should pair Anthropic-for-chat with Voyage AI / Ollama / OpenAI for embeddings |

---

## Stack Patterns by Variant

**If user has a real OpenAI / Anthropic key:**
- `LLM_PROVIDER=openai` (or `anthropic`)
- Service uses `llm-agent-providers` adapter; no Ollama service needed
- Compose can drop the `ollama` service entirely

**If user has only local hardware:**
- `LLM_PROVIDER=ollama`
- Compose includes the `ollama` service + a one-time `ollama pull` step
- For non-GPU machines: `qwen3:0.6b` or `tinyllama`; for GPU: `llama3.1:8b` or larger

**If user wants production observability:**
- Drop `grafana/otel-lgtm`, add separate `otel-collector` + `tempo` + `prometheus` + `loki` + `grafana` services with persistent volumes
- Switch exporter from `otlptracehttp` (demo) to `otlptracegrpc` (prod) via env var
- Set retention / sampling policies in collector config

**If user is on Kubernetes:**
- `otwld/ollama-helm` chart for Ollama (StatefulSet + GPU support)
- Standard Go-service Helm chart (Deployment + Service + ServiceMonitor)
- OTel Operator for collector lifecycle (out of scope to author; document the integration point)

---

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| `github.com/openai/openai-go/v3` | Go ≥ 1.22 | We ship Go 1.26; safe |
| `github.com/anthropics/anthropic-sdk-go` v1.x | Go ≥ 1.22 | Same |
| `github.com/ollama/ollama/api` v0.23.x | Go ≥ 1.22, Ollama server ≥ 0.5 | Match minor of Go client to running Ollama server within one minor |
| `go.opentelemetry.io/otel` v1.43.0 | Go ≥ 1.22 | API/SDK are stable; metric SDK at v1.43 too |
| `otlp{trace,metric,log}{http,grpc}` | otel/sdk same minor | The exporter version is split: trace at 1.43.0, metric at 0.65.0, log at 0.19.0. Bump as a set. |
| `gen_ai.*` attribute names | unstable | Pin a single semconv generation; bump `llm-agent-otel` major when changing |
| `testcontainers-go` | Docker ≥ 20.10 | Both PR CI and nightly need Docker available; GH Actions Linux runners have it pre-installed |
| `grafana/otel-lgtm` image | none / standalone | Single-process image; not for prod; pulls Loki/Tempo/Prometheus/Grafana as one |

---

## Sources

### Provider SDKs
- `/openai/openai-go` (Context7) — streaming, tool calling, embeddings code samples (HIGH)
- [OpenAI Go releases](https://github.com/openai/openai-go/releases) — v3.35.0 confirmed (HIGH)
- `/anthropics/anthropic-sdk-go` (Context7) — `Message.Accumulate`, `BetaToolRunner` patterns (HIGH)
- [Anthropic Go releases](https://github.com/anthropics/anthropic-sdk-go/releases) — v1.41.0 confirmed (HIGH)
- [Ollama API package on pkg.go.dev](https://pkg.go.dev/github.com/ollama/ollama/api) — v0.23.2 confirmed; streaming/tools/embeddings methods listed (HIGH)
- [Ollama SDKs in Go comparison](https://www.glukhov.org/post/2025/10/using-ollama-in-go/) — recommends `ollama/ollama/api` as production choice (MEDIUM)

### OpenTelemetry
- `/open-telemetry/opentelemetry-go` (Context7) — exporter setup, `WithBatcher` patterns (HIGH)
- [OTel Go releases](https://github.com/open-telemetry/opentelemetry-go/releases) — v1.43.0 confirmed (HIGH)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otel/protocol/exporter/) — HTTP/protobuf SHOULD be default (HIGH)
- [GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/) — **Development** status confirmed as of 2026-05-10 (HIGH)
- [GenAI agent and framework spans](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/) — `invoke_agent`, `invoke_workflow`, `create_agent` patterns; `execute_tool` for tool spans (HIGH)
- [Semantic-conventions releases](https://github.com/open-telemetry/semantic-conventions/releases) — v1.41.0 latest; gen-ai still development (HIGH)

### Multi-repo Go modules
- [Go Workspaces tutorial](https://go.dev/doc/tutorial/workspaces) — `go.work` semantics (HIGH)
- [LogRocket: Go workspaces multi-module local development](https://blog.logrocket.com/go-workspaces-multi-module-local-development/) — gitignore guidance (MEDIUM)
- Existing `llm-agent` README — `replace` escape hatch already documented for the project (HIGH; in-tree)

### Deployment
- [grafana/docker-otel-lgtm](https://github.com/grafana/docker-otel-lgtm) — single-container LGTM stack (HIGH)
- [otwld/ollama-helm](https://github.com/otwld/ollama-helm) — Helm chart with StatefulSet + GPU support (HIGH)
- [Ollama Kubernetes deployment guide (Kubert)](https://mykubert.com/blog/ollama-kubernetes-deployment-cost-effective-and-secure/) — StatefulSet + volumeClaimTemplates pattern (MEDIUM)

### Testing
- [pkg.go.dev/net/http/httptest](https://pkg.go.dev/net/http/httptest) — stdlib server stub (HIGH)
- [Testcontainers Ollama module (Go)](https://golang.testcontainers.org/modules/ollama/) — official upstream module (HIGH)
- [Running Testcontainers tests on GitHub Actions (Docker blog)](https://www.docker.com/blog/running-testcontainers-tests-using-github-actions/) — CI integration pattern (MEDIUM)

---

*Stack research for: Go LLM agent framework — provider adapters + OTel + reference deployable service*
*Researched: 2026-05-10*
*Verification: SDK versions cross-checked against GitHub releases pages on 2026-05-10; OTel `gen_ai.*` semconv status verified as **Development** on the same date — all callers MUST treat the namespace as opt-in until upstream promotes it.*
