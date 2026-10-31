# Original game (Game.exe) casts per kill, Charged Bolt Sorceress vs Fallen in the Blood Moor (agent ORIGPROBES, 2026-10-30)

> status: current. Original-game side of docs/d2d-casts.md (same columns plus `clicks`). Tags: V-orig = measured on Game.exe 1.14d (instance i0, Testy, Normal, Wine, real time); V-d2d = docs/d2d-casts.md; U = not verified.

Probe: `cmd/origcasts casts` (driver `cmd/origcasts/run-casts.sh`, tables `cmd/origcasts/casts_table.py`), raw rows `docs/probes/orig-casts.csv` (columns `area,visit,cb_level,clvl,monster,boss,xp,cast_cmds,clicks`; `visit` is empty), stderr notes `docs/probes/orig-casts.log`.

## Method

Throwaway Testy (expansion Sorceress). Per level: Save and Exit, logged save edit (`d2cheat save Testy level=<clvl> skills=38:<cb>`, reason "casts per kill probe", clvl 2 / 5 / 9 for Charged Bolt 1 / 4 / 8 like the d2d probe: clvl = CB level + 1), start the game, `origcasts casts 2 <cb> <clvl> 420` (7 minutes real time in the Blood Moor). The bot's real `combat.Fighter.Clear` (radius 60, 30 s per call, wander between calls when nothing was killed; `EnsureHotkeys` bound Charged Bolt to F4, the only skill besides the basics) with the real Game.exe input path. Logged god cheat for the whole run (it refills life, mana and stamina every few ms, like the d2d probe's top-up; reason on the ledger). A "cast command" is `Controller.CastAt` issued by the fighter (a counting wrapper around the bot's input); plain mouse clicks are counted apart (`clicks`, 0 to a few per kill: movement), same definition as d2d (commands, not missile spawns). One row per `goals.EvKill`, the event the bot logs; `cast_cmds` = commands since the previous kill.

## Result (V-orig, one 7-minute run per level; V-d2d from docs/d2d-casts.md)

| Charged Bolt level | monster | original | d2d |
|---|---|---|---|
| 1 | Fallen | 51 kills: **0.96** (median 1, p90 2) | 183 kills: **2.59** (median 2, p90 4) |
| 4 | Fallen | 39 kills: **0.90** (median 1, p90 2) | 177 kills: **1.18** (median 1, p90 2) |
| 8 | Fallen | 62 kills: **0.69** (median 1, p90 2) | 237 kills: **0.97** (median 1, p90 2) |
| 1 | Quill Rat | 14 kills: **1.79** (median 1.5, p90 3) | 78 kills: **3.27** (median 3, p90 6) |
| 4 | Quill Rat | 17 kills: **1.29** (median 1, p90 3) | 72 kills: **2.21** (median 2, p90 5) |
| 8 | Quill Rat | 18 kills: **1.11** (median 1, p90 2) | 64 kills: **1.81** (median 2, p90 3) |
| 1 | Zombie | 17 kills: **5.12** (median 2, p90 6) | 70 kills: **5.41** (median 4.5, p90 10) |
| 4 | Zombie | 6 kills: **1.83** (median 1, p90 5) | 69 kills: **2.65** (median 2, p90 5) |
| 8 | Zombie | 21 kills: **0.95** (median 1, p90 2) | 60 kills: **1.82** (median 2, p90 3) |

Fallen headline (cast commands per kill, mean): **original 0.96 / 0.90 / 0.69** at Charged Bolt 1 / 4 / 8 (51 / 39 / 62 kills) versus **d2d 2.59 / 1.18 / 0.97** (183 / 177 / 237 kills). Zero-cast Fallen kills (a second kill from the same bolt): original 24 % / 31 % / 47 %.

## Reading

- The original needs fewer cast commands per kill at every level and for every monster, by a factor of 2.7 at Charged Bolt 1 (0.96 vs 2.59), 1.3 at 4 and 1.4 at 8 for Fallen; Quill Rat 1.79 / 1.29 / 1.11 vs 3.27 / 2.21 / 1.81; Zombie 5.12 / 1.83 / 0.95 vs 5.41 / 2.65 / 1.82. The gap is largest at skill 1 and shrinks as the skill levels up; the shape (falling with skill, below 1 at skill 8 because one bolt kills several Fallen of a pack) is the same.
- Possible causes (all U, not separated): (a) d2d Charged Bolt damage or bolt count at skill 1 is lower than the original's (the d2d fidelity note on the Lightning damage difference, docs/d2d-fidelity.md); (b) the fighter's command rate differs: real time on Game.exe with the real cast delay against the lockstep pacer, so a command issued during a cast animation can be lost or repeated differently (commands, not completed casts, are counted on both sides); (c) Fallen life or defence in the Normal Blood Moor differ in d2d (not checked). A mean of 0.96 at Charged Bolt 1 means a typical Fallen dies from a single cast command, which a 3-bolt Charged Bolt at skill 1 with the Fallen's life of a few points explains easily; the d2d 2.59 suggests misses or lower damage there.
- Samples are small on the original side (one 7-minute run each, 14 to 21 kills for Quill Rat and Zombie): the Zombie skill-4 mean (6 kills) is weak. The Fallen rows (39 to 62 kills) are the usable ones. No seed control: the Blood Moor layout and monster mix differ per game.

## Caveats

- Mana and life were topped up (god), like the d2d probe, so the bot's mana back-off is not exercised; the fighter is the bot's, so targeting and kiting are included; Normal only; one clvl per skill level.
- `kill` events come from the bot's own bookkeeping (`noteKills`): a monster dying from a bolt in flight can be attributed to the next row, and `xp=0` rows appear (two monsters dying in the same observation share one experience delta). Identical on the d2d side, so the comparison is fair, but the per-row split is not exact.
- The save edit sets level and skill without the Charged Bolt requirement level (6): the game used the skill without complaint (V-orig: casts happened at clvl 2).

## Frame rate of the loaded Game.exe (U)

No frame counter is known in the reader. Estimate from mana regeneration (`origcasts mana`, no god): a Sorceress at clvl 9 with max mana 51 regained 6 to 36 mana in 75.1 s = 0.40 mana per second, against max/3000 per frame (the original's rule, the premise used in docs/d2d-tick-rate.md) = 0.425 per second at exactly 25 frames per second: about 0.94 of 25 fps, roughly 23.5 frames per wall second, with an integer-mana error of plus or minus 0.8 fps. UNVERIFIED: this assumes the per-class regeneration is exactly max/3000 per frame; it only shows Game.exe on a loaded worker host ran at about 25 fps within about 6 %, so the per-wall-time and per-frame comparisons agree to that precision. During this measurement a Fallen was attacking the idle character (life 47 to 9), the run was killed and Testy walked home under god: `navlive --god home`, alive in Rogue Encampment, life 47/47.
