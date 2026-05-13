# Requirements: Phase 7 deprecation removal

**Status:** active as of 2026-05-12 by explicit early gate override
**Core Value:** the core `llm-agent` module remains stdlib-only and zero-dep;
providers, telemetry, and reference services stay opt-in sister repos

The full shipped `v0.3` requirement set remains archived at
`.planning/milestones/v0.3-REQUIREMENTS.md`. This active file pulls forward
only the deprecation-removal work for Phase 7.

## Active

- [x] **DEPRC-01**: Audit complete — zero internal users of `llm.Client`
      remain inside the core repo.
- [x] **DEPRC-02**: `llm.Client`, `llm.LegacyClient`, `GenerateRequest`,
      `GenerateResponse`, `StreamChunk`, and `StreamUsage` are removed in the
      `v0.4.0` core release line.
- [x] **DEPRC-03**: `CHANGELOG.md`, `DEPRECATIONS.md`, and
      `docs/migration-v0.2-to-v0.3.md` clearly document the v0.4 breaking
      removal.
- [ ] **DEPRC-04**: Sister repos move to `llm-agent v0.4.x` and coordinated
      release tags are cut only after compatibility updates land.

## Carry-forward Notes

- This gate was opened early by explicit operator instruction on 2026-05-12.
- Phase 7 scope is intentionally narrow: remove the deprecated v0.2 surface and
  coordinate the release. New feature work still requires a distinct milestone.

## Next Action

- Finish the release-coordination tail of `DEPRC-04`:
  - publish the final `llm-agent v0.4.0` tag first
  - then bump sister-repo `require github.com/costa92/llm-agent ...` lines off
    `v0.3.0-pre.2` to the published `v0.4.0` tag
  - cut coordinated sister-repo release tags after the version bump lands
- Use `.planning/ROADMAP.md` as the active phase ordering source.
