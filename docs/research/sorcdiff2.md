# SORCDIFF2: counted differentials for what the lightning Sorceress actually casts

> status: research, 2026-11-03, agent SORCDIFF2 (card sorcdiff2). Offline (no game). Extends [sorcdiff.md](sorcdiff.md): that doc showed the gate character casts only Charged Bolt, Frozen Armor, Frost Nova and Fire Bolt and left three items open (mana payment timing, Frozen Armor trigger and group, the Charged Bolt pipeline). All runs are the real d2d build (upstream main `e1eea02a`, PR 181 merged) against game.exe's own code in the emulator, on the Windows build worker's Linux lane (one job at a time under its lock). Nothing pushed upstream, no PR, no comment. Tags: **V** verified (run or read in this card, the way is said), **S** secondary (an earlier doc), **I** inference, **U** unverified.

## Answer

| # | Item | Reproduced on the real build? | Counted | Fix |
|---|---|---|---|---|
| A1 | Mana paid at the command (d2d) vs at the action frame (game.exe), Charged Bolt, Frost Nova, Fire Bolt, Frozen Armor | **yes** | 42,168 of 42,168 cases where the cast began differ (53,628 cases in all; 11,460 refused for lack of mana agree) | `local/sorc-mana-timing-r2` commit 2: 10,560 differ |
| A2 | A cut cast (get-hit before the action frame) costs mana on d2d, nothing in game.exe | **yes** | 21,084 of 21,084 cut-before-event cases: d2d paid, game.exe did not | same commit: 0 of them differ |
| A3 | **New:** d2d's action frame comes one tick before game.exe's at FCR 9-18, 42-61 and 105-199 | **yes** | 132 of the 401 FCR values (all four skills); 10,560 cases once the payment is fixed | `local/sorc-mana-timing-r2` commit 3: 0 of 53,628 differ |
| B1 | Frozen Armor trigger (D5): `damage > 0` of any element vs physical record > 0 | **no harness built** | the game side needs the monster hit record chain (not run); see section 3 | none |
| B2 | Armor group (D2): Frozen, Shiver and Chilling Armor (and 9 more skills' states) stack on d2d, switch each other off in game.exe | **yes** | 48 of 5,476 ordered skill pairs differ | `local/sorc-armor-group-r2` commit 2: 0 differ |
| C | Charged Bolt do end to end: count by level, spawn, target, level and index of each bolt | **no deviation** | 2,400 cases (levels 1-60, random start and target), 0 differ | none needed |

Counts come from `tools/emu/sorcdiff2_run.sh` (stages `mana`, `bolts`, `armor`), logs of 2026-11-03 (V).

## 1. How game.exe casts, read and run (V unless tagged)

Disassembled with `tools/emu/disasm.py` and run natively under unicorn on the build worker.

| Address | What it does | Tag |
|---|---|---|
| `FUN_0057fe90` | the player mode start: mode tables, the animation state, then **`FUN_005539b0`**, then clears the unit's missile-made flag (`+0xc4` bit 6), then `FUN_0056faf0` -> `FUN_0056f640` (the skill start) | V read |
| `FUN_005539b0` | the animation scheduler: walks the position from the start frame in steps of the rate (`+0x4c`), and for each AnimData event marker it passes files an event `(type 0, tick, value, index)` at tick `start + n`; at the end a type 1 event | V read + run |
| `FUN_00580460` | the player mode event: a value 1 or 2 runs `FUN_0056fc50` (the do, which calls `FUN_0056f7f0` and so pays) unless the missile-made flag is set | V read |
| `FUN_005c9300` (Charged Bolt do) | sets that flag at its first instruction, then one `FUN_0059fa30` per bolt | V read + run |
| `FUN_0056c160` / `FUN_0056bfe0` | the mana check (start) and the payment (do); payment refused when short, the cast stands | V read + run |

Run (`python3 cast_mana.py anim`): the four skills have `srvstfunc` 0 (none), so they pay in the do; for the Sorceress SC records (14 frames, speed 256) the action event (value 1) is at tick **7** at 0 FCR and at 6 / 5 / 4 as the cast rate rises (FCR 20, 66, 200), the end event at 14 at 0 FCR down to 8 at FCR 200; every Sorceress SC record (one per weapon class) gives the same events. d2d's `Fight::cast` / `cast_missile` paid at tick 0, and its action frame was scaled by the cast length, not the cast speed.

Assumption (U): a mode replaced before the event (a get-hit) leaves the queued event unrun. The handler is the SC mode's, and the mode start calls `FUN_005533d0` and `FUN_00553990` before scheduling, which may clear queued events (not read); the cut cases rest on this, and on the interrupt landing strictly before or after the event (the case generator never puts it on the event's tick).

## 2. Mana timing (A1, A2, A3)

Harness: `tools/emu/cast_mana.py` (game side: the cast rate branch of `FUN_00623f50`, `FUN_005539b0` with the real AnimData SC record, `FUN_0056c160`, `FUN_0056bfe0`) and `tests/game/test_cast_mana.cpp` (d2d side: the real `Fight::cast` / `cast_missile` and `World::tick`, mana read each tick against a control run with no cast so regeneration cancels). A case is `id skill lvl mana fcr cut`: skills 36, 38, 44, 40 at levels 1-20, six mana pools around the cost, FCR from the breakpoints plus every FCR 0..400 at levels 1 and 20, and a get-hit after tick `cut` (none, or two ticks either side of the event). Output `id began paid_tick paid fired_tick`.

| build | commit | cases | differ |
|---|---|---|---|
| A: upstream main + harness | `cb07926f` | 53,628 | **42,168** (every cast that began) |
| B: + pay on the action frame | `e4181289` | 53,628 | **10,560** (all: d2d's action frame one tick early) |
| C: + action frame on the scheduler's tick | `9f835b6e` | 53,628 | **0** |

On A: 7,028 uncut casts differ (paid at tick 0, game.exe at 7 / 6 / 5 / 4), 14,056 casts cut after the event differ (same reason), and in the 21,084 cut before the event d2d pays and game.exe pays nothing. The 11,460 refusals agree. On B the residual is all one cause: the action frame. d2d scaled `action_ms` by `length / (cast_ticks x 40)`, and the cast length has the plain-cast "one tick early" end (`FUN_006217c0`), so at 132 of 401 FCR values it fired one tick before game.exe's scheduler (`ceil(256 x frame / speed)`); e.g. FCR 9: game.exe 7, d2d 6. This also moves the missile's release by a tick; it is separate from the payment and worth a PR line of its own.

Effect on the bot (I): the mana bar drops 4-7 ticks (0.16-0.28 s) later in game.exe than on d2d, so the bot's `ManaBurnTracker` (books at the command) and any mana-out fallback sample one tick-window early; a cast cut by a get-hit costs mana on d2d that game.exe keeps. Both matter only where mana is tight (level 1 Fire Bolt costs 2.5 mana, Charged Bolt 3, in the oracle's payments) and under heavy hit rates. Not measured in a gate run.

Fix (private fork, remote only, on upstream `e1eea02a`, numeric noreply identity, `Agent: Claude Code`, one change per commit):

| commit | subject | files |
|---|---|---|
| `cb07926f` | tests: when a Sorceress cast pays its mana vs cast_mana.py | `tools/emu/cast_mana.py`, `tests/game/test_cast_mana.cpp`, `tools/emu/test_tooling.py` (unit tests of the oracle's helpers), `tools/emu/README.md` |
| `e4181289` | game: a cast pays its mana on the action frame, not at the command | `Fight::owe` / `settle_cast`, `fight/tick.cpp`, `missiles.cpp`, `strike.cpp`, `setup.cpp`, `fight.hpp`; only skills with no start function (`srvstfunc` none); Inferno, Telekinesis, Thunder Storm, Hydra and the ammo skills still pay at the command |
| `9f835b6e` | game: a cast's action frame on the scheduler's tick, not prate's | `rules::cast_action_ticks`, `Player::action_ms`, `setup.cpp`, `strike.cpp` |

Conflicts (V, `git merge-tree`): the branch conflicts with the held batch 2 (`local/batch2-bundle-r3`, private) in `fight/tick.cpp` and `rules/combat.hpp` (its `pay` and mana check changes; same function neighbourhood), and in `combat.hpp` with `local/sorc-lightning-seq` (adjacent additions). A rebase after batch 2 lands is a mechanical hunk fix (U, not tried). The earlier push `local/sorc-mana-timing` has a harness whose `srvst` assertion was wrong; it is superseded by `-r2` and left (no force-push).

## 3. Frozen Armor (B1, B2)

**B2, armor group: reproduced and fixed.** `tools/emu/armor_group.py` runs `FUN_0056c740` natively over the States table game.exe loads: for each other state of the same States.txt group (word at `+0x1e`) the unit has, it clears it. `tests/game/test_armor_group.cpp` starts X then Y through the real `Fight::start_state` for every ordered pair of the 74 skills that give a state (5,476 pairs). On main 48 pairs differ (d2d keeps X, game.exe drops it): Frozen (40), Shiver (50), Chilling Armor (60) with each other and with Bone Armor (68) and skills 209 / 210; other groups are 223, 228, 232, 233 and 350 among themselves, and 258 with 267 (names not checked). After `35345dd9`: 0. The bot casts only Frozen Armor (two points, no Shiver or Chilling point in plan `fallback-lightning-v2`), so this changes nothing in a gate run (I). Branch `local/sorc-armor-group-r2` (`65ccfb0b` harness, `35345dd9` fix: `StateInfo::group` loaded from States.txt, `Fight::add_state`).

**B1, trigger D5: not built.** d2d freezes the striker when the monster blow's `taken.damage > 0` (whole points, every element, `ai.cpp`); game.exe tests the hit record's physical field `+8` (256ths) after the damage stage. A harness needs game.exe's record chain for a monster's melee hit (the physical part after absorb and reduction), which neither `armor_hit.py` (it takes the record as given) nor the batch 2 damage-roll work runs. The two differ for an elemental-only blow or a sub-point physical one (I); whether any monster the bot meets before clvl 16 has an elemental-only melee attack was not counted (U). Not a proven deviation.

## 4. Charged Bolt pipeline (C)

`tools/emu/cast_bolts.py` runs `FUN_005c9300` natively (its calc `FUN_00646ca0` for the count, the params block, the launch loop; hooked: skill records and levels, the stat getter, the srvmissilea row lookup, the caster's subtile, the target point, and `FUN_0059fa30`, whose params are read and nothing made). `tests/game/test_cast_bolts.cpp` casts through the real `Fight::cast_missile` and the world tick and reads the launches. 2,400 cases (levels 1-60, 40 random start / target subtiles each): count, skill, level, start subtile, target subtile and every bolt's index **0 differ** (levels 1 and 20 give 3 and 22 bolts on both). Branch `local/sorc-bolts` (`73784683`, harness only; no fix needed).

What the rest of the pipeline stands on (not re-run here): the hit roll `missile_hit` (60,000 cases, merged, S), damage per bolt `skill_damage` (116,000, 0 differ on `005c7cab`, sorcdiff.md) and the wiggle path (`missiles.py`'s port of `FUN_005c9290`, S). Unproven: the per-bolt wiggle seed (game.exe seeds from the target x plus the index, `FUN_005c9290` -> `FUN_00650e40`; d2d's `bolt` index feeds its own flight rule) is read-equal only, and no harness runs both flights tick by tick (U). Frost Nova (do 22, nova ring) and Fire Bolt (plain `srvmissile`): their pay timing is in section 2; ring aim, hits and freeze are the merged / re-run differentials of sorcdiff.md; no new piece found.

## 5. Ready for the independent Opus check (ranked by how much a gate run meets it)

1. **A1-A3 as one PR candidate**: the payment move and the action-frame tick (two commits, one claim each). Check the cut assumption (section 1, U): read `FUN_005533d0` / `FUN_00553990` for whether the mode start clears queued events, and the server's order of event dispatch against the get-hit on the same tick. Check that `settle_cast` / `owe` cover every cast path the bot uses (`cast_summon`, `start_swing` still pay at the command: intended, they have start functions or ammo).
2. **Per-bolt flight of Charged Bolt tick by tick** (wiggle seed and path, d2d real flight vs game.exe's missile step), the one place the pipeline has no real-build counted check.
3. B1, the Frozen Armor trigger: needs the monster-hit record chain; low value for the bot (act 1 melee is physical).
4. B2 as a small independent PR (no bot effect).
5. A rebase of A1-A3 onto batch 2 (conflicts in section 2).

## 6. Compute and reproduction

The Windows build worker's WSL lane, one job at a time under `C:\d2lab\linux.lock` (waited for it up to 40 minutes behind other agents' jobs), g++ 15.2, ccache; the worker host ran no build or emulator job. Script copy: `tools/emu/sorcdiff2_run.sh` (stages `dis`, `emu:SCRIPT:ARGS`, `mana`, `bolts`, `armor`; input a bundle of the private branches). Stage times: `mana` 161 s warm (cold 254 s incl. three builds), `bolts` 102 s, `armor` 25 s warm.

Re-check: `uv run python cast_mana.py gen 1 > cases.tsv; uv run python cast_mana.py game cases.tsv > g.txt; ../../build/tests/test_cast_mana --cast-mana cases.tsv | grep -E '^[0-9]+ ' | diff g.txt -` in `tools/emu` of the fork branch; the same shape for `cast_bolts.py` and `armor_group.py` (README rows in the branches).

## Shared-file lines for the merger

See `openspec/changes/sorcdiff2/merge-lines.txt`.
