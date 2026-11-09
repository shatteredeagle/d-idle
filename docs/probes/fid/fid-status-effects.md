# fid-status-effects (TASK-87, matrix rows MA11 SK08): monster states, the freeze and chill rule, d2d against game.exe

> status: finished 2026-11-08 by agent decomp6 (Sonnet). Result: **freeze rule C3/C4/C5, chill length, freeze end and Static Field are EQUAL on current upstream main (0 differences, all four merged by the maintainer); the put rule of the timed states (curses, Taunt, Inner Sight, Attract) still differs from game.exe in 54,685 of 100,000 sequences (carried deviations D1 to D5 of covstates2, unchanged by the upstream commits since 2026-11-02).** No PR candidate from this card (the put rule is a missing-rule change and needs a new differential on the fix; not relevant to the early Sorceress or Paladin). Poison state and the stun end were not re-run (U).

Tags: V run (game.exe's own code under the emulator against d2d's real code), V read, D (tables), I, U. Worker host; scratch tree = upstream main a5b970d5 plus the two Lightning commits of the TASK-83 candidate (they change `skill_missile_hits` for LastCollide rows without NextHit only; the freeze path `Fight::strike` is not touched). Scripts `tools/emu/d6_base.sh status`, `tools/emu/d6_cs2.sh`.

## What was upstream already (checked first)

`git log upstream/main`: C5 (b3cd6f8a, "a Glacial Spike freeze is resisted and divided as FUN_0057b230 does"), C3 (50724ee1, "a special monster is chilled, not frozen"), C4 (a777f203, "Blood Raven's and the Countess's freezes aren't halved"), the oracle (1c8c228c `freeze_rules.py` + `test_freeze_rules`), freeze resume (46f77ca4), Frozen Armor striker (36cbc5d3), state 0x36 (43386974), the freeze's end apart from a stun's (e6eeb842, 2026-11-08) and the frozen swing (8a70b282). So `pr-freeze-proof.md` (local candidate `fix/freeze-rules-v2`) is **superseded; do not open it**.

## Q1. The freeze rule (AC #2): 0 differences V run

`freeze_rules.py gen 100000 20261101` (225,748 cases: Glacial Spike's record +0x34 through resistance, pierce, difficulty division, boss / champion / unique / superunique / minion / plain, coldeffect), game.exe FUN_0057c6c0 -> FUN_0057af80 / FUN_0057b230 against d2d's real `Fight::strike` (`test_freeze_rules`): **cases 225,748, mismatches 0, resfail 0.** Same number as the 2026-11-01 run on the candidate before the merge.

## Q2. Chill length of a cold hit, and the cold-immune quirk (AC #3): 0 differences V run

`chill_length.py 100000 {1,7}` against `test_cold_length --cold-length {1,7} 100000` (d2d's real `missile_blow` + `Fight::land`): 121,312 cases each (21,312 boundary + 100,000 random), **0 differ in both seeds**. Coverage (read from the generator): 3 difficulties (MonsterColdDivisor 1 / 2 / 4); cold resistance -150, -100, -99, -50, -1, 0, 1, 25, 50, 75, 90, 99, 100, 101, 150 (random: 10 % at 100 to 150, 15 % at -150 to 0); pierce 0 to 60 (only below 100); MonStats coldeffect signed incl. 0 and negative; a chill already running (prior 0 and nonzero, lengthened only). The Ice Bolt quirk "chill vs a cold-immune monster": at resistance 100 or more the length resisted is 0, FUN_0057af80 does nothing at length <= 0, so **no chill and no damage**; d2d equal (the res 100 / 101 / 150 cases above). Coldeffect 0 monsters (not chillable) also equal.

## Q3. The freeze's end, when the frozen monster thinks again: 0 differences in its domain V run

`freeze_resume.py gen 50000 3` (57,776 cases, 16 MonStats rows' aidel, kinds, a prior freeze, a second freeze) against `test_freeze_resume`: every case where the freeze lands in game.exe (19,130 plus the per-aidel groups, all aidel values 6 to 15 and "0 -> 15") **0 differ**; the 38,646 cases of kinds boss column / champion / unique / superunique are cases where game.exe does not freeze (FUN_0057b230's special branch chills them: Q1) and the test's d2d side applies the freeze function directly, so the table's "differ" there is the harness's domain, not a rule (Q1 covers the real path). Plain and minion kinds: 0 differ.

## Q4. Static Field chill (SK07 neighbour): 0 differences V run

`static_field.py 100000 1` against `test_static_field`: 309,664 lines, 0 differ.

## Q5. The timed-state put rule (curses, Taunt, Battle Cry, Inner Sight, Slow Missiles, Attract): STILL DEVIATES V run

`tools/emu/d6_cs2.sh` (covstates2's `cs2_put.py` game.exe FUN_0056e970 / d2d's real `put_state` through `cs2_d2d_test` built against the scratch tree / the Python model): `d2d-real vs model: differing 0` (the five rules of the model are what d2d does) and **`game.exe vs d2d-real: cases 100,000, differing 54,685`** (accept flags differ in 8,681). The 2026-11-02 number for the same seed 20261102 was 54,685: unchanged. Deviations (covstates2-trace.md, tags there):

| id | rule | d2d | game.exe |
|---|---|---|---|
| D1 | the curse slot | Taunt (27) and Battle Cry (89) in a `cry` slot apart from the curse slot | every curse-bit state (26) shares one node: the last cast replaces the old one |
| D2 | non-curse states | one slot: Inner Sight, Slow Missiles replace each other | one node per state id |
| D3 | Attract (57) | no block | a unit with state 57 refuses every curse put |
| D4 | length | at least 25 frames | kept as given; 0 = never ends |
| D5 | a second cast | Battle Cry on a running Battle Cry does nothing | same skill same level extends, lower refused, higher replaces |

The group removal (D7: Frozen / Chilling / Shiver / Bone Armor, Quickness / Fade) is merged upstream (1f4773f3). Relevance for the bot: Sorceress and early Paladin cast no curse and no shout; the Paladin's auras (Conviction state 28, Holy Freeze, Sanctuary) use other paths (sk-resist-trace.md; Holy Freeze's slow is merged, 22dc3dbe). **Low** for early play; matters for Necromancer, Barbarian and merc curses.

## Q6. The state rows (AC #1, "149 untested")

Rows used by the freeze/chill/curse/stun/poison paths and where they stand (D = States.txt, tags as above):

| State | id | Status |
|---|---|---|
| frozen, chilled | 1, 11 | V run, 0 differences (Q1 to Q3) |
| Frozen Armor | 10 | V run 184,771 cases earlier (armor_hit, merged); Shiver 88, Chilling 20, Energy Shield 30 read only |
| stunned | 21 | V run 38,596 cases, 0 differ in the end (verifycovstates-trace.md, 2026-11-02); not re-run |
| poison | 2 | V run earlier, **deviation in both directions** (5,417 of 39,994 and 2,726 of 19,998; whole-point rates equal), **not re-run on this head (U)** |
| curses 9, 17, 19, 23, 27, 55 to 60, 87, 89, 113, 114 | | Q5 above, V run, deviating |
| lowerresist 61 / conviction 28 | | V run earlier (sk-resist-trace.md): negative change / 5 on an immune (D1 there); not re-run (U) |
| justhit | 86 | V run (NextHit: next_hit.py, TASK-83 Q3: 0 differences) |
| remaining rows of the 185 | | no oracle; list in docs/coverage/rule-coverage.md; U |

## Deviations (d2d against game.exe)

| id | what | tag | differential |
|---|---|---|---|
| D1 to D5 | the put rules of Q5 | V run, 54,685 of 100,000 | yes, built (`cs2_run.sh` / `d6_cs2.sh`); the fix needs a patched `put_state` and the call sites (`fight.cpp`, `states.cpp`); not built here |
| D6 | poison state rate (both directions) | V run earlier, U now | yes (`vcs_run.sh`), not re-run |
| D7 | `curse_resistance` (stat 109) absent in d2d | V run (game side, 640 cases), I (d2d absence) | game side only; the d2d side needs a player with the stat |

## Bot findings (described, not changed)

B1 No decision code reads a monster's chill, freeze or stun state (grep of `internal/combat`, `internal/danger`: only the player's armor states, `castlog.go:128`, `necro.go:33-42`): Ice Bolt is chosen by damage and element, never for the slow. Not a bug, a missed value (chilled monsters walk at the coldeffect %, V in walk-speed-trace.md). With this card's result the chill length is known to be game.exe's, so a future "kite a chilled monster" rule can rely on it.
B2 `internal/combat/rules.go:238` (`data.Cold: {"Holy Freeze"}`) lists Holy Freeze as the cold counter; its slow is merged upstream (22dc3dbe) and equal, nothing to change.

## PR candidate

None. The freeze rule is already upstream; the put-rule deviations D1 to D5 are rule additions with no early-game effect and no fix built against the existing differential (`mismatches before > 0, after 0` not demonstrated), so nothing is prepared.

## Drill recipe (lab binary, docs/forkdbg.md)

1. Chill: Sorceress, `debug spawn zombie1 1`, `debug skill` Ice Bolt, cast; `dump monsters` / `debug states` for state 11 and its end tick: expect chill = (ELen x resisted) / MonsterColdDivisor frames from the oracle's `chill_length.py` for the monster's coldeffect and the cold resistance; on game.exe the monster's walk speed 100 + coldeffect %.
2. Put rule: `debug states set` Amplify Damage then Taunt (or Battle Cry) on a spawned Fallen: game.exe keeps only the later (one curse slot); d2d keeps both (D1).
3. Attract: put state 57 then a curse: game.exe refuses it, d2d applies it (D3).
