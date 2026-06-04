[English](./README.md) | [简体中文](./README.zh-CN.md)

# Demo 07: Policy / safety middleware

Demonstrates the three built-in gates from the `policy` package
(Phase 36, CC-2): PII redaction, prompt-injection scanning, and
max-input-length enforcement.

## Run

```sh
cd examples
go run ./07-policy
```

Deterministic; no network or providers — uses the canonical
`scriptedllm` mock per `CLAUDE.md`.

## What the demos show

| Demo | Gate | Action | What you should see |
|------|------|--------|---------------------|
| demoPIIRedaction   | `policy.NewPIIRedactor()`      | `Replace` (pre-call) | The LLM receives a redacted version of your prompt; the response is still produced normally |
| demoInjectionBlock | `policy.NewInjectionScanner()` | `Block` (pre-call)   | `errors.Is(err, policy.ErrBlocked) == true`; the LLM is NEVER reached (counter == 0) |
| demoMaxInputLen    | `policy.NewMaxInputLen(4096)`  | `Block` (pre-call)   | Oversized input is denied before any network round-trip |

## Canonical setup

```go
wrapped := policy.Wrap(model,
    policy.NewPIIRedactor(),
    policy.NewInjectionScanner(),
    policy.NewMaxInputLen(4096),
)
agent := agents.NewSimpleAgent(wrapped, agents.SimpleOptions{Name: "my-agent"})
```

Three gates registered in order; first non-`Allow` decision wins;
`Block` short-circuits; `Replace` rewrites the request and subsequent
gates see the rewrite.

## Composition with OpenTelemetry observability

In production you typically stack `policy` on top of `otelmodel.Wrap`
from the sister repo `llm-agent-otel`:

```go
// requires github.com/costa92/llm-agent-otel/otelmodel
// (lives in a sister repo; NOT imported by this example)
wrapped := policy.Wrap(otelmodel.Wrap(provider),
    policy.NewPIIRedactor(),
    policy.NewInjectionScanner(),
    policy.NewMaxInputLen(4096),
)
```

Outer-most `policy` denies before observed; middle `otelmodel`
observes; inner-most provider makes the call. The compose-test in
`policy/integration_test.go` (Decision G of
`.planning/phases/36-policy-safety-middleware/36-RESEARCH.md`)
verifies this stack works WITHOUT importing `otelmodel` into core —
when `llm-agent-otel` bumps to match core v0.6.x in v1.3, a
sister-repo example will demonstrate the real-world stack.

## Audit log

`policy.WrapConfig(model, policy.Config{Gates: ..., OnDecision: f})`
accepts an optional `OnDecision func(policy.Decision)` callback that
fires synchronously for every non-`Allow` decision. Use it to wire
in your logger (`slog`, `zerolog`, etc.).

```go
cfg := policy.Config{
    Gates: []policy.Gate{policy.NewPIIRedactor(), policy.NewInjectionScanner()},
    OnDecision: func(d policy.Decision) {
        slog.Info("policy decision", "gate", d.Gate, "reason", d.Reason, "action", d.Action)
    },
}
wrapped := policy.WrapConfig(model, cfg)
```

See `policy/doc.go` for the full surface + ratified Q1-Q5 decisions.
