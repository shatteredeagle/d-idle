# Monster AI, third pass: the Act 2 to 5 thinks and the bosses (agent covmonai2, 2026-11-02)

> status: research, current. Continues docs/research/monster-ai-trace.md (pass 1: 25 AIs, section 12 lists all 146 AIs with think addresses) and docs/research/monsterai2-trace.md (pass 2: D1..D9 and B1..B4 there are that card's; here D1..D5 and B1..B2 are this card's). Tags: V read = read in the game.exe disassembly (`mai2_dis.sh`, addresses given); V run = game.exe's own think run under the emulator against d2d's real `rules::mon_think` (counts below); D = 1.14d patched tables (`mpq-table.sh monstats`); S = an existing upstream or earlier-trace statement (named); I = inferred; U = unverified. d2d = upstream f8aab0e8 (`components/rules/monsters.hpp`), read in a scratch tree on the worker host. Offline only: no game, no slot, nothing pushed or posted. The task brief's "covmonai-trace.md" does not exist in the tree; the first pass is monster-ai-trace.md and the second monsterai2-trace.md.

## Verdict

1. **81 of the 121 untested AI rows are now run-tested; 78 equal d2d, 3 deviate.** game.exe's think was run natively for 83 AIs (Act 2 to 5 monsters, the Act 2 to 4 bosses Duriel, Summoner, Mephisto, Izual, Diablo, Baal's minions, towers, traps) and replayed through d2d's real `mon_think` on 139,500 cases (final runs, seed 20261105; V run). Compared per case: the act, its numbers and unit, the pace, the effects, the three AI control words and the unit seed after. **Duriel, Summoner, Mephisto, Izual, Hydra, Trapped Soul and Uber Izual: 0 of 3,000 differ each** (V run). Diablo: 61 of 3,000 differ (D4, D5, and 18 not isolated). Arach 378 of 1,500 and Fetish 64 of 1,500 differ (D1 to D3). 78 names have 0 differences (27,000 + 42,000 + 60,000 + 1,500 cases in the four ledger lines). Two runs stay without a ledger line: Baal Crab (2 of 1,500 differ) and Vampire (11 of 1,500), neither isolated (U).
2. **Five deviations, none a PR candidate** (D1..D5): all in the think's output, found on this card's harness, not the upstream one; no fork branch could be made; the effect of each is U or I (section "Deviations").
3. **Six harness artifacts were found and fixed before any difference was called a finding** (section "Dead ends"): the first runs showed 1,634 differences on 121,500 cases (seed 20261103); the final runs show 516 on 139,500 (the case sets differ because the generator's sequence follows the AI list). The removed ones were labels and unreachable scenes, not rules.
4. **Boss think parameters (D)**: Duriel has aip4 and aip5 empty, so his out-of-melee branch can only walk (his Charge is never cast by the think, I from D plus the equal think); every boss' think period (`aidel`) shrinks with difficulty (Mephisto 15 / 10 / 6 frames, Izual 15 / 12 / 8, Summoner 15 / 13 / 10, Diablo 15 / 13 / 12). The bot's `rangedtype` flag is empty for all of them (B1).

## Q1. Which thinks, which rows (S for the addresses: monster-ai-trace.md section 12; D for the columns)

Every AI below has its think at MonAI row + 8 (address column); the d2d port is a branch of `mon_think` (all 83 are in `traced_ai`, S section 12; V run here: none returned `untraced`). F4 below is Bighead and Vampire, two Act 1 and 2 rows upstream's `mon_ais.py` does not list (added by address from section 12). Run groups: F1 3,000 cases per AI, F2 and F3 1,500 per AI; raw output `docs/research/covmonai2-runs.txt`.

| group | AIs (think address; differ of cases) |
|---|---|
| F1 bosses, 3,000 each | Duriel 5f67b0 0; Summoner 5f85c0 0; Mephisto 5f78b0 0; Izual 5f89b0 0; Diablo 5e9170 **61**; Uber Izual 5f8c80 0; Uber Mephisto 5f81c0 0 and Uber Diablo 5e9df0 0 (both a bare `ret 4`: game.exe does nothing, `none` in all 3,000, so equal by emptiness); Hydra 5e9e60 0; Trapped Soul 5e9f10 0 |
| F2 Act 2, 3, 4, 1,500 each | Mummy 5f2850, Greater Mummy 5f2b10, Vulture 5f3170, Baboon 5f0cd0, Sand Raider 5f0700, Claw Viper 5f1b60, Sand Leaper 5f20a0, Scarab 5f2540, Swarm 5f2460, Panther Woman 5f22b0, Panther Javelin 5e1080, Pin Head 5f6340, Sand Maggot 5f1800, Sand Maggot Queen 5f9cf0, Maggot Egg 5f6530, Maggot Larva 5f6220, Mosquito 5f3730, Thorn Hulk 5f4850, Will-O-Wisp 5f39b0, Zakarum Zealot 5f6e60, Zakarum Priest 5f72d0, Frog Demon 5f8260, Tentacle 5f8f80, Tentacle Head 5f9270, Fetish Shaman 5f9a80, High Priest 5e0490, Fetish Blowgun 5e1250, Bat Demon 5f5040: **all 0 differ**; Arach 5f4510 **378**; Fetish 5f53e0 **64** |
| F4 Act 1 and 2, 1,500 each | Bighead 5eff50 0; Vampire 5f4a70 **11** (not isolated, U) |
| F3 Act 4 casters, Act 5, 1,500 each | Vile Mother 5fa010, Vile Dog 5fa280, Finger Mage 5fa380, Regurgitator 5fa710, Doom Knight 5faa90, Abyss Knight 5fab80, Oblivion Knight 5faf00, Megademon 5e0c80, Reanimated Horde 5e1540, Siege Beast 5e1900, Minion 5e1b60, Suicide Minion 5e1d30, Succubus 5e1e00, Succubus Witch 5e2120, Overseer 5e27a0, Minion Spawner 5e2bd0, Imp 5e2ff0, Catapult 5e34c0, Frozen Horror 5e3530, Blood Lord 5e36f0, Catapult Spotter 5ee040, Npc Barb 5edc50, Generic Spawner 5e61b0, Death Mauler 5ee260, Putrid Defiler 5efa90, Baal Tentacle 5ef820, Baal Minion 5ef910, Ancient 5ef1a0, Nihlathak 5ee5d0, Claw Viper Ex 5f1de0, Baal Crab Clone 5fd210, Uber Baal 5fd200 (bare `ret 4`), Siege Tower 5e1860, Hell Meteor 5f56d0, Sarcophagus 5f6a10, Spirit 5e3840, Arcane Tower 5e0f60, both arrow traps 5fb6c0 / 5fb7e0, Desert Turret 5e0980: **all 0 differ**; Baal Crab 5fcfe0: 2 differ (U) |

The Act 2 rows the bot meets in Normal (mummies, vultures, sand raiders, scarabs, baboons, panthers, Claw Vipers, maggots, Pin Head) are all in F2 and equal d2d: their rules in the first pass's section 12 (placement) and the aip columns below. Param columns (D, `monstats` Normal / Nightmare / Hell; `aip1..aip8` are the think's percentages and distances, `aidel` the think period in frames):

| row | Vel / Run | aip1 | aip2 | aip3 | aip4 | aip5 | other | aidel |
|---|---|---|---|---|---|---|---|---|
| duriel | 10 / 15 | 5 / 5 / 6 | 33 | 50 | - | - | Skill1 Charge (seq), Skill2 Jab (seq), Skill3 Smite (seq), Skill4 Holy Freeze (aura, level 1) | 15 / 15 / 15 |
| mephisto | 8 / 8 | 15 / 20 / 25 | 25 / 33 / 33 | 25 / 33 / 33 | - | - | six skills, all mode A2: Prime Lightning, Prime Bolt, Prime Poison Nova, Mephisto Missile, Frost Nova, Blizzard | 15 / 10 / 6 |
| summoner | 5 / 5 | 85 / 93 / 98 | 5 | 63 | 40 | 120 / 100 / 80 | aip6 33 / 20 / 10, aip7 5 / 8 / 11, aip8 40; Glacial Spike, Frost Nova, Fire Ball, Vampire Firewall, Weaken (all SC) | 15 / 13 / 10 |
| izual | 10 / 16 | 45 / 50 / 50 | 50 | 66 | - / 75 / 100 | 20 / 5 / - | aip6 3 / 4 / 4; Frost Nova (SC) | 15 / 12 / 8 |
| diablo | 6 / 6 | - | - | - | - | - | no aips: his act comes from the weighted chooser; seven skills (Lightning SC, Cold S2, Fire S1, Wall S3, Run seq, Prime Firewall S3, Prison S3) | 15 / 13 / 12 |
| mummy1 | 3 / 3 | 5 / 9 / 15 | 60 | 100 | 65 | 10 | | 15 / 14 / 13 |
| vulture1 | 4 / 4 | 70 / 75 / 80 | 8 | 75 / 65 / 50 | 30 | 40 | | 15 / 14 / 13 |

The aip values of the Normal column of Duriel are 5, 33, 50 (the columns `aip1(N)`, `aip1(H)` are the Nightmare and Hell values; in the first pass's table the Normal and Hell values were printed).

## Q2. What the bosses do (V read of d2d's port, equal to game.exe by V run; numbers from the table above)

d2d's port was written from the disassembly (its comments cite each `FUN_`); the differential shows game.exe's own think and the port agree on every case of the run, so the rules below hold for both (V run for the equality, D for the numbers, I for what they mean in play).
- **Duriel** (`0x5f67b0`): his aura (Skill4 Holy Freeze) is started when it is off; out of melee `roll < aip5` casts Charge, else walks in; in melee `roll < aip2` Smite, else `roll < aip3` Jab, else `roll < aip4` A2, else A1 (monsters.hpp:1797-1806). With aip4 and aip5 empty (D), **he never charges from the think and never swings A2: Smite 33 %, then Jab 50 % of the rest, else A1** (I from D plus the equal think).
- **Summoner** (`0x5f85c0`): the first think opens his quest (state 1); within 5 and `roll < aip6` he backs off 6; `roll < aip1` (85 / 93 / 98 %) he casts, else wanders 4: `aip2` 5 % Weaken; the element the target resists less (cold when its cold resist <= fire resist), flipped when `aip3` (63 %) is below the roll; Frost Nova ready after `aip4` frames within `aip7`, Glacial Spike or Fire Ball at the second target within `aip8`, Vampire Firewall every `aip5` frames, each at the target's spot (monsters.hpp:2107-2135).
- **Mephisto** (`0x5f78b0`): `hurt = (100 - life %) / 5`; picks a case and keeps it in state 3: past 20 off case 4 (circle the target 6); under 21 % life within 5, 40 % case 1 (runs off 8); under `hurt + 50` % case 2 (casts in a row); else a fallback (35 % stand 10, else within 6 walk in, else circle 4). Casts go to his chooser's unit (`FUN_005f77c0`): `aip2` % the weakest within 32, else with a player nearest `aip3` % the weakest again, **where game.exe returns the wrong find (d2d's comment cites bugs.md #21 and reproduces it; the differential agrees: 0 of 3,000)**. Skills in a row from Skill1: past Normal with no line or past 30 off Skill6 (Blizzard); under 15 off past Normal Skill5 (Frost Nova); then Skill4, 3 times Skill2, else Skill1 (monsters.hpp:2579-2596).
- **Izual** (`0x5f89b0`): the first think greets (a4q1 dialog); in melee swings left or `aip1` % A1, else Frost Nova at the target's spot at `aip4`; out of melee within 10 the nova at `aip3` %, `aip2` % walks in, else stands or keeps 6..9 off (monsters.hpp:3146-3160).
- **Diablo** (`0x5e9170`): his kept point is his own spot the first think; a pending act (state) first, else the chooser's weighted pick over 17 acts (weights depend on reach, line of sight, the target's resists and life, the count of players looked at, the score), rolled on the unit seed; Skill8 first (not act 5) at the target. Acts: 1 slow move to the target's spot (pace 20), 2 A1, 3 A2, 4 S4, 5 Skill1 (state 5 pending, state 0xc off), 6 Skill3, 7 Skill2, 8 Skill4, 9 Skill7 (prison), 10 Skill5, 12 circle 4, 13 Skill6, 14 back to his point (pace 50 / 100), 15 Skill7 at the target player's command unit (monsters.hpp:2939-3038; the disassembly's switch at `0x5e92f1`, table `0x5e9704`, V read).
- **Hydra** (`0x5e9e60`, the Act 3 hydra): fixed, `gone` in 2,858 of 3,000 cases (a timed mode), skill or idle else (V run; equal).

## Q3. Deviations found (V run; the d2d side is the replay of rules::mon_think, the game side is game.exe's own code)

See the Deviations table. In short: Fetish back-off pace (D1), Arach charge as run (D2) and lay unit (D3), Diablo's pending Skill1 unit (D4) and act 15 (D5).

## Q4. The three groups' common caveats (what the 0 means)

The case line fixes the scene: the AI row's aips, skills and modes, the difficulty, the frame, the distance and melee flags, life %, the AI control words, spots, leader and group state. Answers the harness gives to calls outside the think (the claim excludes them): got-hit (`FUN_005dd2b0`), life % (`FUN_00621f20`), the second target (`FUN_005ddc30`: set equal to the primary one at the same distance, as `ai.cpp` feeds it), the search results (`FUN_005dd0b0`), skill usable (`FUN_005fd470`), the distance helper `FUN_006416d0` (answers the scene's distance; hooked, not run), the room scan `FUN_00619790` (empty) and the units' stats. 0 differences therefore means: given the same inputs the two thinks choose the same act with the same numbers, pace, effects, control words and seed; it does not cover how `ai.cpp` produces those inputs (limit L1). Where d2d computes a distance from positions (Ancient, Diablo's point tests) the positions were made to agree with the scene's distance (Ancient) or are U (Diablo).

## Deviations, d2d vs game.exe

| # | Rule in game.exe | d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | Fetish: the back-off (14 off, pace 50 set first) that cannot set off leaves pace 50 pending (the pace is a pending request, `FUN_005de190` stores it whatever follows; S pass 2 Q6) | `*input.pace = 50; if (!away(14, false)) *input.pace = 0;` (monsters.hpp:1752-1753) | V run: 57 of the 64 Fetish differences in 1,500 cases are exactly "back-off asked 14, ok 0, pace 50 vs 0" (acts idle 36, none 21); V read of d2d's line. Effect: what the pace does to the next move U (pass 2 Q6: the pace adds to the monster base velocity percent). 7 more Fetish cases (walk vs none, pace 50 vs 0, control words differ) are not isolated (U) | yes: this card's `mai3_run.sh Fetish`, the replay's back-off ask list; fix shape, not built: drop the reset at line 1753 (expected 0 differences after, not run) |
| D2 | Arach: the charge (state 2) is `FUN_005ded40` = a move of type 0xd (S pass 2 Q6: a walk-mode move at Velocity with the think's pace) | `MonAct::walk` (monsters.hpp:1646, 1651, 1659, 1662) | V run: 337 of 1,500, act only (game run, d2d walk; every other field equal, also the pace). Effect U: both end up at Velocity; what request type 0xd changes is not read | yes (same) |
| D3 | Arach: the lay (Skill1 at its mode when out of Spider Lay's state) is cast at no unit | `{ MonAct::skill, 0 }` leaves `unit` at its default `target` (monsters.hpp:1669), while d2d's own comment (1642-1643) says "at no unit" | V run: 41 of 1,500, the unit field only; V read of the d2d line and of the comment. Effect U (what `ai.cpp` does with a target on a lay) | yes (same) |
| D4 | Diablo: a pending act 5 (state 5, set by a Skill1 cast) with the target gone casts Skill1 with the third argument of `FUN_005dead0` = the current target (null): at none (`0x5e93b5`..`0x5e93cf`: `push ebx` is the target unit); same for act 4 (S4, one case) | `cast(0)` with the default unit `target` (monsters.hpp:3009); Skill8 alone is `input.target ? target : none` (3001) | V run: 30 of 3,000 Skill1 plus 1 S4 case, unit field only. Effect U (a cast at a missing target: what `ai.cpp` does with `ThinkUnit::target` and no target) | yes (this card's `mai3_run.sh Diablo`) |
| D5 | Diablo act 15 (Skill7 at the target player's command unit, `0x5e9655`..`0x5e9697`): `FUN_005dead0(mode, skill, 0, unit id, 2)`: the unit is passed as (id, type 2), the harness shows x = id (7) and y = 2 | `cast(6, ThinkUnit::none)` (monsters.hpp:3032): the unit is dropped | V run: 12 of 3,000 differ in x, y only; the meaning of the two arguments is I (id and type of the unit `FUN_00552f60(game, 2, id)` found). Reached only with the player's command unit near his point in the Chaos Sanctuary (`leader_near`) | yes (same) |

Not isolated, so not findings (U): Diablo, 18 of 3,000 differ in act or n with a different seed (weighted pick; the point-far tests use unit sizes that the case line does not carry and the replay defaults to 2: I); Baal Crab, 2 of 1,500 differ in the skill slot (game slot 1, d2d slot 4; raw skill ids 0x13b..0x13e; U); Fetish, 7 cases above; Vampire, 11 of 1,500 (7 casts where game.exe's slot is 0, 1 or 2 and d2d's is 3, with distinct scene skill ids; 4 where the harness's flags-0 idle meets a d2d `none`): not isolated, so no ledger line. Arach's earlier 56 `none` vs `idle` (a back-off asked 4 or 8 that could not set off) were a harness label (dead end 7): 0 after modelling it.

## Bot findings (described, not changed)

- **B1: caster bosses are "melee" to the bot's step-back.** `internal/data/monsters.go:122` fills `Ranged` from `rangedtype`; `monstats` has it empty for Andariel, Duriel, Mephisto, Diablo, Summoner, Izual and the Hydras (D, `mai3_tab.sh`). `internal/combat/fight.go:788-789` skips only `Ranged` monsters when listing kite threats, and `fight.go:635-651` steps back from "ONE melee monster within 3 below 60 % life and falling" (retreat `KiteRetreat`, `internal/env/tuning_combat.go:85`). Against Mephisto (all six skills mode A2: casts, V read of the row), the Summoner (all SC) or Izual (Frost Nova SC) a step back cannot break contact; against Duriel (Vel 10, walks in every think, D plus Q2) it cannot either. Tag: D plus V read of the bot lines; the effect on a fight is U.
- **B2: no bot code models a boss' think period.** No `aidel` read exists in the Go code (grep of internal/, 2026-11-02); the periods are 15 / 10 / 6 frames for Mephisto, 15 / 11 / 9 for Andariel (pass 1), 15 / 12 / 8 for Izual in Normal / Nightmare / Hell (D). A Hell Mephisto decides every 6 frames (0.24 s at 25 frames per second, I), 2.5 times as often as in Normal, so timed dodges and potion windows that hold in Normal do not carry to Hell (I, not measured).

## PR candidates

None. D1 to D5 were found on this card's harness (the replay program is cut from the scratch tree's test by `mai2_patch.py`, not the upstream corpus run), no fork branch could be made in this run, and each effect is U or I. If a PR is wanted: D1 (one line, 57 of 1,500 cases) and D3 (set `unit = none`, 41 of 1,500) are the smallest; an independent verifier must re-run them on the existing harness (mismatches before > 0, after 0).

## Dead ends and harness artifacts (fixed before the final runs; each would have been a false "deviation")

First runs (seed 20261103, same harness, `mai2_ais.py`): 19 Act 2 and boss AIs 28,500 cases 798 differ; 26 Act 3 and 4 AIs 39,000 cases 576 differ; 36 Act 5 AIs 54,000 cases 260 differ.
1. `keep`'s y was zeroed for every AI (the second pass's normalisation for Wraith and Skeleton Archer): the Vulture and Izual ports carry a real mode in y (2, 9, 8), so **719 Vulture and 15 Izual cases differed by the label**. `mai3_ais.py` zeroes it only for Wraith and Skeleton Archer: 0 after.
2. The replay patch added a rally effect whenever d2d set the rally flag; the Scarab's port sets the flag and pushes the effect, so **32 Scarab cases were counted twice**. `mai3_fix.py` adds it only when absent: 0 after.
3. **Ancient 253 of 1,500 (534 of 3,000 on the re-run, seed 20261104)**: d2d measures `apart` from positions less 2 per axis (default unit size 2), the harness answers `FUN_006416d0` with the scene's distance and the spots were independent. Putting the target `dist + 2` east of the monster: 0 of 3,000 and 0 of 1,500. The claim therefore rests on `FUN_006416d0` being that distance (hooked; V read of d2d's comment only).
4. **Will-O-Wisp 32 of 1,500 (20 of 3,000)**: all 20 cases have the dance state 5 with word 3 = 0 (counts 0, 1, 2; V run, `mai3_mm.py`): the game reads its step tables at an index d2d's table holds as `{0, 0}`; the think never writes word 3 = 0 in a dance state (it writes its place + 1, 1..5, S d2d comment), so the case is unreachable (I). With states limited to 5..12 and word 3 to 1..5 (1..4 past state 5): 0 of 3,000 and 0 of 1,500.
5. **Catapult Spotter 4 and Putrid Defiler 2 of 3,000**: the thinks cast raw skill ids (288, 300, 304) and the harness labels a cast by its slot when the id equals a scene skill id: a random scene id equalled the raw one. Scene ids 280..320 replaced by 99: 0.
6. Hell Meteor 1 of 3,000 in one run, 0 in the next (same label collision family, I).
7. **Arach 56 and Vampire 34 `none` vs `idle`**: a back-off with flags 0 that cannot set off makes game.exe think again with `aidel` (the harness models that idle for the curse AIs only; d2d's Arach comment, monsters.hpp:1643-1644, says the same). `mai3_ais.py` applies it to Arach and Vampire: Arach 0 of those after (378 left, all D2 and D3). Vampire: scene skill ids repeated across slots and a cast is labelled by the first slot with its id; made distinct: 7 of 1,500 casts still differ in the slot and 4 flags-0 idles meet a d2d `none` (U).
8. `mai3_ais.py` runs only AIs in upstream's `mon_ais.py` table (81 here) plus Bighead and Vampire by address (`EXTRA`). **Not run** (no hook set, or no think): 7TIllusion, Ancient Statue, Assassin Sentry, Baal Taunt, Baal Throne, Baal To Stairs, Blade Creeper, Bone Wall, Buffy, Cycle Of Life, Death Sentry, Druid Bear, Druid Wolf, Evil Hole, Good Npc Ranged, Idle, Inviso Pet, Inviso Spawner, Jar Jar, Mosquito Nest, Navi, Necro Pet, Npc, Npc Out Of Town, Npc Stationary, Raven, Shadow Master, Shadow Warrior, Skeleton Mage (27 monstats rows, d2d has a port and upstream's corpus holds 36 cases: S), Totem, Towner, Trap-Melee, Trap-Missile, Trap-Nova, Trap-Poison, Vendor, Vines, Wussie. Baal Crab and Vampire stay untested (unexplained differences, above).

## Limits

- L1: the inputs `ai.cpp` builds (target choice, second target, distances, leader state, line of sight, `dying`, resists) are scene values here, as in the first two passes: the differential proves the think, not the feed.
- L2: damage, to-hit and speed are not in the think; D2's and D1's speed effects are U (see pass 2 Q6).
- L3: seeds: 20261105 for every count in the table and the ledger rows; 20261103 and 20261104 for the artifact runs. Cases are random scenes from upstream's `mon_ais.py` generator (aips, skills, life, distance, melee, flags, states); a branch the generator rarely reaches may hold a difference: the acts reached are in `covmonai2-runs.txt` (every AI reached at least two different acts; Uber Mephisto, Uber Diablo and Uber Baal only `none`, as their code is a bare return).
- L4: the d2d side is the real `mon_think` compiled in the replay, but the replay program is edited by `mai2_patch.py` and `mai3_fix.py` (assert to counter; rally); the edits are visible in those files.
- L5: no live run: every game-side claim is the emulator's.

## Drill recipe for the fork verbs (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md; not run)

1. D1, D2 (Fetish, Arach speed; Act 1 and 2): `debug spawn fetish1 1 near`, `monsters` each tick with `step 1`; `debug mon set <id> life ...` so the target is above `aip4` % life and the Fetish squares up, then watch whether it keeps backing off at pace 50 when blocked (walls behind it). game.exe: pace 50 stays on the next move; d2d: resets. For Arach (`debug spawn arach1`) read cells per second while it charges: Velocity either way (D2).
2. D4, D5 (Diablo): `debug goto` the Chaos Sanctuary, `debug spawn diablo 1`, `debug cast`/`monsters` over 100 thinks; to hit D4 the target must disappear during state 5 (move the lead player out of his sight right after Lightning Hose starts). U whether the fork can express it.
3. Duriel (Q2): `debug spawn duriel 1 near`, 200 ticks in melee, count A1, A2, Smite, Jab starts (expect no A2, Smite near 33 %): confirms aip4 = 0 reading.
4. Boss cadence (B2): `debug spawn mephisto 1` in Normal and Hell, count actions per 100 frames (expect 1 per 15 and 1 per 6 frames at most).

## Matrix and ledger proposals

- docs/coverage/ledger.tsv (applied here, seven lines, `mai3_ledger.py`): 9 names TESTED-0-DIFF with 27,000 cases (F1, 3,000 each), 28 names with 42,000 (F2), 40 names with 60,000 (F3), Bighead with 1,500 (F4); Diablo, Arach and Fetish TESTED-DEVIATION (3,000, 1,500, 1,500 cases). Baal Crab and Vampire get no line. `python3 tools/coverage/coverage.py --write` regenerated the table: monster.ai untested 121 to 40 of 146.
- docs/coverage/gaps.tsv row 2 (monster.ai): untested falls to 40 (the names under "Not runnable"), mostly town NPCs, traps and scripted Baal / Ancient statues; the row stays but its priority text should say "Act 2 to 5 combat thinks are covered; the feed (ai.cpp) is not".
- docs/research/d2d-fidelity-riskmap.md rank 5 `mon_wake`: add "boss and Act 2 to 5 thinks: 0 of 78 AIs differ; open: Fetish pace, Arach run and unit, Diablo unit (D1 to D5), effects U".

## Reproduce (the worker host; all through `tools/emu/run-on-worker.sh`)

```sh
tools/emu/run-on-worker.sh --ttl 1800 mai3_run.sh <AI,AI,...> <cases> 20261105   # prints `mai2 AI cases differ`, `acts AI act N`
tools/emu/run-on-worker.sh mai3_mm.py run3 [AI]      # classifies the mismatches of the last run (act pairs, fields, asks)
tools/emu/run-on-worker.sh mai3_dbg.py <AI> <cases> <seed> <act|act:n> <N>   # the raw native calls of N matching game.exe cases
tools/emu/run-on-worker.sh mai3_tab.sh <AI,AI|cols>  # monstats AI columns (patched stack)
python3 -I tools/emu/mai3_pack.py                    # docs/research/covmonai2-runs.txt from the saved outputs
```

Files (all new, none of upstream or of the earlier passes edited): `tools/emu/mai3_ais.py` (scenes, per-AI label fixes; imports `mai2_ais.py`, `mai_ais.py`, upstream `mon_ais.py`), `mai3_run.sh`, `mai3_fix.py` (replay patch), `mai3_mm.py`, `mai3_dbg.py`, `mai3_tab.sh`, `mai3_tab.py`, `mai3_names.py` (rows still untested), `mai3_ledger.py`, `mai3_pack.py`. The first-run command lines (seed 20261103) used `mai2_run.sh` with the AI lists of the three groups.

## Glossary note

No missing term; "think period" (`aidel`) and "pace" are used as in the earlier passes.

## Shared-file lines for the merger

See `openspec/changes/covmonai2/merge-lines.txt`.
