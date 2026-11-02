# Swing speed: IAS, WSM and the attack frame count (SKIAS)

> status: PARTIAL, desk trace. No game.exe code was run for this item (the job had no emulator budget), so **no number here is V run**. The d2d side is read and its arithmetic is reproduced by a script; the game.exe side is carried from existing traces and flagged U where nothing was read. Honest limit, not a finding.

## Questions

| Q | Question | Answer | Tag |
|---|---|---|---|
| Q1 | What does d2d compute? | `effective_speed(ias) = 120 x ias / (120 + ias)` (ias > 0); `EIAS = clamp(effective_speed(ias) - WSM + skill_ias, -85, 75)`; `speed = max(rate x (100 + EIAS) / 100, 1)` (integer); `ticks = ceil(256 x frames / speed)` (components/rules/combat.hpp:170-191) | V read (d2d) |
| Q2 | Where is it used? | swing length via `attack_ticks` (components/game/fight.hpp:450, :509); monsters take `attackrate` as IAS (riskmap: fight.hpp:515) and `0062a710` attack frames are taken as attack ticks (fight.hpp:787) | S docs/research/d2d-fidelity-riskmap.md |
| Q3 | What does game.exe compute? | The mode-rate function `00623f50` is the single place that stores the animation rate (+0x4c). Its moving branch and the get-hit branch are V read / V run in movement-and-recovery-trace.md (rate = AnimData speed x (stat-derived % + 50) / 100, effective stat with factor 120). Its attack branch (modes 5 and 6, the IAS / WSM terms) was NOT read in this job | S for the other branches, U for the attack branch |
| Q4 | Breakpoints for any class | Frames and rate come from the class AnimData and PlrMode tables (rate 256 is the d2d default). Generic d2d table produced by `tools/emu/skias_bp.py FRAMES RATE WSM [SIAS]` | D (d2d arithmetic only), U against game.exe |

Reproduce (pure arithmetic, no seed, exhaustive over IAS 0..200):

```
python3 -I tools/emu/skias_bp.py 16 256 0    # example: 16 frames, rate 256, WSM 0
```

Output for 16 frames / rate 256 / WSM 0: ticks 16 at IAS 0, then 15, 14, 13, 12, 11, 10 first at IAS 9, 18, 30, 48, 75, 125. With WSM 20 the first ticks are 21, 20 (IAS 2), 19 (IAS 6), 18 (IAS 11). Tag D (it checks the script against the d2d formula as written, not against the game).

## Deviations d2d vs game.exe

| D | Rule on both sides | Tag | Differential |
|---|---|---|---|
| D1 | d2d's EIAS clamp (-85..75) and `effective_speed` cap 120 are the 1.10 community rule; game.exe's attack branch of `00623f50` is unread, so whether 1.14d keeps the same cap, clamp and integer division order is open | U | could be built: run the attack branch of `00623f50` for modes 5 / 6 under tools/emu (pattern: tools/emu/cast_rates.py and the walk `rate` oracle), inputs = player with weapon speed and stat 93 (IAS) / 68 (WSM item field) / skill speed set directly, AnimData speed from the class table, oracle = d2d `attack_speed` on the same inputs; 100,000 random cases plus every IAS 0..200 x WSM -60..+60 |
| D2 | Skill speed bonus (`sias`): d2d adds it after WSM, uncapped before the clamp; game.exe's order (before or after the effective-IAS cap) not read | U | same differential, vary the skill bonus |
| D3 | Monsters: d2d takes `attackrate` as IAS and `0062a710` frames as ticks (riskmap) | S (riskmap lists it as unmeasured) | build with the monster attack branch; part of the monster-attack trace work |
| D4 | The tick unit: d2d's `ticks` rounds up with `speed` floored; game.exe's per-tick animation step (rate/256 frames a tick, then the frame at which the swing event fires) not read | U | same differential, count ticks until the animation ends |

## Bot findings (described, not changed)

| B | Finding |
|---|---|
| B1 | The resource simulator models melee as one swing per 1.0 s (see sk-melee-trace.md B1, internal/resource/sim.go:94); there is no frame count, IAS or WSM term, so any breakpoint gear choice is not represented. Not changed. |

## Matrix cells proposed

None: the sorc skill matrix rows for plain Attack keep "Speed / IAS not traced [U]" (sk-melee-trace.md line 108) until the D1 differential runs.

## Not covered (limits)

The game.exe attack branch (Q3), class AnimData tables from the MPQ (needs the table dump on the worker host), Whirlwind / Frenzy / Fury style skill speed, Zeal (multi-hit sequences), the throw and bow modes, and the difference between the first and later frames of a swing. Coverage ledger: no row proved, so docs/coverage/ledger.tsv is not touched.

## Drill recipe (lab binary, docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md)

Not run. In the lab: give a Sorceress a weapon of known WSM and gear giving a known IAS, swing at a dummy target 20 times and read the frame count between the attack start and the hit event from the fork's unit dump; repeat at IAS values straddling the table above (8/9, 17/18, 29/30) on d2d and compare with the game's own client. Confirms D1 and D4 when game.exe is available as the oracle.

## Shared-file lines for the merger

See openspec/changes/skias/merge-lines.txt.
