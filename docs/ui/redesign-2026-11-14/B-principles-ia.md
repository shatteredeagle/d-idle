# B. Principles and information architecture

> status: design proposal (TASK-416, 2026-11-14). It builds on `docs/ui-design-system.md` (the shell, tokens, panel contract and compatibility promise stay as they are); where this page is stricter, it is meant to be.

## 1. Eight rules

1. **One question per tile.** A tile answers one of: what is it doing, what needs me, what did I order, is it healthy, am I progressing, what happened. A tile that answers two is two tiles or has a drill-down. The title is the question's short form ("Orders", "Needs you", "Health", "Progress"), never a data source ("Tally", "Marks", "Why").
2. **One sentence on top, composed once.** The status band carries a single sentence built by the server: state word, place, reason. "FARMING Cold Plains, for level 10, to unlock Blood Raven." No tile below repeats it, and no script composes its own version.
3. **Every number earns its place.** A number is on the default view only if the player would act on it or it answers "progressing?". It has a unit, a fixed width and, where a trend matters, a sparkline instead of a second number. Test for each: "what would you do differently if it changed?" No answer, no place.
4. **A fact appears once per screen.** Life is in the band. The current order is in Orders. A second mention is a link, not a copy.
5. **Alerts are not status.** Status is always there and calm. An alert appears only when something waits for the player, says what, and carries the one button that answers it. When nothing waits, the alert tile takes no space.
6. **Two levels, then a page** (unchanged from the design system). Level 1 is the default view. Level 2 is a click on a tile: a popover, a sheet or a tab. Deeper is a page (Inventory, Skills, Policy).
7. **Colour means one thing.** Green is fine, amber is "look when you can", red is "act or lose something", blue is information and focus, gold is the brand and the thing you are pointing at. Item rarity colours stay inside item names only.
8. **Empty is small.** A tile with nothing to say shrinks to one muted line or hides. It never fills its cell with an explanation.

## 2. What the player sees in three seconds

Reading order on every width: **state and why, needs you, the picture, orders, health, progress.**

| # | Question | Tile | Content (and only this) | Source |
|---|---|---|---|---|
| 1 | What is it doing and why | `statusline` (status band) | dot + state word + place + reason; click opens "what would change it" and "what comes next" | `/api/glance` `sentence` (from `/api/hero` `state`, `area`, `why`) |
| 2 | Life and mana | `hero-meters` (status band, kept) | two thin bars, one number each | `/api/hero` |
| 3 | What needs me | `needs-you` | zero to four rows: severity, one line, one button. Hidden when empty; the band shows the count | `/api/glance` `needs` |
| 4 | The game | `video` | the picture | relay |
| 5 | What did I order | `orders` | verb row, safety banner when active, the rows, the plan row | `/api/commands` |
| 6 | Is it healthy | `health` | five lights with a word each: Life, Gear, Readiness, Mercenary, Supplies | `/api/glance` `health` |
| 7 | Am I progressing | `progress` | level meter with ETA, xp per hour with a 30 minute sparkline, chapter N of M with its band, gold per hour | `/api/glance` `progress` |

Second look (same screen at 1440, one scroll at 900, a tab on a phone): the minimap, the act map, the Journey strip, the log.

### What a `needs-you` row is

| Kind | Severity | Text example | Button | Source today |
|---|---|---|---|---|
| Death, alarm | red | "Died in Blood Moor to a Fallen Shaman" | See what happened | `/api/alerts` severity `alarm` |
| Gear broken | red | "Weapon broken. Repair is about 180 gold" | Repair now / Farm until repair | `/api/repair` `trip.button` |
| Open question | amber | "Go for Blood Raven at level 9?" | Go now / Not yet | `/api/thoughts`, `/api/campaign` |
| Journey blocked | amber | "Journey blocked: the Den needs 2 healing potions" | Go now / Hold | `goals.Status.JourneyBlocked` (V in the bot; U on the watch side) |
| Order failed | amber | "Repair failed: no way home. Retry in 14 min" | Retry | `/api/commands` |
| Risky area | amber | "Lightning resistance is under the line for Blood Moor" | Open Hero sheet | `/api/hero` `stats.readiness` |
| Mercenary dead | amber | "The Rogue is dead. Revive costs 600 gold (you have 46)" | Revive / Farm gold | U: merc state is not on a watch endpoint today |
| Finds waiting | blue | "3 finds to flip" | Open cards | `/api/reveal` |
| Update ready | blue | "Update 0.1.2 is ready" | See notes | `/api/update/status` |

At most four rows show; more collapse into "+2 more". Red sorts first. A row never repeats what Orders already shows as queued.

### Health lights

| Light | Green | Amber | Red | Source |
|---|---|---|---|---|
| Life | above 50% | 25 to 50% | under 25% | `/api/hero` (the thresholds `hero-meters` uses today, V) |
| Gear | nothing under the warn line | a slot amber | a slot broken | `/api/repair` |
| Readiness | `light: green` | `amber` | `red` | `/api/hero` `stats.readiness` |
| Mercenary | alive | none hired | dead | U |
| Supplies | potions and scrolls at the kit plan | below plan | none | U: belt and tome counts are in `/api/inventory`; the plan numbers are the bot's |

A click on a light opens its detail in a popover: the gear chips, the four resistances, the belt. That is where today's `gear` chips and `hero-vitals` resistances live in the default view.

## 3. What moves to drill-down

| Leaves the default view | Lives at |
|---|---|
| xp `cur / next`, gold on hero and in stash, resistances as numbers, name and class | Hero sheet (`hero-vitals`, opened from `statusline` or the Readiness light; a tile in Standard and Debug) |
| "Would change it" and "Next" of the Why tile | the `statusline` popover |
| Gear chips per slot | Gear light popover, and the Inventory repair card |
| Tally banked, this trip, marks | `progress` popover |
| Map: trail window, coverage, heat, events | "Layers" popover on the minimap |
| Map: timeline, replay, PNG and JSON export, stall percent | Debug preset only |
| Uptime, wall clock | `statusline` tooltip |
| Journal, Chronicle, Diary, alert history, the thoughts feed | `log`, tabs Now / Journal / Story / Alerts |
| Rules list | Policy page; Orders shows "Rules: N" as a link when N is above zero |
| Quests, Cain's notes, Still locked, Spend, Waypoint atlas | Standard (folded) and Debug; the atlas also as marks on the act map |
| Theme, Edit layout, Help | one "..." menu at the end of the band; Inventory and Skills stay as the two named links |

## 4. Density tiers

One setting, three values; it is the preset's `density` plus which tiles it names. The tokens already have `dense` and `comfortable` (V `tokens.css`).

| Tier | For | Numbers on the first screen (fixture state) |
|---|---|---|
| Simple | a friend's first week; a phone | about 8 |
| Standard | the daily view | about 16 (today's first screen at 1440: over 60, V from the audit counts) |
| Debug | a bug report | everything that exists today |

## 5. Units and words

- Life and mana: `36 / 120`. Percent only where a bar cannot show (text sentences).
- Time left: `28 min`, `1 h 10 min`; under a minute `<1 min`. ETA to a level: "level 10 in about 25 min". Clock times in the viewer's local zone, 12 or 24 hour as the browser says; never UTC.
- Rates: `xp/h` and `gold/h` as `12.4k`, three significant digits, with the sparkline. "measuring" becomes a dash with the tooltip "needs 2 minutes of play".
- Gold: `1,240`. No split digits.
- State words are the set `/api/hero` `stateLabel` already produces; the sentence uses them, tiles do not invent others.
- A button says what happens, in the words of the result: "Repair now", "Farm until repair", "Go now", "Add to queue". Never "OK" or "Confirm" alone.
