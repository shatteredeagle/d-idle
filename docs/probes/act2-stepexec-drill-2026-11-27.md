# Act 2 step executor drill (card stepexec1, 2026-11-27)

> status: current (lab drill of TASK-976 AC #4; docs/design/stepexec-p9.md is the design).

## Setup

- Worker host lab slot through `tools/d2dslot.sh`. Lab d2d is the featuresoak build of 2026-11-26 (the featuresoak campaign's lab binary). The bot is origin/main at 93b68e23f, after the stepexec2 merge.
- Command (labrun job 20261127-025021-2ffa, run from the synced worktree):
  `env D2BOT_RUN=act2drill PG_DIR=<dir> PG_D2D_BIN=<featuresoak lab d2d> PG_DATA=<MPQ dir> tools/agentrun.sh act2drill --ttl 7200 -- taskset -c 0-7 nice -n 19 tools/provingground/run.sh -id drill.act2.assemble.stepexec,drill.act2.duriel.stepexec -class Paladin -seed 11 -repeat 3 -parallel 1`
- Knobs are the scenario files' own: `quests.stepExec` (1) and `resource.pointsDecisions` (1).
- Heroes are Proving Ground throwaways made by SETUP verbs before the start. Provenance is CHEATED, so the results are mechanics only.
- No writes after the start (the `no_writes` assertion is green on the assemble runs).
- Load: load1 on the worker host was 16 to 28. No run hit a wall-clock timeout. Each run ended on its own verdict.

## Results

| scenario | runs | verdict |
|---|---|---|
| drill.act2.assemble.stepexec (Paladin 11) | 3 | GREEN 3/3. `step executor horadric_staff/transmute outcome=done`, hst in the pack, Cain's menu opened |
| drill.act2.duriel.stepexec (Paladin 11) | 3 | BLOCKED 3/3: `campaign stalled ... no progress for 60 virtual min` |

### Duriel: why it stalls (V, the same in all three runs)

1. The hero reaches the Canyon of the Magi. findTrueTomb stops in tomb 66 because the orifice is in its map.
2. The executor runs `seven_tombs/place_staff`. The first try is interrupted by a fight.
3. Every later try ends with `interact done, but post [lacks_item(hst)] not observed after 4 looks`.
4. The board parks the chunk after 3 tries, and the bot halts on the stall rule.

The d2dctl journal shows what the bot sent each time:
- `cmd interact 105`, then `cmd hand 25`, then `cmd place 105 25` (d2dinput `PlaceItem`), 6 times per run.
- Every reply is empty: the hash is `e3b0c442...`, the hash of empty text.
- Yet hst stays in the pack, and record 14 never gets its placement bit.

So the place verb is accepted and nothing happens. The cause could be either of these:
- (a) This lab d2d build does not run the devp28 place handler the design doc cites (U). The binary does carry the strings `place <object> [item]` and `The Horadric Staff: placed in the orifice`.
- (b) Unit 105 or item 25 is not the orifice and staff pair the verb expects (U).

It is not load: the runs fail on the same post at the same place every time.

## Quest record 10 (horadric_staff) and record 14 (seven_tombs)

All values below come from the bot log line `memory quest words` and the final `debug quest list`. All three assemble runs show the same values.

| when | record 10 | source |
|---|---|---|
| assemble start | 0x0000 (absent) | first `memory quest words` |
| after the transmute (executor done), before the Cain talk | 0x0800 | log line right after `step executor ... done` |
| after the first Cain talk (plain `Talk`, pick "") | 0x0c78 | log line right after Cain's `npc: menu open` |
| after the second talk (pick "horadric staff") | 0x0c78 (no change) | final `debug quest list`: 10 = 0xc78 |

Duriel runs:
- Record 10 is 0x0001 by SETUP and stays there.
- Record 14 becomes 0x0004 when the hero enters tomb 66, before any place attempt.
- The placement never happened, so record 10 after the placement is NOT observed.

### Decision for the talk row's post

The Cain hand-in moves record 10 from 0x0800 to 0x0c78 on the first talk.
- Bit 0 stays clear: the quest is not completed.
- The 0x0400 bit (and 0x0078) are new.

A talk row for `horadric_staff/talk_cain` can use the post "record 10 has bit 0x0400 set" (or "word != the value before the talk"). It must not use "bit 0 set".

Caveat (U): these words come from the lab d2d's quest model, not from game.exe.

## Rerun after placefix (origin/main 6391311d7, labrun job 20261127-044935-9921)

The rerun used the same command as above with `-id drill.act2.duriel.stepexec` only. The Duriel setup now writes record 10 = 0x0c78.

Result: BLOCKED 3/3 again, with a new cause (V, the same in all three runs).
- The d2d reads record 10 back as 0x0c08, not 0x0c78 (first `memory quest words` line).
- The board then refuses the chunk. Taps log: `not startable seven_tombs/duriel: state locked ... "locked until The Horadric Staff"`.
- The hero idles in Lut Gholein until the 60 virtual min stall rule ends the run. It never travels.

The placement was never reached, so record 10 after the placement is still NOT observed.

Next step, for the placefix owner:
- Option 1: make `requiresCleared` treat the word the d2d keeps (0x0c08) as cleared.
- Option 2: find which setup word survives the write. 0x0c78 masked to 0x0c08 suggests that bits 0x0070 are not stored by the lab quest model (U).
