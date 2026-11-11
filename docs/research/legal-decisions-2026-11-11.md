# Legal decisions of 2026-11-11 (user), and what was done

> status: record for the grill session. Source: the user's answers to docs/research/legal-note.md (2026-11-11). Not legal advice. `docs/decisions.md` is written by the coordinator or the grill session only; the paste-ready lines below are the proposal (docs/process/grill-queue-2026-11-11.md was not on origin/main when this was written, so this doc is the source for it).

## Decisions (one line each)

1. 2026-11-11: GO for a private beta to one or two friends.
2. 2026-11-11: delivery is a private tarball over a private link (no registry, no public image).
3. 2026-11-11: the fork source tarball and exact commit ship WITH each pack; no public fork.
4. 2026-11-11: the user reviews GPL compliance himself with the 7-item checklist (tools/friendpack/checklist.sh writes the record docs/friend-checklist-<version>.md).
5. 2026-11-11: art on the stream overlay is allowed for a non-monetised stream, but stays OFF by default until he asks (proposal: docs/paperdoll-art-build.md "overlay.art").
6. 2026-11-11: the stream is not monetised; no inquiry to Blizzard now.
7. 2026-11-11: verify the Tyniann licence text and the Freepik icon terms; remove if unverifiable. Result: Tyniann verified (THIRD_PARTY.md); Freepik icon to be replaced by an original asset (user addendum: regenerating assets is fine; originals preferred over attribution), inventory in docs/friend-assets.md.

## Paste-ready lines for docs/decisions.md (grill session)

- 2026-11-11: Private beta to one or two friends approved; delivery by private tarball over a private link; the fork source tarball and commit ship with every pack, no public fork (docs/research/legal-decisions-2026-11-11.md).
- 2026-11-11: The user reviews GPL compliance per release with tools/friendpack/checklist.sh (7 items; a record docs/friend-checklist-<version>.md per release); `build.sh` runs it as its last step and fails unless `--force`.
- 2026-11-11: Stream overlay art may be enabled for a non-monetised stream only; default off; the stream is not monetised; no Blizzard inquiry now.
- 2026-11-11: Third-party or unverifiable assets are replaced by original ones where possible instead of attributed; the Tyniann licence is verified (view/fork/study, no product redistribution; ideas only), the Freepik icon in d2d is replaced on a fork-only branch (docs/friend-assets.md).

## What was done (TASK-348)

- Tyniann: LICENSE read from github.com/Tyniann/d2r-offline-farming-bot (V, GitHub API, 2026-11-11): THIRD_PARTY.md row and UNVERIFIED line replaced.
- Checklist tool: tools/friendpack/checklist.sh + test in tools/friendpack/pack_test.go; wired as the last step of build.sh (`--force` to proceed on FAIL).
- Disclaimer line in README.md, docs/friend-guide.md and the pack README.txt; the pack NOTICE now lists the third-party components.
- Open: the GPL reviewer (the user), the Freepik icon patch (see docs/friend-assets.md), the fork `LEGAL.md` still names the Freepik skull for launcher-only files.
