# docs/archive

Superseded planning and review documents, kept for history (everything is also in `git log`). Do not extend them; current status lives in [../INDEX.md](../INDEX.md) sections 1 and 2, open defects in [../open-findings.md](../open-findings.md).
Their internal facts were tagged V/S/U when written and are not re-verified. Path strings inside them may still say the old location.

- `milestones/` : milestone definitions A to G, S, risks, next-3-weeks (rescored 2026-10-28, stale per INDEX). `docs/milestones/milestones.json` stays in place: `tools/publish` and the board read it.
- `review/` : design review of 2026-10-28 (findings, design-gaps, gotchas, parallel-work, roadmap).
- `review-overnight.md`, `review-day.md`, `review-evening.md`, `review-night.md` : merge-risk review bodies; open items merged into [../open-findings.md](../open-findings.md).
- `wave2.md` : consumer-side interface contracts between agents (the contracts are in code now).
- `goals-catalog.md` : 127-objective goal catalogue (research; was `docs/goals/catalog.md`).
- `prior-art/meta/` : prior-art survey of other game bots and agents (Alas, MAA, playerbots, AutoExile, Screeps, Pokemon/Voyager agents).
- `BACKLOG-2026-10-30.md` : the backlog before the 2026-10-31 trim (current list: [../BACKLOG.md](../BACKLOG.md)).
- `research/` : finished upstream PR drafts (posted PRs #134, #135, #136; executed review replies; superseded draft v1). Index with status: [../research/README.md](../research/README.md).
- `merc-farm.md`, `sandbox.md` : superseded or paused-path docs (doctrim, 2026-10-31); the old path keeps a stub.

## Research archive, 2026-11-11 (TASK-311)

Rule (docs/process/coordinator-efficiency.md, rule 8): research older than 7 days with no inbound link (README and INDEX lists do not count) and a finished one-off purpose is moved here with `git mv`; a one-line stub stays at the old path so no link breaks. Nothing is deleted.

Candidates found: 39 research docs older than 7 days with no inbound link by file name. Archived (15, verifier and review one-off reports and one historical note): verify-alignment2, verify-dmgroll4, verifycovmissiles-trace, verifyitemweapons, verifylevelsareas, verifypalaura, verifypalskill2, verifyskamazon-trace, verifyskdruid, verifyteleroom-trace, verifywaypoint2-trace, review-batch2-C-opus, review-community-prs-20261102, review-pr180-20261103, redivivus-sister-project.

Kept on purpose: PR bodies, drafts and evidence packs not yet posted (the upstream batches still use them), verifiers named by another doc (verify-smiteblock, verifypaldur), tool examples, run analyses and docs with a `current` status banner or that decisions link to. Re-run the check later: a doc qualifies when `git grep -F <file name>` finds no file other than the doc itself, the research README, INDEX, CHANGELOG, BACKLOG and STATUS.
