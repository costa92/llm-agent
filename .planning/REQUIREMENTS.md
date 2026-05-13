# Requirements: Phase 7 deprecation removal

**Status:** complete as of 2026-05-13 by explicit early gate override
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
- [x] **DEPRC-04**: Sister repos now target `llm-agent v0.4.0`, verification
      passed against the coordinated release line, and coordinated release tags
      have been cut.

## Carry-forward Notes

- This gate was opened early by explicit operator instruction on 2026-05-12.
- Phase 7 scope is intentionally narrow: remove the deprecated v0.2 surface and
  coordinate the release. New feature work still requires a distinct milestone.

## Next Action

- Open the next milestone instead of extending Phase 7.
- Use `.planning/ROADMAP.md` as the active phase ordering source.
