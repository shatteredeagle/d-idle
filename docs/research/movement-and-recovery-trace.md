# Walk/run speed and hit recovery traced in game.exe 1.14d vs d2d (agent DECOMPREC, 2026-10-30)

> status: research, current (card decomprec). Offline only: no game instance was started, nothing was pushed, posted or filed on GitHub. Tags: V = read in game.exe 1.14d (`/tmp/casttrace/all.txt`, the full capstone dump of the read-only copy, md5 2b3600c1...) or run in game.exe's own code under unicorn by this agent (`tools/emu/move_recovery.py`); V-d2d = read in d2d's source (upstream `~/d2d-lab/src` e36922ea and the fresh clone `~/d2d-lab/pr/gh-rate` at upstream/main 32768ddb, both the same for the lines cited) or measured on the lab binary; D = patched 1.14d tables read with `mpq-cat` (docs/process/mpq-data-reading.md recipe, sanity line `Id<TAB>hcIdx` seen); S = upstream `docs/research/re/*.md`, community tables, or our own docs; U = unverified. Classic 1.14d, Normal, one player.

## Verdict in four lines

1. **Walk/run speed is the same number on both backends**: game.exe moves a unit `velocity / 16` subtiles a tick (walk 6: 0.375, run 9: 0.5625; 1.875 / 2.81 cells per second, the same on a diagonal), and d2d's `cells_per_sec` is that constant. The gap is the mode: **the d2d bot only walks** (d2d's run toggle is off by default and `internal/d2dinput` never sends `cmd::Run`), while a Game.exe character runs while it has stamina. Ratio original : d2d = 1.5 while running, 1.0 once stamina is spent (about 19 s of running from full for a clvl 1 Sorceress).
2. **Hit recovery length: d2d's get-hits are about half as long as game.exe's.** game.exe's get-hit rate is `AnimData speed x (50 + effective FHR) / 100`; d2d uses `(100 + E)` for the player and the plain AnimData length for monsters. A Sorceress at 0 FHR staggers 15 ticks in game.exe, 8 in d2d; a Zombie 16 vs 8, a Fallen 13 vs 7, a Fallen Shaman 9 vs 5. Real deviation; PROOF for a PR incomplete (the server's monster mode end is U), recorded in docs/research/upstream-pr-candidates-movement.md.
3. **Hit recovery decision**: game.exe rolls it by hit class (Charged Bolt: no get-hit under 1/16 of max life, 50 % from 1/16, 75 % from 1/8, sure from 1/4; frozen units never); d2d: a monster at 1/8 always, a player at 1/12. Second candidate, smaller effect: Act 1 Normal monsters have 1 to 12 life, so a bolt of 4 to 8 is over 1/4 of it on both.
4. For kiting, the Fallen question is moot: a Fallen has 1 to 4 life (D) and dies to any bolt. A Zombie (7 to 12 life) survives one bolt and is staggered 0.64 s in game.exe against 0.32 s in d2d: d2d's Zombies close in twice as fast between bolts.

## 1. Movement (addresses; explanations in our words)

| Address | What it does | Tag |
|---|---|---|
| `00623f50` 0x624113..0x6241c9 | a moving mode (`006214a0`): walk % = `00621740`(unit, row 4) + stat 0x43 (velocitypercent), at least 25 (`cmp esi, 0x19`); animation rate = `006213d0` base x % / 100 (stored at +0x4c); path velocity = `00621360` base x % / 100 through `00648690` | V |
| `00621740` row 4 | stat 96 (faster run/walk) through the 12-byte table at `006e8e24`: effective = 150 v / (150 + v): 10 -> 9, 20 -> 17, 30 -> 25, 50 -> 37, 100 -> 60, 200 -> 85 | V (oracle `frw`) |
| `00621360` | velocity base: a player's CharStats `WalkVelocity` (record byte +0x40) << 8, always the walk value; a monster's MonStats word +0x32 (Velocity) << 8 | V |
| `006213d0` | animation base: a player's 0x65 (101) in RN, else 0xd5 (213); a monster's MonStats word +0x36, or +0x38 (Run) in mode 0xf; upstream's `walk_rates.py` already diffs this branch against d2d's `moving_anim_rate` | V (read), S (diff) |
| `00620e80` | running adds `Run x 100 / Walk - 100` to stat 0x43 (+50 for the stock 6 / 9), so run velocity = 1536 x 150 / 100 = 2304 = 9 << 8 | S (char-select.md; the velocity arithmetic V by the oracle) |
| `00648690` | stores the velocity at path +0x7c and +0x84; a change resets +0x38 to 0xf | V |
| `0064fe40` 0x64fef6..0x64ff1b | the per-tick step: `00064fc60` gives the unit direction vector (length 4096), the step is `velocity x vector >> 8` in 16.16 subtiles, stored at path +0x72 / +0x76: velocity / 16 subtiles a tick, the same on a diagonal (0.5624 vs 0.5625) | V (oracle `step`) |
| `0057f240`, `00580500` | stamina: a running tick outside town costs `RunDrain x 2 x (armor speed / 10 + 1)` 256ths (Sorceress RunDrain 20 (D): 40/256 = 3.9 a second); regen per tick: max / 256 standing (7.2/s at 74), max / 512 walking but not below 1.0 stamina, none running; at 0 the next move is a walk | S (d2d `stamina_drain` / `stamina_regen`, combat.hpp:504..527, cite those functions; the 3.9/s matches docs/sme/mechanics-combat.md:216) |
| path type 7 (`00679ed0`) | a player's click: a far target (18+ subtiles) takes the straight "toward" path and stops at the first obstacle; a near one runs A* (200 nodes); monsters block the player (collision 0x1000), a merc does not (0x2000); doors with BlocksVis block until opened | S (upstream collision.md; d2d ports all three pathers and diffs them with `moves.py`, 15,000 cases) |

| Quantity | original | d2d | ratio orig/d2d | Evidence |
|---|---|---|---|---|
| walk, subtiles per tick (cells/s) | 0.375 (1.875) | 0.375 (1.875): `cells_per_sec(6)` = 6/16 x 25/5; measured 1.87 in town | 1.00 | V oracle; V-d2d gamedata.hpp:808, docs/d2d-fidelity.md:227 |
| run, cells/s | 2.8125 (velocity 2304) | 2.8125 when `running` (`run_velocity`) | 1.00 | V oracle; V-d2d fight.cpp:3640, world.cpp:2658 |
| diagonal step | same speed as straight (0.5624 / 0.5625) | float step along the path, same | 1.00 | V oracle `step`; V-d2d `move_frame` |
| what the bot does | runs while stamina > 0 (client toggle, default run), walks at 0 | **always walks**: `running` defaults false (world.hpp:91), set only by `cmd::Run` (world.cpp:2245), which `internal/d2dinput` never sends | 1.5 running, 1.0 at 0 stamina | S (the toggle default), V-d2d, V (grep of internal/d2dinput: no run verb) |
| stamina from full, clvl 1 Sorceress | 74 (D) / 3.9 per s = about 19 s of running, refilled standing at 7.2/s | the same port (stamina_frame, world.cpp:2690) but unused: the bot never runs | | S |
| walk animation rate at 100 % | WL 213, RN at 150 % 151 | the same (`moving_anim_rate`, diffed by `walk_rates.py`) | 1.00 | V-d2d test_combat.cpp:31 |
| 100-cell straight route, walking | 53.3 s | 53.3 s (56 s measured for the 480-subtile Blood Moor -> town leg with a detour, docs/d2d-walk.md leg 4) | 1.0 | analytic; V-d2d |
| the same route, running from full stamina | 19 s run (53 cells) + 25 s walk = 44 s | 53.3 s | 1.21 | analytic (U: the original's run/walk switching at 0 stamina is per move, not per tick, and the bot's standing time between moves refills stamina) |

So per-kill-time-split.md's rank 4 ("walking speed and route: not measured") resolves to: the engine speed is identical, the d2d bot is up to 1.5 x slower on foot because it never runs. The fix is on our side (send d2d's run toggle from `internal/d2dinput`, then d2d's own stamina port decides), not a d2d bug; the d2d run/stamina path itself is not oracle-checked (S).

## 2. Hit recovery: the decision (`0057cb00`, returns 1 = too small, 0 = get-hit)

| Step | Rule | Tag |
|---|---|---|
| `00639df0`(unit, state 1) | a frozen unit (state 1) never gets a get-hit | V (read; state 1 = freeze is S, states.md) |
| damage struct +0x28 | when not 0 and equal to the damage (+0x4c, 256ths): too small (U what +0x28 holds; absorbed?) | V (read), U (meaning) |
| +0x4c < 0x100 | under 1 point: too small | V |
| hit class (+0x60) -> divisor d | 8 for classes 2, 6, 10, 11; 32 for 4, 8; 64 for 5; 16 for 1, 3, 7, 9, 12 and every other value (0, 13 and up). Charged Bolt's Missiles.txt HitClass is 64 (D) -> 16 | V (jump table at `0057cbf8` read, oracle `hit`) |
| thresholds | damage < max/d: too small; < 2 max/d: get-hit on `00472210`(seed, 2) != 0 (50 %); < 4 max/d: on rand(4) != 0 (75 %); else get-hit. Max life from `00625d10` (stat 7, 256ths) | V (oracle: hit class 64 at 62 / 63 / 125 / 250 permille) |
| monster | needs MonStats2 mGH (`0046c140` bit 3); fallen1, zombie1 have it (D) | V |
| callers | `0057cee0` at 0x57d0c8 (monster) and 0x57d37a (player), only on result flag 4 (the hit landed) and not while state 0x15 (stun) is on; a get-hit then goes through `005a7e60`(mode 3) / `005a7c20` for a monster and `005809d0`(mode 4) for a player, so a unit already in GH restarts it | V (read) |

d2d (V-d2d): `hurt` flinches a monster when `damage x 8 >= max` (ai.cpp:2853), never below; the player at `x 12` (fight.cpp:4797), the merc at x 12 (fight.cpp:4692); no hit class, no roll, no frozen test. Upstream lists both as ponytails (combat.md:196).

## 3. Hit recovery: the length

| Address | What it does | Tag |
|---|---|---|
| `00623f50` 0x62409f..0x6240b1, 0x62435c..0x624381 | player mode 4 or monster mode 3 (GH): rate = AnimData speed (+0xc) x (`00621740`(unit, row 1 = stat 99, factor 120) + 50) / 100, at least 0, at most 0x7fff, stored at +0x4c | V (oracle `rate`: 128 / 138 / 148 / 161 / 179 / 320 at effective 0 / 4 / 8 / 13 / 20 / 75, speed 256) |
| `006217c0`, `00463390` | a plain mode ends after `ceil(frames x 256 / rate) - 1` ticks (the client tick, docs/research/cast-trace-gameexe.md); the server's monster mode end was not read here | V (client), U (server) |
| block (mode 9 / 6) | the same with (50 + FBR), 100 instead of 50 under state 0x65 | V (read, 0x62403d..0x624084; not run) |

| Unit, GH record (D, animdata.d2) | game.exe ticks at 0 FHR | d2d ticks | ratio | Evidence |
|---|---|---|---|---|
| Sorceress SOGHHTH 8 frames, speed 256 | 15 (14 / 13 / 12 / 11 / 10 / 9 / 8 / 7 / 6 / 5 at FHR 5 / 9 / 14 / 20 / 30 / 42 / 60 / 86 / 142 / 280: the community table) | 8 (`speed_frames` uses 100 + E, fight.cpp:456) | 1.9 | V oracle + S table; V-d2d |
| Fallen FAGHHTH 7, 256 | 13 (0.52 s) | 7 (`length_ms`, ai.cpp:172) | 1.9 | V; V-d2d |
| Zombie ZMGHHTH 5, 152 | 16 (0.64 s) | 8 (5 x 67 ms) | 2.0 | V; V-d2d |
| Fallen Shaman FSGHHTH 5, 256 | 9 | 5 | 1.8 | V; V-d2d |
| Gargantuan Beast BTGHHTH 4, 256 | 7 | 4 | 1.8 | V; V-d2d |

A monster's FHR is stat 99, 0 for every Act 1 Normal monster (U: MonProp rows not checked), so the monster column is the plain (50 + 0) case.

## 4. Kiting: seconds staggered per Charged Bolt hit

Life (D): Fallen 7 x 21..61 % = 1 to 4, Zombie 7 x 101..181 % = 7 to 12, Fallen Shaman 9 x 58..101 % = 5 to 9, Quill Rat 1 to 5 (MonLvl L-HP 7 / 9 at level 1 / 2, MonStats minHP / maxHP %); Charged Bolt level 1: 4 to 8 lightning per bolt (D, Skills.txt EMin 4 EMax 8).

| Monster | one bolt kills? | get-hit chance per surviving hit, game.exe (d2d) | staggered per hit, game.exe (d2d) |
|---|---|---|---|
| Fallen | always (4 to 8 vs 1 to 4) | moot | moot |
| Zombie | 4 vs 7..12: no; 8 vs 7..8: yes | bolt >= 1/4 of life (4/12 = 33 %): sure (sure) | 0.64 s (0.32 s) |
| Fallen Shaman | 4..8 vs 5..9: mostly | sure (sure) | 0.36 s (0.20 s) |

So on both backends every surviving Act 1 monster flinches on every bolt; the only difference is that d2d's flinch lasts about half as long, so a d2d Zombie regains about 0.3 s of approach per bolt (a Zombie walks velocity 1 = 0.31 cells/s, so 0.1 cell; a Beast or a Shaman more). Small for the Sorceress's kiting in Act 1; it grows with monster life (Act 2 onward a bolt stops being a sure flinch, and the 50 / 75 % bands start to matter) and with melee characters whose own GH is 8 instead of 15 ticks.

## 5. What settles the rest (not done)

- The server-side mode end for a monster's GH (whether the `- 1` of the client tick applies): run `005a8030`'s caller under unicorn or count ticks in a d2d-style probe on the original (live, not allowed here).
- The original bot's run share: log the player's mode (2 WL / 3 RN) per nav step on the live side; nothing in the soak logs carries it.
- `0057cb00`'s +0x28 field and MonProp FHR rows (U above).
- d2d's run and stamina path against game.exe (`0057f240` / `00580500` / `0057f090`): not oracle-checked, S only.
