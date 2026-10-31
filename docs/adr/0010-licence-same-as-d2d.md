---
status: accepted
date: 2026-10-30
decision-makers: user, coordinator
---
# 0010: d2lod-bot is licensed under the same licence as d2d (GPL-3.0-or-later)

## Context and Problem Statement

ADR 0009 made d2d (psi29a/d2decomped, GPL) the primary backend. Grill topic 0002 (GPL / licence plan, ADR 0003, docs/d2d-container/gpl.md) was open and blocked any image leaving the NAS. The repo had no LICENSE (THIRD_PARTY.md: "private, not for redistribution until a license review").

## Decision Outcome

User, verbatim: "Let's finalize our decision and use the same license as d2d".

Upstream licence, verified read-only on 2026-10-30 (V, `gh api` on psi29a/d2decomped): LICENSE is the GPLv3 text; the README ("GPL-3.0-or-later") and LEGAL.md state GPL-3.0-or-later; source files carry `SPDX-License-Identifier: GPL-3.0-or-later` (apps/d2d/items.hpp, tools/emu/keys.py). GitHub's metadata only says `GPL-3.0` (it reads the LICENSE text, which has no "only/later" choice). We use the identifier of the headers: **GPL-3.0-or-later**.

Grill topic 0002 (licence route) is RESOLVED by this ADR; its remaining parts are the open follow-up below.

## Consequences

What changes:
* Our code may be published or distributed under the GPL. Distributing binaries or the container image to others requires offering the corresponding source, including any patches baked in. Private or internal use is not distribution.
* LICENSE added; new source files SHOULD start with `// SPDX-License-Identifier: GPL-3.0-or-later` (docs/process/spdx.md); existing files are not mass-edited.
* Contributions from agents are under the same licence.

What does NOT change:
* The rule "private d2d fork patches live only on the worker host and the forge <owner>/d2d-fork, never in this repo" stays (ADR 0003, AGENTS.md) until the user decides otherwise.
* Game data (the user's own licensed Diablo II files) is never in any repo or image.
* Nothing is posted to GitHub without the user's go.

Third-party code policy:
* No code copied from projects without a compatible licence. Redivivus has no licence: numbers and paths only (V-S, per THIRD_PARTY/prior-art notes; not re-checked here). Kolbot has no licence file: ideas only. Tyniann is source-available (UNVERIFIED terms): ideas only, check before release.
* D2MOO is MIT (V, `gh api repos/ThePhrozenKeep/D2MOO --jq .license`, 2026-10-30), GPL-compatible; copying would still need its notice kept. Decompiled Blizzard code never enters the repo.
* The zlib-licensed explode port (internal/data/mpq/explode.go) is GPL-compatible; keep its notice.
* The Go module has no third-party dependencies (V: `go list -m all` prints only d2lod-bot), so no dependency licence conflicts.

### Open follow-up question for the user

Now that the repo is GPL, may the lockstep patches live in the repo or be offered upstream, and is the repo going public?

## Addendum 2026-10-30 (user): follow-up answered

* The lockstep patches STAY in the private downstream fork (the forge `<owner>/d2d-fork`) for now; they do not enter this repo. User: "I haven't heard if the maintainer wants [the lockstep mode] pulled in. For now we can keep our downstream fork with our patches."
* The d2d maintainer has not said whether he wants lockstep upstream. If he does, it is offered as an opt-in PR (drafts stay in the repo, nothing posted without the user's go).
* The repo "will be going public soon". Readiness audit and publish plan: docs/research/public-readiness-audit.md (change `publicprep`).
* The AGENTS.md patch rule ("patches never go into this repo") is unchanged.

## Sources

docs/licensing.md, docs/process/spdx.md, ADR 0003, ADR 0009, docs/d2d-container/gpl.md, THIRD_PARTY.md
