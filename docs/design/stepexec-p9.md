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

## Rule (acceptance criterion 2)
No new quest chunk may add a hand-written click-wait-pick-up handler: a new step of these four verbs is a questspec row with
an `exec` block, run by `RunStep`. Old handlers move over one at a time, each behind its scenario and a knob; the old code is
deleted after a 24 h watch with auto-revert (docs/process/dark-launch.md). Next in line: the Cain hand-in once a fact
proves it, the cairn stones, the Act 2 talk steps (Atma, Jerhyn, Tyrael).
