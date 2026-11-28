# Fork requests of 2026-11-28: one prioritized list

> status: current (card forkreq, TASK-1134, 2026-11-28). A reading list for the owner's next fork session. Nothing here is built. No fork checkout was opened, no fork source was read and nothing was pushed or posted by this card. Every request below is a pointer to the doc that carries the detail; where the two disagree, the source doc wins.

Tags: V = stated as verified in the source doc named in the row (this card re-ran nothing), S = read in a doc or log, U = unverified. Every size and risk is this card's guess from the source text (U): S = under half a day, M = about a day, L = more.

"Lab acceptance" means a test the fork session can run headless on the builder (the differential pattern of `tests/input_items_diff_d2d.py`, or a Proving Ground scenario). No live game is needed for any row.

## Do first (data safety, then the base everything else sits on)

1. **A1: the character file is written to a temp file and renamed** (requests 1 to 3 of [fork-save-atomic.md](fork-save-atomic.md)). A hero was lost to this on 2026-11-28. Until it lands every save on a full or failing disk can cost a hero again; the bot's rolling copy only limits the loss to about ten minutes ([docs/ops/save-backups.md](../ops/save-backups.md)).
2. **A2: keep the last good file, refuse a bad one loudly** (requests 4 and 6). Same incident: the only `.bak` was a level 1 file from days before, and a 0 byte file must never be read as "no save".
3. **B0: decide the branch before any patch is written** (section B). A1 and A2 should be written once, on the branch that will be switched to. Needs the owner's go; see the open question in B.

Everything after that is ordered by what it unblocks, not by danger.

## Order at a glance

| # | Request | Size (U) | Risk (U) | Needs first |
|---|---|---|---|---|
| A1 | save: temp file, fsync, rename; a failed write keeps the old file | M | medium: the save path of every hero | B0 |
| A2 | save: `.bak` only of a file that passes the load check; bad file refused at load | S to M | medium: a too strict check refuses a good hero | A1 |
| B0 | branch decision and the pre-switch checks of the rebase round | S (decision) + M (test runs) | low | owner's go |
| A3 | side files (`.key`, `.map`, `.ma0`.., shared stash) the same way | S | low | A1 |
| C1 | open object 153 (Tyrael's door) at Duriel's death | S | low to medium: a quest path | B0 |
| D1 | `order=<id>` on the four direct item verbs, `item order <id>`, caps `item.order` | M to L | medium: touches the side file written with the save | A1, A3 |
| D2 | merc-equip order keeps the displaced unit and bytes; caps `item.full.merc` | S | low | D1 |
| E1 | game-side count of writes after start (read verb, or verbs dropped at hand-over) | M | low: a read | B0 |
| F1 | differentials for `cmd repair` and `menu choose` | M (tests only) | low | B0 |
| G | read-only answers from the fork source (nine questions) | S each | none | none |
| B1 | the switch itself (image, lab binary, sandboxes) | M | high: every running hero | A1, A2, B0, owner's go |
| A4 | free bytes of the save folder in the status reply (optional) | S | low | A1 |

## A. Save safety

Source: [fork-save-atomic.md](fork-save-atomic.md) (seven requests, card savesafe). Incident: [docs/probes/needs-owner-2026-11-28.md](../probes/needs-owner-2026-11-28.md).

**Evidence.** V (seen on the damaged files): the worker host's root disk filled; at the next save the sandbox d2d left the hero's `.d2s` (character Questy) at 0 bytes, about 04:08 EST by the coordinator's brief (S: the time is not in the source docs); the only `.bak` was a level 1 file from days before. U: that the fork writes in place is concluded from the 0 byte file; the save code was not read. **First step of A1 is to read that code path and correct this section if it is wrong.**

| # | What | Lab acceptance | Risk (U) |
|---|---|---|---|
| A1 | Requests 1 to 3: write `<name>.d2s.tmp` in the same folder, check every write and the close, `fsync` the file, `rename`, `fsync` the folder. On any error (ENOSPC, EIO, short write) remove the temp file, leave the live file untouched, answer an error on the save verb, one log line with file and errno; the game keeps running. | (a) Save folder on a small tmpfs filled to the last byte, trigger a save: the previous `.d2s` is byte-identical and loads; the verb answers an error. (b) Kill the process between write and rename (a test hook or a signal in a loop): the file found is the old or the new one, both load. (c) A normal save still produces a file the unchanged loader and the bot's own reader accept. | A rename across a bind mount fails if temp and target are not in one folder; Wine or container volume semantics for `fsync` of a folder (U). |
| A2 | Requests 4 and 6: before the rename, move the live file to `.d2s.bak` only when it passes the load check (signature, size field, checksum). At startup a zero-length or checksum-failing `.d2s` is an error that names the file; never "no save", never overwritten by a new hero of the same name. | (a) Start with a 0 byte `.d2s`: exit with an error, no new hero written, `.bak` untouched. (b) Corrupt one byte of the live file, save: the good `.bak` is not replaced. (c) Two good saves in a row: `.bak` is the first one. | A check stricter than the loader refuses a hero that would have loaded. The bot's startup repair (card savesafe) restores from its own copies when d2d refuses: the two must be tried together once. |
| A3 | Request 5: `.key`, `.map`, `.ma0`.. and a shared stash file get the same write. | The tmpfs test of A1 with each side file as the one that does not fit. | Low. D1 writes its order table into the side file, so A3 comes before D1. |
| A4 | Request 7, optional: the status reply carries the free bytes of the save folder. | The value matches `statfs` within one block. | Low. The bot has its own disk guard; this only removes a second code path. |

After A1 to A3 are verified the bot's planned save-point refusal on a low disk (`bot.Deps.DiskLow`) may be relaxed; that is a decision for the card that verifies them, not for the fork session.

## B. Rebase state

Source: [docs/probes/fork-rebase-20261128.md](../probes/fork-rebase-20261128.md).

**State (V in that doc).** Private branch `didle/rebase-20261128` is pushed to the private remote, built (image number 31, no existing binary replaced) and passed the regression set except `paygold` (fail, fail, pass; same shape as the flake recorded for the two earlier binaries, not shown to be a regression, U). It merges 294 upstream commits and contains the forkcaps patch branch. One meaning conflict was fixed: the `buy` verb now sends the stock item's unit id. Nothing was switched: the lab binary and the sandboxes run the previous round.

| # | What | Why | Lab acceptance | Risk (U) |
|---|---|---|---|---|
| B0 | Owner decides: (1) forkcaps stays in the integration branch, (2) new patches (A, C, D, E) are written on patch branches off `didle/rebase-20261128`. Then the checks the round skipped: the other fork verb tests, upstream's own ctest (the range is over 100 commits), the bot-side real-d2d tests, one Proving Ground scenario each for a fight row and a vendor row, and `paygold` three times on the old and three on the new binary. | Writing A1 on the old round means porting it over 294 commits later; writing it on a branch that is then not switched to leaves the sandbox unprotected. The buy message changed, and `paygold` guards the vault gold write. | The ready-to-run checklist at the end of the rebase doc, steps 1 to 3. `paygold`: the failing lines of both binaries are the same set. | Low for the checks. |
| B1 | The switch: image step in the night window, then lab binary and sandboxes, with backup and rollback as in [docs/process/sandbox-devp30.md](../process/sandbox-devp30.md). Each is a separate go of the owner. | Several dark features wait for verbs the sandbox image does not have (U per row in [docs/probes/dark-queue-plan-2026-11-27.md](../probes/dark-queue-plan-2026-11-27.md)): `item merc-equip` (card mercvault: "needs the fork build with item merc-equip on the sandbox image"), `debug warptriggers` for `nav.warpRead`, `debug levelmap` for `quests.layoutRead`. | After the switch `caps` on the sandbox lists the expected names; one hero plays a town visit with a buy; the save of A1 is in the image. | High: 294 commits of game behaviour nobody has played (U in the rebase doc), on every running hero. Do it after A1 and A2 so the new image is the first one with a safe save, and take a copy of every hero file first. |

**Open question for the owner (this card cannot answer it).** If the switch will not happen within a day or two, A1 should ALSO be cherry-picked onto the round the sandbox runs today and that binary rebuilt, because the sandbox is exposed until then. Whether the save code differs between the two rounds is U.

## C. Duriel and Tyrael

Source: the `result:` line of TASK-1126 (card durieltyrael) on branch `agent/dur2`, not merged when this was written: `git show "agent/dur2:backlog/tasks/task-1126 - durieltyrael-the-Duriel-chunk-parks-at-Talk-to-Tyrael-npc-not-found-and-the-lab-drill-ends-BLOCKED.md"`. Runs: [docs/probes/lab-night-results-2026-11-28b.md](../probes/lab-night-results-2026-11-28b.md).

Note: the brief for this card placed the object in "Durance of Hate 3". The task card says level 73, Duriel's Lair, and that is what this row uses.

| # | What | Why | Lab acceptance | Risk (U) |
|---|---|---|---|---|
| C1 | At Duriel's death, open object 153 (`tyrael's door`) at subtile (110,207) of level 73, as game.exe does. | V offline in the card (the fork's level dump and Objects.txt for level 73; bot logs of 10 runs): Tyrael's preset is at subtile (78,88); the arena's only way to him is the 1x5 opening at (110,207), closed by object 153 (collision in modes 0 and 1, not selectable, no operate function). The fork never opens it (S: the card quotes the fork's own note, "Tyrael's prison is not opened"), so the hero's walk ends in the arena in every run and `drill.act2.duriel.stepexec` is BLOCKED 3 of 3. U: the exact game.exe trigger (death event, quest state change, or a timer) was not traced; the card states the behaviour, not the function. | (a) Fork test: kill Duriel with a lab verb, step N ticks, the object's mode is the open one and the five cells at (110,207) are walkable in the collision read. (b) Without the kill the door stays closed. (c) Proving Ground: `drill.act2.duriel.stepexec` Paladin 11 GREEN 2 of 3 (criterion 3 of TASK-1126), then the `quests.stepExec` ledger row is re-read. | The scenario's Duriel is already dead when the hero arrives (CHEATED setup): if the door only opens on the death event, the setup needs the kill inside the game, or the door must also open from the quest state at level load. Decide which and write it in the test. |

This is an upstream candidate (a deviation from game.exe), but only with a trace of the game.exe trigger and the differential the project's PR standard asks for; nothing is posted without the owner's go.

## D. Order ids on the direct item verbs

Source: [fork-order-id-spec.md](fork-order-id-spec.md) (cards fix1 and vault2), findings F1 and F2 of [fable-review-2026-11-28/01-item-flows.md](fable-review-2026-11-28/01-item-flows.md), N5 and N6 of [10-fix-review-items-and-board.md](fable-review-2026-11-28/10-fix-review-items-and-board.md), the sweep rows marked needs-fork in [docs/probes/review-sweep-2026-11-28.md](../probes/review-sweep-2026-11-28.md) and [review-sweep2](../probes/review-sweep2-2026-11-28.md).

| # | What | Why | Lab acceptance | Risk (U) |
|---|---|---|---|---|
| D1 | `item equipdirect`, `item wearset`, `item unequipdirect`, `item merc-equip` accept `order=<id>` (1 to 64 of `[A-Za-z0-9._:-]`). The last 64 answered orders (id, verb, full reply) are kept and written into the character's side file with the save. A repeated id does not run again and answers the stored reply plus `replayed`. New read verb `item order <id>` (stored reply or `err unknown-order`). `caps` lists `item.order`. The measured-run refusal comes first, unchanged. | These verbs remove an item from the game and the reply is the only copy. V offline (the bot's tests): a lost reply can only be judged by looking, and looking has blind spots: a two-hander that pushes the shield off, bytes the engine changed between read and swap, a second process. N5 and N6 (U on the fork): create keys live per game process, so after a d2d restart a second attempt may run again or answer `dup` for the wrong reason. | The four cases of the spec: (a) equipdirect with `order=a`, drop the reply, `item order a` returns the same `removed=` bytes; a second call with `order=a` answers `replayed`, body unchanged. (b) The same across a d2d restart. (c) wearset of a two-hander over weapon and shield: `item order` returns both removed items. (d) A measured run refuses all of it. | Medium: a table saved with the hero; a table written out of step with the save recreates the loss it is meant to close. Hence after A1 and A3. |
| D2 | Per `item merc-equip` order the table also keeps the displaced unit id and its bytes. `caps` says whether `item full` lists the mercenary's items (`item.full.merc`). | V offline (card vault2): the bot has no slot read for the mercenary and guesses the displaced item from what is new on the hero; an item picked up in between that he could wear is filed as displaced. Round 3 of the review asks to file only the unit the fork named. | merc-equip over a worn piece with `order=b`: `item order b` names the displaced unit and bytes; the hero's listing before and after agrees with it. | Low. |

When `item.order` is listed the bot side is already specified in the spec ("The bot side once the capability is listed"); no bot work is asked of the fork session.

## E. A write count the bot cannot fake

Source: finding E4 and fix 8 of [fable-review-2026-11-28/06-proving-ground.md](fable-review-2026-11-28/06-proving-ground.md); still open in review 12 and in the sweep (needs-fork, high).

| # | What | Why | Lab acceptance | Risk (U) |
|---|---|---|---|---|
| E1 | A read verb that answers the number of gameplay-affecting lab verbs the game ran since a mark (or since start), or a way to drop the lab verbs at hand-over so later ones are refused. | V (read in the review): on field starts "writes after start" comes from the bot's own command journal while the lab d2d keeps its verbs on; 69 of 140 field scenarios do not assert it. Only town starts have an independent count. The crutch rule rests on that number. | Run a field-start scenario, send one lab write behind the bot's back through a second control connection: the game-side count is 1 where the bot's journal says 0. With the drop variant: the write is refused after hand-over. | Low (a read). The drop variant changes which verbs a running lab d2d accepts: every lab tool that writes mid-run must be checked. |

## F. Differentials the fork owner runs (tests, no behaviour change)

Source: section 3 of [input-i4.md](input-i4.md) (card inputl4). The bot side is built and dark behind `input.planner` (0).

| # | What | Why | Lab acceptance | Risk (U) |
|---|---|---|---|---|
| F1 | Three differentials on the forkcaps code (now inside the integration branch): (1) `cmd repair -1` against a click on Repair All, worn wand 7/15, digest durability and gold; (2) `cmd repair <id>` against the single Repair button, with the refusals (no gold, nothing to repair, a non-repairable item); (3) `menu choose Trade` against a click on the dialog line for Akara and Charsi, digest the UI flags, refusal with no dialog open. | The knob's enable criteria ask for one differential per verb. U (recorded in the devp14 sandbox doc): the headless hero never reached the vendor, so neither verb was ever run; the tests must walk him with `debug goto` first. | The three tests pass in lockstep with equal digests; then the caps note file is updated by a card that owns it. | Low. |

## G. Questions the fork source answers (read-only, no patch)

Each is tagged U in the doc named, and a bot card waits on the answer. One short note per answer (file and line in the fork, on a pushed commit) is the deliverable.

1. Does `item full -1` list the mercenary's own items on the real build? The bot's fake does not; if the build does, a copy given to the mercenary looks "still there" ([fork-order-id-spec.md](fork-order-id-spec.md)).
2. Does the control channel ignore an unknown `key=value` argument, or refuse the line? Decides whether the bot may send `order=` before `caps` lists it (same doc).
3. `cmd select` of a skill that was never learned, `cmd skillpt` without the level or prerequisite, `debug points`: what does the fork answer? The smoke fake accepts all three (U in `smokegame.go`), so a hero that binds an attack it cannot learn passes the smoke (D1 of [12-fix-review-security-pg-data-pipeline.md](fable-review-2026-11-28/12-fix-review-security-pg-data-pipeline.md)). With the answer the fake can refuse what the fork refuses.
4. `debug hit`: can it damage and kill the mercenary, and what does it take as target? Two drills are planned on it: R3 for `bot.portalGuard` (`debug hit 20`) and the missing revive drill for `bot.mercReviveUnverified` ([dark-queue-plan](../probes/dark-queue-plan-2026-11-27.md)). If it cannot target the mercenary, that is a small verb request (S).
5. Tools of the Trade: the quest record stays 0 after the hand-in and the reward word 0x204e does not come (`town.plan.malus-errand-gold` RED, `necro.act1-tools` RED; "fork or bot" is U in [lab-night-results-2026-11-28.md](../probes/lab-night-results-2026-11-28.md), "the fork's Imbue line, backend limit" in [necromancer-gap-2026-11-27.md](../classes/necromancer-gap-2026-11-27.md)). Which quest states does the fork set at the Malus hand-in?
6. The Summoner: after the kill and the journal touch the quest word reads 0x2006 and bit 0 never comes (`drill.act2.summoner.stepexec` RED, "a or b, U" in [lab-night-results-2026-11-28b.md](../probes/lab-night-results-2026-11-28b.md)). Does the fork set the done bit, and on what event?
7. Barracks: `d2dworld: arrived in level 28: 0 adjacent entries` in every arm and seed, and with `nav.warpRead` on the warp read did not give the way back to level 27 (`nav.barracks-return` RED, same docs; card navbarracks owns the bot side). Does the fork's adjacency and warp read for level 28 name level 27?
8. Do create keys (`key=vk-...`) survive a d2d restart? (N5, N6; folded into D1 if the answer is no.)
9. `combat.minionRead`: the read verb works but was never compared with a hero that has minions (the lab hero had none). Not a fork gap; listed so nobody files it as one.

## Not in this list

Older open fork requests were not re-ranked here: the lockstep `poll_levels` wait ([docs/d2d-determinism.md](../d2d-determinism.md)), the points verbs proposal ([docs/process/points-verbs-devp14.md](../process/points-verbs-devp14.md)), the skill-bonus stats in the dump (candidate FORKSTATS in the backlog), and whatever is still open in [docs/probes/stepgrounds-fork-request.md](../probes/stepgrounds-fork-request.md) (its banner says F1 and F2 are done).

## Rules that bind the fork session

From AGENTS.md: every fork commit is pushed to the private remote the same day; a doc cites only a pushed commit; no patch text enters this repo; nothing is posted upstream without the owner's go; the switch of a binary or image is the owner's decision each time.
