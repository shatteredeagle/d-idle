# d2fidprobe run recipe (same-save probe, slice 1)

> status: first live run done 2026-11-07 (scenarios 1 to 4, see the last section); the recipe below is VERIFIED LIVE except where marked. The tool and its tests exist (offline, replay only); see [implementation status](../research/fidelity-probe-design.md). Card `fidprobe-core` (TASK-55). The first live run waits for the worker host to be free after the Gate 2 g2g wave.

## Per scenario (all read-only: no input, no memory writes, no cheats, d2d lab verbs off)

1. Capacity: `labrun` on the worker host (`top -bn2 -d1`) (idle + nice at least 25 percent; `tools/capacity.sh --add 2` as the ops rules say).
2. Saves: a directory outside the repo with `<role>.d2s` per scenario (roles: `d2fidprobe -list`). Work on COPIES: the host rewrites the save on exit (d2d-lan-oracle.md V). Never commit them.
3. game.exe side: a hidden instance i1..i5 (private Xvfb), never i0, never another agent's. Only multi/OOG tooling starts it: the coordinator starts it and logs in the character copy (`tools/game.sh start <id>` + OOG login), muted (`-w -ns`, `tools/mute-d2.sh`).
4. d2d side: one slot from `tools/d2dslot.sh`, started through `tools/agentrun.sh <agent> --ttl SECS -- ...`, the copy placed as `save/<Name>.d2s`, character selected, in town. No `--lab-verbs`.
5. Wait until both are in town and at rest, then run through the ops wrapper so the ptrace grant and the instance flock apply, from a build of `cmd/d2fidprobe`:
   `tools/live.sh <agent> d2fidprobe -scenario 1 -instance iN -sock SLOT.sock -tables CACHEDIR -out OUTDIR`
   (`-tables` is the extracted game table cache that `d2dsnap -tables` takes; without it item classes are unknown and the items group is UNREADABLE.)
6. Read the table: any MISMATCH is a matrix row update and, for a d2d deviation, an oracle task; it is never added to `known.json` without a row and a reason. Stop the instance (`tools/game.sh stop <id>`), `tools/agentreap.sh <agent>`, `tools/agentreap.sh --list`.

## Run 2 additions (2026-11-07; V live, see [fidelity-probe-run1.md](../research/fidelity-probe-run1.md))

- Third reference, the d2s file reader (`internal/fidprobe/file.go`, stdlib, on `internal/cheat/d2s`): side source `-a file:X.d2s` (or `-b file:X.d2s`) reads header (class, level, difficulty, act), the stat block with its 8 fractional bits (str/ene/dex/vit, points, exp, gold, current and maximum life, mana, stamina), the 30 class skill levels, quests and waypoints. It does NOT read items, selected skills, area or resists: those cells show UNREADABLE, never EQUAL.
- `-ref X.d2s`: the file's sheet is also diffed against side A and side B and appended to the report (file against game.exe, file against d2d).
- `-dump F.json`: both sheets with every field (set members included) as JSON, for re-scoring without the game.
- Scenario 15 `load-life-mana` (group `load`): life, maximum life, mana, maximum mana, stamina and maximum stamina as floats (stat/256) right after a load. Read game.exe 1 to 2 s after `oog start`; d2d after `key Return` in lockstep with no step. Between repeats: `oog leave`, restore the save copy, `oog start`; d2d restarted with a fresh copy.
- Instance note: the agent name is not in `tools/instances.conf`, so name the hidden instance with `D2_INSTANCE=i5` for `tools/live.sh` and `oog` (or `-instance i5` for the probe), as in [d2d-lan-oracle.md](../d2d-lan-oracle.md).
- Normalisation (data in `rules.json`): `skills.base` lists the class's own 30 skills; zero stat entries are dropped on both sides; stat ids only one reader lists go to `statsx`, which no scenario compares.

## Offline replay (no game)

`d2fidprobe -scenario fresh-sorceress-1 -a json:A.json -b json:B.json -reads 1`, with `A.json` from a `d2.Snapshot` JSON and `B.json` from `d2dsnap -progress`.

## Checklist for the first live run (slice 1, scenarios 1 to 4)

- [x] worker host has capacity; hidden instance i5 started with `tools/game.sh start i5` + `helper`, logged in with `oog start NAME`
- [x] (substitutions, see below) saves for roles `sorc-fresh`, `pal-fresh`, `sorc-6`, `sorc-6-gear` exist as copies
- [x] confirmed: game.exe `FindServerPlayer` works in town; d2d `Progress` answers on the lab binary with lab verbs off (quests and waypoints read)
- [x] scenarios 1 to 4 run; three-read stability passed on both sides (one retry, see scenario 3)
- [ ] reports saved to `docs/research/fidelity-probe-run1.md`: NOT written; the findings are in the section below (task owns the file; coordinator decides)
- [x] record `writes after start: 0` from `d2cheat ledger --since-run` for the record

## First live run 2026-11-07

Tags: V = seen in the run output of this session, S = inferred. Host shared with Gate 2 wave g2h (3 runs, 3 slots), never touched. Raw reports (text and JSON per scenario) stay on the worker host cold dir, not in the repo.

### Setup (what was actually done)
- Capacity: `top -bn2 -d1` before: idle 22 + nice 47 percent (about 70 percent free of 16 cores); `tools/capacity.sh` is documented as hanging and was not used. One game.exe instance plus one niced d2d (nice 19) added.
- Gate speed ("virtual per wall" from the `pace` log lines of the three g2h runs, windows of 5 minutes): the windows are very noisy on their own (single runs swing between 1.6 and 8.5 in the hour before the probe). Mean of the 60 minutes before against the last 10 minutes during the probe: s11 5.41 to 5.30, s13 5.19 to 5.66, s17 4.44 to 1.81; sum 15.04 to 12.77 (minus 15 percent). INCONCLUSIVE as an effect of the probe (s17 alone had 1.6 windows an hour earlier), but at the edge of the stop rule; the probe was already finished (about 7 minutes of load) and was not repeated.
- Instance: i5 (idle, stopped, no other agent mapped to it besides the multi agent). Saves are COPIES kept in the cold dir (`fidprobe/saves/<role>.d2s`, originals in `fidprobe/orig/`); no save was edited.
- d2d: slot 3 (free), binary `d2d.lab3-labverbs-f75816c6-linuxbuild` WITHOUT `--lab-verbs` (flag defaults off), started through `tools/agentrun.sh` + `d2dslot.sh run -n 3`, `--start-screen charselect --start-name NAME`, then `key Return`. The rebase of the private-branch fixes onto this lab binary is NOT done and stays open (separate step).
- game.exe: `tools/game.sh start i5`, `helper i5`, `oog "start NAME"` (main/OOG tooling; no mouse or key input beyond the menu flow).
- Probe: `tools/live.sh multi d2fidprobe -scenario N -instance i5 -sock <slot3 sock> -tables <slot3 e1data/cache/tables> -saves <cold>/saves -out <cold>/out`. The `-tables` cache must be the one with `monstats.tsv` etc. (the d2lod user cache only had `armor.json` and `gear.json`: first attempt failed with `open monstats.tsv`).
- Substitutions (no fixture existed, characters may not be created): sorc-fresh = Freshd (clvl 1, expansion), pal-fresh = Pally (clvl 4, NOT fresh), sorc-6 and sorc-6-gear = both the same file Freshe (clvl 6, charged bolt, 12 inventory items): scenario 4 therefore tests the same character as 3.
- `writes after start: 0`, `lab verbs: off` printed by every report. No `d2cheat` call, no ledger entry.

### Results per scenario (exit 3 = MISMATCH on all four; none passes yet)
| # | Scenario | EQUAL | KNOWN | MISMATCH | UNREADABLE |
|---|---|---|---|---|---|
| 1 | fresh-sorceress-1 (Freshd) | 33 | 1 | 2 | 7 |
| 2 | fresh-paladin-1 (Pally, clvl 4) | 33 | 0 | 3 | 7 |
| 3 | sorceress-6-skills (Freshe) | 31 | 0 | 5 | 5 |
| 4 | sorceress-6-vendor-gear (Freshe) | 25 | 0 | 5 | 5 |

Scenario 3 first run: `gameexe unstable between read 1 and 3 (derived.mana)`: the save holds mana 24.4 of 45 and game.exe regenerates; the tolerance rule (three equal reads) refused the run correctly, and a retry a few minutes later passed (mana 40, still regenerating but stable over the 3 reads).

### What matched (V)
base (level, exp, gold, stash gold, str/dex/vit/ene, points, difficulty, area, class), max life/mana/stamina, current stamina, cold/fire/lightning/magic resist, selected left and right skill, quest words and waypoint sets (difficulty 0 to 2), on all four loads. Matrix rows touched: ST07, ST01, PR05 (evidence appended; the other rows were not verified by this run).

### What diverged or is unexplained (each needs triage, none is hidden in `known.json`)
1. **Current life and mana at load (V, three-way).** File Freshe: life 89.5/91, mana 24.4/45. game.exe after load: 89 and 27 (regen afterwards). d2d: 91 and 45 (full). Pally: file life 77.8/88, game.exe 77, d2d 88. So d2d heals to full on load; game.exe keeps the saved value. Relevance for the bot: a bot start with low life/mana behaves differently on the two sides. Candidate d2d deviation (oracle task); whether real 1.14d rejoin keeps the saved value is the game.exe result here (V on this box).
2. **Max mana fixed point (V).** Pally file 19.5, game.exe 19.5 (raw 4992), d2d 19.0 (4864). The integer `max_mana` agrees, the raw stat does not: d2d truncates at load.
3. **Poison resist on Freshe (V, cause U).** d2d 6, game.exe 0, scenarios 3 and 4. Probably an equipped item, not resolved.
4. **`skills.base` (V, S for cause).** game.exe lists 8 extra skills at level 1 (ids 0,1,2,3,217 to 220: Attack, Kick, Throw, Unsummon and the like); the d2d read shows only class skills. Reader scope, not a game difference. `skills.bonus` on the clvl 1 Sorceress: game.exe 36=1 (staff), d2d none; the KNOWN rule SK14 covered it with a reason that does not fit this direction (the +1 comes from the item and is missing on the d2d side): the rule text should be revisited.
5. **`stats.vector` (V, S for cause).** game.exe omits zero entries, d2d lists them (13, 14, 15, 39, 41, 43, 45, 4, 5 equal 0): the tool should treat absent as 0. game.exe also carries entries d2d does not (172, 194, 31 defence, 67 to 69, 72 and 73, 350/351 on the Paladin): reader scope until proven otherwise.
6. **Not compared (UNREADABLE by design of the readers):** `items.*` (d2d `inventory.item_flags` lacks the start-item flag bits), `npcs.units` (no super unique id), `objects.units` (no open/closed mode). game.exe equipment counts: Freshd 1, Pally 2, Freshe 4; the d2d side was not read. So IT04, IT02, ST06, VN07, MA03 stay unverified.

### Dead ends and notes
- `tools/capacity.sh` hangs; used `labrun` on the worker host (`top -bn2 -d1`) instead. The board server answered 403 for `agentstatus` (local files written).
- The d2d class number differs from the d2s class byte (Freshd shows class 4 in the d2d state line); the tool compares correctly, so this is a naming difference only (S).
- The three-read rule needs the character at rest: wait for mana regen or use a save with full mana.

### Next steps (not done here)
Fix the zero-versus-absent and extra-skill normalisation in `internal/fidprobe`, add a d2s reader so the file is the third reference (it already explained the life/mana result by hand), a real fresh Paladin fixture, items on the d2d side, then rerun scenarios 1 to 4 and 5 onwards; rebase the private-branch fixes onto the lab binary before measuring d2d fixes.
