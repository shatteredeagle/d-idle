# HIREREADER design: confirming "hire the merc" on d2d (read-only feasibility, 2026-11-02)

> status: design, current (card hirereader). Evidence and tags are in [hirereader-trace.md](hirereader-trace.md) (Q1..Q5, B1..B5, D1..D4); this file only decides what to build, in what order. Nothing here was built or run live. Tags as there: V run, V read, D, S, I, U.

## The criterion

Gate 2 asks the 8 h run to "hire the merc" (`openspec/specs/soak-gates/spec.md:9`; the policy is `openspec/specs/merc-policy/spec.md`; the brief's `gate2-analysis-20261102.md` is not in the tree, trace dead ends). On d2d the merc is the free Rogue (claim) or a paid hire. A confirmation the memory reader can give has three parts, each a pure function of dump and state reads the bot already makes:

| Part | Observable | Today | After the fixes |
|---|---|---|---|
| Hired | `present` false to true, with a purse drop (hire) or none plus burial state 4 to 5 (claim) | yes for the Rogue (V read: dump + `state gold=`) | yes for every class once B1 is fixed |
| Alive and following | `present`, `!dead`, `life[0] > 0`, merc gap to the player at most 24 (game.exe's own run threshold, V run in mercstats Q4) a few seconds after the player stopped, same area | Rogue only (B1) | every class |
| Revived | `dead` seen true, then `present && !dead`, purse drop equals the quoted price | **no**: B2 hides the dead merc once the corpse is gone, B4 leaves the quote unread | yes (bot-side only) |

## What is missing, and the cheapest sound way to get it

1. **Bot only, no d2d change, no game:** fix B2 (read `dead` when `present` is false; `internal/d2dsrc/player.go:381-389`, `internal/d2dmerc/d2dmerc.go:67-69`), B1 (map `GU`, `IW`, `0A`; `internal/d2dsrc/tables.go:92`), B4 (parse the price in the "Resurrect" line before clicking; `internal/d2dmerc/act.go:64`), B3 (a `Hirer` over `menu` rows for level, life, defence, cost). These make "hired", "alive and following" and "revived" confirmable for any merc. Sound because every input is a value d2d already prints and the money rules they rest on are proven (purse 200,000 and offers 108,000 and revive price 256 cases, 0 differ).
2. **Smallest d2d change (private fork, lab binary; proposal, not built):** extend the `merc` object of `dump` (dump source `apps/d2d/devctl_dump.hpp:210-214`, one `std::format` call in two branches) with `type` (= `header.merc_type`, the hireling Id), `name` (`merc_name`), `exp` (`merc_exp`) and `level` (`merc_stats(scene->rules, type, exp).level`, the function `apps/d2d/town/draw.cpp:73-74` already calls). `header`, `scene` and `town` are in scope of the dump lambda (V read, `devctl_dump.hpp:48-60`). Put the four keys in the `present:false` branch too (the revive price of a gone merc needs `level`). Size: about 45 bytes of JSON per dump, about 3 lines of source; no new verb. What it buys: exact level for the revive pre-check (B5 gone), class and aura from the Id by the hireling table (D, the aura is a function of the Id only, trace Q2), hire-versus-claim and replace detection by a changed `type`/`name`/`exp`. `level` is d2d's `merc_stats`, which equals game.exe's level-up in 2,940 cases (mercstats Q1, V run); the dump key is d2d state, not a game.exe read.
3. **Optional, only for the Defiance policy:** a `hire` section in `dump` listing the open window's offers (`Town::hire_offers`, `apps/d2d/town.hpp:392`): per row `id` (hireling Id), `name`, `level`, `life`, `def`, `cost`; `[]` when no window is open; about 80 bytes a row, ten rows. Bot-side inference without it is not sound: Ids 6, 7, 8 are identical in every number the row shows (D), and the hire list is rolled from the game seed (not readable). Without this the bot can still hire, but not choose the aura; it can read the aura of what it hired only after item 2.
4. **Game.exe memory reader (Game.exe backend):** out of scope; docs/merc-wire.md "Plan for the ORIGINAL game" stays as it is, U.

## How each piece is proven against game.exe

| Piece | Oracle | State |
|---|---|---|
| Offer rows and cost | `FUN_006637f0` (upstream `hire.py`, `mst_oracle.py hire`) vs `rules::merc_offer` | 108,000 offers, 0 differ (mercstats Q2) |
| Purse payment, hire and revive | `FUN_00576d90` (+ `FUN_006637b0`) vs `rules::hire`, `rules::resurrect_merc`: `tools/emu/hrd_pay.py`, `hrd_d2d_test.cpp`, `hrd_build.sh`, `hrd_run.sh` | 200,000 cases, 0 differ (trace Q5) |
| Revive price | `FUN_006637b0` | 256 of 256 |
| Level from experience, stats | `FUN_00572840` (`mst_oracle.py levelup`) vs `merc_stats` | 2,940 of 2,940 |
| Merc follows the owner | `FUN_005e52d0` (`mst_think.py`) vs `rules::hireable_think` | 100,000, 0 differ |
| The new dump keys | no game.exe oracle (they are d2d state). Proof is a live drill on the lab binary: recipe items 2 and 5 in the trace, expected values from the tables (D) | not run |
| Revive result (life, mode, cell) | D1: emulator run of `FUN_00579c00` with recorders; not built | I |

Only a rule that differs from game.exe could make a PR candidate; none does, so the d2d dump change stays a private-fork lab change (fork rules: committed and pushed the same day, never in this repo).

## Order of work (cheapest first)

1. B2, B1 (two small bot fixes plus fixtures from the drill recipe items 1 and 2): turns the revive and the Act 2 merc visible; no d2d change; one lab drill to record the fixtures. Opens the Gate 2 "hire the merc" check for the Rogue today and for every class after B1.
2. B4 (quote parse): exact revive pre-check; replaces the level estimate B5.
3. The dump keys (item 2 above): a three-line lab change; then `level` and aura become exact and B5 is retired.
4. B3 hire driver over `menu`, then the `hire` section if the Defiance choice matters for the run.
5. D1 and D3 differentials only if a live drill disagrees with the revive result or the replace rule.

Limits: no number in the design is a live observation of this card; the live items (drill recipe) are U until run on a throwaway character in the lab, and the Game.exe path is untouched.
