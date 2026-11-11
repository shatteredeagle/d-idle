# Upstream PR queue (d2d fidelity fixes against game.exe 1.14d)

> status: current, 2026-11-11 (card TASK-332, agent prprep). Audit of every local candidate against upstream main `b6b59ae9` (2026-11-11). Nothing in this file has been posted. Standing rules: memory d2-pr-proof-standard (differential against the original code, reproducible commands, independent Opus verifier BEFORE posting), at most 3 open upstream PRs, one subsystem per batch, CodeRabbit inline replies only, never answer humans.

Open upstream PRs of ours today: none (the last ones, #242 load life and mana and #245 respawn refill, merged 2026-11-08; the palimpsest trace PRs are not tracked here). So all 3 slots are free. Open PRs by others that touch our files: #426 (`fight/missiles.cpp`, Holy Bolt heals a pet), #371 (`rules/uniques.hpp`), #401 (`game/ai.cpp`), #301 (`fight/tick.cpp`); rebase on whichever merges first.

## 1. Ranked queue

| Rank | Candidate | Branch (private fork remote) | Tag | Readiness | What is still missing |
|---|---|---|---|---|---|
| 1 | Lightning hits one unit a frame, skips the last unit (FUN_005ae1f0, FUN_005a8730) | `fid/missile-last-unit-r1` head `b7954bd0` | V run: parent 44 of 1,192 casts differ (seed 1; 43 seed 7), with the claim 0 of 1,192 (both seeds); `test_next_hit` neighbour 0 of 598 | READY FOR VERIFIER. Draft: [drafts/lightning-last-unit.md](drafts/lightning-last-unit.md). Rebased onto current main; the old head was narrowed to non-Pierce rows after it broke `test_game_missiles` | verifier; Pierce rows (arrows, javelins, Blessed Hammer) keep the old list, extending the oracle to them is a follow-up; unit order inside one subtile (U) |
| 2 | Cold Enchanted chills for level x 5 + 100 frames (FUN_005a1c70) | `fid/cold-enchanted-length-r2` head `05d5d754` | V run: cold-only cases 7,869 mismatches before, 0 after (157,618 cases); `test_boss_cold_length` (real `make_boss`) fails on the parent, passes with the claim | READY FOR VERIFIER. Draft: [drafts/cold-enchanted-length.md](drafts/cold-enchanted-length.md) | verifier; clamp 110 source (U); two-element uniques and minions differ separately (not this claim) |
| 3 | A saved purse or stash over its cap is 0 on entering a game (FUN_0056b180, FUN_0053ff00) | `local/fid-load-gold-clamp` head `42d40b09` (base 1453f853, merges clean on main) | V run: 363 of 539 differ before, 0 after (2026-11-08, not re-run on current main) | Draft exists: [../research/pr-load-gold-draft.md](../research/pr-load-gold-draft.md). Low value (needs an edited or imported save, U) | rebase on `b6b59ae9`, re-run `fl_gold.py` against `test_game --load-gold`, move the draft here; ask whether the maintainer wants a clamp that zeroes a player's gold |
| 4 | A raised monster gets MonStats2 `ResurrectMode` (S1 for Fallen and Mummy, SQ for skeletons), d2d sets NU (FUN_005ccb10, FUN_005fd7b0) | none (no d2d-side test) | game side V run (703 rows, 0 disagree with the column); d2d side V read (`ai.cpp` around the raise) | NOT READY. Report: `fid-revive-mode.md` of TASK-316 (on its own branch, not merged to main at this audit) | build a `test_game` scenario (a Fallen Shaman, a Fallen corpse in mode 12 within aip4, step until the corpse moves, read `mode`), oracle = `d9_revive.py` `M` lines; effect is about 24 ticks of S1, small |
| 5 | Pierce rows follow the last-unit rule too (arrows, javelins, Blessed Hammer) | none | S (same filter FUN_005a8730); no oracle | NOT READY | extend `light_pierce.py` to rows with ToHit / hit / damage functions; reconcile with #225's pierce charges |
| 6 | Hire list made once per NPC in game.exe, regenerated on every open in d2d (FUN_00578b30 flag at +1, FUN_00576070) | none | V read both (`fid-stock-oracle.md` D1) | NOT READY | an emulator run of FUN_00576070 twice with the flag handling, `OpenHire` twice in `test_game` with a level change between; open question whether the list differs at all (U) |
| 7 | Teleport mana on a refused hop (SKTELE D3 / D4) | none | U (`sk-teleport-trace.md`) | NOT READY | lab drill with the fork verbs (a refused hop and the mana before and after), then an emulator run of FUN_005ca360 |
| 8 | Timed-state put rule: curses, Taunt, Inner Sight, Attract (covstates2 D1 to D5) | none | V run: 54,685 of 100,000 sequences differ | NOT READY; a missing-rule change, needs a new differential on the fix | scope it as one rule at a time; low value for the early Sorceress or Paladin |

## 2. Dropped (checked, no longer a candidate)

| Item | Why |
|---|---|
| Load life and mana, respawn refill, noshield order, Energy Shield order, Blessed Hammer undead share, batch 2 A / B / C (#185, #187, #189), fidelity bundle 1 (#177), Blizzard offsets (#163), curse-immune (#160), freeze rules (#159) | merged upstream |
| `local/smite-block-v1-final`, `local/missilepoison-skillrows`, `local/ground-burst-v1`, `local/skmelee3-d1-final` / `d2-final`, `local/vendor-repair-gate-v4`, `local/nmlevel-v5`, `local/freeze-rules-v5`, `local/monsterpace-b-v2` | their subjects are in upstream main (found by subject in `git log upstream/main`) |
| `fid/missile-last-unit`, `fid/cold-enchanted-length` (old heads, base a5b970d5) | replaced by the `-r1` / `-r2` branches above; `fid/cold-enchanted-length-r1` (98e10ccf) is stale, superseded by `-r2` (rebased, scope fixed, glue test) |
| `local/sorc-lightning-seq`, `fix/lightning-sequence`, `fix/skills-doc-cast`, `fix/cast-ticks`, `local/sorc-bolts`, `local/sorc-mana-timing` | old pre-split branches; upstream now has `rules/sequences.hpp` and SQ cast lengths (`fight/strike.cpp`); whether Lightning's seqtrans length is covered was not re-checked (I) |
| `fix/waypoint-hostile-window` | merged as #153 |

## 3. Bundle plan under the 3-open-PR rule

- Slot A: Lightning (rank 1). One subsystem (player missiles), two commits (oracle, claim). Follow-up PR for the Pierce rows only after the oracle exists.
- Slot B: Cold Enchanted (rank 2). One subsystem (monster mods), two commits.
- Slot C (free): gold clamp (rank 3) once rebased and re-run, or resurrect mode (rank 4) once its test exists; they are different subsystems (world load, monster AI), so each is its own PR. Do not open a third PR before one of A / B merges unless the user says so.
- Order of posting: verifier clears the branch first (d2-reviewer, singly, own harness, not given this file's conclusions), then the user's push go for the batch, then `gh pr create --body-file` from the draft. CodeRabbit threads are answered inline only.
- Each branch is pushed to the private fork remote only (done 2026-11-11: `fid/missile-last-unit-r1`, `fid/cold-enchanted-length-r2`). Posting needs the GitHub fork copy (`ghfork`) pushed and the verifier clearance.

## 4. Reproduce on the worker host

Scripts of this prep (tools/emu, prefix `pp_`): `pp_setup.sh` (worktrees from upstream main plus cherry-picks), `pp_build.sh light|cold TARGET...`, `pp_light.sh before|after SEED N`, `pp_light_all.sh`, `pp_cold.sh before|after SEED N`, `pp_cold_count.py`, `pp_cold_test.sh`, `pp_cold_check.sh`, `pp_cold_guard.sh`, `pp_cold_fix.sh`, `pp_cold_push.sh`. Fork checkout `~/d2d-lab/fork/src`; worktrees `~/d2d-lab/wt-pp-light`, `~/d2d-lab/wt-pp-cold`; data `D2_MPQ_DIR=/tmp/ug-emu` with the 1.14d patch installer.
