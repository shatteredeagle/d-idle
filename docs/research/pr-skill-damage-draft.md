# PR draft: a skill's element damage with synergies and masteries as game.exe computes it

> status: DRAFT (agent DECOMPSKILL, 2026-10-31). Nothing is pushed, posted or filed. Branch `fix/skill-damage` in `~/d2d-lab/pr/skilldmg` on the worker host (commits 20441196, d72ee7f3, c6be9247 on upstream/main f8aab0e8). Proof: docs/research/pr-skill-damage-proof.md. The body below is in their template (.github/pull_request_template.md), ready for `--body-file`. posted:

````
## What changes
Two commits, one claim each, on `rules::elem_damage` (the skill's element damage in 256ths that `missile_damage`, the melee `skill_element`, the auras and the tooltip read):

1. A skill's **min** element damage takes its synergy only when the base min is over one point (0x100) or the row has an EMinLev1. game.exe's FUN_00644d50 tests that before adding the EDmgSymPerCalc share; the max (FUN_00644e40) has no such test. d2d added the share to the min always, so Lightning, Chain Lightning, Lightning Bolt, Charged Strike, Lightning Strike, Lightning Fury, Power Strike, Holy Shock, Shock Field, Charged Bolt Sentry, Claws of Thunder, Lightning Sentry and Death Sentry (EMin 1 at HitShift 8, or 2 at 7, no EMinLev1) rolled a min above 1 with synergies; now their "1-x" stays 1, as in the game.
2. The synergy and mastery shares go through MulDiv (FUN_00483360: above 0x100000 it divides first), with d2d's existing `mul_div`; d2d multiplied and divided in plain integers. Up to one point off on hits over 4,096 points (Fire Claws, Meteor, Blizzard at high levels with big shares).

Not changed: the level brackets, the length, the mastery mapping, the missile's roll and the life arithmetic (#135), `elem_length` and `skill_phys` (0 mismatches / outside the claims). Siblings: the change is the skill's own calc; the missile and melee paths read it unchanged.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_0059f900` -> `FUN_0064b860` | a missile's damage struct: FUN_00644d50 / FUN_00644e40 / FUN_00644f20 at the missile's level with the mastery flag; `FUN_0064ac60` sets the missile's stats 0x30..0x3b from it and no mastery stat, so FUN_005a8910's mastery read at roll time is 0 (the mastery counts once) |
| `FUN_00644d50` | min: `(EMin + FUN_00644b70 brackets) << HitShift`; `LAB_00644db3`..`LAB_00644dc2`: the synergy share only when the value is over 0x100 or EMinLev1 (+0x1e8) is set; `MulDiv(value, calc, 100)` at `LAB_00644e0b`; then FUN_00644c90 with the flag |
| `FUN_00644e40` | max: the same without the gate (`LAB_00644e9e`..`LAB_00644eac`) |
| `FUN_00644c90` | the mastery: byte table by EType (fire 0x149, lightning 0x14a, cold and freeze 0x14b, poison 0x14c), `MulDiv(dmg, stat, 100)` |
| `FUN_00646c00`, `FUN_00646460` | skill(id, op): the unit's record (FUN_00643810), its level (FUN_006442a0, flag 1 with items; op 41 `blvl` flag 0), then the operand |
| `FUN_00483360` | MulDiv: `a * b / c` up to 0x100000, `a / c * b` above |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure      # test_game needs D2_MPQ_DIR and D2_PATCH_INSTALLER
# the oracle: game.exe's own FUN_00644d50 / FUN_00644e40 / FUN_00644f20 (mastery flag) on the loaded tables,
# 16,000 grid cases (every elemental skill, levels 1..40, alone / neighbours at 20 / masteries) + 100,000 random,
# against d2d's real elem_damage / elem_length on the same cases:
cd tools/emu && uv run python skill_damage.py gen 100000 1 > cases.tsv && uv run python skill_damage.py game cases.tsv > g.txt
../../build/tests/test_game --skill-damage cases.tsv | grep -E '^[0-9]+ ' | diff g.txt -    # empty (4,549 lines differed before the first commit, 344 after it)
```

Limits: the oracle hooks the stat getter (clvl and the four masteries), the unit's skill record lookup and its level (one level per skill: base = with items; game.exe's flag 0 / 1 split in FUN_006442a0 matches `base_level` / `level` by reading); the calc machine, the operand table, skill() and MulDiv run natively. Synergy skills in the cases are the skill's class neighbours (no calc references another class). `elem_length` and `skill_phys` keep their plain integer shares (lengths matched in every case; the physical part is not this claim). Static Field (FUN_005c96a0) differs elsewhere (difficulty floor, life - 1, negative resistance): a separate candidate. `tools/emu/skill_damage.py` and the `--skill-damage` mode touch `tools/` and `tests/` (CODEOWNERS); the two fix commits stand without them.

## Screenshots
Not applicable.

## Checklist
- [x] A test in `tests/` (fix or port): `test_game --skill-damage` (the oracle's mode); existing test_combat / test_game unchanged and green
- [x] The finding in `docs/research/re/`: skills.md "Elemental damage" (the gate, MulDiv, the mastery stats), tools/emu/README.md row
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
````

## Commands for the coordinator (NOT RUN; each needs the user's go)

```sh
ssh <the worker host> 'cd ~/d2d-lab/pr/skilldmg && git fetch upstream && git rebase upstream/main fix/skill-damage && python3 tools/pr_guard.py upstream/main && ninja -C build test_combat test_game && ./build/tests/test_combat && D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./build/tests/test_game > /dev/null && echo READY'
ssh <the worker host> 'cd ~/d2d-lab/pr/skilldmg && git push origin fix/skill-damage'
# save the body above (inside the four-backtick fence) to ~/d2d-lab/pr/skilldmg/PR_DESCRIPTION.md (untracked), then:
ssh <the worker host> 'cd ~/d2d-lab/pr/skilldmg && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/skill-damage --title "Element damage: the min takes its synergy only over one point; MulDiv shares" --body-file PR_DESCRIPTION.md'
```
