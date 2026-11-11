# Independent verification of upstream PR candidates 1 and 2 (2026-11-11)

> status: WIP (agent prverify). Verdicts below are filled as the runs finish.

Verifier: d2-reviewer (Opus), singly, read-mostly. Material: `docs/upstream/pr-queue.md` and `docs/upstream/drafts/*.md` on branch `agent/prprep` (tip 49cfd0f1), the private fork on the worker host. Tags: V read (disassembly read here), V run (run here), S / U.

## Setup (V run)
- Upstream main moved since the prep: `962ffce2` (2026-11-11, #414 trace docs and an Imbue include fix), not `b6b59ae9`. Both branches rebase onto it with no conflict (detached worktrees `~/d2d-lab/wt-pv-light`, `~/d2d-lab/wt-pv-cold`, script `~/d2d-lab/pv/pv_setup.sh`): Lightning `9e756c76` (oracle) + `4b80e1e2` (claim); Cold `8c327675` (oracle) + `37bd27ad` (claim).

## Candidate 1: Lightning, one unit a frame, last unit skipped (`fid/missile-last-unit-r2`, 89059e2d)

**Verdict: NEEDS-WORK.** The game.exe rule is right and the proof for the `lightningbolt` row holds, but the d2d change reaches many more player skills than the PR says, some of them by a path the proof does not cover.

### What holds
- V read, FUN_005ae1f0 (0x5ae1f0..0x5ae3fa, disassembled here with upstream `disasm.py`): range − 1 (0x5ae26a..0x5ae294), then for each subtile the move entered (0x648f40 count and list) FUN_0064d9b0, then FUN_00641cb0 through the filter; a unit found goes to FUN_005adf10(unit, 0) and the function returns (0x5ae3cd..0x5ae3e0). One unit a missile a frame: confirmed.
- V read, the missile srvdofunc table at 0x73c768 (read from the loaded image): entry 1 = 0x5b0bc0 = `jmp 0x5ae1f0`; entry 3 = 0x5ae480, which jumps to FUN_005ae1f0 both when the missile moves and (after 0x64cb90) when it stands. So srvdofunc 1 and 3 rows share this rule in game.exe.
- V run, the differential (`~/d2d-lab/pv/pv_light.sh`, rebased tree on 962ffce2, N 400 = 1,192 casts a seed):

| tree | seed 1 | seed 7 | seed 13 |
|---|---|---|---|
| parent (oracle commit), T lines that differ | 44 (53 casts with any H/T difference) | 43 (54) | 47 (54) |
| claim, T lines that differ | 0 | 0 | 0 |

  The prep's 44 / 43 / 0 / 0 reproduce exactly (they counted T lines). Seed 13 is new.
- V run, full `ctest -j8` (D2_MPQ_DIR and the 1.14d installer set) on the claim: 233 of 237 pass or skip; the 4 failures (`test_mpq`, `smoke_d2d`, `quests_act1`, `quests_act4`) fail identically on the parent, so they are this host's data set (no video MPQs), not the change. `test_game_missiles` (the #225 pierce-charge test), `test_game`, `test_monsters`, `test_combat` pass. `test_light_pierce` and `test_next_hit` skip without arguments in ctest (they are oracle modes, as upstream's other oracle tests).
- V run, Bone Spear (a throwaway variant of the test with the skill and row from the environment, never committed; oracle copy with `bonespear` added to its row list): parent 12 T lines / 16 casts differ from game.exe, claim 0 (seed 1). So the rule also fixes Bone Spear, which shares the flags.
- Hygiene: pr_guard 0 FAIL / 0 WARN (V run here on the rebased claim); no private host, path or fork-only code in the diff (V grep); `Workspace/private/diablo2` is upstream's own default. Overlap: open #426 (`missiles.cpp`, a different function) and #405 (`re/missiles.md`, a different section): textual rebase only. No open or merged upstream PR has this fix (read-only `gh` search "lightning", "LastCollide").

### Findings
1. **HIGH: scope is wider than the PR says, and partly unproven.** The new branch fires for every row with LastCollide, no NextHit, no Pierce that reaches `Fight::skill_missile_hits` without an earlier hit-function return. Player skills whose missiles have those flags (V run over Skills.txt / Missiles.txt): Lightning, Lightning Fury (`furylightning`), Inferno (`infernoflame1`), Bone Spear, Arctic Blast (srvdofunc 1, CollideKill 0: the same path as Lightning; Bone Spear verified above), and Blizzard (`blizzard1`), Poison Explosion (`poisonexplosioncloud`), Poison / Plague Javelin clouds (srvdofunc 3, Vel 0: d2d strikes these through `standing_struck`, a different test; upstream's own doc says srvdofunc 1 never strikes at velocity 0, and what 0x64cb90 does for a standing do-3 missile with several units in reach was not run by anyone). The draft and the commit say "in the player's skill set that is `lightningbolt`": false. Blizzard is the Sorceress's main skill. Recommendation: either (a) narrow the condition to what is proven (`missile_info.srv_do == 1`, CollideKill 0 rows; keep `struck` for the standing srvdofunc 3 rows) and name the srvdofunc 1 skills in the body, or (b) keep the scope and extend the proof: parameterise `test_light_pierce` over the srvdofunc 1 skills (cheap, the oracle accepts any row with no hit / damage / ToHit function) and add a multi-unit standing-shard case to `sk_ground_hit.py` with FUN_005adf10 run, not hooked. (a) is the minimal PR.
2. **MEDIUM: the 0-of-1,192 depends on d2d's own unit order.** The oracle replays d2d's contacts in monster-index order and breaks at the first; game.exe walks the subtiles in path order. With the contacts re-ordered along the heading (approximate path order, throwaway variant `PV_PATHORDER=1`), the claim differs from game.exe in 18 T lines (19 casts) on seed 1 and 15 on seed 7; the parent in 37 (53 casts). So the claim still moves d2d closer (37 to 18), but the residual is not "unit order inside one subtile" as the draft's limits say: it is the order across the subtiles of one frame. Recommendation: say so in the body ("which unit of a frame goes first: d2d's monster order, game.exe's path order; N of 1,192 casts differ by that"), or sort the frame's candidates by distance along the flight in `Fight::world` for these rows (then re-run).
3. **LOW: `SK6_REVERSE` / "decomp6" in `light_pierce.py`** (line 132): an env knob and a name from our lab workflow in an upstream oracle. Remove it or describe it neutrally.
4. **LOW: number mismatch.** `docs/research/re/missiles.md` (the claim commit) says "check 2,584 casts"; the README row and the draft run N 400 = 1,192. Make them agree.
5. **LOW: commit message names `FUN_005a8850`** (first commit); the address is the filter entry 0x5a8850, not a function start (the draft correctly says "the filter at 0x5a8850").

## Candidate 2: Cold Enchanted chill length (`fid/cold-enchanted-length-r3`, 1b4036a3)

**Verdict: NEEDS-WORK (one mechanical fix, then PASS).** The claim is right and fully proven for the cases it names; the first commit does not compile on its own.

### What holds
- V read, FUN_005a1c70 (0x5a1c70..0x5a1dee): the unit's level (stat 0xc via FUN_00625480) is taken as at least 1 and at most MonLvl rows − 1 (0x5a1ce5..0x5a1d1c, rows at tables +0xb74); cold min / max (stats 0x36 / 0x37) = MonLvl damage x the MonUMod percent rows (0x1c / 0x1f with the boss flag, 0x10 / 0x13 without); then stat 0x38 += level x 5 + 100 (`lea eax, [eax+eax*4+0x64]`, 0x5a1dcd, through the adder FUN_006272b0). So the length is ADDED, and the 110 cap is the MonLvl table size (111 rows, V run), not an unknown: the draft's "clamp source (U)" is resolved.
- V run, edge levels through game.exe's own FUN_005a2120 (`~/d2d-lab/pv/pv_cold_levels.py`, umod_stats' harness with the base level forced to 0..200): every case equals 100 + 5 x min(max(final level, 1), 110), the final level including the unique's level add (+3); uniques, minions, Cold with Fire before or after, all 60 cases.
- V run, the differential on the rebased tree (`pv_cold.sh`; grouping by my own classifier `coldclass.py`, not the prep's):

| cold_len mismatches | seed 1, 157,618 cases: parent | claim | seed 9, 87,618 cases: parent | claim |
|---|---|---|---|---|
| champion / unique / super unique, Cold Enchanted, no Fire / Lightning Enchanted | 9,636 of 9,636 | 0 | 4,044 of 4,044 | 0 |
| Cold together with Fire or Lightning Enchanted | 3,647 of 3,647 | 1,873 | 1,069 of 1,069 | 566 |
| minions of a Cold Enchanted boss | 5,657 of 5,657 | 5,657 | 2,900 of 2,900 | 2,900 |
| no Cold Enchanted | 0 | 0 | 0 | 0 |

  The remaining rows are the two pre-existing deviations the draft names (d2d keeps one element per boss, the last Enchanted mod; minions carry no mods). The prep's 7,869 differs from my 9,636 by grouping only (it split off multi-mod rows); its claim-side 1,625 + 248 = my 1,873.
- V run, `test_boss_cold_length` (real `make_boss`): fails on the parent (the `dur` assertion), passes on the claim. Full `ctest -j8` on the claim: the same 4 data-set failures as candidate 1's parent and claim (`test_mpq`, `smoke_d2d`, `quests_act1`, `quests_act4`), everything else passes or skips; `test_game`, `test_monsters` pass.
- Hygiene: pr_guard 0 FAIL / 0 WARN (V run); no private or fork-only content (V grep). Overlap: #371 (unique mods 21, 35, 40-42) and #401 (`ai.cpp`) touch neighbouring files with no semantic overlap (`gh pr diff`, no `elem_len` / FUN_005a1c70 lines). No upstream PR or commit has this fix.

### Findings
1. **HIGH (blocks posting as is): the oracle commit `511d907d` does not build.** `tests/monsters/test_umod_stats.cpp` there calls `boss_elem_length`, which only the claim commit adds (`error: 'boss_elem_length' was not declared`, V run on 8c327675, the rebased twin). So "the oracle first, it shows the difference on the parent" is not true of the commits as pushed, and the prep's parent numbers cannot have come from that commit. Fix: in the oracle commit print `boss.elem_len` (what `make_boss` used before); the claim commit switches that line to `boss_elem_length(boss, level)`. With exactly that throwaway edit the parent numbers above are produced. Push as a new branch name (`-r4`), never a force-push.
2. **LOW: the cap is data.** `std::min(level, 110)` in `boss_elem_length`: game.exe caps at MonLvl rows − 1. Either take the row count from the loaded MonLvl table or keep 110 with the comment "MonLvl's last row (FUN_005a1c70 0x5a1cf5..0x5a1d1c)" and drop the "the oracle shows" wording. Also the floor of 1 (level 0 counts as 1) is not modelled; no reachable boss has level 0 (S), say so or add `std::max(level, 1)`.
3. **LOW: the draft's honest limits** should replace "clamp 110 source (U)" with the read above, and say the length is added to whatever 0x38 the unit had (Ghostly's 150 too: d2d's single `elem_len` keeps whichever mod comes last; whether a real boss carries both is U).
