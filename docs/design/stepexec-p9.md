# P9: the generic step executor (card stepexec1, TASK-976)

> status: current (2026-11-27). Source: docs/research/fable-logic-review-2026-11-25.md 2.4 (classes F, H), 7.1 P9; ADR 0021 (questspec), docs/outcomes.md.

## What it is
One engine, `questspec.RunStep` (internal/questspec/exec.go), runs a step of the five verbs `interact`, `pickup`, `talk`, `use`,
`transmute`:
APPROACH the target, ACT, WAIT for the declared fact (the step's `post` clauses, read from a fresh snapshot, never from the
executor's own memory), retry within the step's `tries` with doubling backoff. A post that already holds before the first act
is Done and touches nothing (resumption after a restart). V: unit tests `internal/questspec/exec_test.go`.

## The row is data
A step of `data/questspec/*.json` carries an optional `exec` block: `verb`, `on` (the thing acted on, by class id, never a
display name or a numeric literal in Go), `then` (a second verb per wait poll, e.g. pickup after the click), `tolerance`,
`wait_polls`, `wait_ms`, `backoff_ms`, `backoff_cap_ms`, `wake`. `Validate` rejects a bad verb, no post or no tries.

## Outcome
`questspec.Outcome` has the five kinds of internal/bot/outcome.go (Done, Progress, Refused{need, wake, why}, Failed{err},
Interrupted{err}). questspec cannot import bot, so the kinds are mirrored and the bot adapter maps them; this avoids
touching the hot outcome.go. Progress (the backend moved but has not arrived) does not use a try. A refusal is data, not a
failure. A backend error that is not one of the typed ones is a failed attempt, retried in budget.

## Adapter
`internal/bot/stepexec.go`: Backend over `TaskDeps.Objects`, `Deps.Loot` and the pack scan; one call site in
`taskActs.Interact`, behind `quests.stepExec` (default 0, `[dark-launch]`, ledger row in data/darklaunch.json).
Approach is a no-op there: the object and dialog adapters walk themselves (U: a separate approach verb is for chunks
whose adapter does not).

## Migrated
Cain scroll `search_for_cain/take_scroll` (Tree of Inifuss click and pick-up). Scenario `cain.scroll.stepexec` (Paladin and
Sorceress, same assertions as `cain.scroll`). Not run live or on the lab yet (U).

Card stepexec2: the TALK and USE adapters (`execBackend.Act`). TALK says the step's `exec.pick` to the NPC named from the class id
(the inverse of `talk.ByAct`, so no name is written in Go) through `TaskDeps.Dialog`; `taskActs.Talk` takes the executor path when
`quests.stepExec` is 1 and the data has a talk step with an `exec` row for that NPC class and pick (`execTalkStep`, data-driven).
USE right-clicks the named pack item through the object adapter (`UseItem`); no chunk uses it yet. The second migrated chunk is
Akara reading the bark: `search_for_cain/read_scroll` (pick `bark`, post `has_item(bkd)`, read from the pack), scenario
`cain.read.stepexec` (Paladin and Sorceress, same assertions as `cain.read`). V: unit tests `internal/bot/stepexec_test.go` (two
classes, must-fail when the dialog says yes and the pack keeps the scroll). U: not run on the lab. `Backend.Act` gained the `pick`
argument.

## Act 2 rows (card stepexec1, branch stepexec2, 2026-11-27)
`data/questspec/act2.json` holds the chains `horadric_staff` and `seven_tombs`, checked like Act 1 against both capability
sets and the board (rule 11). Two steps carry an exec row:
- `horadric_staff/transmute`, verb `transmute` (new; the row's `inputs` are base codes `msf`, `vip`, the target is the
  product `hst`): `taskActs.Transmute` takes the executor path when the knob is 1 and the input names match a row
  (`execCubes`). Post `has_item(hst)|quest_state(horadric_staff,done)`, read from the pack and the quest READER (never the
  task layer's proxies). `Backend.Act` now takes `questspec.ActArgs{Pick, Inputs}` instead of a pick string.
- `seven_tombs/place_staff`, verb `interact` on the orifice (class 152, adapter name `tal-orifice`, which finds the true
  tomb itself): `taskActs.Interact` looks every object up in `execInteracts` (the Cain tree goes the same generic way now).
  Post `lacks_item(hst)`: the orifice takes the Staff out of the pack. The quest word is no post here: the drill notes say
  Cain's talk may already move record 10, and d2dctl V devp28 reads 0x2001 after the placement, so a word post could read
  Done before the click. `lacks_item` is a new reader (proxy over dump items, S) in both caps files.
Not given an exec row, on purpose: `horadric_staff/talk_cain`. No game fact proves the hand-in (lab word 0x0c78 after the
talk, bit 0 clear, drill.act2.assemble notes); a talk row needs that fact first (live check below). Item codes `box`, `msf`,
`vip`, `hst` joined `questItemCodes` (S misc.txt, internal/loot/questcodes_test.go), so the pack counts them by code.
V: unit tests internal/questspec/exec_test.go (transmute) and internal/bot/stepexec_act2_test.go (Sorceress and Paladin,
must-fail when the Cube or the orifice say yes and the pack does not change, knob 0 keeps one call each). U: not run on the
lab. Scenarios `drill.act2.assemble.stepexec`, `drill.act2.duriel.stepexec` (tools/provingground/scenarios/stepexec-act2.json).
Live drill wanted: the two scenarios 2 of 3 each; read record 10 before and after the Cain talk and after the placement, to
decide the talk row's post.

## Boss steps and the Act 2 hand-in (card bossdone, 2026-11-27)
- A kill step's post is read from game state (`taskActs.bossDead`, internal/bot/stepexec.go): the data row of the kill
  step, found by its monster name (`execKills`, no exec row needed), holds when a kill of it was seen this game, its dead
  unit is in the snapshot, or its quest word bit is set. `KillBoss` checks it before the boss run and again when the run
  ends in `boss not found` (V drill act2drill5: the general fight killed Duriel, then the boss run found nobody). Every
  boss: without a row it reads `killed(name)` alone.
- New fact `quest_bits(q,mask)`: every bit of the hex mask set in the reader's raw word of task quest q (memory reader in
  both caps files). `kill_duriel` post: `killed(duriel)|quest_bits(duriel,0x0020)|quest_state(duriel,cleared)`; record 14
  bit 5 at Duriel's death is V on the lab (0x0004 to 0x0024), S in the d2d port's SevenTombsQuest.
- Hand-in rows (talk kind): `seven_tombs/talk_tyrael` (exec, pick `reward`, post bit 0x2000, Tyrael sets bits 13 and 3),
  `seven_tombs/talk_jerhyn` (exec, pick `act2`, post bit 0x0010, Jerhyn's reward line), and a new chain `act2_done` with
  `talk_meshif` (no exec row: the sail changes act; post bit 0x0001, the record 14 completion set by Meshif's line 450).
  The bit sequence is S (d2d port rules/quests.hpp, traced from game.exe); the lab has only shown bit 5 (U the rest).
- Open (task layer, internal/task/act2.go, not this card): the Tyrael node's done is `not(stage == Cleared)`, and a kill
  raises the stage to Cleared until bit 0, so the node stays needed after the talk; the Jerhyn node is skipped once the
  act 2 word is done, which may let the hero sail before Jerhyn's reward line, so record 14 bit 0 would stay clear.

## Rule (acceptance criterion 2)
No new quest chunk may add a hand-written click-wait-pick-up handler: a new step of these four verbs is a questspec row with
an `exec` block, run by `RunStep`. Old handlers move over one at a time, each behind its scenario and a knob; the old code is
deleted after a 24 h watch with auto-revert (docs/process/dark-launch.md). Next in line: the Cain hand-in once a fact
proves it, the cairn stones, the Act 2 talk steps (Atma, Jerhyn, Tyrael).
