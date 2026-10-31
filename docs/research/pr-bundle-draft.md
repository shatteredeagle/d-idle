# Draft PR: Get-hit frames drawn once at the get-hit rate; drlg-dump opens the folder's patch

> status: draft, current (agent BUNDLEPR, 2026-10-31, card upstream-bundle). NOT posted. Branch `fix/bundle-get-hit-render-lut` in the upstream clone worktree `~/d2d-lab/pr/bundle` on the worker host: two commits on upstream/main afa0cdae (4224a03d renderer, e0731ebd Lut Gholein), author shatteredeagle, no trailer. The body below follows upstream's `.github/pull_request_template.md`; the maintainer asked for bundles (2026-10-31). Title: `Get-hit frames drawn once at the get-hit rate; drlg-dump opens the folder's patch`. Evidence: pr-bundle-evidence-pack.md.

---

## What changes

Two independent claims, one commit each (you asked for a bundle; they touch no common file):

1. **Get-hit frames** (4224a03d). With #136 a monster's or the merc's get-hit holds for game.exe's length, but d2d drew the GH animation at its base rate and looped it. game.exe's client steps the frame position by the get-hit rate only while position + rate stays under the end, and draws position >> 8: the animation plays once and stands on its last stepped frame. d2d now draws a monster's and the merc's GH that way (`rules::get_hit_frame`, `Unit::get_hit_rate`). A Zombie at 0 FHR shows `0 0 0 0 1 1 1 2 2 2 2 3 3 3 4 4 4` over its 17 ticks. Other modes, objects and the player's frames are unchanged.
2. **drlg-dump and the patch** (e0731ebd). `tools/drlg-dump` opened only `$D2_PATCH_INSTALLER`, never the `patch_d2.mpq` in the MPQ folder, so without the installer it ran on CD-era data: no `monpreset.bin`, no DS1 monster mapped, no "roll to stay" step (FUN_00667620), and Lut Gholein's rooms were seeded two level-seed steps late. It now opens the MPQs as `open_game_data` does (installer if set and present, else the folder's patch). This is a tool fix, not a generator bug: `components/drlg` already does the roll.

## Evidence from game.exe 1.14d

### Claim 1: get-hit frames

| Address | What it shows |
|---|---|
| `004b13a0` (from `004808d5`) | the client's per-tick update of a monster: `FUN_004af400` gives the mode's row, `FUN_004b1280(unit, row)` steps the animation, the row's column 2 ends the mode on the step's "over" flag |
| `004af474`..`004af499` | the row: `0x7252e0 + mode x 12`; row 3 (GH) is `[0, 1, 2]` |
| `004b1306`..`004b1341` | column 1 = 1: `FUN_006217c0` (over: +0x44 + rate(+0x4c) >= +0x48) and, not over, `FUN_00623e00` (+0x44 += rate); the wrap at `00623ed1` is never reached this way |
| `00621839`..`00621845` | the drawn frame: +0x44 >> 8 (`FUN_00621810`, called by the unit draw at `004715ca`) |
| `006243bf`..`006243d7`, `006244d3`..`006244f4` | the client's mode start (`FUN_00624390`, a monster): +0x44 = 0 for GH, the rate by `FUN_00623f50` (#136), +0x48 = frames << 8 |
| `00463466`..`004634f9`, table `00711e00` row 4 | the player's tick steps the same way and ends a tick before the server's event; the player's own frames are NOT changed here (`// ponytail:` in `set_pmode`) |

### Claim 2: drlg-dump and the patch

| Address | What it shows |
|---|---|
| `FUN_00667620` | the preset-unit roll to stay: id 0xcc, 0xcd, 0x173 or 0x174 (byte table at `0x6677dc`) steps the seed at `0x667721..0x667744` and stays when `low % 3 == 0`; every other unit is kept without a step |
| `FUN_00667970` | calls it on the level seed at generation for a LvlPrest row with Scan or Pops (Lut Gholein: Scan 1, Pops 3), before the rooms are allocated |
| `FUN_0066b3e0` | room allocation steps the level seed once per room (`0x66b451`) |
| `0x744304` + 0xa80 | the MonStats row count the roll compares the id against (`0x6676f3`): the id is the MonPreset-mapped MonStats id, which a DS1 only yields through `monpreset.bin` (in the 1.14d patch only) |
| emulator trace | `lut_trace.py 161464580 40` (memory-write watch on level +0x1c4, nothing hooked out): two seedings, two writes at `0x667744`, 49 at `0x66b451` |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure

# claim 1 (oracle hit_render.py attached to this PR; needs tools/emu/emu.py on PYTHONPATH and your game.exe)
uv run python hit_render.py gen 100000 7 > cases.tsv
uv run python hit_render.py --exe <your game.exe> game cases.tsv > game.tsv
g++ -O2 -std=c++20 -o hit_render_harness hit_render_harness.cpp && ./hit_render_harness < cases.tsv > before.tsv
build/tests/test_combat --get-hit-frames < cases.tsv > after.tsv
uv run python hit_render.py compare game.tsv before.tsv cases.tsv 1    # 1488667 ticks in 49641 of 55089 cases differ
uv run python hit_render.py compare game.tsv after.tsv cases.tsv 1     # 0 ticks in 0 cases differ

# claim 2 (no installer in the environment, patch_d2.mpq in $D2_MPQ_DIR)
cd tools/emu
env -u D2_PATCH_INSTALLER D2_PATCH_INSTALLER= uv run python diff_drlg.py 1-20 40 tiles units seeds   # before 0/20 each, after 20/20 each
for L in 1 2 75 103 109; do env -u D2_PATCH_INSTALLER D2_PATCH_INSTALLER= uv run python diff_drlg.py 1-10 $L tiles units; done   # after 10/10 each
build/tools/mpq-cat/mpq-cat "$D2_MPQ_DIR" 'data\global\excel\monpreset.bin' | wc -c    # 920 with the patch, "not found" without
```

Claim 1: 10,836 boundary cases (every FHR 0..300 for 18 AnimData get-hit records, both kinds) plus 100,000 random (speed 1..1024, frames 1..60, FHR 0..1000, seed 7); the 55,089 monster cases cover 1,747,580 hold ticks; mismatches before 1,488,667, after 0. `test_combat` asserts the Zombie's 17 frames, the Sorceress's, a tick past the hold, rate 512 and rate 0. Claim 2: level 40 seeds 1-20 tiles / units / seeds 0/20 before, 20/20 after; the other five levels 10 seeds each, all 10/10 after (before: level 1 units 0/10, level 2 units 6/10, levels 75 and 103 units 0/10, level 109 tiles and units 0/10).

An independent adversarial verifier read each claim (verdict HOLDS WITH LIMITS for both); its limits are listed below.

## Limits

Claim 1:
- The player's own drawn frames keep d2d's prate stretch (136,651 of 1,653,995 player ticks differ from the stepped rule), and game.exe ends the player's GH on the hold's last tick: a separate change.
- A rate at or above frames x 256 (no vanilla record): the client ends at tick 1 where the server holds 2.
- The merc's FHR is its items'; a joined game's merc has no FHR on the wire (drawn at 0, `// ponytail:`); a party member's merc and pets in GH keep the plain loop (ponytail).
- Negative FHR (no vanilla item), and a frozen monster (`004b13be` tests state 1 before any step) are not modelled. The sub-tick order of the server's mode packet and the client's first step, and what sets flag `+0xc8` bit `0x2000` (tested in `FUN_00463720`, after the hold), are not traced.
- The verifier's adversarial run: 5,544 cases, 6,273,036 ticks at FHR >= 0, 0 differ.

Claim 2:
- With `D2_PATCH_INSTALLER` set and present the stack is unchanged: a no-op for a setup with the installer, including `diff_drlg.py`'s default path. Unpatched upstream main passes 20/20 there; the 0/20 "before" is the tool with the variable unset. The gate was never wrong on your machine; the fix is for everyone without the installer.
- Without the installer upstream's tool exited "MPQ open failed" on a missing default; now it notes the missing file and falls back.
- The in-game generator was not re-measured (its loader already opens the patch). The trace names addresses, not unit ids; the ids come from `monpreset.bin` plus the before / after `units` dumps.
- Harrogath's tile change shows the patch layer carries data files too: anything measured with `drlg-dump` and no installer may have been measured on CD-era files.
- The game wraps `push_patch` in try/catch and falls through on a corrupt patch; the tool throws out of main, as it did for every archive before. `mpq-cat` / `MpqCache` still look for lower-case `patch_d2.mpq` only (a possible follow-up).
- `tools/` needs CODEOWNERS review; `lut_trace.py` is attached to the PR, not committed.

`presubmit.py --quick`: 0 FAIL. WARNs answered: siblings (claim 1: merc and monster, not player / pet / co-op, see limits; net roles: the client draws, the hold is the host's, #136); doc-claims "never loops" (claim 1's differential); error-path `fs::exists` at `tools/drlg-dump/drlg-dump.cpp:76` and `:81` (same call as the existing d2exp / d2data line; a missing path is `false`, not a throw).

## Screenshots

None: claim 1 is exact per tick by the differential, claim 2 changes dumps only.

## Checklist

- [x] A test in `tests/` (claim 1: `test_combat`; claim 2: a tool, covered by `diff_drlg.py`, none added)
- [x] The finding in `docs/research/re/` (claim 1: combat.md "Drawn frame"); claim 2 moves no row
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN against upstream/main and per commit)

---

Agent: Claude Code
