# Monster wake-up and aggro: game.exe 1.14d traced vs d2d (agent DECOMPWAKE, 2026-10-31)

> status: research, current. Riskmap rank 5 `mon_wake` (docs/research/d2d-fidelity-riskmap.md section 5). Question: when does an idle monster notice the player and start acting, and does d2d do it on the same rule (kills/h, crowd size, deaths)? Tags: V = read in game.exe with `disasm.py` or run in game.exe's own code under unicorn by this agent (`tools/emu/wake_cadence.py`); V-up = upstream's own oracle (`tools/emu/search.py`, `sight.py`, `moves.py`, re-read, not re-run); V-d2d = read in d2d (upstream/main f8aab0e8, worktree `~/d2d-lab/pr/wake` on the worker host); D = 1.14d patched MonStats read with `mpq-cat` (docs/process/mpq-data-reading.md); S = upstream `docs/research/re/monster-ai.md`; U = unverified; I = inference. Offline only: no game instance, nothing pushed, posted or filed; the upstream branch `fix/mon-wake` exists with no commits.

## Verdict in four lines

1. **No deviation found in the wake rule.** game.exe's notice is "a think comes due and the target search finds a player under 35 subtiles (7 cells), no sight needed outdoors"; d2d runs the same search (`search_target`, upstream oracle 20,000 cases) on the same cadence (this agent's oracle: 141,088 cases, 0 differ). A clean negative: no PR candidate, no patch written.
2. **The radius is data and it is universal**: no row of 1.14d MonStats.txt sets `aidist` (0 of 735, D), so every monster, Act 1 to 5, uses FUN_005dd7f0's default 0x23 = 35 subtiles; `aidel` is 15 (Normal) / 14 (Nightmare, Hell) for every Act 1 type (D). There is no stealth, no difficulty modifier, no "always aggro" flag in the search (V-up trace: it reads no player stat).
3. **Group members do not wake together** in either: the Fallen leader's rally (FUN_0058f730) only copies command 1 into each minion's record (V); the minion still needs its own search to find a target before its think proper runs (driver order, V). Being hit never sets the aware flag (S, V-d2d).
4. Three structural items stay U (section 4): which rooms think (near list vs d2d's 30 cells), the first think after spawn, and a think event due during a get-hit. None of them changes who is noticed at what distance; they can shift a wake by up to one think period (15 frames) in corner geometries.

## 1. game.exe's chain (V unless marked)

| Address | What it does | Tag |
|---|---|---|
| FUN_005b1740 (ECX game, EDX unit) | the think driver, run when the unit's type-2 event comes due: with no MonStats or MonStats2 row it returns; FUN_005b10e0 (stun / door / flee), FUN_005b1650 (the search step), FUN_005b13e0 (speech, teleport, leash), then the MonAI's think at entry +4. Each step that returns non-zero ends the think (`0x5b180b`, `0x5b181e`, `0x5b182f`) | V |
| FUN_005b1650 | search type 1: FUN_005de890, result `== 0` ends the think (`0x5b169b sete`); type 2: FUN_005dd7f0 only, never ends it | V |
| FUN_005dd7f0 | the target search: lists 0..7 (players and pets), 8, 9; best from MonStats aidist, 0 -> 0x23; `nearest` = the player's FUN_005dc530 distance; a player at 0x37 or more is skipped with its pets; sight only in a non-Outdoors preset room until flag 8 | V-up (`search.py` 20,000 cases) + S |
| FUN_005de890 | no target: FUN_005dd2b0 (last mode, monster data +0x54 via FUN_005734e0, is 3 GH or 0x13) and mWL (FUN_0046c140(class, 2)) -> FUN_005de200 wander 5; else FUN_0064d910(room, x, y, pattern, 0x40) and mWL -> wander 5; else FUN_005de080 stand: `nearest` >= 0x23 -> 25 frames, < 0x19 -> 10, else `nearest` - 10 (`0x5de995`..`0x5de9b0`) | V (oracle `stand`) |
| FUN_005de080 (unit, n) | n 0 -> 1; a unit not in NU goes to NU (FUN_005a7e60 mode 1, FUN_005a7c20); FUN_00540e60(2, 0) drops the pending type-2 events; FUN_005417d0 schedules the think at frame + n | V |
| FUN_005a73e0 (ECX game, EDX ctx) | NU or a failed mode start: if FUN_005415a0(unit, 2) (the earliest pending type-2 event's frame, 0 none) > game +0xa8, nothing; else state 0x15 -> frame + 0x2d; else frame + MonStats +0x4f (classic) or +0x4f + difficulty (expansion), 0 -> 15 | V (oracle `aidel`) |
| FUN_005a8030 | a mode's end: WL / RN run the driver at once; other modes go to NU through FUN_005a7c20 (-> FUN_005a73e0) | S + V (the driver call at `0x5a80f5`) |
| FUN_0058f730 | the leader's command to its group: for each minion of its group list (+0x34, FUN_00552f60 by id) FUN_0058ec90 gets the record and the 5 words of the command are copied in (`0x58f7c6`..`0x58f7e7`); no event, no flag, no mode | V |
| FUN_005734e0 | the last mode: monster data +0x54 (1 when none): monster modes are 0..0xf, so FUN_005dd2b0's 0x13 branch never fires for a monster | V |
| Think during GH / KB | the timer handlers FUN_005a8330 / FUN_005a83e0 test only frozen (state 1) and dead before calling the driver (`0x5a83ab`..`0x5a83d1`); whether FUN_005a7c20 drops the type-2 events when GH starts was not read (its callees FUN_00553990 / FUN_005539b0 are unread) | U |

RNG: the search draws nothing on the unit seed except Confuse's kind 3 (one `& 1`, S); a no-target wander draws FUN_005de200's three (upstream models them in `mon_ais.py`, `rules::think_wander` makes the same three, V-d2d); a stand draws none. So an idle monster's seed does not move while it waits (V, by the branches above).

## 2. d2d, item by item (V-d2d, `components/game/ai.cpp` and `components/rules/monsters.hpp` at f8aab0e8)

| Item | d2d | Verdict |
|---|---|---|
| Target search | `search_target` (ai.cpp:508): `best = aidist > 0 ? aidist : 35`, `rules::search_pick` (monsters.hpp:912) with the 0x37 skip, dead player 0x7fffffff, sight only indoors until `sighted`; the area flag | SAME (V-up oracle; ponytails: list 9's path test, mode 5's order, a dead pet) |
| No-target cadence | ai.cpp:808-820: `has_walk && (left_mode == "GH" \|\| blocked 0x40)` -> `think_wander(seed, 5)`, else `idle(nearest < 25 ? 10 : nearest < 35 ? nearest - 10 : 25)` | SAME (oracle `stand`, 85,820 cases, 0 differ); `has_walk` is the WL animation's presence, game.exe's is the MonStats2 mWL bit (I: equal for every placed type, U not checked row by row) |
| Think period | `idle()` ai.cpp:618-621 (`max(frames, 1) * 40 ms`), `set_off` :631 `aidel ? aidel : 15`, `next_act` kept across `set_mode` (:166-175) | SAME (oracle `aidel`, 55,268 cases, 0 differ); the attack-end schedules (:1865, :1912, :2686) use `aidel * 40` with no 0 -> 15: only the 19 unplaced rows with a blank aidel (chaoshorde, darkguard, bloodmage, D) would differ, none in Act 1 |
| Stunned | holds NU to `stun_until` (:1797) instead of game.exe's 3-frame stands / +0x2d schedule | same outcome (stands), different bookkeeping (I) |
| Rally | the Fallen think (monsters.hpp:1520-1545): leader, `dist < 15`, `roll < aip1` -> `rally`; ai.cpp:1471-1473 sets `ai_command = 1` on the group; consumed only inside the think proper, which runs only with a target | SAME shape (V-d2d read; the Fallen think itself is hand-ported with asserts in `tests/test_monsters.cpp`, not oracle-run: U) |
| Hit wakes? | `aware` is set only from `search_target`'s result (:668); a GH only sets `left_mode` | SAME (S for game.exe: flag 8 is set only by a found target) |
| Which monsters think | `Fight::near_players` (fight.cpp:4630): within 30 cells of a hero on both axes | DIFFERS in scope (section 4) |
| First think | `make_monster` leaves `next_act = 0` (ai.hpp:188): a spawned monster thinks on its first updated frame | U on game.exe's side (section 4) |

The 58 `ponytail` notes in ai.cpp: the wake-related ones are at :497-505 (mode 5's candidate order, a dead pet, list 9), :600 (the seed as the look left it), :608-612 (FUN_0064d910's one-subtile test, the in-melee path test), :867 (the Shaman's corpse search as 80 subtiles), :1049 / :1067 / :1213 / :1234 (packs in list order, the nearest instead of a radius search), :2729 (a spawn's spot). None changes the notice distance or the cadence; the rest are attacks, casts, missiles and Act 2-5 thinks.

## 3. Differential (V): `tools/emu/wake_cadence.py`

game.exe's FUN_005de890 and FUN_005a73e0 run natively under unicorn (upstream harness `emu.py`, read-only `game.exe` copy on the worker host, never run as a game). Hooks (all outside the claim): FUN_005dd7f0 returns the case's found / not and writes `nearest` to the out-distance; FUN_005734e0 the last mode; FUN_0046c140 the mWL bit; FUN_0064d910 the door word; FUN_00649180, FUN_00620bb0 zero; FUN_005de200 / FUN_005de080 record their argument; FUN_005415a0 the pending frame; FUN_00639df0 state 0x15; FUN_005417d0 records the scheduled frame. The d2d side is a Python copy of the rules cited above.

| Table | Cases | Differ |
|---|---|---|
| stand: found x last mode 0..0xf x mWL x door x nearest 0..100, exhaustive 13,600 + random | 85,820 | **0** |
| stand, last mode 0x13 (FUN_005dd2b0's other value; a player's KB, not a monster mode) | 5,259 | 654 (unreachable: informational) |
| aidel: 0..255 x pending {none, past, now, +1, +50} x classic / expansion x difficulty, exhaustive 5,120 + random, not stunned | 55,268 | **0** |
| aidel, stunned (game.exe +0x2d; d2d holds NU instead) | 54,972 | 0 vs the +0x2d model (informational) |

```sh
# The worker host; upstream harness in the clone, read-only game.exe copy
cd ~/d2d-lab/pr/wake/tools/emu && cp ~/code/d2lod-wt/decompwake/tools/emu/wake_cadence.py . && source /tmp/casttrace-venv/bin/activate
D2_MPQ_DIR=~/d2d-lab/data python wake_cadence.py --exe /tmp/casttrace/game.exe 100000 7     # about 15 s; ends with "ok"
D2_MPQ_DIR=~/d2d-lab/data python disasm.py --exe /tmp/casttrace/game.exe 5de890            # and 5a73e0 5de080 5dd2b0 5734e0 5415a0 5b1740 5b1650 58f730
```

No "before" row: nothing was patched. A broken model (`last_mode == 3` dropped from `d2d_stand`) gives 704 mismatches on the same cases (every got-hit + mWL row), so the check bites.

## 4. What stays open (U), and what it can move

- **Which rooms think.** game.exe runs the units of the active rooms (the near list FUN_0066c370: rooms under 6 tiles = 30 subtiles apart, S); d2d every monster within 30 cells = 150 subtiles. A monster 31 to 35 subtiles from a player standing at its room's edge can sit in a room the near list does not hold: game.exe's does not think (and does not wake), d2d's wakes. Everything within 30 subtiles of the player is active in both (the player stands inside its room). Bound: one sliver of the 35-subtile ring, geometry-dependent (I). Beyond 0x37 nothing is taken in either.
- **The first think after spawn.** d2d thinks on the first updated frame; game.exe's spawn chain (FUN_005b2a00 -> FUN_00552d60, S) was not followed to its first FUN_005417d0 / FUN_005a73e0. At most one think period (15 frames = 0.6 s) at the room's first population; a room populates when it enters the near list, before the player is within 35 subtiles of most of it (I).
- **A think event due during GH.** The timer handlers do not test GH; whether FUN_005a7c20 drops the pending type-2 events when GH starts is unread. d2d thinks at the GH's end when `next_act` passed during it, else at `next_act`. At most one get-hit length of difference per hit on a standing monster (I).
- The Fallen / Zombie / Skeleton / QuillRat thinks themselves (what a woken monster does: the Fallen walks in only from aip2 = 10 subtiles, else 30 % wanders 3 or stands 10) are upstream hand ports with asserts, not oracle runs (`mon_ais.py` covers Act 2-5 and four Act 1 AIs). Not this card's question; a candidate for a later oracle if kills/h still disagree.
- Not run live on either backend; the original's wake distance (plan-v2's live probe) remains the only way to see the whole system at once.

## 5. Verifier verdict (independent adversarial agent, Fable, read-only)

VERIFIER_PLACEHOLDER
