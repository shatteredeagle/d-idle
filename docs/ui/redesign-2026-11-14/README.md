# Watch UI redesign, design phase (2026-11-14)

> status: design proposal for the user's review (TASK-416, branch `agent/uidesign`). Nothing is built; no file under `internal/watch/` changed. Read from `origin/main` at 883770a9 and from pictures of the fixture viewer; no live game, no sandbox. Tags in the detail pages: V verified (how is said there), S secondary, U unverified.

Clickable mockup (private link, fake data, placeholder shapes): https://claude.ai/artifact/6PFYh9zdx1A8LUA15FYuq8 . The same file is [mockups/watch-orders.html](mockups/watch-orders.html); pictures: [1440](mockups/shots/mockup-1440.png), [900](mockups/shots/mockup-900.png), [390](mockups/shots/mockup-390.png).

## One page

### The ten biggest findings

1. The same fact is on screen three or four times: life and mana in three tiles, the current order in three, order lines in two logs.
2. Nothing says what needs the player: a broken weapon, a risky area, two alerts and an open question are in four tiles with four looks.
3. Commands are scattered over six places and two pages, with five separate queues and no shared status.
4. The first screen spends its top on the onboarding card and three text tiles; the game picture is cut by the fold at 1440 and absent at 900 and 390.
5. The minimap tile has thirteen debug buttons and three lines of legend above the map.
6. There are two maps that do not know each other, and the one beside the picture cannot take a click.
7. State words disagree, because three tiles each compose their own sentence from different endpoints.
8. Empty states fill whole tiles ("Nothing queued...", "No rule learned yet", "xp/h measuring").
9. "Am I progressing" has no answer in one place; kills and gold per hour are nowhere.
10. Pause, revive mercenary and sell now do not exist as commands, although the bot has most of the machinery.

### The proposed default view (Standard, 1440 px)

```
 d.idle  * FARMING Cold Plains, for level 10, to unlock Blood Raven     life == 86  mana === 105   (2 need you)  Inventory  Skills  ...
+--------------------------------------+--------------------+--------------------+
| game picture                         | minimap  [Layers]  | NEEDS YOU          |
|                                      |                    |  Weapon broken   [Repair now]
|                                      +--------------------+  Go for Blood Raven? [Go now][Not yet]
|                                      | act map            +--------------------+
|                                      |  click a zone      | ORDERS             |
|                                      |  pins, route,      |  [Go to][Town][Repair][Pause][More]
+--------------------------------------+  level bands       |  > Farm Cold Plains  12 min left
| PROGRESS  lvl 7->10 62% 25 min       |                    |  1 Identify all      next
|  xp/h 12.4k ~  chapter 3 of 9  gold/h|                    |  ~ then the plan ...
+--------------------------------------+                    +--------------------+
| journey strip (as built)             |                    | HEALTH o o o o o   |
|                                      |                    | LOG                |
+--------------------------------------+--------------------+--------------------+
```

Three seconds: the sentence (what and why), Needs you, the picture, Orders, Health, Progress. About 16 numbers on the first screen instead of more than 60.

### The Orders model in six lines

1. A player command is `{intent, args}`: go, fight here, town, repair, farm gold, identify, stash, take, gold, equip, trash, gear set, journey go/skip/hold; later pause, sell now, revive mercenary.
2. One Orders panel shows every open command from all five queues with one state set: queued, waiting, active, done, failed, cancelled; plus blocked rows ("Journey blocked: reason").
3. Row order is the bot's real priority ladder: safety banner, player commands, then the plan row. No drag to reorder.
4. **Primary surface: a hybrid (open question 1).** The Orders panel is primary; the act map is its place picker and draws the queue back as pins, route and failed marks. Departures from "map for everything", flagged for overruling: repair, errands, pause, mercenary and gear sets are not spatial and live in the Orders verb row (and on the town node as the same buttons); the local minimap is for watching and takes no commands.
5. Confirm only what cannot be undone or costs currency (trash, charter, revoke all, a too-hard zone inline); everything else gets a five-second Undo.
6. API: new read `GET /api/commands` (`watch.commands/1`, a join in the Hub, no bot change) and a write dispatcher `POST /api/commands` over the existing routes; nothing existing changes shape.

### Presets

Simple (picture + Needs you, Orders, Health, Progress, Journey), **Standard** (the proposed default, above), Stream (big picture, minimap, progress, orders, log; no buttons), Debug (everything that exists today, nothing hidden). Classic is today's layout and stays loadable. Old saved layouts are untouched: no id is removed, new panels are opt-in, and a preset hides old tiles with the editor's own `hidden` flag.

### Build slices

S1 foundations (opt-in panels, layout `status` key, presets, tokens) . S2 `/api/glance` and the status sentence . S3 Needs you, Health, Progress . S4 Orders read-only (`/api/commands`) . S5 Orders write path and the act map menu . S6 log, minimap declutter, phone . S7 rollout banner and polish. Bot cards in parallel: pause, sell now, revive mercenary, until-level, mercenary and supplies data, journey-blocked on the glance.

### What the user decides

Twelve questions in [H-open-questions.md](H-open-questions.md). The first three: the primary command surface (recommended: hybrid), whether the minimap ever takes commands (no), and which preset a new hero starts on (Simple).

## Details

| Page | Content |
|---|---|
| [A-audit.md](A-audit.md) | every tile with a verdict, the noise ranking, what is missing, the command table; pictures in [audit/shots/](audit/shots/) |
| [B-principles-ia.md](B-principles-ia.md) | eight rules, the three-second hierarchy, Needs you rows, Health lights, drill-down map, density tiers, units |
| [C-orders.md](C-orders.md) | the primary-surface decision with the four options, the two maps, intents with V/U backend support, states, rules, API draft, keys, phone |
| [D-layouts.md](D-layouts.md) | presets, positions, the migration table, rollout; drafts in [layouts/](layouts/) (`python3 -I docs/ui/redesign-2026-11-14/layouts/gen.py` rewrites and checks them) |
| [E-visual-system.md](E-visual-system.md) | tokens, components, accessibility, motion |
| [G-build-plan.md](G-build-plan.md) | slices with acceptance, tests, risks; review points; what not to build |
| [H-open-questions.md](H-open-questions.md) | twelve questions with a recommendation each |

## What was and was not checked

- V: panel manifests, layout files, `shell.json`, routes and order code were read; the fixture viewer was rendered at 1440, 900 and 390 px with headless Chrome on the worker host; the layout drafts pass the generator's checks; the mockup was rendered at the three widths in the dark theme and looked at once.
- U: nothing ran against a live bot; the mockup's light theme was seen once before a fix and not again; the drafts were not loaded in the real editor (they name panels that do not exist yet); whether the shell scheduler makes one or four `/api/hero` requests; every row marked U in C.3.
- The design adds no polling: `/api/glance` and `/api/commands` replace readers of `/api/uptime`, `/api/why`, `/api/tally`, `/api/alerts`, `/api/campaign` (chip), `/api/orders` and `/api/repair` in the new presets.
