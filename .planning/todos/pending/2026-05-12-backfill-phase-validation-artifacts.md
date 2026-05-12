---
created: 2026-05-12T12:42:00+08:00
title: Backfill phase validation artifacts
area: planning
files:
  - .planning/v0.3-MILESTONE-AUDIT.md:59
  - .planning/PROJECT.md:63
  - .planning/ROADMAP.md:60
  - .planning/milestones/v0.3-ROADMAP.md:251
---

## Problem

`v0.3` shipped with archive-quality debt: formal `*-VERIFICATION.md` coverage
is uneven after Phase 0, and Nyquist validation artifacts are missing for
Phases 2-6. The milestone is correctly closed, but later audits will have to
reconstruct evidence from summaries unless these artifacts are backfilled.

## Solution

Create a planning-only cleanup pass that:

- decides whether to backfill `*-VERIFICATION.md`, `*-VALIDATION.md`, or both
  for Phases 2-6
- links each artifact to existing test/runtime evidence rather than inventing
  new claims
- updates milestone-close notes once the archive debt is reduced

This should be treated as documentation/verification debt, not feature work.
