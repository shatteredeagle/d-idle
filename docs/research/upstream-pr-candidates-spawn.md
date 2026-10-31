# Upstream PR candidates from the spawn trace (agent DECOMPSPAWN, 2026-10-30)

> status: current. Companion of `docs/research/spawn-density-trace.md`. UPSTREAMGAPS owns `docs/research/upstream-pr-candidates.md` (its item 3 is the density verdict); this file only records what the champion/unique/superunique trace adds. Nothing was pushed, posted or filed.

## Result: no PR candidate

- Density, group sizes, the unique/champion roll (FUN_005be020), the counter (FUN_005a0320) and the superunique spawn's count (FUN_005a49b0 -> FUN_005a09e0 -> FUN_005a0320) all match d2d's `components/game/objgroups.cpp` (`populate()`, `unique()`, `monster()` with `++pop.uniques`) and `components/rules/monsters.hpp`. Spot check of item 3's oracle: `diff_drlg.py 7-7 3 monsters` = 41/41 rooms.
- Upstream's `docs/research/re/monsters.md` already states the counting ("FUN_005a0320 counts a unit as it first sets flag 8 ... a superunique that is also an act boss counts once"), so there is no doc claim to add either. One sentence could be offered later if the maintainer wants it spelled out: "a level's superunique counts toward MonUMin/MonUMax, so Normal Cold Plains (1 / 1) has no random champion or rare once Bishibosh's room is up" (evidence: `tools/emu/spawn_bosses.py 1-10 3`, seed 3). Not worth a PR on its own; could ride on a later monsters.md change.
- Our own probe is what needs the fix: `cmd/d2dspawn` `gap = 30.0` -> 8 (or a walk), see the trace doc section 5. That is a d2lod-bot change, not an upstream one.

## If a later run finds a real difference

Follow `docs/research/cast-trace-gameexe.md` "The three branches": a worktree of `~/d2d-lab/src` under `~/d2d-lab/pr/<name>`, author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no Co-Authored-By, one claim per PR, `PR_DESCRIPTION.md` from their template with the `Agent: Claude Code` line, `python3 tools/pr_guard.py` 0 FAIL / 0 WARN, evidence = the oracle line and the seed. The user is asked before any push.
