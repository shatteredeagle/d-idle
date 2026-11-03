# SORCDIFF: which Sorceress skills the lightning build uses, and what is proven

> status: research, 2026-11-03, agent SORCDIFF (card sorcdiff). Offline (no game). Counted differentials ran on the real d2d build (upstream main `005c7cab`) against game.exe's own code in the emulator, on the build-worker WSL lane. Nothing pushed upstream, no PR, no comment. Tags: **V** verified (run or read in this card, the way is said), **S** secondary (an earlier doc of ours, not re-run), **I** inference, **U** unverified.

## Answer

1. **The gate character is the lightning Sorceress**: every committed Gate 1 minute archive says plan table `fallback-lightning-v2` (V: `zcat docs/probes/soak-logs/gate1*-minutes.jsonl.gz | grep -c '"table":"fallback-lightning-v2"'` on the eight archives), the combat profile `sorceress/lightning` (V: `internal/combat/classdata/profiles.json`). A Barbarian job (barbdiff) was a mistaken premise.
2. **No gate run got past clvl 15** (V: last minute line of the eight Gate 1 archives: clvl 14 or 15; 55,449 kills, 10,176 virtual minutes in all). So what the bot really casts in a gate is **Charged Bolt (the main attack), Fire Bolt (start skill, mana-out fallback), Frozen Armor, Frost Nova** and nothing else. Teleport (clvl 18), Chain Lightning, Lightning, Lightning Mastery (clvl 26-30 and up) are the plan's later bands and were never reached in a measured run.
3. **Energy Shield and Thunder Storm are not used by the build at all** (V: build json has no point in either; the lightning profile lists Energy Shield only as a second buff group that needs a learned skill; Thunder Storm appears in no profile; `grep` of `internal/` and `cmd/` finds no cast path). The brief expected them near the top of the unproven list; by the bot's own table they are never bought, so their read-only status (D3, D4 of the armor trace) costs the bot nothing today. Telekinesis is one prerequisite point that is never cast (no F-key), Warmth one point at clvl 91.
4. **Of the skills the bot uses, one proven deviation was still open on upstream main `005c7cab`: the cast length of Lightning and Chain Lightning** (19 ticks in game.exe at 0 FCR, 13 in d2d). Counted on the real build: **802 of 3,609 compared lines differ on main, 0 on the fix** (all 401 FCR values of both skills; the seven plain casts, 2,807 lines, equal). Fix: private branch `local/sorc-lightning-seq` (two commits). It matters from clvl 26-27 (Chain Lightning is first in the attack order there), so it did not touch any gate run so far.
5. Everything else the bot uses is either **merged upstream**, **in a held batch**, or **clean on re-run** (table 2). What is left unproven for used skills is **stateful** (mana paid at the action frame, a cut cast) or has **no harness** (Lightning's piercing, units per frame): the ready-for-Opus list at the end.

## 1. Which skills the bot uses (V unless tagged)

Sources: `internal/plandata/build-fallback-lightning-v2.json` (points per level, summed by script), `internal/combat/classdata/profiles.json` (attack order, buffs, F-keys), `internal/combat/skills.go` (Telekinesis dropped from the keys), `internal/leveler/build.go`, the matrix `docs/research/sorc-skill-matrix.md` (S) and the gate minute archives.

Table v2 points (V, summed): Charged Bolt 20 (first at clvl 2), Chain Lightning 20 (27), Lightning 20 (40), Lightning Mastery 20 (26), Static Field 13 (clvl 86 only), Frozen Armor 2 (clvl 5, 22), Frost Nova 1 (6), Telekinesis 1 (11), Teleport 1 (18), Warmth 1 (91). Fire Bolt is the start skill. The leveler refuses a spend while ReqLevel > clvl and a missing prerequisite, so the first Lightning Mastery point is at clvl 30 at the earliest and Chain Lightning waits for Lightning (sk-passives-trace B1, S).

Attack order from clvl 26 (and when Chain Lightning is known): Chain Lightning, Lightning, Charged Bolt, Fire Bolt; below that Charged Bolt, Fire Bolt, Ice Bolt (V profile). Buff groups: Frozen Armor / Shiver / Chilling, then Energy Shield (never learned). Other casts: Frost Nova on a crowd of 3 or more within 4 subtiles with 25 mana, Teleport in nav hops (A2 on), Static Field only on legacy characters (not on v2).

### Cast frequency (what data exists)

The archived minute lines carry kills, clvl and area but **no cast counts** (V: keys of a minute line), and the full bot logs of the gate runs are not in the repo. So casts are ranked from kills and clvl time, with one measured rate:

| clvl | virtual minutes (8 runs) | kills (8 runs) |
|---|---|---|
| 1-5 | 868 | 3,909 |
| 6-10 | 2,376 | 12,248 |
| 11-13 | 3,775 | 21,639 |
| 14-15 | 3,058 | 17,653 |

(V: computed from the eight `gate1*-minutes.jsonl.gz` files, sum of kills between consecutive minute lines per clvl; total 55,449 kills.) Charged Bolt real casts per Fallen kill were 1.16 / 0.73 / 0.61 at Charged Bolt 1 / 4 / 8 on d2d (V-d2d, `docs/d2d-casts-real.md`, earlier agent), so the eight runs cast Charged Bolt about **34,000 to 64,000 times** (I: kills x 0.61..1.16). Frozen Armor: one cast per buff edge (new game, after a town trip, a death; Fighter.Buff only from sweepPass, sk-armor-trace B1, S), so hundreds, not tens of thousands (I). Frost Nova: no count (U). Fire Bolt: clvl 1 and the mana-out fallback (U, small). Telekinesis, Teleport, Chain Lightning, Lightning: 0 in the gate runs.

Rank by how often the bot casts it in a measured run: **1 Charged Bolt, 2 Frozen Armor, 3 Frost Nova, 4 Fire Bolt**, then (not reached) Teleport, Chain Lightning, Lightning, Lightning Mastery (a passive: affects every Lightning and Chain Lightning hit).

## 2. Status per skill (state of upstream main 005c7cab)

Ledger columns are `docs/coverage/ledger.tsv` rows `skills.sor` (V read); "re-run" is table 3 below (V, this card).

| id | skill | bot use | ledger | proven clean on main (counted) | proven deviation, PR state | read-only or untested |
|---|---|---|---|---|---|---|
| 38 | Charged Bolt | main attack to clvl 25 | TESTED-DEVIATION (SKMISS) | roll and hit pipeline (missile_hit), element damage with synergies (skill_damage), calc1 of the count (22,400 calc cases, sorcrest, S), wiggle (upstream missiles.py), cast length (this card, 401 FCR) | D1 roll top inclusive: merged in #166 (S, pr-candidates-20261102) | count and spawn equal by read (S); mana paid at the command, game.exe at the action frame: SKMISS D2, **stateful, no harness** |
| 36 | Fire Bolt | start skill, fallback | same row | as above | D1 as above | same |
| 40 | Frozen Armor | buff, all runs | TESTED-DEVIATION (SKARMOR) | striker effect: armor_hit (re-run, table 3); cast length | D1 striker effect: merged in #159 (S, git log of 005c7cab: 36cbc5d3, 50724ee1) | D2 armor group, D5 trigger (any damage vs physical record): read only; plan casts one armor |
| 44 | Frost Nova | crowd cast | TESTED-DEVIATION (SKMISS, SKNOVA) | ring aim (merged), justhit hits (next_hit, re-run), freeze rule (freeze_rules, re-run) | D1 NextHit justhit: merged in #158; D2 ring directions: merged (02055e45) | D3 Howl only; D4 units per ring missile per frame folded into D1; D6 monster novas on the player U |
| 42 | Static Field | legacy characters only | TESTED-DEVIATION | static_field and chill_length (re-run) | merged #155 / #156 | none known |
| 49 | Lightning | clvl 26+ band | same row as Charged Bolt | roll, damage, calc | **cast length D3: this card, `local/sorc-lightning-seq`** (802 -> 0 with Chain Lightning) | **D2 piercing, units per frame: read on both sides, a model only (sk_light_rehit.py, tag I), no real-build harness** |
| 53 | Chain Lightning | clvl 26+ first pick | same | hop pick FUN_0056b7e0 merged; justhit merged #158; calc1 (hops) | cast length D3: this card | D4 hop pick order (unit id vs vector index): S, equal while creation order |
| 63 | Lightning Mastery | passive, clvl 30+ | READ-ONLY (stale: row predates skill_damage) | mastery share of the element damage is part of skill_damage.py (masteries 0/50/98/158/278 in the case grid, #146 merged) (re-run, table 3) | none | mastery stat at creation once: S |
| 54 | Teleport | nav hops, clvl 18+ | TESTED-DEVIATION (SKTELE) | cast length (this card) | D1 room, D2 blocked target: **batch 2 (not open upstream yet; branch local/batch2-bundle-r3, private)** | D3 monster at target, D4 mana on a refused hop (stateful), D5-D7 U |
| 43 | Telekinesis | point only, never cast | READ-ONLY | cast length of the cast path (this card) | none | D1 knockback 35 percent vs 100: unused by the bot |
| 37 | Warmth | clvl 91 only | READ-ONLY | none | none | regen model: U, unused before clvl 91 |
| 58 | Energy Shield | profile only, never learned | READ-ONLY | none | none | D3 stays at 0 mana, D4 arithmetic: **unused**, no harness |
| 57 | Thunder Storm | not in any profile | READ-ONLY | none | none | unused |

The unused skills of the Sorceress ledger rows (Inferno, Blaze, Fire Ball, Nova, Shiver Armor, Fire Wall, Enchant, Meteor, Chilling Armor, Fire Mastery, Hydra, Frozen Orb, Cold Mastery, Ice Blast, Glacial Spike, Blizzard) are not part of the lightning build; their status stays as in the ledger.

## 3. The counted differentials (real d2d build on the build-worker WSL lane)

Setup (V): a git bundle of the private fork carries `local/sorc-base` (= upstream main `005c7cab`) and `local/sorc-lightning-seq`; build-worker WSL (Ubuntu 26.04, g++ 15.2, ccache, game files and emulator venv as in the ops repo build-worker doc); two builds: A = harness commit only (main plus `tests/game/test_cast_seq.cpp`, `tools/emu/cast_rates.py cast`), B = A plus the fix; game.exe's side is `tools/emu/cast_rates.py` and the other oracles, run natively under unicorn on the same machine. Script: `tools/emu/sorcdiff_build-worker_run.sh` (stages `build`, `cast`, `harness`). One job at a time was the rule; see "the build worker overlap" at the end.

### 3.1 Cast length of the plan's Sorceress casts through the REAL `Fight` (new)

`test_cast_seq --cast-ticks` starts each cast through `Fight::cast` / `cast_missile` (the same rig as `test_sk_mana`), with `player_combat.fcr` set to 0..400, and prints the ticks the cast mode lasts (`pmode_until` less the cast tick over 40 ms). game.exe's side: `cast_rates.py cast` (FUN_00623f50 cast branch, FUN_00621260 / FUN_006217c0 / FUN_00623e00 stepping, as the existing `cast_rates.py` default, which also passes on both builds: 0 differing lines of 2,807 on A and B).

| build | compared lines (9 skills x 401 FCR) | differ | by skill |
|---|---|---|---|
| A: main + harness (`2f06b25c`) | 3,609 | **802** | Lightning (49) 401, Chain Lightning (53) 401; Fire Bolt, Charged Bolt, Frozen Armor, Static Field, Telekinesis, Frost Nova, Teleport 0 |
| B: head (`e4cf2253`) | 3,609 | **0** | none |

Examples (V, line `cast ID FCR TICKS`): game.exe `cast 53 0 19`, `cast 53 7 18`, `cast 53 194 11`; d2d on main plays Lightning and Chain Lightning as a plain 14-frame SC cast: 13 ticks at 0 FCR. Effect: d2d over-rates the Chain Lightning band by 32 percent in cast time (I, SKLIGHT D3: 13 against 19 ticks); not seen in any gate run yet (clvl 26+).

### 3.2 Re-runs of the existing harnesses on `005c7cab` (build A; the skills the bot uses)

Run on build A (main plus the harness commit), stage `harness2` of `tools/emu/sorcdiff_build-worker_run.sh` (V, alone on the build worker, 299 s), except the first row (stage `harness`, shared the build worker, finished before the VM reset killed the rest). Counts are compared cases; "differ" is game.exe against the real d2d build.

| harness (skills it covers) | compared | differ |
|---|---|---|
| skill_damage gen 100000 1 (element damage with synergies and masteries: Fire Bolt 36, Charged Bolt 38, Static Field 42, Telekinesis 43, Frost Nova 44, Lightning 49, Chain Lightning 53, Thunder Storm 57, with Lightning Mastery levels) | 116,000 (about 1,120 to 1,250 per bot skill) | 0 |
| static_field.py 20000, seeds 1 and 7 | 229,664 lines each | 0 and 0 |
| chill_length.py 20000, seeds 1 and 7 (cold length) | 41,312 lines each | 0 and 0 |
| next_hit.py, test_next_hit seeds 1 and 7, N 1500 (Frost Nova ring hits, justhit) | 5,027 and 4,964 H/T lines | 0 and 0 |
| armor_hit.py gen 20000 1 (Frozen Armor striker rule, by monster class) | 109,604 | 0 in every class |
| freeze_rules.py gen 20000 1 (Frost Nova / Glacial Spike freeze) | 145,748 | 0 |

Not re-run: missile_hit (60,000 cases, killed by the the build worker VM reset; merged state S from #166). Static Field, cold length, nova hits, Frozen Armor and element damage are therefore clean on main `005c7cab` for these case sets. Smaller N than the 100,000 of the first plan, because three the build worker jobs ran at once and the VM had to be reset.

## 4. Fix branch (private fork, the private remote only)

`local/sorc-lightning-seq` on upstream main `005c7cab` (base branch `local/sorc-base`), numeric noreply identity for author and committer, bodies of at most three lines, last line `Agent: Claude Code`, one change per commit:

| commit | subject | files | claim |
|---|---|---|---|
| `2f06b25c` | tests: a Sorceress cast's length in the real Fight vs cast_rates.py | `tests/game/test_cast_seq.cpp`, `tools/emu/cast_rates.py`, `tools/emu/README.md` | harness: base 802 of 3,609 differ |
| `e4cf2253` | game: Lightning and Chain Lightning cast at the sequence's length | `rules/combat.hpp` (`sequence_ticks`), `rules/skills.hpp` and `gamedata_load.cpp` (`seqtrans`), `fight/setup.cpp` (`set_pmode`), `test_combat.cpp`, `test_cast_seq.cpp`, `fight.hpp` comment | head: 0 of 3,609 differ; `test_combat` asserts 19 / 18 / ... / 11 ticks (the old candidate `fix/lightning-sequence` 2036fd3f, re-made on the new file layout) |

Not done by the fix (ponytail comment in `set_pmode`): the missile still leaves at the SC action frame, not at the sequence's event-1 step (the 8th), and `start_sequence` is not run for casts. Maintainer rules (`docs/research/pr-upstream-rules-20261103.md`) are met by shape; the branch was not run through `tools/presubmit.py` or the tidy stage (U). Upstream main has moved to `7d999228` since (PR 180); the branch is on `005c7cab` as the brief said, a rebase has not been tried (U).

## 5. Ready-for-Opus list (ranked by how often the bot meets it)

1. **Mana paid at the action frame (SKMISS D2) for Charged Bolt and every other cast.** d2d pays at the command; game.exe pays in the do at the action frame, so a cast cut by a get-hit costs mana on d2d only (about 7-9 ticks earlier drop). Stateful: needs a drill or a tick-level harness around `Fight::tick`; the arithmetic (check, floor, payment) is batch 2 commit 1 and proven (S, `pb2_mana.sh`).
2. **Lightning's piercing, units per frame (SKLIGHT D2, SKNOVA D4).** One unit a frame in game.exe against every touching unit in d2d: 3.15 vs 2.86 hits per cast in the model (I, 20,000 casts). Buildable like `next_hit.py` (FUN_005ae1f0 and FUN_005a8730 natively, d2d flights from `test_next_hit`); the `next_hit.py` harness handles NextHit rows only (`nova`, `frostnova`), so it needs the `lightningbolt` row and a pierce line. Matters from clvl 40 (Lightning first point in the planner; B1 of sk-passives-trace).
3. **Frozen Armor D5 (trigger) and D2 (armor group).** Trigger `damage > 0` points vs physical record > 0; one group-1 state in game.exe. No harness; small.
4. **Energy Shield D3 / D4, Thunder Storm, Telekinesis D1, Warmth regen.** Not used by the build; only worth a harness if the plan buys them (the table does not).
5. **Chain Lightning on an immune target (SKLIGHT B2)**: bot-side (it drops Chain Lightning for an immune), not a d2d deviation; from NM.

## 6. Notes

- **the build worker overlap.** Three jobs shared the VM while the first `harness` stage ran (my skill_damage took 22 min); the VM was reset by the coordinator and that stage was killed. `build`, `cast` and `harness2` ran alone.
- **Ledger rows to correct (proposed, not edited)**: `skills.sor 43,37,61,63,65` says READ-ONLY for the masteries, but the mastery share of the element damage is inside the `skill_damage.py` differential (merged #146); only the passive stat bookkeeping and Warmth stay read-only.
- **Premise corrections for the brief**: Energy Shield and Thunder Storm are not cast by the bot, Charged Bolt is the skill that decides a gate run, and the cast counts are not in the minute lines.

## Shared-file lines for the merger

See `openspec/changes/sorcdiff/merge-lines.txt`.
