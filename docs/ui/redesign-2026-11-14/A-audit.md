# A. Audit: what /watch shows and how the player commands the bot today

> status: design input (TASK-416, 2026-11-14). Read from the code on `origin/main` at 883770a9 and from pictures of the fixture viewer. Tags: V = read in the code or seen in the picture named, S = from a doc, U = not checked.

How the pictures were made (V): `labrun <worker> --pull docs/ui/redesign-2026-11-14/audit/shots -- sh docs/ui/redesign-2026-11-14/audit/shots.sh` runs the frozen fixture viewer of `tools/uishots` and headless Chrome on a private display on the worker host. No live game, no sandbox, no token. The fixture is a level 6 Sorceress with one order.

| Picture | Width | What it proves |
|---|---|---|
| [shots/watch-1440-fold.png](audit/shots/watch-1440-fold.png) | 1440 x 900 | the first screen |
| [shots/watch-1440-full.png](audit/shots/watch-1440-full.png) | 1440, whole page | the page is about 2.5 screens of content |
| [shots/watch-900-fold.png](audit/shots/watch-900-fold.png) | 900 x 900 | the game picture is below the first screen |
| [shots/watch-390-fold.png](audit/shots/watch-390-fold.png) | 390 x 844 | a phone's first screen has no game picture and no order |
| [shots/watch-390-full.png](audit/shots/watch-390-full.png) | 390, whole page | about six phone screens of stacked cards |

Not in the fixture (U how they look next to the rest): the `journey` panel (its route is off in the fixture viewer, so the panel hides), the onboarding card's text (the card renders as an empty titled box in the picture, V), and the nine unlock-gated cards.

## 1. The ten worst problems

1. **The same fact is shown three to four times.** Life and mana are in the top band (`hero-meters`), in `hero-vitals`, and as a percentage in `why`. The current order is in `activity` ("Farm Cold Plains for 30 min", 0%, "about 30 min left"), again in `now` (same title, same 0%, same minutes), and as a pin in `orders-map`. "Order given" and "Order started" are in `thoughts` and again in `journal`. (V, 1440 full picture.)
2. **Nothing says what needs the player.** A broken weapon (`gear`: "weapon 0/18 BROKEN"), a risky area (`hero-vitals`: "risky for Blood Moor"), two alerts (top band) and an open question (`thoughts`) are four different tiles in four different places, with four different looks. There is no single list of "things waiting for you".
3. **The control surface is scattered across six places and two pages.** Zone orders in `orders-map` plus `now` plus `queue`; repair in `gear`; farm gold in `gear`; journey intents in `journey`; item errands and gear sets on `/watch/inventory`; rules in `rules`; policy on `/watch/policy`. Each has its own queue, its own state words and its own cancel. See section 3.
4. **The first screen spends its best space on the least useful tiles.** At 1440 the top 180 px is the onboarding card (empty in the picture), then three text tiles; the game picture starts at 615 px and is cut by the fold. At 900 and 390 the game picture is not on the first screen at all. (V.)
5. **The map tile carries thirteen debug controls above the map.** `+`, `-`, Hide, Coverage, Heat, Events, Timeline, Export PNG, Export JSON, 30 s, 1 min, 5 min, 30 min, then three lines of legend text and "stall 100%". They take 190 px before the map starts and serve bug reports, not daily watching. (V.)
6. **Two maps that do not know each other.** `map` (the local minimap from `/api/senses`) and `orders-map` (the act graph from `/api/orders/map`) sit a screen apart. The one you can command from is the one further down; the one next to the game picture cannot take a click.
7. **State words disagree.** The top band says "WALKING TO COLD PLAINS", `why` says "Fighting in Blood Moor, my life is 72%" while the meter shows 36/120 (30%). Part of this is fixture data, but the design invites it: three tiles each compose their own sentence from different endpoints (`/api/uptime`, `/api/hero`, `/api/why`).
8. **Empty states take full tiles.** "Up next (0): Nothing queued. 'Go now' replaces the current order; a real queue comes in a later update", "No rule learned yet", "No open question. The hero carries on", "xp/h measuring". Four tiles whose content today is an apology.
9. **No progress answer.** "Am I getting anywhere" needs xp per hour, the level meter with its ETA, and the journey chapter. They are in `hero-vitals` (xp, "measuring"), `journey` (meter, ETA) and `journey-chip` (chapter). Kills and gold per hour are nowhere on the page.
10. **The legacy half of the page costs connections and has its own timers.** `app.js` opens a second `EventSource('/api/events')` next to the shell's, and runs four `setInterval`s (250 ms, 2 s, 5 s, 5 s); `update.js` adds one more; `/api/config` is fetched by five scripts. With the video stream that is three long-lived connections of the browser's six per host before any panel polls. (V: grep of `web/*.js`; U how often it actually stalls a page.)

## 2. Every tile

Columns: **n** = numbers and labels a reader has to parse in the fixture picture; **act** = how many of them the player would act on. Rate = the manifest `poll` in seconds (0 = on the SSE change event plus the 5 s safety poll).

### 2.1 Shell panels (`web/panels/<id>/`)

| Panel id (v) | Title | Band | Endpoint, rate | Shows | n | act | Verdict |
|---|---|---|---|---|---|---|---|
| `hero-status` (1) | Hero status | status | `/api/uptime`, 0 | state word, uptime, clock | 3 | 1 | **MERGE** into the new `statusline`; uptime and clock demote to its tooltip |
| `hero-meters` (1) | Life and mana | status | `/api/hero`, 2 | who and where, life and mana bars with `cur/max` | 6 | 2 | **KEEP**, thinner: bars with one number each, no labels row |
| `alerts-count` (1) | Alerts | status | `/api/alerts`, 5 | "alerts N" | 1 | 1 | **MERGE** into `needs-you` (count stays in the band as its badge) |
| `journey-chip` (1) | Journey | status | `/api/campaign`, 10 | "chapter: band, next: chapter" | 3 | 1 | **MERGE** into `progress` |
| `onboarding` (1) | Getting started | side | `/api/onboarding`, 3 | welcome card, first 30 minutes list | - | - | **KEEP as it is** (not redone). Only its place changes: it leaves row 1 once dismissed or done, and takes no cell when hidden (the shell already does that) |
| `hero-vitals` (3) | Hero | side | `/api/hero`, 2 | name, class, level, life, mana, xp cur/next/pct, xp/h, gold, stash, four resistances, readiness light and reason | 17 | 3 | **DEMOTE** to drill-down ("Hero sheet"). The three that matter (readiness light, gold when short, level) move up into `statusline`, `needs-you` and `progress` |
| `hero-detail` (3) | Hero stats | side | merged | already `mergedInto: hero-vitals` | - | - | stays a stub |
| `why` (2) | Why | side | `/api/hero`, 2 | Now, Would change it, Next | 3 lines | 1 | **MERGE** into `statusline` (Now is the sentence; the other two open on click) |
| `activity` (2) | Doing now | side | `/api/hero`, 2 | area, activity line, order card with meter, up next | 6 | 2 | **MERGE** into `orders` (the current order row) |
| `video` (3) | Game | stage | mjpeg relay | the game picture, Full screen, Collapse | 0 | - | **KEEP**, first on every width |
| `map` (7) | Map | side | `/api/senses`, 0.5 | minimap with trail, fog, units; 13 controls; legend; stall % | 20+ | 0 | **KEEP the map, DEMOTE the controls**: zoom stays, everything else goes behind one "Layers" button; exports and the timeline are Debug preset only |
| `gear` (4) | Gear condition | side | `/api/repair`, 5 | summary, Repair now or Farm until repair, chips per slot | 9 | 2 | **MERGE**: the alert ("1 broken, about 180 gold") goes to `needs-you` with its button; the chips become a drill-down of the health strip |
| `tally` (2) | Tally | side | `/api/tally`, 0 | banked, this trip, next purchase | 3 | 1 | **MERGE** into `progress` (one line) |
| `currency` (2) | Marks | side | `/api/currency`, 0 | currency chips | 2 | 0 | **MERGE** into `progress` drill-down; d.idle mode only, as today |
| `journey` (2) | Journey | side | `/api/campaign`, 5 | chapter path, Up next, level meter with ETA, chapter detail with intents | 12 | 3 | **KEEP** the strip as built. Its "Go now" also appears as an order in `orders` (C-orders.md), so there is one queue |

### 2.2 Pre-shell cards (`watch.html`, `data-lid`, drawn by `app.js`, `unlockui.js`, `alerts.js`)

| Card id | Title | Data | Shows | Verdict |
|---|---|---|---|---|
| `now` | Hero now | `/api/orders`, `/api/uptime` | the order, meter, kills and xp since the order, route crumbs, Cancel order, Come home | **MERGE** into `orders` (alias `now` -> `orders`) |
| `queue` | Up next | `/api/orders` | queued orders, Cancel each | **MERGE** into `orders` (alias) |
| `orders-map` | Where to? Tap an area | `/api/orders/map` | the act graph, six legend chips, List view | **KEEP as the place picker**, restyled as panel `zone-map`; the legend moves into a popover |
| `thoughts` | Thoughts | `/api/thoughts` | the open question with answer buttons, the feed | **SPLIT**: an open question is a `needs-you` row; the feed merges into `log` |
| `journal` | Journal | `/api/journal` | permanent lines | **MERGE** into `log` (tab "Journal") |
| `chronicle` | Chronicle | `/api/chronicle` | story lines, run timeline, recent reasons | **MERGE** into `log` (tab "Story") |
| `diary` | Diary | `/api/voice` | one page per milestone | **MERGE** into `log` (tab "Story") |
| `alerts` | Alerts | `/api/alerts` | alert list, sound and notification switches | **MERGE**: open alarms are `needs-you` rows; the list is a tab of `log`; the switches go to Settings |
| `rules` | Rules | `/api/rules` | learned rules, Revoke all | **DEMOTE** to the Policy page link in `orders` ("Rules: 3"); the card stays loadable |
| `spend` | Spend | `/api/tally` | what the tally buys | **DEMOTE** under `progress` |
| `quests` | Quests | `/api/questlog` | quest list | **DEMOTE**: the Journey strip already answers "what next"; card stays for Standard and Debug |
| `cards` | Find cards | `/api/reveal` | unidentified finds to flip | **KEEP** (it is the fun loop), Standard preset; a waiting card is also a `needs-you` row of kind info |
| `bestiary` | Cain's notes | `/api/bestiary` | monster notes | **DEMOTE** to Standard and Debug |
| `atlas` | Waypoint atlas | `/api/atlas` | waypoints found | **MERGE** into `zone-map` (a waypoint mark on the node is the same fact) |
| `locked` | Still locked | `/api/unlocks` | what opens later | **DEMOTE** to Standard and Debug |

No tile is DROPPED outright: the compatibility promise keeps every id loadable, and each merged id gets an alias (D-layouts.md). "Drop" in practice means "not in any new preset except Debug".

### 2.3 Noise sources, ranked by screen area wasted in the default view

1. The map tile's control block and legend (190 px by 345 px, used rarely).
2. Duplicate order display (`activity` and `now`: two tiles, about 430 px by 480 px each, one fact).
3. `hero-vitals` at full size (seventeen numbers; three are decision-relevant).
4. Empty-state tiles (`queue`, `rules`, `thoughts` with no question, onboarding when blank).
5. The six legend chips of `orders-map`, permanently visible.
6. Top band: six buttons of equal weight ("Theme: dark" has the same weight as "Inventory").
7. Text logs twice (`thoughts` feed and `journal`).

### 2.4 What is missing

| Question the player has | Today | Needed |
|---|---|---|
| What is it doing and why | three tiles, three sentences | one sentence: state, place, reason, with the Why code behind it |
| What needs me | four places | one list, sorted by severity, each row with its one action |
| Is it healthy | life and mana (band), gear (tile), resistances and readiness (Hero), merc (only a note in Journey), potions and scrolls (nowhere) | one strip of five lights: life, gear, readiness, mercenary, supplies |
| Am I progressing | xp numbers, "measuring" | level meter with ETA, xp per hour as a sparkline, chapter N of M, gold per hour |
| What did I tell it and what happened | `now`, `queue`, gear trip line, inventory queue, journal | one Orders list with state and reason per row |
| Is the journey stuck | not on the page; `goals.Status.JourneyBlocked` and Why code `journey_blocked` exist in the bot (V `internal/goals/runner.go`, `internal/bot/whyline.go`) | a `needs-you` row "Journey blocked: <reason>" with Go now / Skip / Hold |

## 3. Every way the player commands the bot today

| # | Command | Where | What happens | Feedback shown |
|---|---|---|---|---|
| 1 | Go to an area and farm N minutes or until cancelled | `orders-map` node, then the order sheet (duration chips, route, "Go now") | `POST /api/orders {kind:"go", area, minutes | until:"forever", replace:true}`; every open order is cancelled first; the Hub maps it to a `goals.FarmArea` goal (V `orders.go`, `http_answer.go`) | toast "Order given: ..."; `now` and `activity` show it; a pin on the node; journal line |
| 2 | Rest in town ("Come home") | `now` button, or the town node | `{kind:"home"}`; becomes a one-minute idle goal (V; U that it always walks home) | toast; `now` |
| 3 | Fight here | no button on the page | `{kind:"fight"}` exists in the API (V) | - |
| 4 | Cancel an order | `now` "Cancel order", `queue` "Cancel" | `POST /api/orders/cancel {id}` | toast "Cancelled: ..." |
| 5 | Pay a charter for a zone above the hero's range | `window.confirm` after the order is refused with `charter-needed` | resend with `approve:true`, tally is charged | browser confirm box |
| 6 | Repair now | `gear` tile, Inventory repair card | `POST /api/repair/request`; safe moment, loot, portal, smith, back (S `docs/repair.md`) | the gear tile's `trip.line` and phase; Why `repair_wait` |
| 7 | Farm until repair | same button when gold is short | request `kind: farm`; switches the gold-farm order on for the repair target | `farming for repair: 140 of 300`; Why `repair_farm` |
| 8 | Cancel repair | same button while open | `POST /api/repair/cancel` | tile returns to ready |
| 9 | Farm gold on or off | `gear` tile | `POST /api/farmgold {on}` | Why `farm_gold` |
| 10 | Journey: Go now, Skip, Hold or Resume, Ask first or Auto, go-at band, mode Guided, Free or Orders | `journey` panel chapter detail | `POST /api/campaign/intent` | the strip's states; Why line |
| 11 | Answer a question, confirm a risky step | `thoughts`, Return screen | `POST /api/answer`, `/api/confirm` | the card closes; a rule may be learned |
| 12 | Revoke a rule, revoke all | `rules` | `POST /api/revoke`, `/api/revoke-all` | list updates |
| 13 | Item actions: equip, unequip, trash, mark sell or keep, unpin | `/watch/inventory` item tooltip | `POST /api/inventory/{equip,unequip,trash,mark,unpin}` into the invdesk queue (at most 8) | the inventory page's queue list; Why `player_items` |
| 14 | Identify, buy scrolls, Cain | `/watch/inventory` | `POST /api/inventory/{identify,buyid,cain}` | same queue |
| 15 | Stash, take, deposit or withdraw gold | `/watch/inventory` | `POST /api/inventory/{stash,take,gold}` (errand: walks home at a safe moment) | same queue |
| 16 | Gear set save, apply, cancel, rename, delete | `/watch/inventory` | `POST /api/inventory/sets/*` | sets card |
| 17 | Policy rows and pacts | `/watch/policy`, `/watch/pacts` | `POST /api/policy`, `/api/pacts/*` | the sheet |
| 18 | Layout: edit, save, reset | top band "Edit layout" | `POST /api/layout` | the editor |
| 19 | Update: check, download, apply, roll back | banner (`update.js`) | `POST /api/update/*` | banner |
| 20 | Pause, resume, stop | **not on the page.** `bot.Runner.Pause()` exists (V `internal/bot/runner.go`); no watch route calls it (V: no match in `internal/watch`) | - | - |
| 21 | Revive the mercenary | **no command.** The merc policy revives in town when gold allows (S `docs/campaign.md` 6a) | - | a note in the Journey panel while he is dead |
| 22 | Sell now | **no command.** Marked items sell on the next town visit | - | `sellPending` count on the inventory page |
| 23 | Skill points | **no write path** (S `docs/skilltrees.md`) | - | - |
| 24 | Knobs | not on the page; config and CLI flags of `cmd/d2bot` (U: not audited) | - | - |

Five separate queues carry these: the watch orders (max 5 open), the invdesk queue (max 8), the repair request (one), the gold-farm switch (one), and the campaign intents. A player who asked for a repair, an identify and a zone change sees three unrelated status lines on two pages, and cannot tell which runs first. The bot does know: the priority ladder of `docs/inventory-actions.md` (safety, player commands, chores, roam). The page never shows that ladder.
