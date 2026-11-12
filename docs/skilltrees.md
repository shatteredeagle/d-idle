# Skills page: the hero's skill trees on the watch screen (TASK-355, skilltrees)

> status: current 2026-11-11. Read only. Tags: V verified (say how), S secondary, U unverified.

Page `/watch/skills` (link "Skills" in the watch header and on the inventory page), data `GET /api/skills` (schema `watch.skills/1`). It draws the three skill trees of the hero's class in the game's grid (rows 1 to 6 by required level, columns 1 to 3), one square per skill, with prerequisite arrows, the points spent and a tooltip per skill. Nothing on it changes the character: the player (or the build plan) still allocates in the game.

## What it shows
- Per skill: icon, hard points, `+bonus` when items or shrines add levels, state (see below), arrows from each prerequisite.
- States: `learned` (hard points > 0, gold edge), `bonus` (only +skills, green edge), `learnable` (level and prerequisites met, grey), `locked` (character level or a prerequisite missing: dim, dashed edge).
- Per tree "N points spent"; page header: class, level, unspent points, total spent.
- Tooltip (hover, keyboard focus, tap; one element `#sk-tip`, `role=tooltip`, Escape closes; same look as the item tooltip): name, short and long description, passive or aura, required level, prerequisites (red when not learned), skill level (`base`, `+bonus`, maximum), mana cost now and at the next level, synergies, and the reason when the skill cannot be learned yet.

## Where the data comes from (all at runtime; no game text, art or numbers are committed)
| Fact | Source | Tag |
|---|---|---|
| Rows, columns, tab (page), required level, max level, prerequisites, passive/aura, synergy names | `skills.txt` + `skilldesc.txt` through `data.Skill` (`internal/data/skills.go`, user's own MPQs; cache schema bumped to 5 for the new `IconCel`, `str short`, `str long` columns) | V: read from the real 1.14d tables (a Paladin and a Sorceress rendered against a real install) |
| Names, descriptions | string tables by key (`data.DB.Text`) | V: rendered from the real install; a missing key leaves the text out, never the raw key |
| Icons | `internal/art` key `skill-<charclass>-<IconCel>` from `data\global\ui\SPELLS\<Xx>Skillicon.DC6` | V: `IconCel` is the DC6 frame number (steps of 2, the Paladin file has 60 frames); U: the even frame is the unpressed look (rendered, looked plausible) |
| Hard points | `d2.Skill.Level` (UnitAny skill list, +0x28) | V live (docs in `internal/d2/skills.go`) |
| Bonus points | `Player.Effective - Level` (`d2.SkillBonus`) | single-skill stat 107 V live; all-skills, class, tab and shrine sources U (`internal/d2/skilllevel.go`); the d2d backend has no effective stat vector, so there only what the snapshot carries shows; elemental +skills (stat 126) are not mapped (U) |
| Unspent points, level, class | `d2.Player.SkillPoints`, `Level`, `Class` -> `data.CharClass.Code` | V live for level and class; U that `SkillPoints` is the in-game unspent count on every backend |
| Mana cost | `data.Skill.ManaCost` (skills.txt `mana`, `lvlmana`, `manashift`, `minmana`) | V by test against the table formula; not compared with the game's tooltip live (U) |
| Damage, duration, radius | NOT shown | the `calc` and `skilldesc` formulas (`par8`, `skill('X'.blvl)`) are not evaluated here; the tooltip never guesses a number |

Layout (skilltrees2, V): row and column are `skilldesc.txt` SkillRow (1 to 6) and SkillColumn (1 to 3) as the table has them; read for the Paladin, Necromancer and Druid from the real tables, every skill has its own cell and the Paladin Combat tab matches the in-game screen (Sacrifice top left, Smite top right, Holy Bolt row 2 middle, Fist of the Heavens bottom). The server sends the geometry: each node has `x,y,w,h` and each tree `w,h` (300 by 504 grid units, `skilltree.TreeW/TreeH`) and `links[{from,to,x1,y1,x2,y2,have}]`, the prerequisite lines from the bottom edge midpoint of the required tile to the top edge midpoint of the dependent (side midpoints in the same row). The page draws one board per tree with `aspect-ratio: 300/504`, tiles placed in percent (CSS custom properties) and an SVG with the same viewBox stretched to the board: nothing is measured, so the first paint is final (the first version measured tiles in the browser and drew lines before layout settled: too large and off grid until a later redraw). V: a headless render at DOM load (virtual time 0.4 s) and at 6 s gives identical tile and line markup for Paladin, Sorceress, Necromancer and Druid (900 px), and the pictures at 1400, 900 and 390 px were looked at. Tests: `go test ./internal/skilltree -run Layout` (unique cells, tiles inside the board, every line end point is an edge midpoint of its two tiles).

Tab names: the tables do have strings `StrSklTabItem1` to `21` ("+%d to Combat Skills", "Offensive Auras", ... V: read at runtime from the string table) but they belong to the +skills item property, ordered per class group, not to the page number, and no table maps a class page to one (U); naming a tab from them would be a guess, so the page says Tree 1 to 3 in `SkillPage` order (U: the game's tab order may differ; the Paladin pages read Combat, Offensive Auras, Defensive Auras by their skills). Skills of another class that the hero uses through items (oskills) are not trees and are not listed.

## Model (`internal/skilltree`)
`Build(Tables, Hero, iconKey)` returns `View{trees[{tab, spent, nodes[]}], unspent, spent, allocate}`; a `Node` carries `id, name, row, col, icon, reqLevel, maxLevel, prereqs[{id,name,have}], base, bonus, level, state, canAdd, why, short, long, manaNow, manaNext, synergy`. It names no class, skill or tab: the class comes from the hero's `charclass` code, the shape from the tables. `CanAdd(skill, hero)` returns the verdict and the reason (`needs character level N`, `needs <skill>`, `no unspent skill points`, `already at the maximum level`). Rules (U live; game.exe's own check was not decompiled): unspent point, below `maxlvl`, character level >= `reqlevel`, each required skill has at least one HARD point (bonus points do not count, U), no stat requirements (all zero in the class rows of 1.14d, U for other tables). Tests: `go test ./internal/skilltree` (Paladin and Sorceress fake tables, every refusal reason), `go test ./internal/watch -run Skills` (goldens `testdata/skills_paladin.json`, `skills_sorceress.json`, routes, stubs; `UPDATE=1` rewrites the goldens).

## Performance
One cached body (2 s TTL, shared by all open pages, built from the newest snapshot, no game read), the page polls it every 5 s and rewrites the DOM only when the body changed. Icons are `<img loading=lazy>` from `/watch/art/` (decoded once, disk cached by `internal/art`); about 30 requests at most, so no page holds the browser's six connections for long. The art is not pre-extracted (30 icons decode on demand).

## UI compatibility
Additive: a new page, new routes, a new optional `Fun.Skills`. `watch.html` and the inventory page only gained a link. No layout panel was added, so `watch.layout/1`, `layout.json` and saved layouts are untouched (a Skills layout panel is a possible follow-up). Old clients never call the new routes.

## Future: allocation (NOT implemented, no write path is enabled)
Reserved shape: `POST /api/skills/allocate {skill, points}` on the answer listener (same gates as the other answer routes: origin list, token, JSON only, small body, unknown fields refused). Today it answers 501: code `off` while the flag `skills.allocate` (`SkillsAPI.AllocateEnabled`, default off) is off, `not_implemented` when it is on but no `SkillsAllocator` is bound. `View.allocate` stays false until an allocator is bound AND the flag is on; the tile component already supports `data-act` (a "+" on hover or focus, click posts) but never gets it. `canAdd` and `why` are already computed server side, so a button only needs the action.
What the bot side needs (none of it exists):
1. A player command queue like `internal/invdesk` (equip, identify): the request is stored, refused at once with a reason (`CanAdd`), executed at a safe moment (in town, no monsters near) by the runner, withdrawable.
2. A way to spend the point: UI clicks on the skill panel (open with the skill key, click the skill square, one click per point; the square positions are in the same icon grid, U) or a d2d devctl verb. The d2d memory path would be a write and falls under ADR 0014 (SETUP only on a throwaway character), so it is not a player-facing route. U for both.
3. Ownership: who chooses skills is a user decision not yet taken. Precedent: "the bot never makes equipment decisions" (`docs/gear-policy.md`). Today the build plan (internal/plandata) allocates on level-up; a player allocation would need a flag saying the player owns skill choices (the plan then stops allocating, or only fills what the player left), and the plan must re-read the hero's points after a player allocation. Proposed decision text for the coordinator: "Skill points: the build plan allocates unless the player turns on `skills.allocate`; then the plan only suggests."
4. Live checks before enabling: that one click adds one point on d2d and game.exe, that the prerequisite refusal in the game matches `CanAdd`.

## Not verified (needs a browser or the live game)
- Hover and tap behaviour of the tooltip at the screen edges (the page was rendered headless at 1400 and 390 px; hover was not exercised).
- That the tree order (`SkillPage` ascending) and the even icon frame match the game.
- Live hero data (hard points, unspent count) on the sandbox hero: not touched.
