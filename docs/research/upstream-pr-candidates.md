# Upstream PR candidates (agent UPSTREAMGAPS, 2026-10-30)

> status: DRAFTS. Nothing is pushed, no PR, issue or comment exists. Each push and each PR needs the user's own go, asked per item (upstream AGENTS.md section 2). Upstream: psi29a/d2decomped main af825eda; clone of the user's fork on the worker host: `~/d2d-lab/pr/gaps` (remote `upstream` added, fast-forwarded to af825eda, no branch created yet). Author and committer must be `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no Claude trailer (the `Agent:` line goes in the PR body).

At most three candidates, one at a time. Open PRs of ours: #127 (vendor prices). Candidate 1 only; items 2 and 3 found nothing to send.

## Candidate 1 (rank 1): spell damage keeps its 256ths

Branch NOT built (time box). Evidence is complete (docs/research/upstream-gaps/item1-charged-bolt-hit.md); the change is small in idea and invasive in code, so it is left for a session that can run the full presubmit.

Claim (one): game.exe takes a missile's rolled damage off a monster's life in 256ths (record int 0x13, `FUN_0057c6c0`; life stat 6, dead only below 0x100), d2d cuts each elemental roll to whole points (`rules::missile_blow`: `roll >> 8`) and subtracts it from an int `hit_points`, so a Charged Bolt of 2 to 4 deals 2.5 on average, not 3.0.

Proposed change: carry the fraction. `Blow` gets the unrounded 256ths (physical and element, after resistance) and the monster a `life_frac` (0..255) so that life = hit_points x 256 minus life_frac and the monster is dead when that is under 256; `hurt()` takes 256ths; the damage-8 hit-recovery test uses the same value. Test: a Charged Bolt volley whose mean per-bolt damage equals the 256ths mean (game.exe: 3.0 at skill level 1), a monster at 1 life dying to a 0.5 point hit, and the existing monster/fight tests unchanged.

Draft PR body (template):

```
## What changes
d2d cut a spell's rolled damage to whole hit points before taking it off a monster's life (a Charged Bolt of 2-4 dealt 2 or 3). game.exe keeps damage and life in 256ths and only kills below 1 point. d2d now carries the fraction. [TO WRITE after the code: before/after numbers]

## Evidence from game.exe 1.14d
| Address | What it shows |
|---|---|
| `005a89a0` | a missile's damage record: lightning rolled min + rand(max - min) in 256ths into record +0x1c (via `005a8910`, mastery stat 0x14a) |
| `005ad730` | the record is handed to `0057c6c0` unchanged (no rounding) |
| `0057c852` | record int 0x13 (total damage, 256ths) |
| `0057c865`, `0057c874` | life (stat 6, 256ths) below 0x100 after the hit is set to 0: dead |
| `005adf10` | the unit-hit entry: no to-hit for a row with ToHit 0 (Charged Bolt) |

## How to re-check
cd tools/emu && uv run python skill_tips.py 20 0 > g.txt && ../../build/tests/test_game --skill-tips 20 0 | grep '^[0-9]' | diff g.txt -
cp missile_roll.py tools/emu/ && uv run python tools/emu/missile_roll.py 3000      # prints ok
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure

Agent: Claude Code (Claude Sonnet 5.5), working for the user
```

Not done: synergy on the real unit path has no unit-level oracle; hit-recovery rule not traced (listed in the evidence note as U).

pr_guard: not run (no branch yet). Run `python3 tools/pr_guard.py up/main` before the PR.

## Candidates 2 and 3

None. Item 2 (monster life): d2d matches game.exe (`monstats.py` 2202/2202); the live spawn dump and the monster-level-at-spawn check remain to do. Item 3 (spawn counts): d2d's generator equals game.exe per room and per seed; the deficit seen in our lab sweep is not in the generator. A candidate would only exist if a running upstream d2d falls below `drlg-dump` for the same seed (see item3 note).

## Waiting for the user's go (do NOT run until told; each needs its own yes)

```sh
# 1. after the branch exists and pr_guard is clean:
ssh worker.example 'cd ~/d2d-lab/pr/gaps && git push origin fix/missile-damage-fine'
# 2. then, with the filled template as the body:
ssh worker.example 'cd ~/d2d-lab/pr/gaps && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/missile-damage-fine --title "Spell damage keeps its 256ths" --body-file PR_DESCRIPTION.md'
```

## Notes on upstream's rules (nothing was refused to us)

- No instruction in upstream's pages conflicted with ours. Their pr_guard blocks tools/CI/process changes without `security-reviewed`: candidate 1 adds `tools/emu/missile_roll.py` (an oracle, like `cast_rates.py`); the maintainer's CODEOWNERS review applies to `tools/`, say so in the PR.
- Gap-map rows closed: none yet (FUN_005a89a0, `005ad730`, `005adcd0`, `005a8910`, `005ada40` are `untouched` in `functions.tsv`; the candidate's doc section in `docs/research/re/missiles.md` would move them to cited, and verified once the oracle is merged).
