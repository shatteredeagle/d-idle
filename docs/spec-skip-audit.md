# Audit of earlier behavioural skip_specs cards

> status: audit of card spec-skip-audit (2026-10-31). Nothing here is a spec; the needs-spec rows are a backlog for later backfill cards.

## Method and limit

The public repo starts at an `Initial public snapshot` (git log has no merge commits for these cards), so `git log --stat` of their merges is not available here. Each verdict is read from the archived card (`openspec/changes/archive/2026-10-30-<id>/`: Why, owns, result line) and from the current code and tests named in the row. Tag in the last column: C = card only, C+code = card plus the code it points to. Rule applied: README "When skip_specs is allowed" (skip is justified for docs-only, tooling and pure refactors, not when code under `internal/**` or `cmd/**` changes a log line, counter, field, decision or timeout).

## Table

| card | verdict | what it changed | capability it would belong to | effort | read |
|---|---|---|---|---|---|
| d2dcombat | needs-spec | new `internal/d2dcombat`, `d2dctl` verbs: casting, belt, targets on d2d (result: 51 kills 0 deaths live) | backend-contract (combat verbs, delta MODIFIED/ADDED) | M | C+code |
| d2dtown | needs-spec | new `internal/d2dtown`, town NPC/vendor/belt refill over d2d verbs (Akara trade live, rest U) | backend-contract (town verbs) | M | C+code |
| d2dcast | skip-justified | docs only (`docs/d2d-upstream-cast.md`), superseded by casttrace and PR 128 | none | none | C |
| d2dlevel | needs-spec | `internal/bot`: Den floor no longer waived far below the floor (decision change, shared bug) | early-game-policy (MODIFIED "Den of Evil first": floor rule) | S | C+code |
| d2dprice | skip-justified | docs only, PR 127 draft; no code of ours | none (upstream-pr covers the posting) | none | C |
| d2dverbs | skip-justified | docs only; verbs live in the private fork, documented in `docs/d2d-lab.md` | none (fork is outside this repo) | none | C |
| equiplive | needs-spec | `internal/town`: per-click log lines and skip-until-level-up for an item the game refuses (decision, log) | early-game-policy ("no armor rebuy loops") or new town-gear | S | C+code |
| invrejuv | skip-justified | inventory logging and the cap, superseded in part by rejuvuse; the cap is now in early-game-policy | early-game-policy (already) | none | C+spec |
| rejuvuse | needs-spec | `internal/resource`, `loot`: drink rejuvenation at the boss threshold; the cap is in the spec, the drink rule is not | combat-loop or new potion-use | S | C+spec |
| npcvisit | skip-justified | `internal/town`: fewer NPC opens and panel closes; efficiency only, no contract number (live effect U) | none | none | C |
| mercwire | skip-justified | wires the already specified merc-policy behind a knob, default off; the policy is in merc-policy, the knob in docs/knobs.md | merc-policy (already) | none | C+spec |
| hitrec | skip-justified | docs and `tools/emu` for upstream PR 134; no bot code | upstream-pr (already) | none | C |
| iowait | skip-justified | docs only, worker host measurements | none | none | C |
| d2dexit | needs-spec | `d2denv`, `d2dsrc`, `d2dworld`: exit to Blood Moor is now found (was a hang waiting for a link) | backend-contract (world exits) | S | C+code |

## Counts

- skip-justified: 8 (d2dcast, d2dprice, d2dverbs, invrejuv, npcvisit, mercwire, hitrec, iowait).
- needs-spec: 6 (d2dcombat, d2dtown, d2dlevel, equiplive, rejuvuse, d2dexit).
- Effort if backfilled: 2 M + 4 S; most fit a MODIFIED or ADDED delta to backend-contract (3 cards) and early-game-policy (2 cards), one new small capability for potion use.

## Doubtful calls

- npcvisit and mercwire are borderline: both change decisions in shared code, but no number or output a consumer relies on is named in the card. They flip to needs-spec if a later reader finds a logged line or counter in their diffs (not checkable here).
- d2dcast, d2dprice, hitrec: their upstream PR claims are covered by the upstream-pr capability and the evidence packs, not by a bot spec.

## Not done

No spec was written for these cards; the backfill needs the merge diffs from the original repo (the ops checkout), which this public clone lacks.
