# Draft PR body: "Draw a get-hit once at its rate, held on the last frame" (agent RENDERGH, 2026-10-31)

> status: draft, current (card rendergh). Not posted. Branch `fix/get-hit-render` in the private clone (commit 57bd4b56, stacked on PR #136's eb89646e; the base has since moved to 9006a688: rebase first, see the evidence pack). Upstream template `.github/pull_request_template.md`; one claim; the oracle attached to the PR conversation, never game data. Title: `Draw a get-hit once at its rate, held on the last frame`.

````markdown
## What changes

With #136 a monster's or the merc's get-hit holds for game.exe's length, but d2d still drew the GH animation at its base rate and looped it, so a long hold replayed it (a Zombie: 5 frames over 17 ticks played three and a half times). game.exe's client steps the unit's frame position by the get-hit rate each tick only while position + rate stays under the end, and draws position >> 8: the animation plays once at the get-hit rate and stands on its last stepped frame; it never loops. Now d2d draws a monster's and the merc's GH that way (`rules::get_hit_frame`; `Unit::get_hit_rate` in the renderer). A Zombie at 0 FHR shows frames `0 0 0 0 1 1 1 2 2 2 2 3 3 3 4 4 4` over its 17 ticks. Other modes, objects and the player's own frames are unchanged.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `004b13a0` (from `004808d5`) | the client's per-tick update of a monster: `FUN_004af400` gives the mode's row, `FUN_004b1280(unit, row)` steps the animation, the row's column 2 (jump `004b15e0`) ends the mode on the step's "over" flag through `FUN_004af6a0` |
| `004af474`..`004af499` | the row: `0x7252e0 + mode x 12` for a mode the MonStats record has (bit at +0x104); row 3 (GH) is `[0, 1, 2]` |
| `004b1306`..`004b1341` | column 1 = 1 (jump `004b1388` -> `004b1327`): `FUN_006217c0` (over: +0x44 + rate(+0x4c) >= +0x48, `006217fc`..`0062180c`) and, not over, `FUN_00623e00` (+0x44 += rate, `00623e55`..`00623e7b`); the wrap at `00623ed1` is never reached this way |
| `00621839`..`00621845` | the drawn frame: +0x44 >> 8 (`FUN_00621810`, called by the unit draw at `004715ca`) |
| `006243bf`..`006243d7`, `006244d3`..`006244f4` | the client's mode start (`FUN_00624390`, a monster): +0x44 = `FUN_00623b10` << 8 (0 for GH), the rate by `FUN_00623f50` (#136's), +0x30 = 0, +0x48 = frames << 8 |
| `004af80c`..`004af851` | the ended mode: NU through `FUN_00480e70` at the first tick with position + rate >= the end, the tick #136's server event fires |
| `00463466`..`004634f9`, table `00711e00` row 4 | the player's tick steps the same way and re-tests after its step: the mode ends on the client a tick before the server's event. The player's own frames are NOT changed here (`// ponytail:` in set_pmode): d2d's prate stretch differs from the stepped rule on about 8% of ticks |

## How to re-check

Differential against game.exe's own code (`FUN_004b1280` with the real row at `0x7252e0 + 36`, `FUN_00621810`, `FUN_00623f50`, `FUN_005539b0` run natively under unicorn; hooked: the stat getters `FUN_00625500` / `FUN_00625480`, the state test `FUN_00639df0`, the event queue `FUN_005417d0`; a MonStats row planted at `0x744304` for `FUN_004b1280`'s head). The oracle `hit_render.py` is attached to this PR (it needs `tools/emu/emu.py` on PYTHONPATH and your own `game.exe`); `hit_render_harness.cpp` is the old renderer's arithmetic for the before.

```sh
uv run python hit_render.py gen 100000 7 > cases.tsv
uv run python hit_render.py --exe <your game.exe> game cases.tsv > game.tsv
g++ -O2 -std=c++20 -o hit_render_harness hit_render_harness.cpp && ./hit_render_harness < cases.tsv > before.tsv
build/tests/test_combat --get-hit-frames < cases.tsv > after.tsv
uv run python hit_render.py compare game.tsv before.tsv cases.tsv 1    # 1488667 ticks in 49641 of 55089 cases differ
uv run python hit_render.py compare game.tsv after.tsv cases.tsv 1     # 0 ticks in 0 cases differ
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
```

10,836 boundary cases (every FHR 0..300 for 18 AnimData get-hit records, both kinds) + 100,000 random (speed 1..1024, frames 1..60, FHR 0..1000, seed 7); the 55,089 monster cases cover 1,747,580 hold ticks. `test_combat` asserts the Zombie's 17 frames, the Sorceress's, a tick past the hold, rate 512 (stands on frame 6) and rate 0.

## Limits

- The player's own drawn frames keep d2d's prate stretch (136,651 of 1,653,995 player ticks differ from the stepped rule), and game.exe's client ends the player's GH on the hold's last tick (what it shows then depends on player data +0x154, not traced): a separate change.
- A rate at or above frames x 256 (no vanilla record): the client ends at tick 1 where the server holds 2; d2d draws the stepped frame on that tick.
- The merc's FHR is its items' (`merc_fighter`); a joined game's merc has no FHR on the wire (drawn at 0; ponytail); a party member's merc and pets in GH keep the plain loop (ponytail).
- The COF's frames per direction are taken as AnimData's (the renderer's `composite_frames` divides by the same `ms_per_frame`).
- Not traced: the sub-tick order of the server's mode packet and the client's first step (both count tick 0 at the set); `FUN_00463720` (the gate before NU at `004af833`; after the hold).
- presubmit: siblings (merc, monster, not player / pet / co-op: the player is listed above as a gap, pets have no GH in d2d, a co-op partner's merc is a pet here); net roles (client, not host: the hold is the host's, #136); doc-claims "never loops" (the differential).

## Checklist

- [x] A test in `tests/` (fix or port)
- [x] The finding in `docs/research/re/` (combat.md "Drawn frame")
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
````
