# Evidence pack: fix/skill-damage (for the user's go / no-go)

> status: current (agent DECOMPSKILL, 2026-10-31). One page. Detail: docs/research/pr-skill-damage-proof.md; body: docs/research/pr-skill-damage-draft.md; trace: docs/research/skill-damage-trace.md; standard: docs/process/upstream-pr-standard.md. posted:

**Claims.** (1) A skill's min element damage takes its synergy share only when the base min is over one point or the row has an EMinLev1 (game.exe FUN_00644d50; the max never gates): the "1-x" lightning skills keep min 1. (2) The synergy and mastery shares are MulDiv (divide-first above 0x100000).

**Proof summary.** game.exe's own FUN_00644d50 / FUN_00644e40 / FUN_00644f20 (mastery flag) run in upstream's emulator on the loaded 1.14d tables (`tools/emu/skill_damage.py`, in the branch; hooks: the stat getter, the unit's skill record lookup, its level) against d2d's real `elem_damage` / `elem_length` (`test_game --skill-damage`), same cases, joined by `diff`. Every link in the chain read in the disassembly (V); the data columns read from the loaded tables (D).

**Numbers.**

| Scenario | Cases | Mismatches before | after claim 1 | after both |
|---|---|---|---|---|
| Grid: every elemental skill x levels 1..40 x {alone, neighbours at 20, + masteries 100, + masteries 37 at clvl 99} | 16,000 | in the total | | 0 |
| Random: levels 1..60 (bracket edges), clvl 1..99, masteries 0..600, 0..6 other skills at 0..40, seed 1 | 100,000 | in the total | | 0 |
| Total | 116,000 | 4,549 (min 4,509 / max 340 / len 0) | 344 | 0 (md5 equal) |
| Verifier's own runs (seed 77: 36,000; adversarial: 104,000) | 140,000 | | | 0 |

Their checks: `ninja test_combat test_game` (D2D_WERROR off), `test_combat` ok, `test_game` rc 0 (27 OK, MPQs + LODPatch_114d.exe), `pr_guard.py upstream/main` 0 FAIL 0 WARN, `presubmit.py --quick` 0 FAIL 4 WARN (ruff absent; siblings and doc-claims answered in the body; build by hand). Three commits authored by the user, bodies one line `Agent: Claude Code`, no trailer.

**Commands (reviewer re-runs on their copy).**

```sh
python3 tools/pr_guard.py && cmake --build build && ctest --test-dir build --output-on-failure
cd tools/emu && uv run python skill_damage.py gen 100000 1 > cases.tsv && uv run python skill_damage.py game cases.tsv > g.txt
../../build/tests/test_game --skill-damage cases.tsv | grep -E '^[0-9]+ ' | diff g.txt -     # empty
```

**What the bot gains.** Nothing for the Blizzard plan's clvl 1-25 skills (Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova: the formula already matched); the lightning plan's Lightning / Chain Lightning lose the about 2 % extra average damage d2d gave them. The value is the closed gap: the skill damage formula is now verified end to end on the unit path (gap rank 4 of the riskmap moves from U to V).

**Residual risks.**
- One level per skill in the cases (base = with items); the `blvl` / `lvl` split is matched by reading.
- The oracle's unit is a zeroed block; calc operands reading other unit fields see 0 on both sides (none in the elemental skills' synergy calcs).
- Synergy skills in the cases are class neighbours (id +-15); no calc references another class.
- `elem_length` / `skill_phys` keep plain integer shares (0 mismatches on lengths; the physical part is outside the claims).
- Static Field (FUN_005c96a0) differs from d2d by reading (difficulty floor, at most life - 1, negative resistance neutralised, 256ths): a separate candidate with its own oracle, not in this PR.
- `tools/emu/skill_damage.py` and the test mode are a `tools/` / `tests/` change (CODEOWNERS); the fix commits stand without them.
- The branch is on upstream/main f8aab0e8 (today's tip); fetch again right before the push.

**Verifier.** Independent second agent (Fable, read-only): **VERDICT: holds (with limits noted).** Fresh seed 36,000 cases 0 diffs; its own adversarial set of 104,000 cases (levels -7..5000, clvl 0..120, masteries -200..700000 through every MulDiv branch, synergy pairs over ALL 357 skill ids) 0 diffs; the author's counts reproduced; the gate, MulDiv, the flag-1 calls and the missile's stat set re-read in the listings. No mismatch. Limits it adds: EType 12 (freeze) takes the cold mastery in game.exe (no stock row has it; d2d has no freeze bucket); the `blvl` / `lvl` split and the level cap are not exercised by the cases; other callers of `elem_damage` take the gate (right for every FUN_00644d50 caller, their game paths not traced). Full report in the proof doc.

**PROOF: complete** for the two claims as stated (differential zero over 116,000 author cases and 140,000 verifier cases, chain fully V/D, their checks green, verifier report attached), with the residual risks above disclosed in the body.****

**Recommendation.** Go, with the user's go for each step: (1) fetch, rebase if main moved, rebuild, rerun the commands (expect READY); (2) push `fix/skill-damage`; (3) open the PR with the draft body (a bundle of two claims in one PR, as the maintainer prefers). If the verifier's verdict is "broken", stop and send the finding back to a worker first.

## Posted log

(empty: nothing pushed or posted)
