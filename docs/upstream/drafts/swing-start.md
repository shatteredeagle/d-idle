# Draft (local, NOT posted): "A player's swing starts at its weapon's start frame and strikes on game.exe's tick"

> status: r1, ready for an independent verifier pass (prswing, TASK-411, 2026-11-14). Branch `fid/swing-start-r1` on the private fork remote, head `ebeb4268`, two commits on upstream main `f9b8cfed`: oracle `5af717cd`, claim `ebeb4268`. Replaces the earlier draft `swing1-first-strike.md` (decomp15, TASK-403; "needs fork patch + independent verifier", on its own branch): its claims are restated here with the numbers of the built fix. Nothing here is posted or pushed upstream. Evidence of the finding: `docs/research/swing1-trace.md` (decomp15's branch). Tags: **V** run (game.exe's functions natively, or d2d's real classes), **S** read, **U** unverified.

PR title: `A player's swing starts at its weapon's start frame and strikes on game.exe's tick`

## What changes

Two rules of one function (`Fight::set_pmode`, an attack mode), both the clock FUN_005539b0 files a swing's events on:

1. **Start frame (Amazon and Sorceress).** FUN_00623b10 is 0 for a monster and for a player mode other than A1 and A2 (S3 and S4 only for a dual wielder). For an A1 or A2 swing it looks up the weapon in hand (FUN_00623990), its class key (FUN_00629fe0: the weapons.txt `wclass` through the map at 0x7446a0) and reads the dword at `0x6e8e60 + 4 (row x 7 + player class)`. Only an Amazon and a Sorceress are not 0: frame 1 barehanded, frame 2 with a 1hs, 1ht, stf, 2hs or 2ht weapon, 0 with a bow, crossbow or claws. d2d played the whole animation, so these two classes' melee swings were 1 to 3 ticks too long (a Sorceress staff swing 18 ticks against 16, about 12 percent of her melee damage per second).
2. **The strike tick (every class).** FUN_005539b0 walks the animation from the start frame at the unit's rate and files the action event on the tick its AnimData flag frame is reached: `max(1, ceil(256 (flag - start) / rate))`, the end event at `max(1, ceil(256 (frames - start) / rate))`. d2d scaled the action frame by the rounded swing length instead, so its strike was one tick late in 1,123 of 8,890 cells of the five classes without a start frame (the same cause as the one-tick residual left in Zeal's chained strikes after `276f85c0`).

`rules::attack_start_frame` holds the table (rows by weapon class, 7 player classes, as `swing_start.py table` prints it from the image); `rules::frame_tick`, `attack_ticks` and `attack_action_tick` the clock; `ItemBase::wclass` the weapon's class (weapons.txt); `Fighter::attack_start` the start frame of the weapon in hand (worn or not: U that game.exe's held weapon ignores the strength and level requirements; the COF record follows the worn weapon the same way); `Fight::action_at()` the strike time the three strike sites read (an attack mode's `action_ms`, any other mode's old formula).

Scope kept narrow: the start frame applies to a plain swing and to a skill whose `anim` is A1 (`Fight::set_pmode` is also reached for a throw, TH, which plays the A1 record in d2d; the game plays the TH record with start 0); Jab and Fend (the `start_sequence` path, a length from the sequence) are unchanged. Kicks (KK) and the S modes get the strike rule only.

Two commits: the oracle (`tools/emu/swing_start.py`, `tests/game/test_swing_start.cpp --swing-start`, README row; it prints what `set_pmode` does on the parent), then the claim (`rules/combat.hpp`, `rules.hpp`, `gamedata_load.cpp`, `fight.hpp`, `fight/setup.cpp`, `fight/strike.cpp`, the test's rule checks and 32 frozen cells, `docs/research/re/combat.md` "A player's attack swing").

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_00623b10` `0x623b10..0x623bfa` | the start frame: not a player 0; a player: the byte table at `0x623c10` (modes 7 to 16) sends A1 and A2 to the weapon path `0x623b88`, S3 and S4 to it only when `FUN_006235a0` (dual wield) is true (`0x623bbc`), every other mode to 0 (`0x623bfa`) |
| `FUN_00629fe0` and the map at `0x7446a0` | the item record's class key (+0xc0) to a row: bow 1, 1hs 2, 1ht 3, stf 4, 2hs 5, 2ht 6, xbw 7, ht1 12; no weapon 0 |
| table `0x6e8e60` (dumped from the booted image, `swing_start.py table`) | 7 dwords a row, d2s class order: row 0 `1 1 0 0 0 0 0`; rows 2 to 6 `2 2 0 0 0 0 0`; rows 1, 7 (bow, crossbow) and 12 (claws) all 0 |
| `FUN_005539b0` `0x5539b0..0x553afc` | `start = FUN_00623b10 << 8`, rate unit `+0x4c`, end `+0x48`; files an event for each flagged AnimData frame the walk passes (flags 1, 2, 4 take one branch of the table at `0x553afc`, 3 another) and the type-1 end event when the position reaches the end |

## Counted differentials (V run, the worker host, tree = upstream `f9b8cfed` plus the branch)

Oracle: `swing_start.py compare` runs `FUN_00623b10`, `FUN_00629fe0` and `FUN_005539b0` natively with the real AnimData flag bytes for each cell `test_swing_start --swing-start` prints: 7 classes x 9 main-hand weapons (none, sword, dagger, great sword, spear, staff, bow, crossbow, claws) x the modes d2d plays (A1, S1, S3, S4, KK) x IAS {0, 10, 20, 30, 40, 60, 80, 100, 150, 200} x weapon speed {-30 .. +30}: 12,460 cells that have an AnimData record and a flag. The rate fed to the game side is d2d's own `attack_speed` (the rate itself: `player_attack_rate.py`). The oracle also checks the closed form against the game's recorded events: 0 of 12,460 differ on the strike and on the end.

| 12,460 cells | differ before (parent) | differ after |
|---|---|---|
| Amazon / Sorceress A1 with a start frame (840): swing end | **815** (d2d 1 to 3 ticks longer) | **0** |
| the same 840: strike tick | **834** (+1 to +4) | **0** |
| Amazon / Sorceress, bow, crossbow, kicks (the strike rule only) | 144 strike | **0** |
| Barbarian, Necromancer, Paladin, Druid, Assassin (8,890): swing end | 0 | 0 |
| the same 8,890: strike tick | **1,123** (always +1 tick) | **0** |
| all | **2,102** cells (815 ends, 2,101 strikes) | **0** |

Per cell before (one line each): Sorceress, staff, IAS 0: swing 18 and strike 11, game.exe 16 and 9; barehanded 16 and 9 against 15 and 8; sword 20 and 12 against 18 and 10. Amazon sword 16 and 10 against 14 and 8; spear 18 and 11 against 16 and 9; bow 14 and 6 on both.

Build and tests (V run, the worker host, GCC 16.2.1, Release, `D2D_WERROR=OFF`): `presubmit.py --quick --base upstream/main` 0 FAIL (6 WARN, answered below), `pr_guard.py` 0 FAIL 0 WARN, ctest 296 of 300 pass, the 4 failures the same as on a pure upstream build with this data folder (`test_mpq`, `smoke_d2d`, `quests_act1`, `quests_act4`; data-folder failures, see docs/fork-rebase-2026-11-13.md section 3). Each commit builds alone (the oracle commit on the parent, the claim on top): Linux GCC only, no Windows, macOS or Clang build run (U).

Unit test (`test_swing_start`, no argument): the table and the clock without data (`attack_start_frame` for every class x weapon class, `frame_tick`), and with the data 32 cells whose answers the oracle printed (Sorceress and Amazon with each weapon class, and the Paladin, Barbarian, Necromancer, Druid and Assassin cells where the strike was a tick late); on the parent the same cells give the "d2d before" numbers in the comments of `kFrozen` (the parent has no `attack_start_frame`, so the test is built against the claim only; the oracle's comparison is the check on the parent).

Zeal side effect (V run, `zg_grid.py`, the przeal harness against game.exe's FUN_00553b10): parent 50 of 150 cells (200 of 600 gaps) one tick late, claim **0 of 150 (0 of 600)**: the residual `docs/upstream/drafts/zeal-gap.md` left is this rule.

## Collateral (what else changes)

- Same function, same clock: Amazon A1 skills with a melee weapon in hand (Power Strike, Charged Strike, Lightning Strike; Fend's sequence keeps its length); with a bow or crossbow only the strike rule applies (Magic Arrow, Fire, Cold, Ice, Exploding, Immolation, Freezing, Guided Arrow, Multiple Shot, Strafe). Sorceress: only the plain attack (her skills are SC).
- A hit-recovery or a cast does not change (SC, GH, BL take the unchanged branches). Whirlwind's gap reads `attack_ticks` of the Barbarian's A1: start 0, same value.
- Not changed, U or out of scope: TH and A2 modes (a throw plays the A1 record in d2d), the S3 and S4 start of a dual wielder (table entries are 0), what the client draws for the skipped start frames (d2d still draws the whole animation over the shorter swing; drill S1 of the trace), the idle tick between held swings.

## Effect on the bot's own scenario (internal, not for the PR body; V run, worker host)

Ten-minute throwaway-hero runs (`rb/botrun.sh` recipe of `docs/fork-rebase-2026-11-13.md`: bot mode, no network, own volume, realtime, `campaign:normal_act1`, one run a cell, two seeds) on the same bot and the same rebased lineage, parent `d2dbot:devp6` (d2d `dd70428d`) against `d2dbot:devsw` (`fork-only/sandbox-swing`, `20a10eda`). Another agent's containers ran on the host at the same time:

| seed | hero | image | clvl | xp | kills | mean kill time | arrivals, restarts |
|---|---|---|---|---|---|---|---|
| 7 | Sorceress | devp6 / devsw | 3 / 3 | 1,545 / 1,527 | 71 / 71 | 8.4 s / 7.0 s | 2, 0 / 2, 0 |
| 7 | Paladin (control: swing length unchanged, strike up to a tick earlier) | devp6 / devsw | 3 / 3 | 1,929 / 1,785 | 91 / 87 | 6.0 s / 6.1 s | 2, 0 / 2, 0 |
| 11 | Sorceress | devp6 / devsw | 3 / 3 | 2,013 / 1,740 | 91 / 80 | 6.5 s / 7.2 s | 5, 1 / 2, 0 |
| 11 | Paladin | devp6 / devsw | 3 / 3 | 1,824 / 1,956 | 79 / 86 | 6.4 s / 6.6 s | 3, 0 / 5, 1 |

Reading: the differences go both ways (Sorceress seed 7 equal, seed 11 minus 12 percent; Paladin minus 4 and plus 9 percent) and the Paladin control, whose swing length is unchanged, moves as much as the Sorceress, so run-to-run noise (monster placement, a restart, arrivals) is larger than any effect at this sample: **no kill-rate change is measurable**. The Sorceress hero is in its first levels and spends part of the ten minutes on spells, so the melee share is small (U); the expected melee effect is the swing 16 against 18 ticks (about 12 percent of melee damage per second). No crash, bounce or unreachable target on either image (the `restarts 1` rows are the bot's own restart, one on each image).

## Answers to the presubmit WARNs

- Unit kinds (player, not merc, pet, monster, co-op partner): the rule is the player's `Fight::set_pmode`; a co-op partner is another `actor` of the same code and gets it. Monsters, mercenaries and pets time their swings through `npc_timing` (`ai.cpp`, `fight/merc.cpp`, `fight/pets.cpp`); FUN_00623b10 returns 0 for a unit that is not a player, so their start frame is 0 on both sides and their own strike rule is a separate question (not checked here).
- Hit paths (melee, skill, not missile): an arrow or bolt skill is an A1 swing of the same class: the strike rule applies to it (three strike sites read `action_at()`), the start frame is 0 for a bow or crossbow; the missile's own flight is not touched.
- error-path: `swing_start.py` checks the AnimData blob's length before each bucket and record (`SystemExit`).

## How to re-check

```sh
python3 tools/pr_guard.py --base upstream/main HEAD
cmake --build build --target test_swing_start
cd tools/emu && uv sync
set -o pipefail
../../build/tests/test_swing_start --swing-start | grep '^D ' > /tmp/swing.tsv
uv run python swing_start.py compare /tmp/swing.tsv      # last line: 12460 cells  same (parent: 2102 differ)
uv run python swing_start.py table
../../build/tests/test_swing_start
```

## Before posting

1. Independent verifier (d2-reviewer, Opus, singly, own harness).
2. CodeRabbit limits, at most 3 open PRs, the user's go for any upstream post.

Agent: Claude Code
