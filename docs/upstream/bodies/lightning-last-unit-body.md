## What changes

A player missile whose Missiles.txt row has srvdofunc 1, LastCollide, and neither NextHit nor Pierce struck every monster it touched, once each per cast (the `struck` list). game.exe's handler for srvdofunc 1 (table at `0x73c768`, entry 1 = `jmp 0x5ae1f0`) hands the missile ONE unit a frame (the first unit FUN_005a8730 passes) and passes every unit except the one the missile struck last; the hit (FUN_005adf10) makes that unit the last. A monster the bolt left for another can be struck again, and two monsters in one frame take two frames.

`MissileInfo::last_unit_rule()` names the rows (`last_collide && !next_hit && !pierce && srv_do == 1`). `Fight::skill_missile_hits` keeps `last_hit` for them as it does for NextHit rows (#158), with no `struck` entry; `Fight::world` ends the missile's frame at the hit. Two commits: the oracle with its d2d side (they fail on the parent), then the claim (three code files, one doc section).

### Skills this reaches

Derived from Skills.txt and Missiles.txt (`a scan of Skills.txt and Missiles.txt`, rows with CollideKill empty so the missile survives its hit): the player skills whose missile is such a row are Lightning (`lightningbolt`), Lightning Fury's bolts (`furylightning`, same flags), Inferno (`infernoflame1`), Bone Spear (`bonespear`) and Arctic Blast (`arcticblast1`). The same table scan also lists rows of Meteor (`meteorcenter`), Howl, Molten Boulder, and four Assassin trap / strike sub-missiles; whether d2d reaches those through `skill_missile_hits` was not checked (U). Firebolt-class rows (CollideKill 1) end on their first hit, so nothing changes for them. A throwaway variant of the test with the Bone Spear row (never committed) shows the same: the parent differs from game.exe on 12 T lines on seed 1, the claim on 0. The same variant with the Arctic Blast and Inferno rows shows no difference on either tree (a standing pack in a line does not exercise their extra behaviour). The Lightning Fury row does not run meaningfully in this harness (about 996 of 1,192 casts differ on both trees: Fury is not cast as a single bolt), so Lightning Fury is reached by the same flags but is NOT covered by this differential.

NOT touched, on purpose: Pierce rows (arrows, javelins, Blessed Hammer: their charge rule is #225's and their oracle needs ToHit and hit / damage functions) and srvdofunc 3 rows (Blizzard shards, Poison Explosion, Poison and Plague Javelin clouds: they stand at velocity 0, d2d strikes them through `standing_struck`, and what FUN_005ae480 / FUN_0064cb90 do for a standing missile with several units in reach was not run).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0x73c768` entry 1 -> `FUN_005ae1f0` | the missile srvdofunc 1 handler (read from the loaded image) |
| `FUN_005ae1f0` | per frame: range minus 1, then for each subtile the move entered, FUN_00641cb0 through the filter; the first unit found goes to FUN_005adf10 and the function returns |
| filter at `0x5a8850` -> `FUN_005a8730` | passes every unit but `FUN_0064a470` (the missile's last unit); a NextHit row also tests state 0x56 |
| `FUN_005adf10` `0x5adfae` | the hit: `FUN_0064a400` makes this unit the missile's last unit |
| Missiles.txt `lightningbolt` | LastCollide 1, NextHit empty, Pierce empty, CollideKill empty, srvdofunc 1 (D) |

## Counted differentials (V run, the worker host, tree = upstream `62732598` plus the branch)

`tools/emu/light_pierce.py` runs game.exe's real filter and hit (FUN_005a8730, FUN_005adf10, FUN_005adb00, FUN_0064a470, FUN_0064a400; owner, hostility, stat getters and the damage roll hooked) on the contacts d2d's own Lightning flight produced. `test_light_pierce --light-hit SEED 400` runs d2d's real `Fight::fire` / `Fight::world` / `skill_missile_hits` on packs of 1..6 standing monsters, sizes 1..3, 8 headings, levels 1..20; each seed is 1,192 casts.

| Tree | seed 1 | seed 7 | seed 13 |
|---|---|---|---|
| parent (oracle commit only): casts whose hit totals differ, of 1,192 | 44 | 43 | 47 |
| with the claim | 0 | 0 | 0 |

Neighbour: `test_next_hit --next-hit 1 400` against `next_hit.py` (NextHit rows, #158): 0 of 598. `test_game_missiles` (the #225 pierce test), `test_game`, `test_monsters`, `test_combat`: rc 0 on the claim.

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build --target test_light_pierce test_next_hit test_game_missiles test_game test_monsters test_combat
cd tools/emu && uv sync
../../build/tests/test_light_pierce --light-hit 1 400 > /tmp/d2d.txt
uv run python light_pierce.py /tmp/d2d.txt > /tmp/g.txt
grep -E '^[HT]' /tmp/d2d.txt | diff /tmp/g.txt - && echo same
```

## Honest limits (to put in the body)

- Which unit of a frame goes first. The oracle replays d2d's contacts in d2d's monster-index order and takes the first the filter passes; game.exe walks the subtiles the missile entered in path order. With the contacts re-ordered along the heading (an approximate path order, a throwaway variant of the test) the claim differs from game.exe in 18 T lines on seed 1 and 15 on seed 7, against 37 and 33 for the parent: the change moves d2d closer, not to equal, when several monsters sit in different subtiles of one frame. Making `Fight::world` order the frame's candidates along the flight is a possible follow-up (not done here).
- Pierce rows and srvdofunc 3 rows keep the `struck` list (see above); extending the oracle to them is a follow-up.
- Overlap check (read-only `gh`, 2026-11-11): the open #426 (`fight/missiles.cpp`, a different function) and #405 (`re/missiles.md`, a different section); a textual rebase at most.
- presubmit `--quick` WARNs: sibling unit kinds and hit paths (the change is player skill missiles on monsters only; melee and the other unit kinds are untouched), doc-claims at `missiles.md` line 548 ("every unit but the one the missile struck last" is the FUN_005a8730 filter text), build skipped by `--quick` (built and tested by hand, see above).

## Checklist

- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same pull request (oracle commit first, claim second; the oracle commit builds on its own)
- [x] The finding in `docs/research/re/missiles.md`
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN
- [x] `python3 tools/presubmit.py --quick --base upstream/main`: 0 FAIL, 4 WARN (answered above)
- [x] `ctest -j8` on the full build: the same 4 failures as upstream main on my box (`test_mpq`, `smoke_d2d`, `quests_act1`, `quests_act4`: no video MPQs here), nothing else

---

Agent: Claude Code
