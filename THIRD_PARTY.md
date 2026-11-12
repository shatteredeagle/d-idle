# Third-party code and provenance

Status: the repo is GPL-3.0-or-later (ADR 0010); the Tyniann licence text was read and verified on 2026-11-11 (below). Inventory of non-original assets in the friends pack and image: docs/friend-assets.md.

| Item | License | Use | Obligation |
|---|---|---|---|
| zlib `explode` (PKWARE implode) Go port, used for MPQ decompression | zlib license (blast.c, Mark Adler) | ported source: internal/data/mpq/explode.go | keep the original notice in the file and ship this entry |
| D2DX | GPL-3.0 | user-supplied dll, loaded by the game | never bundle or redistribute; the user installs it (docs/launch.md 6) |
| Kolbot (prior art) | no license file | ideas paraphrased only, no code copied | keep derived work private; see docs/prior-art/index.md |
| Tyniann `d2r-offline-farming-bot` (prior art), github.com/Tyniann/d2r-offline-farming-bot | "Source Available License", Copyright (c) 2026 Tyniann (V: LICENSE file read 2026-11-11 via the GitHub API; German text controls). Allows: view, fork, study, quote short excerpts with attribution. Forbids: use against Battle.net or any online service, sale or redistribution as a product, claiming Blizzard affiliation. All other rights reserved | ideas only (the idle-farming goal-manager shape); no code, text, assets or data copied | studying and learning from it is within the licence, and copyright does not cover ideas; keep it that way: copy no code or text. Our offline-only, GPL, non-affiliated posture is consistent with its restrictions. If a short excerpt is ever quoted, attribute it here |
| Diablo II 1.14d, MPQ data | Blizzard, user-owned | read at runtime from the user's install | never commit or ship game files, tables or saves (enforced by tools/ci/check.sh). The gear tables (item types, bases, skills.txt prerequisites, charstats, armor.txt) are extracted at setup from the user's own files by `cmd/d2tables` into a git-ignored cache (`~/.cache/d2lod/tables`); nothing extracted is committed and tests use made-up fixture rows |
| Diablo II 1.14d, item and panel art (DC6 sprites, ACT1 palette) | Blizzard, user-owned | decoded to PNG at request time from the user's own install by `internal/art`, cached under `<data>/cache/art` for the watch inventory page | never committed, embedded, put in an image layer, release asset, screenshot golden or docs; `tools/ci/check.sh` forbids `*.dc6` and art PNGs in the tree, `TestEmbeddedWebHasNoArt` and `TestPackageHoldsNoArt` guard the binary; tests use synthetic DC6 files. Showing it on a public stream is the owner's call (docs/paperdoll-art-build.md) |

VERIFIED 2026-11-11: the Tyniann licence text (see the row). Not a legal opinion; "ideas only" is the rule that keeps us outside it.
