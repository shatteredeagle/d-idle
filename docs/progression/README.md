# docs/progression: quests, acts, bosses, mercenary, difficulty (agent PROGDESIGN, docs only)

Closes findings F2 (progression past Act 1 has no owner) and design-gaps D12 (task layer). Nothing here is live-verified by this package; it is
written so code can be dispatched when wave 3 lands, and so the first live session has exact checklists.

Tags: **V** verified (live per repo docs, or a fetched primary source), **S** secondary or snippet, **U** unverified (recollection of Kolbot/SoloPlay,
D2 game knowledge, or a guess). Anything without a tag is a design decision, not a fact. Nothing in a code path may treat U as V.

## Files

| file | content |
|---|---|
| [quests.md](quests.md) | Completion detection per act and difficulty: save bits, memory candidates, UI evidence, proxies; quest index table; what changes per difficulty |
| [acts.md](acts.md) | Act transition procedure per hop (NPC, dialog, portal), cross-act waypoint travel, per-act town facts |
| [bosses.md](bosses.md) | Arena procedure per boss: approach, preset anchors, standing points, hazards, kill detection, exit; the `BossScript` shape |
| [merc.md](merc.md) | Mercenary hire / revive / equip flow with UI states, detection, policy |
| [difficulty.md](difficulty.md) | Difficulty selection screen, unlock rules, gate, resist penalty, the end-of-difficulty hop |
| [tasklayer.md](tasklayer.md) | The step graph: `Step`, `Plan`, preconditions, resume, how `bot.Runner` stays coarse, examples, tests |
| [verification.md](verification.md) | Ranked live-verification list as ready-to-run checklists (instance, minutes, exact checks, expected values) |
| [packages.md](packages.md) | Proposed package split, owners, order, sizes, requests to other agents |

## What this builds on (not repeated here)

- `docs/archive/goals-catalog.md` (127 goal ids), `execution.md` section 3 (per-quest mechanics for all 5 acts: NPC, items, areas, rewards, pitfalls),
  `tracking.md` (state inventory, `Observation`, 24-item spike list), `planner.md` (dependency graph, campaigns). Read them for *what* a quest is;
  this folder decides *how the bot proves it, hops acts, fights the bosses, and structures the code*.
- `docs/archive/prior-art/02-runs.md` (Kolbot boss coordinates, seals, waves), `05-town.md` (NPC tables), `01-leveling.md` (difficulty rule, merc rules), `08-oog.md`.
- `docs/plan/data/{mercs,quest-points,runs,danger-table}.json`.
- `docs/cheats.md`: the save editor already writes `difficulty`, `act`, `waypoints`, `progression`; quest bytes are decoded at 0x14F but not edited (V).

## Findings that change earlier docs

1. **The save file is the only quest source we can read today** (V: layout offsets, cheats.md), and a game leave rewrites it. Detection therefore has three
   tiers (quests.md section 1): save bits after Save and Exit (exact, stale mid-game), memory bits (fast, offsets U), and proxies (item in pack, NPC text,
   UI flag 0x11 edge). The step layer must carry the tier of each decision (`Evidence`), as `Progress.Verified` in tracking.md.
2. **Most act-transition tests need no playthrough.** `d2cheat save ... act=A difficulty=D waypoints=MASK` already starts a throwaway character in any act (V
   for level/gold/waypoints, U for act/difficulty unlocking). One missing tool feature (quest-bit editing in `internal/cheat/d2s`, request R1 in packages.md)
   would let every quest handler and every dialog be tested from a prepared state in about 25 s per setup. This is the biggest time saver in the whole list.
3. **A5 Anya/Malah/Larzuk/Qual-Kehk, A2 Greiz/Meshif/Jerhyn, A3 Hratli/Alkor/Ormus, A4 Tyrael dialog rows are all unread** (U, tracking.md item 16). The dialog
   box layout is known (V town.md): a bottom-anchored list above the NPC, Cancel last, 15-16 px apart; entry *text* cannot be read from memory, so choosing a
   row needs either an index per NPC learned once (verification V-05) or an image anchor.
4. **Merc and difficulty code cannot be written without two reads:** a merc unit reader (owner field, U) and the menu-state detector for the difficulty
   dialog (OOG). Both are in the first six checklists.
5. **Boss coordinates from Kolbot are world coordinates of fixed-layout levels only** (Chaos Sanctuary, Durance 3, Throne, Worldstone Chamber, Arreat Summit
   layout is fixed; U for 1.14d). Random-layout levels (Andariel, Duriel's tomb, Countess, Summoner arms, Nihlathak) must be reached via preset anchors from
   `d2map`, because maps differ per game (V nav.md "seed independence"). bosses.md marks each boss accordingly.

## Order of work (summary; details in packages.md)

1. WP-SAVES `saves.Progress` (exists in the plan) + R1 quest editor => Normal Act 1 to Act 2 provable offline-ish (quests M2).
2. Dialog layer `town/dialog.Talk` + act hop steps => Acts 1-5 hops (V-03 to V-06).
3. Task layer (`internal/bot/steps`) with fakes; quest handlers become steps one at a time.
4. Boss scripts in value order: Andariel, Duriel, Mephisto, Diablo, Baal (critical path), then Countess, Pindleskin, Nihlathak, Shenk (farm).
5. Merc, difficulty hop (needs Baal path; can be tested with a cheated character first).
