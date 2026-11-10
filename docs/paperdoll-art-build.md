# Paper doll game art: what is built

> status: in progress (TASK-244). Plan and licence reasoning: [research/paperdoll-art.md](research/paperdoll-art.md). Tags: V verified (command named), S secondary, U unverified.

The watch inventory page can show Diablo II's own art, read at runtime from the user's own MPQs. No Blizzard art is ever in git, `go:embed`, testdata, golden files, docs screenshots, images or release artifacts; tests use synthetic DC6 files made from made-up pixels. v1 has no palette tints (PL2) and no art on the stream overlay (`/watch?mode=overlay`) until the user decides.

## PR A: `internal/art` (core, no UI, no HTTP)

- `art.Open(installDir, cacheDir)`, `(*Store).PNG(key)`, `art.ValidKey`, `art.DecodeDC6`, `art.ParsePalette`. Stdlib plus `internal/data/mpq`.
- Key grammar (closed): `item-<invfile lowercased, [a-z0-9_]{1,16}>`, `slot-<armor|weapon|helm|gloves|ring|amulet|belt|boots>`, `panel-inv`. Anything else is `ErrBadKey`; a key the install has no art for is `ErrNotFound`. No caller-supplied path reaches an archive.
- Archives: `patch_d2.mpq` (optional) over `d2exp.mpq` over `d2data.mpq`, names matched case-insensitively. Palette `data\global\palette\ACT1\pal.dat` for everything (U that the game picks another act palette in some contexts).
- Cache: memory plus `<cacheDir>/<stamp>/<key>.png`; the stamp hashes the three archive sizes and the Patch_D2 SHA-256, so a patched install starts a fresh directory. A cache that cannot be written only disables the disk cache.
- Item icons: frame 0 of `data\global\items\<invfile>.dc6`, 28 px per grid cell (V: invgsw, invrEl, invhp1, invchm, invamu, invrin are 28x28 on the real install).
- Panel (`panel-inv`), V by rendering the real files and viewing them: `invchar.dc6` and `invchar6.dc6` have 8 frames each; frames 0 to 3 are the character-stats sheet, frames 4 to 7 the inventory sheet. The four frames tile as 2x2 at (0,0) 256x256, (256,0) 64x256, (0,256) 256x176, (256,256) 64x176, giving a 320x432 image with the ten equipment frames, the 10x4 backpack grid and the belt bar. `invchar6` (LoD) adds weapon-swap tabs I/II; `invchar` is plain. We draw `invchar` and fall back to `invchar6` (U: which file the game itself draws).
- Slot silhouettes, V by viewing: `inv_helm_glove.dc6` frame 0 is the glove and frame 1 the helm; `inv_ring_amulet.dc6` frame 0 is the amulet and frame 1 the ring; `inv_armor`, `inv_weapons`, `inv_belt`, `inv_boots` are single frames (sizes 54x82, 54x109, 52x25, 54x52).
- DC6 decoding: bottom-up scanlines, `0x80` ends a line, high bit = skip, else literal run, index 0 transparent, palette stored B,G,R (V: icons render with correct colours).

## Tests

- `go test -count=1 -p 1 ./internal/art` (synthetic archive built with `mpq.Build`, no game files).
- Real install, read-only (skips without it): `D2ART_INSTALL=<install dir> go test -count=1 -v -run TestRealArt ./internal/art` (V on the worker host, 2026-11-10).

## PR B: icon names from the tables (`internal/data`)

- `ItemBase` gains `InvFile`, `UniqueInvFile`, `SetInvFile`; `UniqueItem` and `SetItem` gain `InvFile` and `Row`. `schemaVersion` 3 to 4 so existing table caches re-extract (an old cache has no icon columns).
- `(*DB).InvFile(class, quality, fileIndex) (name, ok)` with `InvPlain|InvUnique|InvSet`; result is lower case without extension. Order: unique -> `uniqueitems.invfile`, else base `uniqueinvfile`, else base `invfile`; set -> `setitems.invfile`, else base `setinvfile`, else base `invfile`; everything else -> base `invfile` (U: the game's exact rule; it is what the columns say).
- Guard: an override row whose `code` is not the item's own base code is ignored, so a wrong id can only show the base icon, never another item's.
- `Row` is the data row in the txt, the "Expansion" separator counted (the table has 404 data rows, 401 loaded). U: that the unique id an item carries is this row index. Evidence: V on the real install (2026-11-10, `TestRealInvFile`): The Gnasher is row 0 (base hax), The Stone of Jordan row 122 (rin), Harlequin Crest row 249 (uap), each on the right base; this is consistent with the id being the row index but does not prove it (no item data was read; reading the id from a live item stays on the live checklist below).
- V (same test): all 1977 (class x plain/unique/set) resolutions name a file that exists in the three archives (0 missing); 140 of 401 uniques carry an `invfile`; `setitems` carries none in this install (127 rows), as the research doc found. One placeholder unique row (Larzuk's Champion) has no `code`.
- Unknown variants (rings, amulets, charms, jewels show several looks in game): v1 always draws the plain `invfile`; the variant graphic index is not read (PL2 tints and variants are out of scope).

Tests: `go test -count=1 -p 1 ./internal/data` (two bases, unique override, set, rune, gem, "Expansion" row counting, schema-bumped extraction); real install `D2ART_INSTALL=<dir> go test -count=1 -v -run TestRealInvFile ./internal/data` (skips without).

## PR C: `status.Item.art` and the display wiring

- `status.Item` gains `art` (`omitempty`): the lower-case icon name, so the page asks `/watch/art/item-<art>.png`. Empty when the catalog cannot name one; the page then keeps the drawn cell. `invdesk.ItemView` embeds `status.Item`, so `/api/inventory` carries it with no other change; the reply `schema` stays (a field was added, none changed).
- `cmd/d2bot` `itemArt` resolves it from the item's class, quality and `FileIndex` through the optional `iconCatalog` side of the data catalog (`bot.NewDataCatalog` implements `InvFile`; test fakes do not, so their items carry no art and existing goldens do not change).
- Tests (two item families, a staff and a scepter, with unique and set fallbacks): `go test -count=1 -p 1 -run 'ItemArt|StatusItemArt' ./cmd/d2bot`.

### FileIndex: what is verified

- U: the memory offset `ItemData+0x28` (`internal/d2/items.go` `idFileIdx`) is the unique or set id of the item. It was never read from a live unique item (no game this run). The d2d source path (`internal/d2dsrc/items.go`) takes the id from the item JSON `unique` and `set` fields, which is the game's own id and the same number space.
- V (offline, real tables, `TestRealInvFile`): the id as a row index of `uniqueitems.txt` (separator row counted) lands on the right base item for the three names checked (The Gnasher 0 hax, The Stone of Jordan 122 rin, Harlequin Crest 249 uap). S: this matches what community save editors use. Not proven without an item read.
- Safety: a wrong id can only produce the base icon (the resolver ignores an override row on another base), never another item's art.
- Live checklist (needs a game, not run): on a throwaway character holding a unique and a set item (the item JSON path or the memory path), read `FileIndex` of the item, look up `uniqueitems.txt` row at that index, and confirm its `code` equals the item's base code and the name equals the item's name.

## PR D: endpoints

- `GET /api/art` -> `{"schema":"watch.art/1","available":bool,"panel":"doll"}` (`no-store`). `available` is true when the store can serve the panel frame (`panel-inv`); false without an install, with unreadable archives, or no source bound.
- `GET /watch/art/<key>.png` (also HEAD) -> `image/png`, `Cache-Control: private, max-age=86400`, `ETag` (hash of the bytes, `If-None-Match` answers 304), `X-Content-Type-Options: nosniff`. A key outside the grammar of PR A (`art.ValidKey`) is 404 before any archive lookup, so traversal strings never reach the store; a missing icon or any decode trouble is the same plain 404 with no reason text (the page then removes just that image and keeps its drawn cell). POST is 405.
- Mounted by `watch.Fun` (`Fun.Art`, set from `cmd/d2bot` for both backends with `<data>/cache/art` as the cache) on the viewer mux, so the viewer's token and passcode gate applies exactly as for `/api/inventory`. The inventory page CSP already allows `img-src 'self' data:` (V: `policyCSP`, `internal/watch/policysheet.go`); no CSP change, and the gate pages (`img-src data:`) are untouched.
- Nothing is embedded or stored: the PNG exists only in memory and in the data-volume cache.
- Tests: `go test -count=1 -p 1 -run Art ./internal/watch` (fake source, no game files): bad keys and traversal strings (source never consulted), headers, 304, HEAD, POST, available true/false for no source, failing source (no leaked path) and unbound, Fun mounts the routes only when `Art` is set.

## PR E: the UI layer (`/watch/inventory`)

- `web/inventory.js` probes once at page load: `GET /api/art` must say `available` and `/watch/art/panel-inv.png` must really load; only then art is on. Never with `?mode=overlay` (the stream view stays art-free until the user decides). Any failure is the drawn page, unchanged.
- Art on: the doll gets `data-art` and the frame `<img class="art-panel">` (the equipment half of the real inventory panel, cropped to 320x246), slots are placed where `inventory.txt` puts them (CSS `--x --y --w --h`, panel-relative pixels), each filled slot with an `art` name gets `<img class="ico" src="/watch/art/item-<art>.png">`, each empty slot the faint slot silhouette (`slot-<helm|amulet|armor|weapon|ring|belt|boots|gloves>`; the empty shield slot shows the weapon silhouette). Backpack, belt and stash items with `art` get the icon and hide their text label; the label stays in `title`. An icon that fails to load is removed and its slot or cell returns to the drawn look (`data-shown` is the flag). Names are checked against `^[a-z0-9_]{1,16}$` before a URL is built.
- Slot geometry, V against the real tables on the worker host (`TestRealPanelGeometry`, all seven classes share it): head 135,8 54x51; neck 209,35 23x24; torso 133,77 56x82; weapon 20,47 55x112; shield 251,47 55x112; gloves 21,181 54x53; belt 136,179 52x25; boots 252,182 54x52; the backpack grid starts at 19,255 with 29 px cells. `inventory.txt` names the ring boxes `rHand` (95,180 23x24, our `rring`) and `lHand` (209,180, our `lring`), not `rArm`; so `rArm` is the weapon hand and `lArm` the shield. V visually: the real frame renders with these boxes inside the slot frames (rendered once on the worker host with the real art for the check, nothing kept).
- CSS: component 13a in `web/components.css` (inside the existing layer, tokens only, no `!important`, no inline style; `tools/uilint` passes). JS writes only existing CSSOM-whitelisted properties.
- DOM goldens (`internal/watch/testdata/inventory_dom_*.txt`, fixture `inventory-art-view.json`; `UPDATE=1 node testdata/inventory_dom.js web/inventory.js` rewrites): `ready_off`, `empty` and `error` were generated from the page as it was before this PR and are byte-identical to the art-layer version (art off changes nothing); `ready_on`, `empty_on` and `image_fails` cover art on with fixture URLs only. No image is fetched in tests and no screenshot with art is kept anywhere.
- Not done: the I/II weapon-swap tabs, jewel/ring/charm variants, PL2 tints (v1 scope), a layout-editor entry for the art mode, a mockup update for component 13a (mockups stay art-free; the drawn doll is unchanged).

## PR F: guard rails

- `tools/ci/check.sh` fails on any tracked or untracked `*.dc6`, `*.dcc`, `*.dt1` file and on PNGs under `internal/watch/web/`, `*/web/art/` or `*/cache/art/`.
- `TestEmbeddedWebHasNoArt` (`internal/watch`): the embedded web tree has no art directory and no image or game file, so no release binary carries art. `TestPackageHoldsNoArt` (`internal/art`): no `go:embed` and no art-like file in the package.
- The container build already aborts on `.mpq` in its context (`docs/d2d-container/licensing.md`); the art cache is created at runtime inside the data volume, never in an image layer. THIRD_PARTY.md has the row; `docs/data.md` points here.
- Posture (not legal advice, from the research doc): runtime extraction from the owner's own install, shown to the owner and to viewers they let in, is the posture the bot already has for the tables; redistribution is what is forbidden. A public stream of `/watch/inventory` would show Blizzard art; the stream overlay (`?mode=overlay`) is art-free until the user decides. Do not share the data volume, the cache directory or screenshots of the art-rendered page.

## Verification summary (what ran, what did not)

- V (worker host, real 1.14d install, read-only, no game running): DC6 decode and PNG of item icons, slot silhouettes and the composed panel (`TestRealArt`); icon names exist for all 1977 class/quality resolutions (`TestRealInvFile`); equipment slot geometry shared by all seven classes (`TestRealPanelGeometry`); one visual check of the page with real art (rendered once and discarded).
- U, needs a live item read: that `ItemData+0x28` is the unique/set id and that the id is the `uniqueitems.txt` row (separator counted). Wrong ids fall back to the base icon. Checklist: on a throwaway character holding a unique and a set item, read `FileIndex`, take that row of `uniqueitems.txt`/`setitems.txt`, compare its `code` and name with the item.
- U: which of `invchar.dc6`/`invchar6.dc6` the game draws; ring box pixels beyond the table (rings are not in `inventory.txt`'s listed slots by name; V that the `rHand`/`lHand` boxes sit on the ring frames).
- Not built (out of v1 scope, safe defaults taken): PL2 colour tints, ring/amulet/charm/jewel variant graphics (the plain `invfile` is drawn), socket overlays, art in the stream overlay, a per-page `art: off|on` setting.
