# Fidelity probe: run 1 and run 2 (TASK-55)

> status: written 2026-11-07 by the fidprobe2 agent. Tags: V = seen in a run output of 2026-11-07 (V-run1 = the first run, V-run2 = this session), S = inferred, U = unresolved. Recipe and raw first-run log: [../probes/fidprobe.md](../probes/fidprobe.md). Design: [fidelity-probe-design.md](fidelity-probe-design.md). Rows: [fidelity-matrix.md](fidelity-matrix.md). No Blizzard data and no save file is in the repo; raw reports, sheets and the saves stay in the worker host's cold dir.

## What changed in the tool (run 2 prep, offline)

- `internal/fidprobe/file.go`: a d2s file reader (third reference), stdlib only, built on `internal/cheat/d2s`. It reads header (class, level, active difficulty and act), the character stat block (str/ene/dex/vit, stat and skill points, exp, gold, stash gold, current and maximum life, mana, stamina WITH their 8 fractional bits), the 30 class skill levels, quest words and waypoint masks. It does not read items, the selected left and right skill, the area or resists (those are absent from the file sheet, so a diff shows them UNREADABLE, never EQUAL). Tests use a byte-synthesized file (no real data); two classes (Sorceress, Paladin). VERIFIED LIVE (V-run2): file against game.exe on a freshly loaded Freshe gives EQUAL on 25 fields including skills.base, both quest sets and all waypoint sets; the only difference is current mana (regen).
- Reader-scope normalisation (data in `rules.json`, no class literal in Go): `skills.base` lists only the character class's own 30 skills with points (game.exe also lists Attack, Kick and the like); `stats.vector` drops zero entries on both sides (game.exe omits them, d2d lists them) and moves the stat ids only one reader lists (31, 67 to 69, 72, 73, 172, 194, 350, 351) to the group `statsx`, which no scenario compares. Cause of those ids: reader scope, S (not checked against the game).
- The KNOWN rule SK14 (`skills.bonus`) was reworded: the missing piece is on the d2d side (it gives base levels only), not extra on game.exe; no game difference is claimed (S).
- New scenario 15 `load-life-mana` (group `load`: life, maximum life, mana, maximum mana, stamina, maximum stamina as floats from the stat vector /256) and CLI additions: `-a file:X.d2s` source, `-ref X.d2s` (file vs each side appended), `-dump F.json` (all fields of both sheets).
- Run 1 re-score: the run-1 snapshots were NOT saved (only the text/JSON reports), so run 1 cannot be re-scored from them. Instead the new run-2 sheets of Freshe (the same character as run-1 scenarios 3 and 4) were diffed with the new normalisation: run 1 had 5 MISMATCH on scenario 3; now 4 cells remain, from 3 causes (current life, current mana, poison resist, plus the `stats.vector` cell that only contains stats 6, 8 and 45 of those same causes). The skills.base and stats noise is gone.

## Run 2 setup (V-run2)

- Instance i5 (hidden), restored the original copies of Freshe and Pally into its save folder before the run (same character names, no create or delete), `game.sh start i5`, `helper`, `oog start NAME`, between repeats `oog leave`, restore copy, `oog start`. d2d: slot 3 through `d2dslot.sh run -n 3` under `agentrun.sh`, nice 19, the run-1 lab binary WITHOUT lab verbs, `--lockstep --no-save`, `--start-screen charselect --start-name NAME`, one `key Return`. Every d2d repeat restarted d2d with a fresh copy of the save.
- Capacity before: `top -bn2 -d1` on the worker host: idle 24 plus nice 48 percent (about 72 percent free). The gate (g2h: s11, s13, s17; slots 0 to 2, characters Gate2i, Gate2s13g, Gate2s17f) was never touched; GATE_MAX_PARALLEL unchanged. `tools/capacity.sh` not used (it hangs).
- Gate speed ("virtual per wall" computed from the `pace` log lines, sums over the three runs): pre-probe mean 8.3 (one hour) and 9.7 (two and a half hours), probe window 16:24 to 16:31 EST 9.1 (d2d part 12.5, game.exe part 8.2). No sustained drop below 90 percent of the pre-probe mean in the 7 minutes of load. Per run the figures swing a lot (s17 0.6, s13 5.7 in the probe window, the reverse in the pre window), as in run 1; `gate-run.sh --status` "speed" is a since-start average and was useless as a 10-minute signal.
- Writes after start: 0. No `d2cheat` call, no ledger entry. Lab verbs off.
- Missing fixtures (not created, per rules): a TRUE fresh (clvl 1) Paladin does not exist among the characters (Pally is clvl 4, Palli clvl 13); a separate sorc-6-with-gear save does not exist (Freshe is the only clvl 6 Sorceress, so scenarios 3 and 4 are the same character). Scenarios 1 and 2 with real fresh data wait for those two.

## Life and mana on load, three-way (V-run2)

Current values with 8 fractional bits. File = the original copy, game.exe = read 1 to 2 s after `oog start` returned (3 repeats), d2d = read after `Return` in lockstep, no step issued (3 repeats). Every repeat gave the same numbers.

| Character | stat | file | game.exe (3 of 3) | d2d (3 of 3) |
|---|---|---|---|---|
| Freshe | life / max | 89.535 / 91 | 89 / 91 | 91 / 91 |
| Freshe | mana / max | 24.39 / 45 | 25 / 45 (28 and 32 in the unhurried first reads: regen) | 45 / 45 |
| Freshe | stamina / max | 102 / 102 | 102 / 102 | 102 / 102 |
| Pally (clvl 4) | life / max | 77.762 / 88 | 77 / 88 | 88 / 88 |
| Pally | mana / max | 19.5 / 19.5 | 19 / 19.5 | 19 / 19 |
| Pally | stamina / max | 101 / 101 | 101 / 101 | 101 / 101 |

Findings:
1. d2d heals life and mana to full at load (6 of 6 loads); game.exe keeps the saved value (6 of 6 loads) with its fraction dropped (89.5 to 89, 77.76 to 77, 19.5 to 19) plus a second or two of regen. REAL d2d DEVIATION (EN04, ST01). Consequence for the bot: a character saved hurt or drained starts full on d2d.
2. Maximum mana 19.5 (raw 4992) on the Paladin in file and game.exe, 19.0 in d2d (3 of 3 d2d loads; 3 of 3 game.exe loads read 19.5): d2d truncates the maximum at load. REAL d2d DEVIATION (ST01), tiny (0.5 mana); the integer `max_mana` agrees.
3. Poison resist: Freshe has stat 45 = 6 in the d2d stat vector and no stat 45 in game.exe's (V-run2, one repeat each, same result as run 1 scenarios 3 and 4). The Paladin shows no difference. The file block cannot carry it (stats above 15 are item sums). Equipment on game.exe is gloves (ethereal), a socketed helm, belt and boots; d2d's items are UNREADABLE so the cause is not found. UNRESOLVED (U). Hypothesis (S, unchecked): a socketed gem in the helm is summed by d2d and not by game.exe, or the reverse bug; needs the d2d item list with socket contents.

## Every run-1 MISMATCH, classified

| # | Field (scenarios) | Class | Evidence |
|---|---|---|---|
| 1 | derived.life, derived.mana (2, 3, 4) | real-d2d-deviation | table above, 12 of 12 loads |
| 2 | derived.max_mana, Pally (2) | real-d2d-deviation | table above (19.5 against 19.0) |
| 3 | derived.res_poison, Freshe (3, 4) | unresolved | above |
| 4 | skills.base extra skills on game.exe | reader-scope, FIXED in the tool | class-range filter; Freshe skills.base EQUAL in the re-score |
| 5 | skills.bonus (1) | known (SK14), reason reworded | staff +1 is missing on the d2d side; reader scope, S |
| 6 | stats.vector zero entries and extra ids | reader-scope, FIXED | re-score: only stats 6, 8, 45 remain |
| 7 | items.*, npcs.units, objects.units | UNREADABLE by reader design | d2d item flags lack the start-item bits; no super-unique id; no object mode; unchanged in run 2 |
| 8 | game.exe unstable between reads (scenario 3 retry) | not a deviation | regen of mana; the three-read rule worked |

## Candidate upstream PR list (in order)

No existing fidelity card covers load-time life/mana (TASK-55 `fidprobe-core` closes the rows ST01 and EN04 only as observations). Proposed, for the coordinator to card:
1. d2d: restore saved current life and mana at character load instead of full (fraction dropped as game.exe does). Rows EN04, ST01. Evidence above (12 of 12). Small, one function.
2. d2d: keep the maximum mana fixed point on load (19.5 not 19.0). Rows ST01. Evidence above. Fold into PR 1.
3. d2d: poison resist 6 on Freshe, after the item list is readable (needs TASK-72 `fid-probe-readers`, item flags and sockets). Not ready.

## Matrix rows

ST01 gets the run-2 numbers (appended in the matrix). EN04 and ST07 keep the run-1 evidence. No other row is verified by this run (IT04, IT02, ST06, VN07, MA03 stay unverified: items UNREADABLE).

## What is still open

- A real fresh Paladin and a separate sorc-6 with gear fixture (see above). Items on the d2d side (TASK-72). Rerun of scenarios 1 to 4 with the new normalisation and `-ref` when the fixtures exist. The rebase of the private-branch fixes onto the lab binary, before measuring any d2d fix.
- The d2d repeat count: d2d was restarted 3 times per character, game.exe loaded 3 times per character through `oog leave` and `oog start` (a real second load, not a second read).
