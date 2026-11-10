# Item tooltip on the inventory page (TASK-269, itemtip)

Hover, tap or keyboard focus on any item cell (backpack, belt, stash) or filled paper-doll slot of `/watch/inventory` shows a D2-style tooltip: black translucent panel, centred text, the display font stack of the tokens (no Blizzard font file). A component of the existing page (additive; compat promise of docs/ui-design-system.md section 0). Tags: V verified by the offline test named, S secondary (docs), U unverified (needs the game or a browser).

## What the snapshot carried before (audit, V: read from the tree)
`status.Item` had id, code, name, quality, ethereal, sockets, filled, level, identified, dur/maxDur, props, art, grid cell. In practice only id, code, name, quality, ethereal, level, identified, art and the cell were FILLED (`cmd/d2bot/display.go` `invSource`); `Props` and `Sockets` were never set, durability only for worn items. The d2d snapshot (`d2dsrc.ItemExtra`) knew name (first tooltip line), quantity, durability, socket count, unique and set index, but no affix or stat list: the fork's `dump items` has none (docs/d2d-backend.md, d2d-spike.md section 7: "no structured affixes, tooltip text only"). The game.exe memory reader has `UnitStats(addr)` but `invSource` does not call it.

## What was missing and what was added (all optional, omitempty, `schema` 1 to 2)
| Field of `status.Item` | Source | Backend |
|---|---|---|
| `tip []string` | the stock d2d `items` verb prints the tooltip text the game builds ("Wand / One-Hand Damage: 2 to 4", S: d2d-spike.md); `d2dsrc` (`Options.Tooltips`, on in `d2denv.Build`) attaches it to `ItemExtra.Tooltip` by item id, cached and refetched only when the set of item ids changes or after 30 s | d2d |
| `sockets` | `ItemExtra.Sockets` (was never read into the page) | d2d |
| `base {name, minAC, maxAC, block, minDam.., twoMinDam.., reqStr, reqDex, reqLvl, speed, maxSockets, dur}` | `data.ItemBase` of the item class (the user's own tables at runtime) | both |
| `sell` | `loot.SellValue` (vendor price; a lower bound for magic, rare, set and unique items) | both |

`View.Schema` is 2 now; nothing was renamed or removed, `internal/status/testdata/item_tip.json` is the golden of the new item shape and a test checks an item without the fields marshals without them.

## What the tooltip shows
Name in the quality colour (white normal and superior, blue magic, yellow rare, gold unique and runeword, green set, orange crafted, grey low quality, and grey for a plain item that is ethereal or socketed, like the game); `Unidentified` in red; with `tip` the game's own lines in order (base lines white, property lines blue), else: base type, base defense or damage range, block, speed, durability, required strength, dexterity and level, `Ethereal`, `Socketed (filled/n)` and the props the snapshot carries; always item level and the sell value; badges SELL, KEEP (our marks) and PIN (worn slot pinned). Set bonuses: shown only if d2d prints them in the tooltip text (U).

## What is NOT done and why
- Magic, rare, set and unique PROPERTIES have no structured source on either backend. d2d gets them as the game's text (above); the game.exe backend shows the base facts and the line "Magic properties are not read on this backend." The server-side formatter from `itemstatcost`/`properties` and the string tables (and a small fallback formatter) needs a per-item stat list: for game.exe that is `Reader.UnitStats(addr)` in `invSource`, for d2d a stat list in the fork's `dump items` (a fork change, not made here). Not built, so no dead formatter code was added.
- The item's own rolled defense and damage are not known without the stat list; the tooltip says "Base defense: a to b" from the table when no game text exists.
- No Blizzard text is committed; every string except the labels of the fallback lines comes from the game at runtime.

## Behaviour (V: `go test ./internal/watch -run ItemTip`, node DOM goldens and pattern tests)
- One element (`#inv-tip`, `role=tooltip`), created once, refilled by `inventory-tip.js`; the page's poll hands the view over as a `d2:inventory` event and the tip re-renders only when the open item's text changed, closes when the item left. The item buttons get `aria-describedby` while it is open and lose their native `title` (no double tooltip).
- Desktop: pointer hover. Keyboard: focus opens it (only after a key press, so a mouse click does not leave it open), blur and Escape close. Phone: tap opens, a second tap or a tap elsewhere closes.
- Never moves the page: `position: fixed`, placed by a transform from the anchor rectangle (right of the item, else left, else below or above, clamped 8 px inside the viewport), no focus call, no scroll call; tests forbid `.focus(`, `scrollIntoView`, `scrollTo`, HTML sinks and timers in the file.
- CSP `style-src 'self'`: its CSS is a file in `@layer components` with new `--d-tip-*` tokens in `tokens.css` (additive; the tip is black in every theme), the position is a CSSOM `style.transform` write. `go run ./tools/uilint` is clean.

## Not verified
U (needs a browser, not run here): the look in a real browser at 390 and 1440 px, tap behaviour on a phone, the position near the screen edges with real tooltip sizes. U live: that d2d's `items` ids equal the `dump items` ids, that stash and cube items are listed by the `items` verb, the exact line order and colour codes of the d2d tooltip text, one more round trip per refresh on the bot's tick. Live checklist: open the page on a d2d hero, hover a magic and a unique item (game text appears), tap on a phone, tab through the backpack, mark an item for sale (badge SELL), equip and pin (badge PIN).
