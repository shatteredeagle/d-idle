# Draft PR body: get-hit length by AnimData speed x (50 + FHR) (agent HITLEN, 2026-10-31)

> status: draft, NOT posted (card hitlen). Proof: docs/research/pr-get-hit-length-proof.md; go / no-go: docs/research/pr-get-hit-length-evidence-pack.md. Branch `fix/get-hit-length` in `~/d2d-lab/pr/hitlen` on the worker host, commit `a2f4a647`. Spacing: PR #134 (decision) and #135 (missile damage) are open; this is the one new candidate, at least one hour after the last push. Wait for #134's outcome first if the maintainer asks for a bundle.

The body (their template, comments removed), inside the fence:

````markdown
## What changes

d2d held a get-hit for the animation's plain length shortened by (100 + FHR) % for the player (`speed_frames`) and for the plain length for monsters and the merc: a Sorceress staggered 8 ticks, a Zombie 9. game.exe rates the get-hit at AnimData speed x (50 + effective FHR) / 100 and holds the mode one tick past the frames that fit: the Sorceress 16 ticks at 0 FHR, 15 / 14 / 13 / 12 / 11 / 10 / 9 / 8 / 7 / 6 at 5 / 9 / 14 / 20 / 30 / 42 / 60 / 86 / 142 / 280 (the known table's 15 / 14 / ... / 5 is what the client draws, one fewer), a Zombie 17, a Fallen 14. Now `rules::get_hit_rate` / `get_hit_ticks` time the player's, a monster's and the merc's get-hit; monsters get a MonProp FHR (`MonProps::fhr`). The block length and the client's drawn frame count are not touched.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0x624089`..`0x62409d` | `FUN_00623f50`'s dispatch: a player's mode 4 or a monster's mode 3 takes the get-hit branch |
| `0x62409f`..`0x6240b1`, `0x62435c`..`0x624385` | rate = (`FUN_00621740(unit, 1)` + 50) x AnimData speed (+0xc) / 100, floored, 0..0x7fff, at unit +0x4c; no other stat and no state test on this branch (the block branch is the one with state 0x65) |
| `FUN_00621740` | row 1 of the table at 0x6e8e24: stat 99 via `FUN_00625500`, 120 v / (120 + v) for v != 0 |
| `FUN_005533d0` | the server's mode start: +0x44 = 0, +0x30 = 0, `FUN_00623f50`, +0x48 = frames << 8 |
| `FUN_005539b0` (0x5539ee, 0x553a36, 0x553ab2..0x553ac0, 0x553ac9..0x553ae2) | the mode's events: rate 0 -> event 1 at frame + 1; else from frame 0 one tick per rate while under the end, then + 1 if no tick passed, + 1, event type 1 there (`FUN_005417d0`) |
| callers (`game-calls.tsv`) | `FUN_0057fe30` (the player's mode-4 handler, table 0x6e1740[4]) and `FUN_005a7c20` (the monster's mode set, 0x5a7e2b) both call `FUN_005539b0` after `FUN_005533d0` |
| `FUN_00581020` 0x5810a7..0x5810b1; table 0x6e2298 | event 1: the player plays NU (`FUN_0057f020`); a monster's mode 3 ends with `FUN_005a8030` -> NU |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
# the oracle (attached to the PR conversation, not in the tree): game.exe's FUN_00623f50 branch and FUN_005539b0 under unicorn
uv run python tools/emu/hit_len.py gen 100000 7 > /tmp/cases.tsv
uv run python tools/emu/hit_len.py game /tmp/cases.tsv > /tmp/game.tsv
build/tests/test_combat --get-hit-length < /tmp/cases.tsv > /tmp/after.tsv
uv run python tools/emu/hit_len.py compare /tmp/game.tsv /tmp/after.tsv /tmp/cases.tsv   # 0 of 110836 differ (main's rule: 102980)
```

## Limits

The FHR range checked is 0..1000 (negative FHR: game.exe divides, d2d keeps v; no vanilla item has one). A monster's FHR is its MonProp's; a state's stat 99 is not read. The block length (50 + FBR) and the drawn frame count stay as they were. presubmit's siblings line: the player (`set_pmode`), the merc and monsters all take the new rule; pets have no get-hit set in d2d; a co-op partner is a player.

## Checklist

- [x] A test in `tests/` (fix or port): `test_combat` asserts and `--get-hit-length`
- [x] The finding in `docs/research/re/`: combat.md "Hit recovery: the length"; no todo / unverified row existed
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN against upstream/main)

---

Agent: Claude Code
````
