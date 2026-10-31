# XP per kill traced in game.exe 1.14d: why d2d pays more in the Den and Blood Moor

> status: research, 2026-10-30, agent DECOMPXP (card decompxp). Offline only: nothing was posted, pushed or filed on GitHub; no game instance was started. Tags: V = read in game.exe or run in the unicorn oracle by this agent; V-orig = measured on the live original (docs/probes/orig-casts.csv, docs/soak-analysis.md); V-d2d = measured on the lab binary (`~/d2d-lab/fidelity/out/lowxp.jsonl`); D = read from the patched 1.14d tables with `mpq-cat` (recipe in docs/process/mpq-data-reading.md, sanity line `Id<TAB>hcIdx` seen); S = upstream's `docs/research/re/*.md` (psi29a/d2decomped) or secondary; U = unverified.

Question (docs/d2d-fidelity.md check 4b): at clvl 4 to 6 d2d pays 28.8 to 31.1 XP per kill in the Den of Evil and 21.2 to 22.0 in the Blood Moor; the live original paid 16.8 to 19.4 and 18.3. Is the formula different?

**Answer: no.** Per monster, the original and d2d pay the same XP at those levels (18 Fallen, 21 Quill Rat, 33 Zombie, 32 Fallen Shaman, 48 Gargantuan Beast). The gaps are what the bot kills (composition) and how it counts kills. d2d's old `kill_exp` (upstream e36922ea, the base of our lab binaries) does differ from game.exe, but only from a level difference of 6 up; upstream fixed it in 01337ba3 on 2026-10-29, so there is no PR to write.

## How it was checked

- game.exe 1.14d: the read-only copy `/tmp/casttrace/game.exe` left by CASTTRACE (docs/research/cast-trace-gameexe.md), disassembled with upstream's `tools/emu/disasm.py --exe` in the venv `/tmp/casttrace-venv` (`D2_MPQ_DIR=~/d2d-lab/data`); never run as a game.
- Oracle: `tools/emu/xp_award.py` (ours; copy it into upstream's `tools/emu/` to run) calls `FUN_0057e2f0` and `FUN_0057e390` under unicorn through upstream's `emu.py`. Grid: 11 exp values x clvl 1..99 x mlvl 1..110 = 119,790 lines, `exp clvl mlvl award`, diffed against a port of the old d2d `kill_exp`.
- Data: `monstats.txt`, `monlvl.txt`, `superuniques.txt`, `monumod.txt` from the patched stack (lowercase `patch_d2.mpq` link, scratch dir deleted).
- Measurements already in the repo: `docs/probes/orig-casts.csv` (per-kill XP by monster on the original, Blood Moor, clvl 2 / 5 / 9), docs/soak-analysis.md section 2 (live area totals), `lowxp.jsonl` + `fid_xp_report.py` on the worker host (d2d sweeps: 598 Blood Moor and 456 Den kills at clvl 4 to 6, with the composition per stop).

## The award path (addresses; explanations in our words)

| Address | What it does | Tag |
|---|---|---|
| `0057e990` | a monster died with a player (or pet) behind it: reads the monster's stat 13 (experience; nothing if <= 0) and stat 12 (its level, mlvl); `0057e7b0` resolves the killer to the owning player; the merc (`00574ec0` type 7) gets its own `0057e480` at the merc's level, x 0x56 / 256 (toward zero) when the merc did not land the blow, through `0057e860`; then, with no party (`00554630` == 0xffff), the player's award is `0057e480(exp, clvl, player, game, mlvl)` added by `0057e510`; with a party `0057e6c0` | V (party and merc paths read, not run) |
| `0057e480` | exp capped at 0x7fffff; exp <= 0 returns 1 (unreachable from `0057e990`); clvl >= the class's MaxLvl (`00611830`, per-class table at `[0x96c8a8]`) returns 0; then `0057e2f0` (level difference), `0057e390` (ExpRatio), + stat 0x55 (item_addexperience) % through `00483360`, then `0057e3f0` (a player's award passes unchanged; a merc's is capped at its Hireling.txt level span / 64) | V |
| `0057e2f0` | the level-difference step, eax = mlvl, edx = clvl, ecx = exp. clvl >= mlvl: exp x `[0x6e1668][min(clvl - mlvl, 10)]` / 256. mlvl > clvl: clvl >= 25 gives exp x clvl / mlvl (any difference, even 1); clvl < 25 gives exp x `[0x6e1694][min(mlvl - clvl, 10)]` / 256. An entry of 256 returns exp untouched | V (oracle) |
| `0x6e1668` | 256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13 (difference 0..10+): 100 % up to 5 levels above the monster, then 80.9 / 62.1 / 43.0 / 23.8 / 5.1 % | V (read from the image) |
| `0x6e1694` | 256, 256, 256, 256, 256, 256, 225, 174, 92, 38, 5: 100 % up to 5 levels below, then 87.9 / 68.0 / 35.9 / 14.8 / 2.0 % | V |
| `0057e390` | Experience.txt ExpRatio: award x ratio(clvl) >> shift, shift = the level-0 row (`00613e60`); with an overflow guard ((award >> shift) x ratio when award x ratio would not fit). Ratio 1024 at shift 10 leaves the award unchanged (oracle: 1000 -> 1000; ratio 768 -> 750) | V (oracle, getter hooked) |
| `00483360` | MulDiv(a, b, c), truncating. Shortcuts: a > 0x100000 and c <= a >> 4: (a / c) x b; b > 0x10000 and c <= b >> 4: (b / c) x a; else 64-bit multiply then divide. So (0x7fffff, 25, 31) gives 6,765,000, not 6,765,005 | V |
| `0057e510` | adds: new = stat 13 + award, capped at `00611800(class, MaxLvl - 1)`; stat 0x1d += award; stat 13 = new; if `00611860(class, new)` (the level for that experience) differs from the level, `00570880` (level up) and `005c0c30(game, 12, ...)` | V |
| `00573cb0` / `006538a0` | the monster's stat 13 at spawn: MonLvl XP for its level x MonStats Exp % / 100 (MulDiv), + the player-count % (`00573910`: 0 for 1 and 2 players, 50 per player from the third); Normal uses the monster's own Level column | S (upstream monsters.md, party.md; confirmed by the per-monster numbers below) |
| `005a0e40` | the `leveladd` unique mod: level + 3, experience x 5 (`lea eax, [eax + eax*4]`) | V |
| `005a0e80` | champion: level - 1 after leveladd, experience reduced to 3x | S (upstream monsters.md) |

Not in the path: no "cleared area" bonus, no XP share outside a party, no cap per kill for a player (`0057e3f0`'s 1/64 cap is the merc's). The Den of Evil quest pays a skill point, not experience (S; Akara's reward in upstream quests.md). The merc's share is its own award, not taken from the player's (V, `0057e990`). With one player the count bonus is 0 % (S).

## Per-monster XP (Normal, expansion, one player)

| Monster | mlvl | MonLvl XP (D) | Exp % (D) | base = floor(XP x % / 100) | d2d fit, clvl 4..6 (V-d2d) | original, clvl 2 and 5 (V-orig) | original at clvl 9 (V-orig) vs table |
|---|---|---|---|---|---|---|---|
| Fallen | 1 | 30 | 61 | 18 | 18.00 | 18 | 7 = floor(18 x 110 / 256) |
| Quill Rat | 1 | 30 | 71 | 21 | 21.00 | 21 | 9 = floor(21 x 110 / 256) |
| Zombie | 1 | 30 | 111 | 33 | 33.00 | 33 | 14 = floor(33 x 110 / 256) |
| Fallen Shaman | 2 | 40 | 81 | 32 | 32.00 | (not in the rows) | 19 = floor(32 x 159 / 256) |
| Gargantuan Beast | 2 | 40 | 121 | 48 | 48.00 | not seen in the Blood Moor rows | U |
| Corpsefire (zombie1 + leveladd = mlvl 4) | 4 | | | 33 x 5 = 165 expected | 19.04 (6 kills, always with 5 minions: the fit cannot separate them) | never reached by the bot (docs/orig-spawns.md: 0 of 4 visits) | U |

Every single-type row of the live original equals the formula to the XP point, including the clvl 9 rows (difference 7 and 8: the 159 and 110 entries). So for mlvl 1 and 2 against clvl 1 to 7 (difference at most 5) both backends pay the base value; the formula cannot produce a 20 % or 60 % gap there.

## Den of Evil +60 % and Blood Moor +20 %: ranked explanation

d2d Den sweep (V-d2d, 456 kills at clvl 4 to 6): Fallen 244, Zombie 84, Fallen Shaman 57, Gargantuan Beast 35, Zombie minions 30, Corpsefire 6; 13,638 XP = 29.9 per kill. With the formula above, the 420 ordinary kills pay 10,668 XP (25.4 per kill), so the 36 boss-pack kills carried 2,970 (82 per kill). Fallen alone pay 18.0; Fallen + Zombie + Shaman at the sweep's ratio 23.3.

Live Den (V-orig, docs/soak-analysis.md 2): 274 kills / 5,304 XP = 19.4 at clvl 4, 50 / 840 = 16.8 at clvl 5, 578 / 10,435 = 18.1 at clvl 6: the value of "mostly Fallen, some Zombies, no boss pack".

d2d Blood Moor sweep (V-d2d, 598 kills): Fallen 357, Quill Rat 123, Zombie 117 = 21.5 per kill. Live Blood Moor at clvl 5 (V-orig, orig-casts.csv, 62 rows): Fallen 39, Quill Rat 17, Zombie 6; the formula on those rows gives 20.3, the rows carry 19.1 (19 rows with 0 XP against 9 rows with a double: about 10 of 62 kills were credited with no XP at all); the soak's 78 kills paid 18.3.

1. **Composition** (most of both gaps): the sweep kills everything on the level; the bot kills what its fighter reaches. The Den's Zombies (33), Beasts (48) and Corpsefire's pack (82 per kill in d2d's numbers, about a fifth of the sweep's XP) are the kills the bot does not make: the live 18 is the Fallen value. In the Blood Moor the bot's rows hold 10 % Zombies against the sweep's 20 % (21.5 -> 20.3).
2. **Kill counting on the live side** (5 to 10 %): about one row in six in orig-casts.csv has no XP beyond the double-count pairs, so `noteKills` credits kills that paid nothing (a bolt still in flight, a monster that died to something else, or a unit that left the client table: U which). 20.3 -> 19.1 -> 18.3.
3. **Formula**: nothing at a level difference of 5 or less. The old d2d formula differs only from 6 up (next section), which starts at clvl 7 against mlvl 1: the fidelity sweeps' clvl 10 to 14 rows differ by rounding (Carver 42 at clvl 11: game.exe 33, old d2d 34).
4. **Rejected**: player-count bonus (0 % for one player), a quest or cleared-Den reward (none in experience), the merc's share (separate award), the experience shrine (stat 0x55, 0 here), the party path (none in single player).

What would settle 1 and 2 (not done, live work): log the monster name and level with every XP gain on the live side (docs/d2d-fidelity.md "must still be verified", item g) and count the 0-XP kills.

## d2d `kill_exp` vs game.exe (the grid)

Old d2d (upstream e36922ea, `components/rules/combat.hpp`: 81 / 62 / 43 / 24 / 5 % above, exp x clvl / mlvl when the monster is more than 5 above, nothing else): 37,288 of 119,790 grid lines differ.

| Class | Lines | What d2d did | game.exe | Example (exp, clvl, mlvl): game.exe / d2d |
|---|---|---|---|---|
| B: mlvl > clvl + 5, clvl < 25 | 20,509 | exp x clvl / mlvl | table 225 / 174 / 92 / 38 / 5 of 256 | (165, 2, 8): 145 / 41 (Rakanishu at clvl 2); (18, 1, 7): 15 / 2 |
| A: clvl > mlvl + 5 | 9,568 | 81 / 62 / 43 / 24 / 5 % | 207 / 159 / 110 / 61 / 13 of 256 | (42, 11, 5): 33 / 34; (100, 10, 1): 23 / 24 |
| C: mlvl in clvl + 1 .. + 5, clvl >= 25 | 4,125 | full | exp x clvl / mlvl | (100, 25, 26): 96 / 100; (1000, 30, 33): 909 / 1000 |
| D: mlvl > clvl + 5, clvl >= 25, exp > 2^20 | 3,086 | plain division | `00483360`'s shortcut | (0x7fffff, 25, 31): 6,765,000 / 6,765,005 |

Upstream 01337ba3 ("Kill experience as game.exe's FUN_0057e480 (emu-checked)", 2026-10-29) replaces `kill_exp` with the two 256ths tables and the clvl 25 rule, adds `kill_award` (MaxLvl 0, ExpRatio, stat 0x55) and the merc share, `rules::scale` for the MulDiv shortcuts, `tools/emu/kill_exp.py` (random cases from the same `FUN_0057e2f0`) and 400 oracle asserts in `tests/test_combat.cpp`; `docs/research/re/party.md` carries the same reading as the table above. Its code was read (S for the port itself: not built or run here); its oracle calls the same function with the same register convention as ours.

**Consequence for us**: every lab binary (`d2d.lab2` .. `d2d.lab5`, base e36922e, docs/d2d-lab.md) still has the old formula. Harmless at clvl 1 to 6 in Act 1 areas; from clvl 7 on, and for any kill of a boss 6+ levels above the character, the replicated soak's XP/h is off by the classes above (B is the large one: a clvl 2 character on Rakanishu gets 41 instead of 145). Recommendation for the fork owner (not this agent's to do): rebase the lab branch onto upstream >= 01337ba3 or cherry-pick it.

## PR candidate

None: the one real bug is already upstream. The draft file docs/research/upstream-pr-candidates-xp.md records this and the two leads that would need a trace first.

## Open and unverified

- Superunique and minion experience: whether `005a49b0` (superuniques) and the minion path take `leveladd`'s x 5 was not read (U). d2d's level 4 for Corpsefire says the level part applies; its 19.04 fit is an artefact of 6 collinear kills. One line of a d2d log (`killed Corpsefire (+N exp)`) on a build >= 01337ba3 would settle d2d's side; the original's needs a live kill with the name logged.
- The party (`0057e6c0`, `0057e5a0`) and merc (`0057e860`) paths were read for the control flow only; the 0x59 / 0x56 constants and the float share are upstream's reading (S).
- The live counter's 0-XP kill rows: cause U.
- The Den's live composition is not logged; the +60 % attribution to composition rests on the sweep's composition, the formula and the Fallen value matching the live average, not on a live per-monster count.
