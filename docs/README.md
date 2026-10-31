# docs/: where to start

d2lod-bot (D.idle) is a Go bot for Diablo II LoD 1.14d. d2d (a lockstep fork of psi29a/d2decomped) is the primary backend ([ADR 0009](adr/0009-d2d-primary-backend.md)); the Game.exe path is frozen and stays the acceptance reference.

## Reading order (same as [../AGENTS.md](../AGENTS.md))
1. [../openspec/README.md](../openspec/README.md): how jobs (change cards) and contracts (specs) are written.
2. [plan/STATUS.md](plan/STATUS.md): where we are; the block at the bottom is generated.
3. Your change card `openspec/changes/<id>/` (the brief: proposal, tasks, `owns:`, `exclusive:`, `gates:`).
4. [../GLOSSARY.md](../GLOSSARY.md) when a term is unclear.
5. [d2-1.14d-memory.md](d2-1.14d-memory.md), then the doc of your area (find it in [INDEX.md](INDEX.md)).
[../PLAN.md](../PLAN.md) is the founding plan (historical).

## Folders
- Top level `*.md`: one doc per area (bot, combat, nav, items, soak, d2d-*). [INDEX.md](INDEX.md) lists every one in a line, grouped by area; the first doc of each group is the one to read.
- [plan/](plan/STATUS.md): STATUS (read first), plan-v2 (the plan with the user's decisions), death data.
- [BACKLOG.md](BACKLOG.md): open items without a card; [needs-you.md](needs-you.md): what waits on the user.
- [decisions.md](decisions.md) and [adr/](adr/README.md): the user's decisions and why (guard files).
- [../openspec/specs/](../openspec/specs/): the numeric gates and contracts; [knobs.md](knobs.md) is generated from code.
- [research/](research/README.md): traces of game.exe vs d2d, upstream PR proofs and evidence packs; topical index with status per doc.
- [process/](process/): how we work (upstream PR standard, workers, capacity, public split).
- [sme/](sme/) and [classes/](classes/): game-knowledge notes and per-class plans; [progression/](progression/), [goals/](goals/).
- [d2d-container/](d2d-container/README.md): the container design; [d2d-probes/](d2d-probes/) and [probes/](probes/): probe notes and raw CSVs.
- [grills/](grills/): the user's grill sessions; [agents/](agents/): how agents read the tracker and domain docs.
- [g1-log.md](g1-log.md): the live soak history, append-only.
- [archive/](archive/README.md): superseded and finished docs; each old path keeps a stub with a `> status:` line pointing there. Nothing is deleted.

## Conventions
A `> status:` line under a title says when a doc is superseded or partly true. Numbers that have an owner are never copied: knob defaults are in [knobs.md](knobs.md), gates in the specs, counts in `git log` and the STATUS block.
