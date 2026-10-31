# Licensing in plain words

Decision: ADR 0010 (user, 2026-10-30). d2lod-bot is licensed under GPL-3.0-or-later, the same licence as d2d (psi29a/d2decomped). Text: LICENSE at the repo root.

## NOTICE

d2lod-bot is free software under the GNU General Public License, version 3 or (at your option) any later version. It comes with no warranty. It contains no Blizzard code or data; Diablo II files are supplied by the user at runtime. Third-party items and their licences: THIRD_PARTY.md.

## What it means for us

* Private repo, private use: nothing is triggered. The GPL only bites when we give copies (source or binaries) to other people.
* Container product: if the image or binary goes to anyone else, they must be offered the matching source, including any d2d patches baked into the image. Game files never go in the image.
* Stream and public page: showing the bot's output on Kick or a web page is not distributing the code; no source obligation.
* Fork patches: unchanged for now. They live in the private <owner>/d2d-fork and not in this repo (ADR 0003) until the user decides otherwise.
* Upstream PRs: a PR to d2d is a GPL contribution authored as the user; drafts stay in the repo and nothing is posted without the user's go.
* Agents: contributions are under the same licence. Do not paste code from projects without a compatible licence (see ADR 0010 for the list).
* Dependencies: stdlib-only Go, so no dependency licence to check.

## Open questions (for the user)

* Answered (user, 2026-10-30, ADR 0010 addendum): the lockstep patches stay in the private downstream fork until upstream wants them (then an opt-in PR); the repo is going public soon (readiness audit: docs/research/public-readiness-audit.md).
* Tyniann licence text is still unverified (THIRD_PARTY.md).
