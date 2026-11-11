# Draft (local, NOT posted): "Lightning hits one unit a frame and skips the last (FUN_005ae1f0)"

> status: ready for the independent verifier (prprep, 2026-11-11). Branch `fid/missile-last-unit-r2` on the private fork remote, head `89059e2d`, two commits on upstream main `b6b59ae9`. Replaces the older `fid/missile-last-unit` (head f45e45c8, base a5b970d5), which does not apply to current main and had a scope problem (see "Honest limits"). Trace: [../../probes/fid/fid-missile-player.md](../../probes/fid/fid-missile-player.md).

Nothing here is posted. PR title: `Lightning hits one unit a frame and skips the last (FUN_005ae1f0)`

## What changes

A missile whose Missiles.txt row has LastCollide and neither NextHit nor Pierce (in the player's skill set that is `lightningbolt`, Lightning) struck every monster it touched, once each per cast (the `struck` list). game.exe's collision handler FUN_005ae1f0 hands the missile ONE unit a frame (the first unit FUN_005a8730 passes, in the order FUN_00641cb0 found them) and passes every unit except the one the missile struck last; the hit (FUN_005adf10) makes that unit the last. So a monster the bolt left for another can be struck again, and two monsters in one frame take two frames.

`Fight::skill_missile_hits` now keeps `last_hit` for these rows the way it already does for NextHit rows (#158), with no `struck` entry; `Fight::world` ends the missile's frame at the hit. Two commits: the oracle and its d2d side first (they exist and fail on the parent), then the claim (a few lines in two files, one doc paragraph in `docs/research/re/missiles.md`).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005ae1f0` | per frame, after FUN_00641cb0 found a unit in the subtiles the missile entered: the first unit the filter passes is handed to FUN_005adf10 and the frame is over |
| `FUN_005a8730` (called from `0x5a8850`) | the filter: passes every unit but `FUN_0064a470` (the missile's last unit); a NextHit row also tests state 0x56 |
| `FUN_005adf10` `0x5adfae` | the hit: `FUN_0064a400` makes this unit the missile's last unit |
| Missiles.txt `lightningbolt` | LastCollide 1, NextHit empty, Pierce empty (D, `tools/emu/mpq-table.sh missiles`) |

## Counted differentials (V run, the worker host)

`tools/emu/light_pierce.py` runs game.exe's real filter `0x5a8850` -> FUN_005a8730 and the hit FUN_005adf10 (plus FUN_005adb00, FUN_0064a470, FUN_0064a400; owner, hostility, stat getters and the damage roll hooked) on the contacts d2d's own Lightning flight produced. `test_light_pierce --light-hit SEED N` runs d2d's real `Fight::fire` / `Fight::world` / `skill_missile_hits` on packs of 1..6 standing monsters, sizes 1..3, 8 headings, levels 1..20, boundary cases plus random packs, and prints the contacts and the hits per monster.

| Run (rebased tree, upstream `b6b59ae9`) | Casts | Casts that differ from game.exe |
|---|---|---|
| parent (oracle commit only), seed 1, N 400 | 1192 | 44 |
| parent, seed 7, N 400 | 1192 | 43 |
| with the claim, seed 1, N 400 | 1192 | 0 |
| with the claim, seed 7, N 400 | 1192 | 0 |
| neighbour: `test_next_hit --next-hit 1 400` against `next_hit.py` (NextHit rows, #158) | 598 | 0 |

Run log and scripts of this prep: `tools/emu/pp_light.sh before|after SEED N` (the before run builds the parent), `pp_light_all.sh`.

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build --target test_light_pierce test_next_hit test_game_missiles test_game test_monsters test_combat
cd tools/emu && uv sync
../../build/tests/test_light_pierce --light-hit 1 400 > /tmp/d2d.txt
uv run python light_pierce.py /tmp/d2d.txt > /tmp/g.txt
grep -E '^[HT]' /tmp/d2d.txt | diff /tmp/g.txt - && echo same
```

`test_game_missiles`, `test_monsters`, `test_combat` and `test_game` pass with the change (rc 0, run with the 1.14d data and patch installer).

## Honest limits (to put in the body)

- Scope. The change is limited to rows with LastCollide and without Pierce. The same FUN_005a8730 filter serves the Pierce rows too (arrows, javelins, Blessed Hammer), but their oracle needs ToHit and the hit and damage functions, which `light_pierce.py` refuses, and their charges are a different rule (FUN_005ada80, #225). They keep d2d's `struck` list. The older version of this branch applied the rule to every LastCollide row and broke `test_game_missiles` (the #225 pierce-charge test hits one monster repeatedly, line 734 on the current tree); narrowing it to `!pierce` is the fix and is what is proven here. A follow-up would extend the oracle to the arrow rows.
- Which unit goes first when two units are found in one subtile is the monster order here; game.exe's is its room's unit list (not read). The earlier trace (fid-missile-player.md) found the two structural counts unchanged in either contact order; that was not re-run on this tree.
- The Chain Lightning part (NextHit rows) is #158's and untouched.
- Overlap check (read-only `gh`, 2026-11-11): the open #426 (Holy Bolt heals a pet) is the only open PR touching `components/game/fight/missiles.cpp`; a textual overlap is not expected (a different function).

## Checklist

- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same pull request (the oracle commit first, the claim second)
- [x] The finding in `docs/research/re/missiles.md`
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN
- [x] `python3 tools/presubmit.py --quick --base upstream/main` (on the head before the README row fix, which changes no code): 0 FAIL, 4 WARN (sibling unit kinds and hit paths: the change is a player skill missile on monsters only; doc-claims line 547 "every unit" is the FUN_005a8730 filter text; build skipped by `--quick`, built by hand above)

Agent: Claude Code
