# D. Layout presets, migration and rollout

> status: design proposal (TASK-416, 2026-11-14). The drafts in [layouts/](layouts/) are `watch.layout/1` files written and checked by [layouts/gen.py](layouts/gen.py) (V: `python3 -I docs/ui/redesign-2026-11-14/layouts/gen.py` prints 0 findings: schema, spans inside 12 columns, no overlap among shown entries, unique order, known ids). They name six panels that do not exist yet, so the editor loads them only after slice S1 adds those ids to `reserved` in `web/shell.json` (U until then). Nothing in `internal/watch/web/` was changed.

## 1. The shell stays

Unchanged: `watch.layout/1`, 12 columns, 48 px rows, the four breakpoints (phone under 640, tablet under 1024, laptop under 1440, wide above), fit rows, the editor (`?edit=1`), `web/layout.json`, `web/layouts/{mockup,live,command}.json`, the alias, compat and reserved blocks of `web/shell.json`, every panel id and every pre-shell card id.

Three small additions are needed. Each is additive and ignored by an older shell:

| Addition | Why | Compatibility |
|---|---|---|
| Layout key `status: ["statusline", "hero-meters", "needs-count"]` (top level, optional) | The status band's panels come from `web/shell.json` today, so a preset cannot choose the new band. With the key, a layout does; without it, the `shell.json` list applies as now | `ParseLayout` ignores unknown keys (V, design system 15.3). `tools/uilint` is strict for repo files and learns the key in S1 |
| Panel manifest key `optIn: true` | The shell packs every panel a layout does not name after the placed ones. Without this key, a user with a saved layout would get six new tiles appended to it the day they ship. An `optIn` panel is mounted only when the layout in force names it | A new optional manifest key; `watch.panel/1` readers ignore it |
| `web/layouts/{simple,standard,stream,debug}.json` and `?layout=<name>` for them | New files next to the three that exist | `?layout=` already loads a preset by name (V, design system 15.5) |

The presets switch old tiles off with the per-entry `hidden: true` that the editor already writes for a tile the user turned off (V `web/layout.js`), so they need no new mechanism: every existing id is named in every preset, either placed or hidden.

## 2. The presets

All four share the status band `statusline | hero-meters | needs-count | Inventory | Skills | ...`.

### Standard (the proposed default), 1440 px and wider

```
+ status --------------------------------------------------------------------------------------------+
| d.idle  * FARMING Cold Plains, for level 10, to unlock Blood Raven   life ===- 36  mana ==== 105  (2) Inventory Skills ... |
+----------------------------------------------+----------------------+----------------------+
| video                                  6 col | map (minimap)  3 col | needs-you      3 col |
|                                              |  Layers  + -         |  ! Weapon broken     |
|                                              |                      |  ? Go for Blood Raven|
|                                              +----------------------+----------------------+
|                                              | zone-map             | orders               |
|                                              |  the act graph       |  Go to  Town  Repair |
|                                              |  pins 1 2, route,    |  Pause  More         |
|                                              |  level bands         |  > Farm until repair |
+----------------------------------------------+                      |  1 Identify all      |
| progress   lvl 7 ===--- 10  25 min  xp/h ~^  |                      |  2 Cold Plains 30 min|
|            chapter 3 of 9  gold/h            |                      |  plan: ...           |
+----------------------------------------------+                      +----------------------+
| journey (as built)                           |                      | health  o o o o o    |
+-----------------------+----------------------+                      +----------------------+
| cards                 | hero-vitals          |                      | log  Now Journal ... |
+-----------------------+----------------------+----------------------+----------------------+
```

Positions (col, row, colSpan, rowSpan; `onboarding` is 1, 1, 12, 3 and takes no cell once hidden, so everything rises three rows):

| Panel | wide and laptop | Why there |
|---|---|---|
| `video` | 1, 4, 6, 11 | the picture is first and never below the fold |
| `map` | 7, 4, 3, 6 | the minimap belongs to the picture |
| `needs-you` | 10, 4, 3, 3 | top right, where the eye lands after the sentence; hidden when empty, and Orders rises |
| `orders` | 10, 7, 3, 7 | next to the act map it takes places from |
| `zone-map` | 7, 10, 3, 12 | the graph is portrait (356 by 636 units, V `ordersmap` `box`), three columns fit it |
| `health` | 10, 14, 3, 2 | one row of lights |
| `progress` | 1, 15, 6, 3 | under the picture |
| `journey` | 1, 18, 6, 4 | as built |
| `log` | 10, 16, 3, 6 | last |
| `cards`, `hero-vitals` | 1, 22, 3, 3 and 4, 22, 3, 4 | second screen |

### Simple

`video` 8 columns, then one right column: `needs-you`, `orders`, `health`, `progress`; `journey` full width underneath. No minimap, no act map tile (it opens from "Go to"). Density `comfortable`. For a friend's first week and for the Electron viewer in a small window.

### Stream

`video` 9 columns by 16 rows; right column `map`, `progress`, `orders`, `log`. No `needs-you`, no onboarding, no buttons that matter to a viewer (the verb row hides when the page has no answer origin, as writes are refused anyway). Density `comfortable`. It is a layout for a second monitor or a capture; `?mode=overlay` stays the OBS strip.

### Debug

The new top (video, a 6 column `map` with all its controls, Orders, act map, log) and then every tile that exists today at its present size: `hero-vitals`, `why`, `activity`, `gear`, `tally`, `currency`, `now`, `orders-map`, `queue`, `thoughts`, `journal`, `chronicle`, `quests`, `cards`, `bestiary`, `atlas`, `locked`, `rules`, `diary`, `alerts`, `spend`. Nothing is hidden. This is where the map's timeline, replay and exports stay visible.

### Narrow widths

Tablet (640 to 1023, which covers the 900 px case): flow by `order`; half-width pairs share a row. Standard: video; needs-you; orders beside zone-map; health beside progress; map beside log; journey; cards beside hero-vitals. Phone: one column in the order video, needs-you, orders, health, progress, map, log, journey. The sticky band plus the picture plus the first Orders row fit an 844 px screen (band 56, picture 293 at 4:3, verb row 52, two order rows 88).

## 3. Migration table (old layouts keep loading)

Rule: **no id is removed and no existing alias changes its target.** A saved layout or one of the three old presets renders exactly as today, because every tile it names is still mounted when named.

| Old id | Kind today | In the new presets | In an old layout | Later, if the tile is ever retired |
|---|---|---|---|---|
| `hero-status`, `hero-detail`, `whyline`, `hero-now`, `voice` | alias (V `shell.json`) | resolve as today | unchanged | keep the alias |
| `hero-vitals` | panel | Standard (second screen), Debug; hidden in Simple and Stream | unchanged | stays (it is the Hero sheet) |
| `why` | panel | hidden; content is the `statusline` popover | unchanged | manifest `mergedInto: "statusline"` + alias `why -> statusline` |
| `activity` | panel | hidden; content is the first Orders row | unchanged | `mergedInto: "orders"` + alias |
| `gear` | panel | hidden; the alert is a `needs-you` row, the chips the Gear light | unchanged | `mergedInto: "health"` + alias |
| `tally`, `currency` | panel | hidden; a line and a popover of `progress` | unchanged | `mergedInto: "progress"` + alias |
| `journey-chip`, `alerts-count`, `hero-status` (status band) | panel | not in the new band (`status` key) | unchanged (old band) | stay as stubs |
| `now`, `queue` | compat card | hidden | unchanged | alias `-> orders`, removed from `compat`, the `data-lid` section stays in `watch.html` as an empty stub |
| `orders-map` | compat card | hidden (the same data draws `zone-map`) | unchanged | alias `-> zone-map` |
| `atlas` | compat card | Debug only | unchanged | alias `-> zone-map` |
| `thoughts`, `journal`, `chronicle`, `diary`, `alerts` | compat card | hidden; tabs of `log` | unchanged | alias `-> log` |
| `rules`, `spend`, `quests`, `bestiary`, `locked`, `cards` | compat card | Debug (and `cards` in Standard) | unchanged | stay |
| `risk` | reserved | - | skipped as today | stays reserved |
| new: `statusline`, `needs-count`, `needs-you`, `orders`, `health`, `progress`, `zone-map`, `log` | - | placed | not mounted (`optIn`) | - |

The "later" column is **not** part of this redesign's slices. It is written down so the retirements, if the user ever wants them, follow the promise: stub directory, `mergedInto`, alias, and a new compat fixture. Tests to add with the presets: `testdata/compat/layout-v1-{simple,standard,stream,debug}.json` (once shipped they are only added to), and one test that loads today's `layout-v1-{mockup,live,command}.json` plus `web/layout.json` under the new shell and asserts the same placed ids and cells as before.

## 4. Rollout

1. **Nothing changes by itself.** `web/layout.json` stays the default until the user says otherwise; a stored per-profile layout always wins, as today.
2. **A switch, not a migration.** The "..." menu of the band gets "Layout: Classic / Simple / Standard / Stream / Debug". Choosing one loads `web/layouts/<name>.json` for this browser (`localStorage`, like the theme), without touching the stored layout. "Edit layout" then edits a copy; Save stores it as the profile's layout as today; "Reset to default" returns to the chosen preset.
3. **"Try the new layout"**: a one-line banner on `/watch` for users on Classic, shown once per hero ("A calmer layout is ready. Try it / Not now"). "Try it" is the same as picking Standard; the menu takes them back.
4. **New heroes** (no stored layout, the onboarding card not yet dismissed) start on Simple or Standard: open question 3.
5. **Flip the default** (`web/layout.json` becomes Standard, the present file moves to `web/layouts/classic.json`) only after the user has lived on Standard and says so. A stored layout is not touched by the flip.
6. The Electron viewer wraps the same page and gets all of it with no change (S `apps/desktop-electron`, ADR 0020).
