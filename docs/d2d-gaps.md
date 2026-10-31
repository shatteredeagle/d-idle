# d2d fidelity gaps: cause analysis (agent D2DTOGGLE, 2026-10-30)

> status: partly superseded: casts per kill (docs/research/gap-casts-per-kill.md), XP per kill (docs/research/xp-award-trace.md) and spawn density (docs/research/spawn-density-trace.md) are explained since; the remaining real gap is spell damage truncation (docs/research/upstream-pr-candidates.md).

Scope: the three open gaps of docs/d2d-fidelity.md. Upstream source read-only at `~/d2d-lab/src` (SRC = file:line there; nothing copied, all described in own words).
Tags: V = verified (how), S = secondary (source or docs), U = unverified. "Original" = Game.exe, which we cannot run here, so no verdict below is confirmed on it.

| Gap | Cause class | Evidence | Fix difficulty | Our patch set should fix it? |
|---|---|---|---|---|
| (a) Tome of Town Portal 250 (d2d) vs 450 (live, clvl 3) | **Unresolved, most likely an upstream missing rule** (book price includes contents, which d2d ignores); not a data difference, probably not our measurement error | see (a) | Medium: needs the original's rule | No. Keep the data; give the bot a price knob (tome 450) instead |
| (b) Sorceress cast period 14 ticks for every spell vs table 13 (19 for Lightning/Chain Lightning) | **Upstream simplification** (one SC animation per class, no per-skill length) plus a possible +1 convention in the FCR formula | see (b) | Medium: a per-(class, skill) frame table plus a check of the +1 | Optional, later: only after the SME tables are confirmed on the original; do not guess |
| (c) Monster wake radius about 7 cells | **Not a gap in the code**: radius is data (`aidist`, default 35 subtiles = 7 cells) in a traced re-implementation of Game.exe's target search | see (c) | n/a | No (nothing to fix; confirm the 7 cells on the original) |

## (a) Tome of Town Portal price

- Data (V, `mpq-cat` on our MPQ copy, `data\global\excel\misc.txt`): tbk cost 250, ibk 200, tsc 100, isc 80; tbk/ibk stackable 1, maxstack 20, spawnstack 5. `npc.txt`: Akara buy mult 512, sell mult 1024 (same row as for the scrolls, which match live 100/80). So the 250 is exactly base cost x Akara's multiplier: **not a data difference** (the table our SME docs call "base cost 250" is the same one).
- Upstream rule (SRC `components/rules/rules.hpp:374-410`, `item_price`): base cost, plus quality extras, plus half of socketed items, times the NPC multiplier and quest multipliers; a quantity above 1 multiplies the price only for **non-stackable** items (line 407). Books are stackable, so their charge count never enters the price. There is no tome branch anywhere (grep `tbk`: only scroll-into-tome insertion, `rules.hpp:558,1818`, `world.cpp:1976,2382`, `fight.hpp:129`).
- Live number: 450 = 250 + 200 (V-live at clvl 3 per our items.md; one sample, character state at the time not recorded). 200 is not 5 x 100 or 5 x 80, so the surcharge is not simply the spawned scrolls at scroll price (U). A plausible reading is that Game.exe adds a per-charge term for books, which d2d does not model; the ID tome (200 on d2d) was never priced live, so the term is unconfirmed (U).
- Not tagged `ponytail:` upstream (no shortcut note for it): the omission is silent.
- Cheapest way to settle it: one live purchase of a Tome of Identify and a second tome of TP on a different character, read the gold delta.

## (b) Cast period

- Behaviour (SRC): every player skill that is not a swing animation goes through one mode, SC: `components/game/fight.cpp:2588` sets `kModeSC` for any skill whose `anim` is not `A1`/`TH`; the length is the class's SC animation frame count from the animation data, shortened by FCR at `fight.cpp:428-431` via `rules::speed_frames` (`components/rules/combat.hpp:193-200`). No per-skill frame override exists, so Fire Bolt and Lightning take the same time: this explains the measured 14 ticks for all 8 spells (V-d2d probe, docs/d2d-fidelity.md check 7a).
- The formula adds one frame before scaling (comment in `combat.hpp:194-197`: Sorc SC "base 14" sweeps 14/13/12 at FCR 0/9/20). The expected table (S, SME: 13 at 0 FCR) is one lower with the same breakpoints, so a +1 convention (frame count including a terminating frame) is likely the cause of 14 vs 13 for the ordinary spells (U); Lightning and Chain Lightning 19 would then need the per-skill table (U, never seen in d2d source: grep of `fight.cpp` finds no Lightning-specific cast length).
- Our measurement error is unlikely: all 8 spells gave an identical period in lockstep (ticks are exact there).
- Proposal: (1) read the original's Sorc cast frames at FCR 0 and 9 for Fire Bolt and Lightning (one session with a stopwatch of frames on the live game, or SME confirmation of the plrmode data); (2) if 13/19 hold, add a small `(class, skill) -> base frames` table in the fork's `set_pmode` and fix the +1. The effect on the bot: casting throughput of the Sorceress is 8 % too slow for bolts and 26 % too fast for Lightning, which matters for the dps/mana plans.

## (c) Monster wake radius

- SRC `components/game/ai.cpp:497-509`: the notice distance `best` is the monster's `aidist` from monstats (per difficulty), 35 subtiles when the column is 0. Candidates farther than 0x23 (35) are dropped in `components/rules/monsters.hpp:955-970` (`search_pick`). 35 subtiles = 7 cells: exactly our measured 7 to 8 cells, and the extra second of latency comes from the think cadence. This is a traced port of Game.exe's search function (the code comments name FUN_005dd0b0 / FUN_005dc380), so the radius is the original's by construction (S).
- Not a bug, not a `ponytail:` shortcut for this value. Open items that are separate (docs/d2d-fidelity.md section 3): d2d simulates monsters 30 cells out (more than Game.exe's active rooms) and a hit does not wake a sleeper outside its radius; both were read from source, neither checked on the original.
- Action: none for the fork; a live check that a stationary player is first noticed at about 7 cells would close it.
