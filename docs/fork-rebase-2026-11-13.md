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

## 2. The new branch (S: git; pushed to the private forge the same day)

`didle/rebase-20261113`, head **`dd70428d9db993a53d5a4b8dcef6ebe719bc58a5`**, 27 commits on upstream main `73bf565b0e054b086ee864507682879e894b0b4a`, one commit per carried patch in this order: lab patch set (lockstep, dump, devctl defer/wait), lab test cheats 1 and 2, debug missiles (2), skill-drill debug verbs, belt fix, `--lab-verbs` gate, overlay (3), SPDX, dump merc, framefeed, cursor, icon (2), art (2), item verbs (6), README-DIDLE (the original plus one update commit naming this branch). Also on the forge: `wip/rebase-20261113` (5f3612c1, the same tree before the fixups were squashed). Nothing force-pushed; every old branch and tag untouched.

Conflicts that needed judgment (each noted in the commit message):
- Lab patch set: upstream now does the Ctrl sale itself (`ctrl_click.hpp`), so the lab ctrl-click sell is DROPPED; upstream's `click x y down|up` and the lab modifiers are merged (any order); `Channel::wait` follows upstream's unified Winsock/POSIX socket layer (WSAPoll on Windows, `kNone`); the pacer calls upstream's `pace_frame(ns, budget)`; the `skip_draw` seam sits next to upstream's LAN-host audio line.
- `--lab-verbs`: upstream's coop, coop-lobby and pad/touch/keyboard test launchers use setup debug verbs, so they pass `--lab-verbs`. The allow-list of read-only sub-verbs is unchanged, so a read-only sub-verb upstream added since 2026-11-01 (for example `collision`) stays gated until someone lists it (S; the bot's verb trace in section 5 uses none of them).
- Overlay: upstream's `Stack::find` takes `strict_installer`; the overlay reads with `false`.
- Cursor off also covers upstream's hardware-cursor path.
- CMake and test lists: union of upstream's and ours.

## 3. Verification (V, worker host)

Build: Release, `D2D_WERROR=OFF`, GCC 16.2.1, all 699 targets (d2d, every test, header checks) built clean (`~/d2d-lab/fork/build-rb.sh`, build dir `build-rb`). Linux GCC only; no Windows, macOS or Clang build was run (U).

ctest (`ctest-rb.sh`, `D2_MPQ_DIR` = the lab game folder, the patch installer set): **297 of 301 pass**, 14 oracle-mode tests skip as designed. The 4 failures are the same 4 on a pure upstream `73bf565b` build of the same targets with the same data (`ctest -R`, V): `test_mpq` (the lab folder's MPQs are not the intact reference set: `intact.files > 2000 && intact.damaged.empty()`), `smoke_d2d` ("no voice": sound reads fail on this data folder), `quests_act1` and `quests_act4` (the quake check times out). They are data-folder failures, not patch failures (the brief called them the video-MPQ ones).

Carried patch tests, all pass: `lab_verbs_d2d`, `test_item_verbs` + `item_verbs_d2d` (2 of 2), `test_overlay`, `overlay_d2d`, `test_framefeed`, `cursor_verb_d2d`, `test_devctl`, `test_click_keys`, `coop_d2d`, `coop_lobby_d2d`, `pad_use_d2d`. There is no dedicated lockstep ctest; lockstep is exercised by the image runs only in realtime mode here (U for `--lockstep` itself).

Dropped patches, upstream's version checked by behaviour (`~/d2d-lab/fork/vet-rb.sh`, recorded game.exe sides where they exist):
| Check | Result |
|---|---|
| Frozen Armor (author oracle, recorded game side) | every class 0 differ (champion, minion, plain, superunique, unique, immune, no resistance, resistant) |
| Freeze resume (author oracle, seeds 1 and 2) | frozen in game.exe: 0 of 12,548 and 0 of 12,516 differ, every aidel class 0 (same as forkrebase.md) |
| Next hit (`test_next_hit --next-hit 1 100000`) | 100,198 cases, 0 T and 0 H lines differ |
| Lightning #444 (`test_light_pierce --light-hit 1 400` vs `light_pierce.py`) | 1,192 T lines, 0 diff lines |
| Cold Enchanted #445 (`umod_stats.py 20000 1`) | uniques/champions: only the documented minion rows and a few multi-mod unique rows differ; the d2d output is byte-identical to the pure upstream build's (`cmp`), so the fork adds nothing there |
| Warp arrival #490 | `test_warp_arrival` passes |
| Zeal | `test_zeal_chain` passes |

## 4. Images and source (not swapped)

- **`d2dbot:devp5`** (worker host, image id `sha256:b124329f75ca...`): bot = origin/main of this repo at `19dad0c`, d2d = `dd70428d` built from source inside the image (context.sh, DEV=1, unstamped so the dev traces are on); `d2d --help` lists `--item-verbs`, `--lab-verbs`, `--lockstep`, `--overlay-dir`. Built next to the running sandbox, which was not touched. The context was assembled on the head box because the worker host has no Go toolchain any more (the iv1 script's `go` path is gone).
- GPL source tarball of the head: `~/d2d-lab/fork/rebase-20261113-out/d2d-didle-dd70428d9db9.tar.gz`, sha256 `f211cf8f2a5aa8fb5ea0367fae88bace3f3f051a77d1f44fa6b48f5c4a3bffe0` (`git archive --prefix=d2d-didle-dd70428d9db9/ dd70428d | gzip -n`; no MPQ or save inside).
- **Friends 0.2.0 candidate = the same commit `dd70428d`** (the same tarball and hash): every sandbox-only extra of the old lineages (warp, Zeal) is upstream now, so there is nothing left to leave out. Compared with the 0.1.x pin `acadbe47` it adds the overlay (off), dump merc (read-only), item verbs (off without `--item-verbs`) and all of upstream since 2026-11-01. A friends image is NOT built.
- Finding: `context.sh` with a non-pinned ref writes `SOURCE.txt` as "Fork: <FORK_URL>, tag didle-gate2-lab3 (<commit>)": the tag label is wrong and a local FORK_URL path ends up in the image. devp5 carries a head-box scratch path there. A friends build must use the published fork URL and fix the label (small container card).

## 5. Bot runs on throwaway heroes (V; `rb/botrun.sh`: bot mode, `--network none`, own volume, game data read-only, Paladin, realtime, `--item-verbs`, 10 minutes; containers and volumes removed after)

| Run | image (d2d) | seed | clvl | kills | xp | arrivals | bounces | restarts, crashes |
|---|---|---|---|---|---|---|---|---|
| new7 | devp5 (dd70428d) | 7 | 2 | 43 | 879 | 2 | 0 | 0, 0 |
| old7 | devp5 bot + iv1 d2d (a42dba25) | 7 | 3 | 91 | 1,919 | 3 | 0 | 0, 0 |
| new11 | devp5 | 11 | 2 | 57 | 1,218 | 2 | 0 | 0, 0 |
| old11 | devp5 bot + iv1 d2d | 11 | 3 | 84 | 1,890 | 2 | 0 | 0, 0 |
| devp5 vs iv1 image (seed 7, own bots) | | 7 | 2 vs 3 | 32 vs 81 | 681 vs 1,671 | 2 vs 2 | 0 vs 0 | 0, 0 |

The hero moved every 30 s in every run (no two status lines at the same spot), no crash, no restart, no bounce; arrivals are too few in 10 minutes to compare with the sandbox's bouncesPer100 of about 1.5. **But the new d2d halves the kill rate with the same bot** (2 seeds). The bot logs 83 `combat ignoring target ... why=unreachable` ("no path and no firing position with a clear line of sight") on new7 against 0 on old7, and 26 against 4 "clear killed nothing". The read verbs the bot's map uses (`debug grid`, `blocked`, `layout`, `level`, `warps`, `nearby`) answer byte-identically on both builds in town and in the Blood Moor (seed 7, `rb/verbs.sh`), so the map is not the cause; the monsters stand elsewhere (upstream's monster AI moved on since 2026-11-01). Not isolated further: a pure upstream d2d cannot drive the bot (no `dump`). Proposed card (bot, high): diagnose the "unreachable" targets on dd70428d before any swap.

(A first pair without `D2BOT_GAMEDIR` idled in town on both images: a harness mistake, discarded.)

## 6. Rollback and swap (NOT run)

Rollback is nothing to do: the old branches (`fork-only/sandbox-itemverbs` a42dba25, `fork-only/art-clean` acadbe47, all lab branches) and images (`d2dbot:iv1`, `devp4`, `didle:0.1.1`) are unchanged; the sandbox still runs `d2dbot:iv1`.

Swap the sandbox to devp5 (only on the user's word, and not before the kill-rate card is answered): on the worker host,
```
cd ~/play-build-20261108 && IMAGE=d2dbot:devp5 sh iv/swap-iv.sh
```
does NOT work as is (the script adds `iv/iv.yml` and the knob only for the image name `d2dbot:iv1`). Use, after the same tuning backup lines of swap-iv.sh:
```
cd ~/play-build-20261108
export D2_GAME_DIR=$HOME/d2d-lab/data D2_WATCH_ORIGIN=http://<worker host>:8891 D2_PUBLISH=0.0.0.0 D2D_CLASS=1 D2D_NAME=D2dbot D2BOT_GOALS=campaign:normal_act1
export D2_DEV_URL=... D2_DEV_PUBKEY=...   # as swap-iv.sh reads them from the running container
D2_IMAGE=d2dbot:devp5 docker compose -p play-build-20261108 -f compose.yml -f override.yml -f dev.yml -f feed.yml -f iv/iv.yml up -d --no-deps d2dbot
```
`iv/iv.yml` passes `--d2d-args --item-verbs`; the hero's `invdesk.verbs` knob is already 1 in its tuning file from the iv1 swap (S: swap-iv.sh).
Rollback after a swap: the same line with `D2_IMAGE=d2dbot:iv1`.

## 7. Where things are (worker host)

Fork worktree `~/d2d-lab/fork/wt-rebase1113` (on `didle/rebase-20261113`), upstream baseline worktree `wt-up1113`, build dirs `build-rb`, `build-up1113`, scripts `build-rb.sh`, `ctest-rb.sh`, `vet-rb.sh`, outputs `ctest-rb.out`, `rb-vet/`, image scripts and run logs `~/play-build-20261108/rb/`.
