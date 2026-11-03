# COVSTATES2: monster-facing and common combat states, game.exe 1.14d vs d2d vs the bot (card covstates2)

> status: card covstates2, 2026-11-02, second pass of the coverage gap "states and auras" (185 rows). Offline: no live game, no d2d slot, nothing pushed or posted. Method as `sk-armor-trace.md` (put rule and group, first read there) and `skcurseval-trace.md` (curse value calcs). Question: for the curses (Amplify Damage, Weaken, Iron Maiden, Life Tap, Decrepify, Terror, Dim Vision, Confuse, Attract), Taunt, Battle Cry, Inner Sight, Slow Missiles and the shape-shift forms: how long does the state last, what does a second cast do (stack, refresh, refuse, replace), what does d2d do instead, and what does our bot assume?

Tags: **V run** game.exe's own code run natively under the emulator on the worker host, with counted cases and a seed; **V read** read in the disassembly (address and instructions, no code pasted; functions are named `FUN_xxxxxxxx`); **D** the patched 1.14d tables as game.exe's MPQ stack serves them (States.txt, Skills.txt, DifficultyLevels.txt, ItemStatCost.txt); **S** d2d source or an earlier trace (named); **I** inferred; **U** unverified. "Real d2d" means `put_state` (components/game/fight/states.cpp, declared in ai.hpp) compiled from the scratch build's source and called by `tools/emu/cs2_d2d_test.cpp`; only the call-site choices around it (which slot, the length floor, the Battle Cry guard) are transcribed from `fight.cpp` (tag S, lines named). Time: 25 frames a second (I; d2d multiplies frames by 40 ms).

## Answer in nine lines

1. **One timed-state put serves nearly every state a monster or a player gets: `FUN_0056e970`** (V read 0x56e970..0x56ebdd; called from 25 sites, listed in Q1). It is not the only one: the shout family (do 68, `FUN_005d8290`) builds its own node and always refreshes (V read, Q5).
2. **A curse is one slot per unit, whatever the curse.** Every state whose States.txt `curse` bit is set shares one list node kind (flag 0x20); a new curse of another state or skill removes the old one, the same skill at the same level refreshes the end, the same skill at a LOWER level is refused (V read; **V run: 100,000 sequences per seed, seeds 20261102 and 7, 0 differences from a Python model of exactly these rules, and from `FUN_0056e970` itself**). 26 states carry the bit (D, loader mapping I); 14 of them are combat curses, including Taunt (27) and Battle Cry (89).
3. **A non-curse state has its own entry by state id** (Inner Sight 17, Slow Missiles 87 can both be up); it follows the same refresh / refuse / replace rule (V run).
4. **An attracted unit refuses every curse, a recast of Attract included** (V read 0x56ea40: has-state 0x39 = 57; V run 6,351 cases of 100,000 depend on it).
5. **Length: game.exe keeps the length it is given; 0 means the state never ends; a refresh with length 0 leaves the end alone** (V read 0x56eb00; V run). The unit's `curse_resistance` (stat 0x6d) cuts a curse's length by that percent and refuses it at 100 or more (V read 0x56ea17..0x56ea4a; V run 640 cases, 0 differences).
6. **The lengths** (V run, calc machine, table in Q4): Amplify Damage 200 frames (8 s) at level 1, 875 at level 10, 7,550 at 99; Weaken 350; Iron Maiden 300; Life Tap 400; Decrepify 100; Lower Resist 500; Dim Vision 175, Terror 200, Confuse 250, Attract 300 before the difficulty divisor (1, 2, 4 for Normal, Nightmare, Hell; Dim Vision, Terror, Confuse, Attract only: V read). **Taunt and Howl have no `auralencalc`: Taunt's length is 0 (D), so the state never ends in game.exe (V run of the put with length 0; U whether something else ends it).**
7. **State groups** (`FUN_0056c740`, V run 370 cases): putting a state of group g removes every state of group g the unit has. Groups (D): 1 = Frozen Armor, Bone Armor, Chilling Armor, `justhit` (86), Shiver Armor; 2 = Quickness, Fade; 3 = Maul, Feral Rage, Werewolf, Werebear, `monsterset`, `delerium` (wolf and bear forms exclude each other).
8. **d2d differs in five rules proven by the existing-style differential (D1..D5 below): 54,685 of 100,000 sequences differ from game.exe (seed 20261102), 54,508 (seed 7); a d2d model with the five rules fixed gives 0 in both**, each fix being needed (leave-one-out table). Two more are read only (D6, D7).
9. **No PR candidate** (reason in the PR section): the fixes are missing rules, not a value, and a fix was not built and run in d2d.

## Q1. What does FUN_0056e970 do? (V read, V run)

Entry: ECX = a request record (+0 the source unit, +4 the target unit, +8 skill id, +0xc skill level, +0x10 length in frames, +0x14 / +0x18 an optional stat-list pair, +0x1c state id, +0x20 an end callback or 0). Returns the list node or 0. Read at 0x56e970..0x56ebdd:

| Step | Address | What | Tag |
|---|---|---|---|
| bounds | 0x56e993..0x56e9b8 | the state id must be in the States table (count at table +0xc4, row stride 0x3c) | V read |
| monster gate | 0x56e9c3..0x56ea08 | a unit of type 1 is refused when MonStats row (stride 0x1a8) byte +0xd has bit 0, and when the MonStats2 row (via `FUN_00451fe0`, stride 0x134) is missing or its byte +5 lacks bit 1. A player (type 0) skips the gate | V read; **V run 8 cases** (`cs2_put.py gate`: type 0 always accepted; type 1 accepted only with +5 bit 1 set and +0xd bit 0 clear) |
| curse? | 0x56ea0a | `FUN_0063a420(state)`: a bit array indexed by state id (V read; the array is filled from States.txt `curse`: I, the loader was not found, Dead ends) | V read, I |
| curse resistance | 0x56ea17..0x56ea3d | `FUN_00625480(unit, 0x6d)` (stat 109, ItemStatCost `curse_resistance`: D); at 100 or more the put is refused; else the length becomes `length - length x res / 100` (`FUN_00483360` = MulDiv) | V read; **V run 640 cases, 0 differences** |
| attract block | 0x56ea40..0x56ea4a | `FUN_00639df0(unit, 0x39)` (has state 57, attract): nonzero refuses the put | V read; V run (D3) |
| find | 0x56ea53..0x56ea78 | a curse looks for the unit's node of kind 0x20 (`FUN_00625760`), any other state for the node of that state id (`FUN_006256b0`) | V read |
| compare | 0x56ea78..0x56eae0 | node's state must equal the new state AND skill AND level, else the node is removed (`FUN_006277e0`, `FUN_00626cd0`) and a new one made. Same state, skill and level: end := now + length (only when length is not 0), end event rescheduled, node returned. Same state and skill, node level HIGHER than the new one: refused (return 0). Lower: replaced | V read; V run |
| make | 0x56eae6..0x56ebdd | `FUN_0064c040`, state ON (`FUN_00639db0`), flags 0x20 for a curse, +2 when the length is not 0, +0x800 / +8 from table bit arrays, `FUN_006251f0` (end frame = game frame +0xa8 + length, or 0 for no length), owner, skill, level, attach, default end callback 0x56e900 | V read |

**Callers (V read, `call 0x56e970` in a full linear disassembly):** 0x56ec9e, 0x57d891, 0x582850, 0x583c52, 0x5a24ba, 0x5ab5ef (Battle Cry's hit function `FUN_005ab500`), 0x5bffa2, 0x5c0418, 0x5c36a6 (the curse do, via `FUN_005c35c0`), 0x5c3d2b (Attract do), 0x5c3e67 (Confuse do), 0x5c6819, 0x5c954f (armor do, after the group call), 0x5c9ccf (Energy Shield), 0x5ca119, 0x5ca66d, 0x5cee38, 0x5d1deb, 0x5d22fe, 0x5d65a5, 0x5d670f, 0x5d7af0, 0x5d91f3, 0x5db1a6 (do 6: Inner Sight, Slow Missiles), 0x5dddea (Howl's run). Of these I read the owner of: the curse do, Attract, Confuse, Battle Cry, do 6, Howl (the table `0x7322b0`, stride 4, gives do 6 = 0x5db1c0, 22 = 0x5c9b50, 30 = 0x5c37c0, 59 = 0x5c3b90, 61 = 0x5c3f20, 68 = 0x5d83e0, 71 = 0x5d8570; V read of the table via the emulator image); the rest are listed, not read (U).

**Harness (V run).** `tools/emu/cs2_put.py game` runs the real function on a fake unit with these Python models in place of the helpers (not the claim): the node list (`FUN_006251f0`, `006252d0`, `006260d0`, `006260f0`, `00626e10`, `00625310`, `006256b0`, `00625760`, `006252f0`, `006260e0`, `00626100`, `006277e0`, `00626cd0`), state on / off (`FUN_00639db0`), has-state (`FUN_00639df0`), the stat getter (`FUN_00625480`), the event scheduler, the client update, the client-flag array, and `FUN_0063a420` (filled from the table, I). Real: `FUN_0056e970`, `FUN_00483360`, `FUN_00451fe0` and the table words it reads.

## Q2. Which states share the curse slot?

**D (table) + I (loader mapping).** States with `curse` = 1 in States.txt: 9 amplifydamage, 19 weaken, 23 dimvision, 27 taunt, 55 ironmaiden, 56 terror, 57 attract, 58 lifetap, 59 confuse, 60 decrepify, 61 lowerresist, 89 battlecry, 113 defense_curse, 114 blood_mana, then 128..137 (the ten shrine states), 141 bloodlust, 156 cloaked. A new put of any of them replaces whichever of them the unit has (Q1, V run with the table-fed bit array). Which of these a combat skill reaches (Skills.txt, D): Amplify 66, Dim Vision 71, Weaken 72, Iron Maiden 76, Terror 77, Life Tap 82, Decrepify 87, Lower Resist 91, Confuse 81, Attract 86, Howl 130 (Terror), Taunt 137 (27), Battle Cry 146 (89), Defense Curse 309 and Blood Mana 310 (monster skills).

d2d (S, fight.cpp:1223): `curse` = the skill's do is 30, 61 or 59, plus Howl (fight.cpp:2932); Taunt (do 71, fight.cpp:1216) and Battle Cry (hit function 21, fight.cpp:2924) go to the second slot `cry`. So Taunt and Battle Cry do not replace a curse in d2d and a curse does not replace them (D1).

## Q3. State groups, FUN_0056c740 (V read 0x56c740..0x56c83a, V run)

ECX = unit, EDX = state, stack arg include_self. The States row (stride 0x3c) word +0x1e is the group (S, `sk-armor-trace.md`; the column-to-offset mapping is I); group 0: nothing. Else every state of the same group the unit has is switched off (`FUN_00639db0`) and its node freed; the new state itself only when include_self is set. **V run: for each of the 185 states as the new state, include_self 0 and 1 (370 cases), with every state present: the removed set equals "every state with the same non-zero group, the state itself only with include_self", 370 of 370.** Group membership (D): **1: 10 frozenarmor, 14 bonearmor, 20 chillingarmor, 86 justhit, 88 shiverarmor; 2: 157 quickness, 159 fade; 3: 117 maul, 120 feralrage, 139 wolf, 140 bear, 176 monsterset, 177 delerium.** Callers of the group removal (V read): 0x55bda3, 0x55bdb6, 0x5bee15, 0x5c6c5b, 0x5c6cb2, 0x5c6f20 (inside the shape-shift do `FUN_005c6ec0` by d2d's comment, I), 0x5c94f4 (armor do), 0x5d7a9c, 0x5d9195. Which callers pass which states is not read except the armor do (`sk-armor-trace.md` Q4).

## Q4. Lengths (V run, `cs2_len.py`)

game.exe's calc machine (`FUN_00646ca0`, as `nec_calc.py`) on each skill's `auralencalc` at skill levels 1..99, character level 30, no masteries, no other skill levels (so synergy terms are 0): the floor of what a character gets. Frames; divided by AiCurseDivisor (Normal 1, Nightmare 2, Hell 4: D, DifficultyLevels) for the four states the do divides (V read: `FUN_005c3400` returns 1 for states 0x17 and 0x38, `FUN_005c37a0` divides with a signed `idiv`; called at 0x5c384e / 0x5c388b in the curse do, at 0x5c3c85 in the Attract do and 0x5c3fd0 in the Confuse do).

| Skill (id) | state | L1 | L5 | L10 | L20 | L99 | min after divisor |
|---|---|---|---|---|---|---|---|
| Amplify Damage 66 | 9 | 200 | 500 | 875 | 1,625 | 7,550 | 200 |
| Dim Vision 71 (divided) | 23 | 175 | 375 | 625 | 1,125 | 5,075 | 43 |
| Weaken 72 | 19 | 350 | 590 | 890 | 1,490 | 6,230 | 350 |
| Iron Maiden 76 | 55 | 300 | 540 | 840 | 1,440 | 6,180 | 300 |
| Terror 77 (divided) | 56 | 200 | 300 | 425 | 675 | 2,650 | 50 |
| Confuse 81 (divided) | 59 | 250 | 450 | 700 | 1,200 | 5,150 | 62 |
| Life Tap 82 | 58 | 400 | 640 | 940 | 1,540 | 6,280 | 400 |
| Attract 86 (divided) | 57 | 300 | 660 | 1,110 | 2,010 | 9,120 | 75 |
| Decrepify 87 | 60 | 100 | 160 | 235 | 385 | 1,570 | 100 |
| Lower Resist 91 | 61 | 500 | 700 | 950 | 1,450 | 5,400 | 500 |
| Battle Cry 146 | 89 | 300 | 540 | 840 | 1,440 | 6,180 | 300 |
| Inner Sight 8 | 17 | 200 | 600 | 1,100 | 2,100 | 10,000 | 200 |
| Slow Missiles 17 | 87 | 300 | 900 | 1,650 | 3,150 | 15,000 | 300 |
| Defense Curse 309, Blood Mana 310 | 113, 114 | 200 | 500 | 875 | 1,625 | 7,550 | 200 |
| **Taunt 137, Howl 130** | 27, 56 | **0** | 0 | 0 | 0 | 0 | **0** |

5,049 cells (17 skills x 99 levels x 3 difficulties), 594 below 25 frames, all of them Taunt and Howl (297 each). The calc cells of the curses equal d2d's evaluator (`skcurseval-trace.md`, 0 of 46,136 and 66,136), so these lengths are what d2d's `calc(skill, auralen, level)` returns too. **Taunt (V read 0x5d8627: its do reads the `+0x60` auralen calc, no divisor call; D: the cell is blank) puts the state with length 0**, which Q1 turns into a node with no end. Whether anything else ends Taunt (a think, a target change) was not looked for (U). Howl's length comes from the missile (d2d: `par[2] + (level - 1) par[3]`, fight.cpp:2932; not traced here, U).

**Stat effects (D, what each state carries):** Amplify damageresist (stat 36) -100 at every level; Weaken damagepercent -33; Decrepify velocitypercent, damage, damageresist, attackrate -50 each; Lower Resist four resists -31 at L1 to -70 (flat from L60); Iron Maiden calc1 200..2,650 percent; Life Tap calc1 50; Inner Sight armorclass `-edmn`; Taunt item_tohit_percent `ln12` and damagepercent `ln34`; Battle Cry skill_armor_percent `ln34` and damagepercent `ln56`; Slow Missiles skill_handofathena `ln12` (values: `skcurseval-trace.md`, `skamazon-trace.md`, `skbarb-trace.md`, calc cells V run equal d2d). What d2d does with them (S, fight.cpp:1494..1504 and 3152..3166): stats 25 -> monster damage, 67 -> speed, 68 / 69 -> attack speed, 36 / 37 / 39 / 41 / 43 / 45 -> the six resists, 171 and 31 -> armor class; Iron Maiden -> reflect from calc1; Terror -> afraid; Confuse / Attract -> alignment. **Not applied by d2d: Taunt's item_tohit_percent, Slow Missiles' stat, Decrepify's separate `damage` stat if it is not 25.** Whether game.exe applies those on a monster, and how, was not run (U).

## Q5. The shout family does not use FUN_0056e970 (V read)

Do 68 (Shout, Battle Orders, Battle Command: `FUN_005d83e0`) applies the caster's `aurastate` to allies through `FUN_005d8290`: length from the `+0x60` calc, then `FUN_006256b0` for an existing node of the state: found -> reuse it, not found -> new node with flags 2 (no curse flag); then state on, new end (`FUN_006260b0`), end event, and the stat apply `FUN_005c6cc0` with the new level. There is **no level comparison**: a lower-level recast replaces a higher one (V read 0x5d8290..0x5d83d7; not run: the helpers are many and the stat apply was not read, U for what it does to the old stat list). d2d `Fight::start_state` (fight.cpp:1325): erase the same skill, push: same result for a recast, and length 0 = until `~0u` (never ends), the same as game.exe for the player's own states (S). The monster side of Battle Cry is the hit function 21 and goes through `FUN_0056e970` (Q1 callers).

## Q6. What does d2d do on the player side?

`Fight::start_state` (S): same-skill erase and push, no group, no level refusal, no curse slot. A monster's curse on the player or the mercenary is `actor->curses[k]`, one entry each, the newest in place of the last, length `max(auralen / divisor, 25)` frames, no `curse_resistance` (S, fight.cpp:4756..4763). game.exe puts a monster's curse on the player through the same function, so the single-slot rule matches; the refusal at a lower level, the Attract block, the length cut and the floor do not (I for the player path: the player skips only the monster gate, V run 4 gate cases, and the sequences were run on a monster unit).

## Deviation list, d2d vs game.exe

Differential: `tools/emu/cs2_run.sh N SEED` (generator, game.exe side, d2d side, model, comparisons, switch table), 100,000 sequences per seed of 1 to 4 puts drawn from 17 monster-facing skills (66, 71, 72, 76, 77, 81, 82, 86, 87, 91, 309, 310, 130, 137, 146, 8, 17), skill levels 1..30, lengths from {25..2000, 25, 26, 175, 1..24, 0, 1}, the puts at one instant. Oracle: accept flag of each put and the final node set (state, skill, level, remaining frames or "never"). **Counts (V run):**

| Comparison | Seed 20261102 | Seed 7 |
|---|---|---|
| cases | 100,000 | 100,000 |
| real d2d `put_state` vs the Python transcription of d2d (validates the model) | 0 differ | 0 differ |
| game.exe vs real d2d | 54,685 differ (accept flag in 8,681) | 54,508 differ (accept flag in 8,762) |
| game.exe vs the model with all five fixes | 0 | 0 |
| fixes but D1 (curse bit slot) | 15,274 differ | 15,090 |
| fixes but D2 (own entry per non-curse state) | 926 | 841 |
| fixes but D3 (attract block) | 6,351 | 6,441 |
| fixes but D4 (no length floor) | 39,995 | 39,718 |
| fixes but D5 (Battle Cry guard) | 2,125 | 2,112 |

Counts are over a generator that injects lengths 0..24 (so D4 is inflated: real curse lengths are at least 43 frames, Q4; only Taunt reaches the floor). Other modes: `cs2_put.py curseres` 640 cases (unit type 0 / 1 x 10 curse resistances x 8 lengths x 4 skills), 0 differences from the read rule; `cs2_put.py group` 370 cases, 0; `cs2_put.py gate` 8 cases.

| # | d2d | game.exe 1.14d | Tag | Differential |
|---|---|---|---|---|
| D1 | Taunt (27) and Battle Cry (89) live in the `cry` slot, apart from the curse slot (fight.cpp:1223, 2924) | every curse-bit state shares one slot: Battle Cry then Amplify Damage leaves only Amplify; Taunt then Lower Resist leaves only Lower Resist | V run (game and real d2d), I for the column-to-bit mapping | yes, built (`cs2_run.sh`); live recipe R1 |
| D2 | one `cry` slot for the non-curse states (Inner Sight, Slow Missiles): the second replaces the first (known: fight.cpp:98 "ponytail") | one node per state id: both up, each its own end | V run | yes, built |
| D3 | no Attract block | a unit with state 57 refuses every curse put, Attract itself included | V read 0x56ea40, V run | yes, built; live recipe R2 |
| D4 | monster states last at least 25 frames (Battle Cry 1): Taunt, with length 0, lasts one second | length kept as given; 0 = no end, a refresh with 0 keeps the old end | V run (put), D (Taunt cell), V read (Taunt do) | yes, built; the end of Taunt by another path is U |
| D5 | Battle Cry on a monster whose Battle Cry still runs does nothing (fight.cpp:2924) | same skill, same level: end extended; lower level: refused; higher: replaced | V run | yes, built |
| D6 | no `curse_resistance` (stat 109): a monster's curse on the player is never shortened or refused (S: no use of the stat in components/game or components/rules by name or id; grep, not a proof) | length cut by the percent, refused at 100 | V run (game side rule, 640 cases), S / I (d2d absence) | game side built; d2d side is an absence, a test would need a player with the stat (item data): not built |
| D7 | `start_state` has no group (wolf and bear forms, Frozen / Chilling / Shiver / Bone Armor, Quickness / Fade coexist; a Shiver Armor cast after Frozen Armor keeps both, `sk-armor-trace.md` D2) | group removal (Q3) | V run (game side, 370 cases), S (d2d read) | could be built: lab drill R3 (`debug states set` of two skills of one group, read `debug states`); a compiled differential needs a `Fight` object |

D6 and D7 are not "found by a differential against real d2d code" in the sense of D1..D5 (D6 is an absence, D7 a read), so they stay read-level in the ledger.

## Bot findings (described, not changed)

| # | File:line | Finding | Tag |
|---|---|---|---|
| B1 | internal/d2/missiles.go:61 | `DefaultCurseIDs.Decrepify = 56`: States.txt row 56 is `terror`; `decrepify` is row 60 (D). With the tables not loaded (`SetTables(nil)`, dodge.go:26) a Terror state on the player reads as Decrepify and a Decrepify as nothing | D |
| B2 | internal/d2/missiles.go:67 | the name looked up for Amplify Damage is `amplifydam`; the table row is `amplifydamage` (D), so `ResolveCurseIDs` never resolves it and the default 9 stays (correct, by luck: the "V against the real file" claim in the comment at :52 does not hold for this one) | D |
| B3 | internal/combat/necro.go:23, :389, :401 | `curseLifeSec = 30` is the cast memory when the state is not readable; the comment says Amplify Damage lasts 24 s at level 1. The calc machine gives 200 frames = 8 s at level 1, 20 s at level 5, 35 s at level 10, 65 s at level 20 (Q4); Weaken 14 s at L1. The memory over-states a level 1..8 curse and under-states a higher one; and the lengths on Nightmare / Hell are the same (only Dim Vision, Terror, Confuse, Attract are divided) | V run (lengths), I (25 frames a second) |
| B4 | internal/combat/necro.go:375 | the comment "a second curse would replace the first, no gain" is confirmed by the put rule (V run, Q1): one curse per unit. Two additions the code does not know: Attract (state 57) blocks every later curse on that unit, and a LOWER-level curse on a unit that has the same curse at a higher level is refused (cast after a +skills item came off) | V run |
| B5 | internal/combat/necro.go:44 | `curseNames` lists Confuse, Attract and Terror among curses the bot may cast; none is ever picked by `curseFor` (necro.go:350..), fine, but if one is added, B4's Attract block applies | I |

## PR candidates

**No PR candidate.** D1..D5 are proven against the real `put_state` with the mismatches before the fix above zero (54,685, 54,508), and a model with the five rules gives 0 after, but the "after" is the Python model of the fixed rules, not a d2d build with the fix: a fix means giving `Monster` one entry per state id and a curse-bit lookup, changing `curse` / `cry` / `frenzy` readers in about ten places (fight.cpp:894, 1489, 3143, 3935, 4011 ...), and a decision on Taunt's end. Not a one-line change, not built, not run. The bot's own recast of a curse needs none of it.

## Dead ends

- **The curse bit array's loader.** `FUN_0063a420` reads a bit array at table +0xf8 (and `FUN_0063a6b0` one at +0x144); a linear search of the disassembly finds no store to +0xf8 outside unrelated code (one `mov [edx+0xf8]` at 0x58a5e2 not read). So "bit = States.txt `curse` column" is I. The states table's group word at +0x1e is S (`sk-armor-trace.md` read it at the loader's use).
- **The monster gate's two bits.** MonStats byte +0xd bit 0 and MonStats2 byte +5 bit 1 are read; the columns they come from were not found (same loader problem). Which monsters refuse every timed state is U.
- **Taunt's end, Howl's length, the stat effects on a monster** (Weaken, Decrepify's separate `damage`, Taunt's to-hit, Slow Missiles): not run, not read to the end.
- **Poison, stun, freeze, chill (the brief's other states):** not repeated: freeze and chill are `freeze-rules-trace.md` / `pr-freeze-proof.md`, poison length and damage `skdefenders-trace.md` Q7 and `missilediff.md`; the stun state (21) and the poison state (2) have no state-table differential and stay UNTESTED in the ledger.
- Running the shout do (`FUN_005d8290`) natively: not tried (many helpers); read only.

## Drill recipe (lab binary, fork verbs: `docs/forkdbg.md`, `docs/d2d-fork.md`, `docs/forkcheats.md`)

Not run here (no game). On the lab binary, one monster from `debug spawn`, `debug points <skill> <n>`, `debug mana full`:

- **R1 (D1):** `debug cast 146 unit <id>` (Battle Cry) then `debug cast 66 unit <id>` (Amplify Damage), then `debug states` (cursed monsters): d2d lists both; game.exe (live comparison is U, not allowed here) is expected to keep only Amplify (state 9) and drop state 89. Repeat with Taunt (137) then Lower Resist (91).
- **R2 (D3):** Attract (86) on a monster, then Amplify (66) on it: d2d shows Amplify replacing Attract; expected game.exe: Amplify refused while state 57 is on.
- **R3 (D7):** `debug states set 223` then `debug states set 228` (Werewolf, Werebear): `debug states` lists both in d2d; expected game.exe: only the second. Also `debug states set 40` and `set 50` (armors, `sk-armor-trace.md` A2).
- **R4 (D4, Taunt):** Taunt (137) on a monster, read `debug states` 2 s later (d2d: gone after 25 frames; expected game.exe: still on).
- To confirm the loader mapping (the `curse` column) on game.exe: cast Amplify and a shrine-state-carrying monster is not needed; compare which of the 26 states replace each other when put through the lab's `states set` (not available for monsters today: a verb `debug mstate <id> <skill>` would be needed).

## Matrix-cell updates (proposed text, `docs/research/sorc-skill-matrix.md`)

- **3.3 Frozen Armor, 3.4 Shiver Armor, 3.9 Chilling Armor, (b) d2d status / (a) rules:** add "state group 1 = frozenarmor 10, bonearmor 14, chillingarmor 20, justhit 86, shiverarmor 88 (D); the group removal `FUN_0056c740` removes every present state of the group, V run 370 cases (covstates2-trace.md Q3). `justhit` is in the group: who calls the removal with a state that evicts it is U (drill: cast an armor on the frame after a hit)."
- **3.11 Energy Shield (a) rules:** add "a put at a lower level than the one up is refused, same level refreshes, higher replaces (`FUN_0056e970`, V run on a monster unit, V read for the player, covstates2 Q1); d2d `start_state` replaces whatever the level (S)."
- **Section 5 / coverage gap 9 (states):** states 9, 17, 19, 23, 27, 55, 56, 57, 58, 59, 60, 87, 89, 113, 114 now have a put-rule differential (D1..D5); stun (21), poison (2), slow (24), the shout states (26, 32, 51), the shrine states and 130 other rows remain untested.

## Ledger rows (docs/coverage/ledger.tsv)

- `states 9,17,19,23,27,55,56,57,58,59,60,87,89,113,114`: TESTED-DEVIATION, evidence this doc, 200,000 cases (100,000 per seed, seeds 20261102 and 7), note: timed put rule (slot, refuse, refresh, length) D1 D2 D3 D4 D5; 61 is already claimed by `sk-resist-trace.md`.
- `states 14,117,120,139,140,157,159,176,177`: READ-ONLY, state group (game side run natively, d2d read, no d2d run) D7; 10, 20, 86, 88 are in group 1 but already claimed.

## Shared-file lines for the merger

See `openspec/changes/covstates2/merge-lines.txt`.

## Limits

- The d2d side is real `put_state` plus transcribed call-site choices; a change in fight.cpp's slot choice, floor or Battle Cry guard would not show in `cs2_d2d_test.cpp` (the transcription is checked against the Python model only).
- The game side runs on a monster unit with Python models of the node list; the real node, event and stat-apply code (`FUN_006251f0`, `FUN_005c6cc0`, `FUN_00639db0`) did not run. "State on / off" is therefore a model: the claims are about which node is kept, its end frame and the accept flag, not about stat lists.
- The curse bit array and the group word are table-fed (I loader mapping).
- Sequences run at one instant: expiry is not modelled on either side (a state past its end is simply absent in game.exe; d2d checks `now < until`).
- Skill levels 1..30 and lengths from a synthetic set; real lengths in Q4.
