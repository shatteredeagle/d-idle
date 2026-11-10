# Paper doll with real Diablo II art: research and plan

> status: research (no code). Question from the user: can the `/watch/inventory` paper doll show the game's own art? Answer: yes, technically easy (the repo already reads the MPQs; DC6 is a 100-line decoder), as a runtime-only feature that never puts art in git, images or releases. Redistribution is the real constraint, not effort. Target: classic 1.14d (not D2R). Tags: V = checked by me this run (command or URL named), S = secondary/snippet, U = unverified or from memory.

## 1. Where the art is (1.14d MPQs)

Probe: a throwaway Go test using `internal/data/mpq` on the install mounted in the sandbox container (read-only mount at `/game`), run on the worker host, deleted afterwards. Nothing was copied into the repo.

| Need | Files | Status |
|---|---|---|
| Inventory item icons | `data\global\items\<invfile>.dc6` (28 px wide per grid cell, e.g. `invgsw.dc6` is 28x28, 1 frame, 406 bytes). 591 files in `d2data.mpq`, 152 more in `d2exp.mpq` (listfile counts under that prefix). | V |
| Equipment panel background | `data\global\ui\PANEL\invchar.DC6` (classic) and `invchar6.dc6` (LoD, d2exp): both 8 frames, 256x256 + 64x256 + 256x176 + 64x176 twice (V: header dump). Decoded and viewed: frames 0-3 tile the character-stats sheet, frames 4-7 tile the inventory panel with the equipment slot frames and the 10x4 backpack grid (V: rendered to a PNG and looked at, then deleted; exact frame-to-quadrant mapping to be pinned in the build PR). A `data\local\ui\invchar.DC6` also exists (localised). | V |
| Slot silhouettes | `data\global\ui\PANEL\inv_armor.DC6` (54x82, 1 frame), `inv_weapons.DC6` (54x109), `inv_helm_glove.DC6` (54x53, 2 frames: helm, gloves), `inv_ring_amulet.DC6` (23x24, 2 frames), `inv_belt.DC6`, `inv_boots.DC6`. These are the faint item-shape ghosts shown in empty slots. | V (listing, 4 headers read; belt/boots headers not read) |
| Gem socket, durability icon, gold button | `gemsocket.DC6`, `duroicon.DC6`, `inv_goldbtn.DC6`, `invwarn.DC6` (in the panel dir) | V (names only) |
| Palettes | `data\global\palette\ACT1..ACT5\pal.dat` (768 bytes, 256 x BGR, no header; V: ACT1 read 768 bytes; BGR order is U beyond the viewed result, the icons showed correct colours with B,G,R order) plus `Pal.PL2` per act (colour-shift tables, used for `Transform`/`InvTrans` tints), `Units\pal.dat`, `STATIC\pal.dat`. | V (names, size), U (PL2 layout) |
| Archive priority | `Patch_D2.mpq` over `d2exp.mpq` over `d2data.mpq`: `internal/data` already opens them in this order (`openArchives`, `readFirst`). Patch_D2 has no listfile (0 names) but `ReadFile` by name works. `d2char.mpq` lists 10031 files, none in the item/panel dirs. | V |

D2R is different (sprite sheets, `.sprite`/CASC); none of this applies to it.

### DC6 format (decodes to RGBA)

Source: OpenDiablo2 `d2common/d2fileformats/d2dc6/dc6.go` (V, fetched from raw.githubusercontent.com: constants `endOfScanLine = 0x80`, `maxRunLength = 0x7f`, 4-byte termination, 3-byte terminator) and the header dump above (V).

- Header (24 bytes, little endian): version (6), flags, format, termination `0xEEEEEEEE`, directions, frames per direction; then `directions x frames` u32 offsets to frame headers. Item icons and panels have 1 direction.
- Frame header (32 bytes): flip, width, height, offset x, offset y, unknown, next block, length; then `length` bytes of scanline data, then 3 terminator bytes.
- Scanline stream: the image is stored bottom-up. Byte `0x80` ends the line; a byte with the high bit set is a skip of `b & 0x7f` transparent pixels; otherwise `b` literal palette indexes follow. Index 0 is transparent.
- To RGBA: look each index up in the 768-byte palette (stored B,G,R), alpha 255, index 0 alpha 0, encode with `image/png` (stdlib). I implemented exactly this in a 40-line probe and it rendered the gem, potion, charm and amulet icons and the panels correctly (V).
- Tiled panels (invchar) are drawn by the game as 2x2 frame grids; compose them into one canvas server-side (or serve four PNGs and let CSS tile them; one composed PNG is simpler).
- Colour shifts: unique/set/gem tints use `Transform`, `InvTrans` (armor, weapons, misc) and `transform`/`invtransform`/`transformcolor` (uniqueitems, setitems) columns (V: header names read) with the PL2 tables. The first version should skip tints (icons look slightly different from in-game for coloured items); doing them right needs PL2 parsing (U, not studied).

## 2. How an item maps to an icon

Columns in the 1.14d `.txt` tables, read from the real files (V, header and row dump):

| Table | Columns | Notes |
|---|---|---|
| `weapons.txt` | `invfile`(45), `uniqueinvfile`(46), `setinvfile`(47) | 308 rows |
| `armor.txt` | `invfile`(33), `uniqueinvfile`(34), `setinvfile`(35) | 204 rows |
| `misc.txt` | `invfile`(21), `uniqueinvfile`(22), no `setinvfile` | 153 rows; runes `r01` -> `invrEl`, gems `gpv` -> `invgsve`, potion `hp1` -> `invhp1`, charm `cm1` -> `invchm` |
| `uniqueitems.txt` | `invfile`(17) | 140 of 404 rows carry an override |
| `setitems.txt` | `invfile`(9) | none carry one in this table (0 refs found); set items use the base's `setinvfile` |

- Every `invfile`, `uniqueinvfile`, `setinvfile` and unique override named in those tables exists as `data\global\items\<name>.dc6` in the three archives: 526 + 286 + 152 + 140 references, 0 missing (V, existence check over all four tables). Names are case-insensitive in the MPQ (`invrEl`).
- Resolution order (U for the game's exact rule, but it is what the columns say): unique item -> `uniqueitems.invfile` if set, else base `uniqueinvfile`, else base `invfile`; set item -> `setitems.invfile` else base `setinvfile` else base `invfile`; everything else -> base `invfile`. Runewords, magic, rare, crafted use the base `invfile`.
- Variants: rings, amulets, charms and jewels show several looks (`invrin`, `invrin1`, `invrin2`, `invamu`, `invchm` exist; `invjwl` was not found in `d2data.mpq` alone, d2exp not checked, U). 1.14d has no `InvGfx1..6` columns (V: header search), so the variant comes from the item's own graphic-index bits (U: how the bot would read it is not studied). First version: always the plain `invfile`.
- Ethereal, sockets filled, durability: not part of the icon. Socketed gems draw `gemsocket` overlays in-game; skip for v1.

### What the bot already has (V, code read)

- `internal/data/mpq`: pure-Go MPQ reader (hash lookup, encryption, implode/zlib/bzip2), `Archive.ReadFile`, `Has`, `ListFile`. Standard library only. Reused as is.
- `internal/data`: `Extract`/`Open`/`LoadInstall` read tables from `Patch_D2` over `d2exp` over `d2data` into a cache under `<data>/cache/tables`. The `ItemBase` row loader (`internal/data/items.go`) reads `invwidth`/`invheight` but NOT `invfile`, `uniqueinvfile`, `setinvfile`; `UniqueItem`/`SetItem` do not carry `invfile` either. These columns must be added to the column lists and structs (additive).
- `DB.InventoryLayout(class)` already returns the `inventory.txt` panel rect and per-slot rects (`rArm torso lArm head neck rHand lHand belt feet gloves`, V in `internal/data/layout.go`): exactly the placement data for composing the real panel.
- `status.Item` (internal/status/inventory.go) has `Code` (in `cmd/d2bot/display.go` this is the numeric class id as a string, V), `Quality`, `Ethereal`, `W`, `H`, but NO unique/set index. The d2d source path reads it (`internal/d2dsrc/items.go` sets `FileIndex` from the `unique`/`set` field, V) and `internal/d2` has an UNVERIFIED `FileIndex` offset. To pick a unique's override the display layer must pass that index (a new `omitempty` field).
- No DC6, palette or PNG code anywhere in the repo (V: grep).
- The container (`docs/container.md`, V): the install is mounted read-only (`/game` in the sandbox container d2play-20261108, V by `docker inspect`: the lab install dir mounted `-> /game ro`; the wizard flow copies it to `/data`). The watch server and the bot run in the same container, so the MPQs are readable at runtime with no new mount. Data volume `/data` is writable (uid 1000) for the cache.

## 3. Design: runtime extraction, no art in git

```
user's install (MPQs, read-only)
        |  internal/art (new): open archives once, DC6 -> PNG, memory + disk cache
        v
<data>/cache/art/<key>.png      (data volume, never in the repo, never in an image layer)
        |
GET /watch/art/<key>.png  ->  web/inventory.js sets <img src> on a paper-doll slot / cell
GET /api/art                ->  {"schema":"watch.art/1","available":true|false,"panel":"doll"}
```

- Package `internal/art` (new, owns the decode; stdlib only): `Open(installDir, cacheDir) (*Store, error)`, `(*Store).PNG(key string) ([]byte, error)`. Keys are a closed grammar: `item-<invfile>` (`[a-z0-9_]{1,16}`), `panel-inv`, `slot-<name>`. The server maps a key to an MPQ name with a fixed prefix; no user-supplied path ever reaches the archive (no traversal). Unknown key: 404.
- Palette: ACT1 `pal.dat` for items and panels (U that the game picks the act palette per UI context; ACT1 rendered all samples I looked at correctly, V).
- Cache: first request decodes and writes `<data>/cache/art/<key>.png`; later requests serve the file; the cache dir carries the install's `Patch_D2` fingerprint (reuse `data.CacheCurrent`-style stamp) so a patched install invalidates it. Cold decode of an icon is a few milliseconds (the probe decoded dozens in one 40 ms test, V).
- Endpoint: served from the viewer origin like the other `/watch/*` assets. The viewer CSP already allows `img-src 'self' data:` (V, `internal/api/api.go`), so no CSP change; the gate pages use `img-src data:` (V, gate.go) and are not touched. Response headers: `Cache-Control: private, max-age=86400`, ETag from the file hash, `X-Content-Type-Options: nosniff`.
- Graceful fallback: `/api/art` says `available:false` when the install or an archive is missing (or decode fails); `inventory.js` then keeps today's drawn cells. A single missing icon (404) removes just that `<img>` and shows the existing text cell. Fallback is the default code path and stays tested; the art path is the addition (compatibility promise: additive).
- No art in git: add `/internal/watch/web/art/` style paths to `.gitignore` only if any dev dumps appear; the cache lives in the data volume. `tools/ci/check.sh` already forbids game files (THIRD_PARTY.md); extend its check to fail on any `*.dc6` or a PNG under `web/art`. Never put PNGs in `go:embed` (embed.go), so release artifacts cannot contain them; add a test that the embedded FS has no `art/` entries. The container build script already aborts on `.mpq` in the context (docs/d2d-container/licensing.md, V), art cache is created at runtime inside `/data`.
- Item art in the API: `status.Item` gains `art string` (`omitempty`), the resolved `invfile` name, computed server-side from the item's class id, quality and unique/set index. The page asks `/watch/art/item-<art>.png`. Old clients ignore the field. Additive per ui-design-system section 0 rule 5; the reply's `schema` stays (fields are only added) and the golden fixture gets the new field in a new fixture, not an edit.

## 4. Licence and redistribution, plainly

(U: general reasoning, NOT legal advice; I could not fetch the Blizzard EULA text, its page is script-rendered.)

- The icons and panel frames are Blizzard copyrighted artwork. The repo is public GPL-3.0-or-later; GPL covers our code, not Blizzard's art. Committing, embedding in the binary, putting in a Docker image layer, a release asset, a screenshot golden containing the art, or a mockup PNG made from it would be redistribution without a licence. Not allowed under the repo's own rule (THIRD_PARTY.md: "never commit or ship game files").
- Runtime extraction on the user's own machine from their own legally owned install, displayed only to them (and viewers they let in) is the same posture as the bot reading `d2data.mpq` tables today. Lower risk than redistribution, but not zero: the page shows the art to anyone with the watch URL (including a stream viewer), and a public Kick stream of the page would show Blizzard art, as streaming the game itself does. Say so in the docs; do not claim it is cleared.
- Friends' builds: they run their own copy of d.idle against their own install (the container mounts the MPQs the same way). No art is in the release, so each friend's page fetches art from their own MPQs. A friend without a copy of the game gets `available:false` and the drawn fallback (the setup wizard already requires the install for the bot to work at all, so this only affects viewer-only setups). Do not share the data volume, cache directory or screenshots of the art-rendered page in the repo, issues or PRs.
- The container image never contains art (cache is written at runtime); GPL-image questions in `docs/d2d-container/licensing.md` are unaffected.
- Screenshot goldens (ui-regression-plan) must be taken with art OFF (fixtures serve `available:false`), so no art enters git through tests. Mockups under `docs/mockups/ui/` stay art-free.
- Alternative with no Blizzard art at all: draw our own item silhouettes (SVG per base type) in the design-system style. Zero risk, zero fidelity. The runtime-extract path keeps this as the fallback anyway.

## 5. Effort and PR plan

Rough size (U, from reading the code): about 2 to 3 focused agent days for v1, mostly tests and the build-flexibility checks.

Principle (ui-design-system sections 0 and 6): additive; a new feature is a new panel or a new endpoint, not a rewrite. The inventory page already carries the paper doll as a component, so art is an optional layer on its slots, behind `available`.

1. PR A, `internal/art` core (no UI): DC6 decoder, palette load, PNG encode, key grammar, cache. Tests use a synthetic DC6 built in the test (made-up pixels, like `mpq/synth_test.go`'s `Build`), never real art; a real-install test skips when the install is absent like `mpq/real_test.go`. Golden: PNG bytes of the synthetic icon. Contract check: `docs/contracts/` entry for the endpoint is added in the same PR if the contracts README requires one for new HTTP surface (U, ask the coordinator).
2. PR B, data columns: `invfile`, `uniqueinvfile`, `setinvfile` (and unique/set `invfile`) in `internal/data` loaders, a `db.InvFile(class, quality, fileIndex)` resolver with the rule in section 2; table-driven tests with two classes of items (a weapon and an armor, plus a unique override and a misc rune/gem), fixture rows made up, plus the real-install test (skips). Cache stamp bump so existing table caches re-extract (check `Extract` stamp rules).
3. PR C, `status.Item.art` and display wiring: new `omitempty` field, unique/set index threaded from `d2.Item.FileIndex`; verify the FileIndex offset live before trusting it (UNVERIFIED today). New fixture `inventory-art` next to the existing golden; the old golden stays untouched.
4. PR D, endpoints: `GET /watch/art/<key>.png` and `GET /api/art` (`watch.art/1`), registered like the other watch assets, tests for 404 on bad key, traversal attempts, headers, `available:false` without install.
5. PR E, UI: `web/inventory.js` and `inventory.css` draw the panel frame and `<img>` per slot/cell when `available`; fallback path unchanged. Rules from section 9: no inline style (CSSOM whitelist only), no `!important`, selectors inside the layer, tokens only; `tools/uilint` must pass; golden DOM for `ready` (art off), `ready` (art on, fixture URLs), `empty`, `error`; the panel manifest/page `version` raised if the DOM changes (compat test loads old goldens). Screenshots with art off only.
6. PR F, guard rails: CI check that no `*.dc6` or art PNG is tracked, embed test, doc note in `docs/data.md` and THIRD_PARTY.md row; CHANGELOG line.

Order A -> B -> C -> D -> E; F can land with A. A to D are invisible to users; E turns it on.

## 6. Open questions (for the user or the coordinator)

1. Is showing art on a streamed page acceptable? (Posture, not a technical issue; section 4.) A per-page setting `art: off|on` default on for the owner and off for a public stream view is an easy safe default.
2. Which palette per context and whether to implement PL2 tints in v1 (section 1; U).
3. Variant graphic index for rings/amulets/charms/jewels (section 2; U): read from the item bits or accept the plain `invfile`.
4. `FileIndex` offset for unique/set must be verified live on a throwaway character before PR C ships (`internal/d2/items.go` marks it UNVERIFIED).
5. Frame-to-quadrant mapping of `invchar.DC6` (frames 4-7 hold the inventory panel, V by viewing; exact placement of each quadrant to be pinned in PR A).
6. Whether the EULA text should be read by the user (I could not fetch it); the doc only states the repo's own rule and general reasoning.
