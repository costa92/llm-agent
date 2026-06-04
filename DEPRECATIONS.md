[English](./DEPRECATIONS.md) | [简体中文](./DEPRECATIONS.zh-CN.md)

# Deprecations

This file is the single source of truth for **what is deprecated and when it disappears**.
Every symbol with a `// Deprecated:` godoc comment in this repo MUST appear here with a
target removal version. Phase 7 of the v0.3 → v0.4 cycle audits this list and removes
each row whose `Removed In` matches the in-flight tag.

> Why this file exists: Pitfall 15 of the v0.3 research bundle (`deprecation kept forever`).
> The `// Deprecated:` comment surfaces in IDE hover, but is invisible at release-tagging
> time — a separate file forces a sweep before every minor cut.

## Active deprecations

*(none)*

## Removed (historical)

- `llm.Client` (interface) — deprecated in `v0.3.0`, removed in the Phase 7
  `v0.4` cut; use `llm.ChatModel`.
- `llm.LegacyClient` (interface) — deprecated in `v0.3.0`, removed in the
  Phase 7 `v0.4` cut; use `llm.ChatModel`.
- `llm.GenerateRequest` (struct) — deprecated in `v0.3.0`, removed in the
  Phase 7 `v0.4` cut; use `llm.Request`.
- `llm.GenerateResponse` (struct) — deprecated in `v0.3.0`, removed in the
  Phase 7 `v0.4` cut; use `llm.Response`.
- `llm.StreamChunk` (struct) — deprecated in `v0.3.0`, removed in the Phase 7
  `v0.4` cut; use `llm.StreamEvent`.
- `llm.StreamUsage` (struct) — deprecated in `v0.3.0`, removed in the Phase 7
  `v0.4` cut; use `llm.Usage`.

## Adding new deprecations

When you add a `// Deprecated:` comment to any public symbol:

1. Add a row to the **Active deprecations** table above with the exact symbol path,
   the version that introduced the deprecation, the target removal version, and a
   migration link.
2. Use this exact godoc format on the symbol so `gopls` and `staticcheck` can warn:
   ```
   // Deprecated: <use what instead>. Will be removed in vX.Y.Z. See <doc link>.
   ```
3. The `Removed In` column MUST point at a real planned release. Vague targets
   ("future", "TBD") are forbidden — they encode Pitfall 15.

## Removal procedure

When a tag matches a `Removed In` value:

1. Confirm `git grep -n '<symbol>' -- ':!DEPRECATIONS.md' ':!CHANGELOG.md' ':!docs/'`
   returns zero internal users.
2. Delete the symbol declaration AND its godoc comment.
3. Move the row from **Active deprecations** to **Removed (historical)**, adding
   the actual removal commit SHA.
4. Add a `### Breaking` entry to `CHANGELOG.md` for the release.
5. Bump sister-repo `require github.com/costa92/llm-agent` lines in coordinated tags.

---

Last updated: 2026-05-10 (Phase 0 of the v0.3 milestone — first entry).
