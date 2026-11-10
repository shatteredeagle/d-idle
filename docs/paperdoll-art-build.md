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
