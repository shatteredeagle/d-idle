# d.idle branding

> status: DECIDED. The user chose the cursor concept on 2026-11-01 ("brand: cursor"). The final set is in `assets/branding/final/` (cards `brand`, `brandfinal`). Nothing is applied to any public surface yet: see the checklist. The ready-to-apply in-repo set is on branch `agent/brandapply` (card `brandapply`), not merged.

> status update 2026-11-11 (TASK-347): the app and repo icons, favicon, README banner and social card are now the ORIGINAL PIXEL-ART set below, made with Aseprite; the vector "cursor" set further down is kept as archive and is no longer used by the Electron app, the watch favicon or the README.

# Pixel-art brand set (current)

Mark: a small gold diamond (the d.idle header glyph) with a dark hourglass cut into it (idle = waiting), on a rounded stone plate. Wordmark: lowercase `d.idle` in a hand-drawn 5x7 pixel font (a 3x5 font for the tagline); both fonts are drawn as strings in the script, no licensed font. No game art or likeness, no third-party artwork.

Palette: 15 colours plus transparent, `assets/branding/palettes/didle.gpl` (GIMP format, loads in Aseprite), every colour taken from `internal/watch/web/tokens.css` (stone ramp for the plate, gold ramp for the gem). Every opaque pixel of every output uses only these colours (`pack.py --check` fails otherwise).

## Files

| Path | What |
|---|---|
| `assets/branding/src/didle.lua` | the whole set, drawn in code; run by Aseprite in batch mode |
| `assets/branding/src/*.aseprite` | editable sources written by the script (mark16/32/64/256, template16, banner, card, hero) |
| `assets/branding/out/png/` | icon-16/32/48/64 drawn natively; icon-128/256/512 nearest-neighbour upscales of the 64 master; `icon-256-native.png` drawn on the 256 grid; `template-16/22/32/44.png` (black on transparent, for macOS `Template` images); `banner-1280x320.png` (320x80 at 4x), `card-1200x630.png` (240x126 at 5x), `hero-1120x135.png` |
| `assets/branding/out/` | `favicon.ico` (16/32/48), `favicon-32.png`, `icon.ico` (16 to 256), `icon.icns` (PNG chunks 16 to 512, written by `tools/branding/pack.py`), the banner and the card |
| `assets/branding/out/electron/` | `icon.png` (512), `icon.ico`, `icon.icns`, `tray.png`, `trayTemplate.png` (22), `trayTemplate` at double size, the name carrying the at-sign 2x suffix (44); `npm run icons` in `apps/desktop-electron` copies them to `build/` |
| `assets/branding/out/d2d/` | for the d2d fork patch: `d2d.png` (256, launcher window icon), `d2d.bmp` (64x64, 32-bit BMP v4 with alpha, what `SDL_LoadBMP` embeds), `d2d.ico`, `d2d.icns`, `hero.png` (1120x135 launcher banner) |

The watch UI favicon is the 32 px PNG inlined as a data URI in `internal/watch/web/watch.html`. d2d fork mapping (V: read in the fork, not changed): copy `out/d2d/*` over the fork's `assets/icon/` of the same names (`d2d.rc`, `CMakeLists.txt` and `window.cpp` need no change; `d2d.bmp` is baked at configure, so re-run `cmake -S . -B build`); the Steam library art (`steam-*.png`) is not replaced here, and the fork's `assets/icon/README.md` credits Freepik for `d2d.svg`, which that patch should drop.

## Regenerate

```
labrun <worker host> -- tools/branding/build.sh      # Aseprite (Steam copy, command line only) + pack + check; then copy assets/branding/{out,src,palettes} back
node apps/desktop-electron/scripts/make-icons.js
```

`build.sh` runs `aseprite -b --script-param out=... --script-param src=... --script-param pal=... --script assets/branding/src/didle.lua` (the `--script-param` flags must precede `--script`), then `pack.py` (ico, icns, bmp, per-consumer folders; stdlib Python) and `pack.py --check` (sizes, 15-colour limit, no partial alpha, template images pure black, expected files present). Preview the small sizes: `python3 -I tools/branding/pack.py --preview assets/branding/out sheet.png`.

## Licence

The artwork in `assets/branding/` (script, sources, outputs, palette) is original work of the project, released under CC0 1.0 (public domain dedication) so anyone may reuse it; the generation scripts are GPL-3.0-or-later like the rest of the repo. Nothing in it derives from Aseprite's or any third party's artwork (Aseprite is used only as the drawing tool). [U: licence choice made by the branding agent; the owner confirms.]

---

# Earlier vector set (archive)

The name is always written with a lowercase d: **d.idle**, never "D.idle". The repo is `d2lod-bot`, the public repo `d-idle`.

## Decision

Cursor: a prompt chevron and a block cursor in a terminal ember palette. The bot sits at the prompt for days, waiting. The other two concepts (`rune`, `tick`) and the first-round cursor drafts stay in `assets/branding/concepts/` as archive; they are not used. Palette and contrast ratios: `assets/branding/final/` colours listed below and `assets/branding/concepts/cursor/palette.md`.

## Final asset set (`assets/branding/final/`)

SVG masters (all shapes, own pixel font, no installed fonts):

| File | Size | Use |
|---|---|---|
| `logo.svg` | 468x108 | chevron + cursor + wordmark, transparent; for dark and light surfaces (P `#e0630f`, Q `#a8700a`, both >= 3:1) |
| `logo-light.svg` | 468x108 | darker colours (`#b34700`, `#7d5200`, 4.9 and 6.1:1 on paper) for light surfaces and print |
| `mark.svg` | 132x84 | chevron + cursor only |
| `mark-mono.svg` | 132x84 | single colour (coal `#0e0c0a`) for stamps, favicons in one ink, embossing; recolour by editing the one fill |
| `wordmark.svg` | 108x28 | wordmark + block cursor, 4 px cells: tight spaces (badges, nav bars) |
| `icon.svg` | 512x512 | rounded tile (rx 104) for avatars and favicons; dark surfaces |
| `header.svg` | 1280x640 | GitHub social preview, README header |
| `header-discord.svg` | 960x540 | Discord banner |

PNG renders (all under 300 KB; the biggest is 6.3 KB): `icon-512/256/128/64/48/32/16.png` (hinted per size: whole-pixel cells, 1 px cells at 16 px), `favicon.ico` (16, 32, 48; PNG-compressed entries), `logo.png` and `logo-light.png` (936x216, 2x), `wordmark.png` (108x28, 1x, scale it by whole numbers with `image-rendering: pixelated`), `header-1280x640.png`, `header-discord-960x540.png`.

Text: `banner.txt` (half-block art, 52 columns, plus the tagline), `banner-ascii.txt` (plain ASCII `#` blocks, for terminals and logs without block characters), `banner-line.txt` (one-line variants: `> d.idle_` and `d.idle: unattended. offline. idle.`; both plain ASCII). `scenes.json` is the rect list every SVG and PNG is made from.

Colours: coal `#0e0c0a` (dark), paper `#f6f1e7` (light), ember `#ff8a1f` and amber `#ffc15a` (header and icon accents, dark only), ash `#ece2d4` (header wordmark, 15.2:1 on coal), muted `#b4a797` (tagline, 8.3:1), icon tile `#15100c` (ember 8.0:1, amber 11.7:1). The generator prints the full contrast table and fails below 3:1.

## Regeneration

```
python3 assets/branding/gen.py      # concepts + final/*.svg, scenes.json, banners (stdlib only)
python3 assets/branding/render.py   # final/*.png and favicon.ico, then decodes every PNG and checks pixel colours
python3 assets/branding/render.py --preview   # ASCII occupancy of the icon at 16, 32, 48, 64 px
```

`render.py` is a small stdlib rasterizer for the project's own subset (axis-aligned rects with opacity, scanlines, one rounded background; exact area coverage, 4x4 supersampling on the corners). It is not a general SVG renderer: new shapes go into `final_gen.py` as rects first.

## Usage rules

- Lowercase d, always: `d.idle`. The dot is part of the name and keeps its accent colour.
- Clear space: at least one wordmark pixel cell (the logo files include it) on every side; for the icon, nothing within the rounded corner.
- Minimum size: logo 120 px wide, wordmark 108 px, mark and icon 16 px (use the icon, not the header, below 64 px). At 16 and 32 px the chevron and the block stay separate shapes (checked by the ASCII preview). Do not scale pixel art to non-integer sizes; in browsers use `image-rendering: pixelated` for the PNGs.
- Do not: recolour outside the palette, stretch, add shadows or outlines, set the name in another font, put header or icon colours on a light background (use `logo-light`), or combine concepts.
- Everything is drawn as shapes by our own pixel font; no installed font, third-party image, Blizzard or Diablo II logo, font, art, sprite, icon or screenshot is used or may be mixed in.
- Licence: original work of the project; the licence follows the repository licence, the GNU General Public License v3 (see `LICENSE`).

## Application checklist (each needs the user's go)

Prepared in-repo on branch `agent/brandapply` (not merged; card `brandapply`; whole-repo vet and `test -short` green on the worker host): README header image, favicon files for d2dview plus the head links as a patch (card `mapitem` owns the two pages), and the banner printed at the head of `d2bot -h` (never on `--version` or the log: scripts parse both).

- [ ] Merge `brandfinal`, then `brandapply`: needs the user's go
- [ ] GitHub social preview (`header-1280x640.png`): needs the user's go
- [ ] GitHub avatar (`icon-512.png`): needs the user's go
- [ ] Discord server icon and banner (`icon-512.png`, `header-discord-960x540.png`): needs the user's go
- [ ] README header on main: needs the user's go
- [ ] Favicons on the status page and pages repo (private ops repo, not this one): needs the user's go
- [ ] The bot's startup banner on main: needs the user's go
