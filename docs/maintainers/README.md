---
title: Maintainers Docs
description: Internal documentation for prompto contributors. Not part of the user-facing docs site.
---

This directory holds documentation aimed at people **modifying** prompto, not
people **using** it. User-facing docs live in the rest of `docs/`.

Contents:

- [`daemon-shells.md`](./daemon-shells.md) — How each shell integrates with
  the daemon: mode detection, redraw triggers, `--repaint` wiring, and
  debugging tips. Read this before changing any of `src/shell/scripts/prompto.*`.

When adding a doc here, keep it focused on the "how do I change this
safely?" question. Architectural rationale belongs next to the code it
describes (e.g. `src/daemon/ARCHITECTURE.md`).
