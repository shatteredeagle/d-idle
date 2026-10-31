# PR draft: get-hit by hit class and share of life

> status: PR #134 follow-up pushed and replied 2026-10-31 (commit e3c1e535). PR #134 is open upstream (agent HITREC, 2026-10-30). The branch exists only in the clone `~/d2d-lab/pr/hitrec` on the worker host (fork of psi29a/d2decomped, `upstream` fetched at 32768ddb), commit `d55973e1`. Each push and the PR need the user's own go (upstream AGENTS.md section 2). Proof: docs/research/pr-get-hit-rule-proof.md; evidence pack: docs/research/pr-get-hit-rule-evidence-pack.md.

## Draft PR body (their template, `.github/pull_request_template.md`)

````
## What changes

d2d put a unit into hit recovery (GH) at a fixed share of its max life: a player at a twelfth, a monster at an eighth, always. game.exe 1.14d (`FUN_0057cb00`) picks the share by the blow's hit class (a sixteenth for most classes, an eighth for classes 2, 6, 10, 11, a thirty-second for 4, 8, a sixty-fourth for 5, a sixteenth for any blow that carries fire, lightning, cold or poison, or a doubled hit) and then rolls: from that share 37.5 %, from twice it 75 %, from four times it always; under a point of damage never. `rules::get_hit` now does that for the player, the merc and monsters, with the blow's hit class and whether an element landed (`Blow::elemental`).

Scope: the get-hit decision on a landed hit. Not changed: the length of the get-hit (d2d's `speed_frames` with 100 + FHR; game.exe uses 50 + FHR, a separate claim), the frozen exemption (state 1; d2d has no freeze state), MonStats2's GH-mode bit, and the stunned monster's sure get-hit (d2d's stunned monster stands anyway). Siblings: the merc uses the same rule (its own seed); pets, thorns, a fire blast and a monster hitting a monster reach `hurt` with class 0 (a sixteenth), which is game.exe's value for a plain record (class 13), not traced per source. Missiles and skills reach the same rule through `Fight::land` with the missile's HitClass (Charged Bolt 64: a sixteenth).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0057cee0` | a unit takes damage: on result flag 4 (the hit landed), unless state 21 (stunned, a sure get-hit), calls `FUN_0057cb00(unit, record, record +0x60)` at 0x57d0c8 (monster, then mode 3 through `FUN_005a7e60`) and 0x57d37a (player, mode 4 through `FUN_005809d0`); a 1 back means no get-hit (`FUN_0064c040`) |
| `0057cb00` | state 1 (frozen): none; record [0x28] poison equal to [0x4c] total: none; [0x4c] under 0x100: none; max life `FUN_00625d10`; divisor by hit class through the jump table at 0x57cbf8 (classes 1..12: 16, 8, 16, 32, 64, 8, 16, 32, 16, 8, 8, 16; else 16); under max / d none (0x57cb76); under max / (d / 2): `FUN_00472210(seed, 2)` == 0 none (0x57cb95); under max / (d / 4): `FUN_00472210(seed, 4)` == 0 none (0x57cbc0); a monster also needs `FUN_0046c140(row, 3)` (0x57cbda) |
| `0057ce30` | the hit class fix-up run at 0x57cf54 (result flag 1, record +0x64 == 0, no 0xf0 bits): the first of the table at 0x6e16d8 whose record field is over 0 gives its bits (0x24 cold 0x30, 0x10 fire 0x20, 0x1c lightning 0x40, 0x28 poison 0x50), else 0x10 for record flag 0x2000, OR-ed with the class or 13 for none |
| `0057b929` | record flag 0x2000: set when the mastery crit (0x57b8b4), critical strike (stat 0x151, 0x57b8e0) or deadly strike (stat 0x8d, 0x57b903) roll lands and the damage is doubled |
| `00472210` | the seed step `low * 0x6AC690C5 + high`, the roll is `low & (n - 1)` (d2d's `Rng` does the same) |
| `0057c8bc` | the record's [0x28] / [0x2c] go to the poison applier `FUN_0057ac50`: +0x28 is the poison per tick |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure   # test_game needs D2_MPQ_DIR and D2_PATCH_INSTALLER
./build/tests/test_combat                                           # the divisor table and the bands
# game.exe's own FUN_0057cee0 on 2264 boundary + 100000 random cases against this branch's rule (oracle script below):
cd tools/emu && uv run python hit_rule.py gen 100000 7 > cases.tsv
uv run python hit_rule.py --exe /path/to/game.exe game cases.tsv > game.tsv                     # 16 s
../../build/tests/test_combat --get-hit < cases.tsv > d2d.tsv && uv run python hit_rule.py compare game.tsv d2d.tsv cases.tsv   # 0 differ
```

Before this branch the same cases differ in 19,343 of 102,264 (8,701 player, 10,642 monster), with the old lines quoted in a 30-line harness.

Not done: the oracle script `hit_rule.py` is not in this PR (it would touch `tools/`, which needs the maintainer's CODEOWNERS review); I can send it as a second PR or attach it here. Limits: d2d compares this frame's whole points of damage against max life in 256ths (game.exe compares the record's 256ths: equal at whole points); the player's rolls take d2d's rng, not the player's own seed (as the other player rolls do); game.exe's total +0x4c includes the record's poison field, d2d's damage does not (a physical + poison blow can roll there and not here); the player's rolls draw from d2d's game rng (0 to 2 draws a struck frame, so later rolls in that frame shift against the old build); the record's +0x64 byte (keeps the class) is untraced and treated as 0; the town / attacker gate and the state 0x41 / 0x42 / 0x44 branches of FUN_0057cee0 are not modelled. `test_game`'s arrow scenario now deals half of max life instead of a quarter: a quarter is the 75 % band for the arrow's class, a half is sure for every class.

## Screenshots
Not applicable.

## Checklist
- [x] A test in `tests/` (fix or port)
- [x] The finding in `docs/research/re/` (combat.md "Hit recovery: the get-hit test"; the ponytail line under Approximations removed; no todo.md / unverified.md row existed)
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code (Claude Fable 5.1)
Traced-by: d2lod-bot agents DECOMPMON and DECOMPREC (the first reads of FUN_0057cb00)
````

## Presubmit answers (V)

- `siblings: unit kinds` (merc, pet, co-op partner): the merc is in the diff (fight.cpp:4800, same rule, its own seed); pets reach `hurt` with class 0 (a sixteenth); a co-op partner is the same player path on its host.
- `siblings: hit paths` (missile, skill): both go through `Fight::land` -> `hurt` with the missile's / skill's `hit_class` and `Blow::elemental`; the harness cases include missile classes 0x40 / 0x20 / 0x43 / 0x12.
- `build: skipped (--quick)`: built and run on the worker host (`/tmp/hitrec-build`): test_combat ok, test_game 23 OK.
