# Verify: the Teleport room rule (VERIFY-TELEPORT-ROOM, card verifytele)

> status: independent re-check, 2026-11-02, offline (no game, no d2d slot). Verifier: Claude Sonnet. Claim under test: docs/research/sk-teleport-trace.md (SKTELE).
> Tags: **V read** disassembled by me with `tools/emu/mm_dis.sh` before I opened the author's trace; **V run** game.exe code run under the emulator on the worker host; **U** not verified.

## What I read first (V read)

- FUN_005ca360 (srvdofunc 27): the target point comes from FUN_0056d2c0; the caster's room FUN_00620bb0 (0 refuses); the Levels row FUN_0061a1b0 / FUN_0061db70 (0 refuses); byte +4 is the Teleport value: 0 refuses, 2 calls the line test FUN_006229f0 with mask 0x804 on the clicked point and refuses when it answers non-zero, any other value goes on; then FUN_00554ea0(caster, room 0, x, y, 0, 0) is the answer.
- FUN_00554ea0: with room argument 0 it first calls FUN_00463740 with room 0 (returns 0 at once), then FUN_00463740 with the caster's room. 0 from both: return 0 (refuse). Otherwise the nearest free spot FUN_0064e7b0 (unit size from FUN_00620510, mask 0x1c09), then the position set FUN_00650be0; 0 from it also refuses.
- FUN_00463740: the point is tested against the room rectangle (x0, x0+w, y0, y0+h); if outside, it walks the room's near list (FUN_00619790) and returns the first room whose rectangle holds the point; none: 0.

So the range is a room rule: the caster's room plus its near list; no distance appears anywhere.

## Run (V run)

Command (worker host, about 50 s): `tools/emu/run-on-worker.sh --ttl 1500 vt_run.sh 20000 4242`
It regenerates cases with MY seed (4242, not the author's 1 or 7), runs the author's native oracle `sk_tele_spot.py game` (FUN_005ca360 down to the position set) and classifies the answers with my own `tools/emu/vt_check.py` against the room rule above. World: 5 x 5 rooms of 40 x 40 subtiles, caster in the centre room, Teleport value 0 or 1 (5 percent zeros), blocked cells, units, targets near, anywhere, on a monster and off the map.

| class | refused | accepted |
|---|---|---|
| Teleport 0, target inside the 3 x 3 rooms | 565 | 0 |
| Teleport 0, outside | 436 | 0 |
| Teleport 1, outside the 3 x 3 rooms | 8,124 | 0 |
| Teleport 1, inside the 3 x 3 rooms | 0 | 10,875 |

Total 20,000: accepted 10,875, refused 9,125.

## Claims

| # | claim | verdict | note |
|---|---|---|---|
| C1 | a hop to a point outside the caster's room and its near list is refused (FUN_00554ea0 + FUN_00463740 over FUN_00619790) | HOLDS (V read + V run, 8,124 of 8,124 refused) | the near list in the oracle is the 8 touching rooms (a hook, see limits) |
| C2 | a point inside that set is accepted, however far | HOLDS (V run, 10,875 of 10,875 incl. snap, none refused for lack of a spot) | in this synthetic world the snap always found a spot; real dense rooms untested |
| C3 | it is a room rule, not a distance | HOLDS (V read; V run: a Chebyshev <= 40 rule disagrees with game.exe in 1,635 cases) | |
| C4 | Teleport value 0 refuses | HOLDS (V read + V run, 1,001 of 1,001 refused) | |
| C5 | Teleport value 2 adds a line test (FUN_006229f0, mask 0x804) on the clicked point | HOLDS WITH LIMITS (V read only) | the oracle hooks FUN_006229f0 to "clear" and never generates value 2; the line walk (FUN_0064e260) was not read: U |

## Differences from the author's numbers

None in kind: author and I agree on the rule. I did not rerun their d2d line copy, so their D1 to D3 counts (blocked target landings, etc.) are not re-checked here, and I did not compare with `diff` output.

## Limits (not covered)

- The oracle and its hooks (room rectangles, near list, collision grid) are the author's; independence is in my own read of the three functions and my own classifier and seed. A bug in a hook (for example a near list that is always the 8 touching rooms) would be shared.
- The real near list (FUN_00619790 on real level data: rooms joined by exits, outdoor 8-tile rooms, larger rooms) is U; no real level was loaded.
- Teleport 2, the player's own tail of FUN_00554ea0 (client packets, pets), and d2d's real build are not covered.

## Reproduction

`tools/emu/run-on-worker.sh --ttl 1500 vt_run.sh 20000 4242 > /tmp/d2agent-pa-verifytele/out.txt 2>&1` (needs upstream's emulator checkout and the patched game data).

## Shared-file lines for the merger

See `openspec/changes/verifytele/merge-lines.txt`.
