# Fork rebase 2026-11-13: the d.idle d2d fork on current upstream main (TASK-395)

> status: current. Branch `didle/rebase-20261113` of the private fork replaces the 52362d52-based lineages (lab2/lab3, d2d.lab3-*, d2d.lab5-mercdump, fork-only/*) as the base for new images. Every old branch and tag is untouched. Patches never live in this repo (GPL, docs/decisions.md); nothing was posted to GitHub or upstream.

Tags: **V** run and seen on the worker host in this card; **S** read in source or git history; **U** not verified.

## Why

User go 2026-11-13 ("rebase the fork"). Every fork lineage sat on upstream `52362d52` (2026-11-01), 2129 commits behind upstream main `73bf565b` (2026-11-13). Upstream has since merged several of our fixes and restructured the client (touch and pad input, LAN games, a unified devctl socket layer, a hardware cursor), so the old branches could no longer be merged forward.

## 1. Inventory (S: `git log`, `git cherry`, file diffs in the fork checkout)

All lineages branch from `52362d52`. The shared stack is the lab patch set (`2ad4099d` .. `7ca4a22c`), the Frozen Armor and freeze-resume fixes (`ca3432ec` .. `ee5e06d8`) and the `--lab-verbs` gate (`f75816c6` = `d2d.lab3-labverbs`, tag `didle-gate2-lab3`). On top:

| Branch (tip) | Adds over the shared stack | Verdict |
|---|---|---|
| `d2d.lab3-framefeed` (80cc7adc) | `framefeed` verb (frames into a shared two-slot file) | CARRIED, shippable |
| `d2d.lab3-nocursor` (c418e235) | framefeed + `cursor on\|off` verb | CARRIED, shippable |
| `d2d.lab5-mercdump` (f64db72a) | `--overlay-dir` table overlay (5fe57b83, 372a0848, e66d0fdf), SPDX headers (ece607de), `dump merc` (f64db72a) | CARRIED; overlay default off, merc dump read-only: shippable |
| `didle/main` on the public GitHub fork (0c4f302e) | mercdump lineage minus `dump merc`, plus README-DIDLE.md | README CARRIED and updated |
| `fork-only/icon-original` (a392bcbd) | nocursor + original pixel-art window icon | CARRIED, shippable |
| `fork-only/art-clean` (acadbe47; friends 0.1.x pin) | icon-original + Steam art without third-party art | CARRIED, shippable |
| `fork-only/itemverbs` (664894bb) | art-clean + item verbs, gate `--item-verbs` (off by default) | CARRIED (same patches as below, byte-identical diff: `diff <(git diff f8ea366c a42dba25) <(git diff acadbe47 664894bb)` empty) |
| `fork-only/sandbox-warp` (f8ea366c; devp4) | nocursor + warp arrival on `Level::warp_spot` (27cb0200, f8ea366c) | DROPPED: upstream #490 (24e8ac4f) is the same rule plus the refusal when no spot exists |
| `fork-only/sandbox-itemverbs` (a42dba25; d2dbot:iv1, the running sandbox) | sandbox-warp + item verbs | item verbs CARRIED, warp DROPPED |
| `fork-only/sandbox-zeal` (90ac00ab) | sandbox-itemverbs + Zeal action-frame fix (44ed27f6) and a test tweak (90ac00ab) | DROPPED: upstream 276f85c0 (an upstream contributor) has component hunks identical to 44ed27f6 and the same test_zeal_chain |
| `didle/lab2-freezeresume` (ee5e06d8), `lab2-*`, `lab3`, `lab5`, older `lab-*` | history | not carried (superseded) |
| `fid/*` | upstream PR branches | #444 Lightning (missile-last-unit-r4), #445 Cold Enchanted (cold-enchanted-length-r5), #490 warp arrival (warp-arrival-r2) are ancestors of upstream main; older `-r*` revisions superseded. `fid/load-gold-clamp-r1` (5db6a582) is not upstream and was never in a fork lineage: left as is |

Dropped from the shared stack because upstream merged them as #159 (`36cbc5d3` Frozen Armor freeze rule, `46f77ca4` freeze-end think): `ca3432ec`, `8c70eb09`, `5ec181e1`, `ee5e06d8`. Upstream holds `tests/game/test_armor_hit.cpp`, `test_freeze_resume.cpp`, `tools/emu/armor_hit.py`, `freeze_resume.py`, `freeze_rules.py` (the python oracles byte-identical, the tests with small later upstream edits).

Private-only vs shippable: the lab and cheat verbs and lockstep stay in every build but are off unless the flag is given (`--lab-verbs`, `--lockstep`, `--time-scale`); the friends release never passes them. Shippable as default-off or harmless: framefeed, cursor, overlay (off), dump merc (read-only), icon and art, item verbs (off unless `--item-verbs`).

RESULTS_PLACEHOLDER
