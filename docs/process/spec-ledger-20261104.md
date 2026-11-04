# Spec ledger, 2026-11-04

Report-only. Nothing under `openspec/specs/` was edited. A second job applies edits from this ledger.

Tags: V = read in the repo (file:line cited), S = secondary, U = unverified.

## How to re-run (V)

```
python3 -I tools/spec/ledger_scan.py > /tmp/ledger.md
```

The script (stdlib, read-only) parses the 17 base specs, every delta under `openspec/changes/archive/*/specs/*/spec.md` (62 ADDED, 16 MODIFIED, 1 REMOVED headers; no RENAMED) and the 10 open-card deltas. Verdict rule: a delta body is compared (difflib ratio on normalised text) with the base requirement of the same name; ratio >= 0.97 is APPLIED; a lower ratio with a later delta of the same name whose text equals the base is SUPERSEDED; a name missing from every base spec is NOT APPLIED-IN FORCE (archived card) or NOT APPLIED-OPEN (open card). Rows that a person read carry `HAND:` in the evidence column (table `OVERRIDES` in the script); a verdict ending in `?` is only the heuristic. The section "Base-spec flags" and the numbers list at the end of the script output are the STALE / DEAD REFERENCE / DUPLICATE / NUMBER material.

Notes on the inventory (V):
- 70 archived delta files (62 changes). Archived change dirs: 656 (status archived) in `openspec/changes/archive/`; archive stubs only matter for dead references.
- `d2d-default-backend` is still `openspec/changes/d2d-default-backend/` with `status: proposed` and no `merge:` in this worktree (tasks.md). The brief says it was archived today; that is not visible here (U). Its delta is treated as OPEN below.
- Capabilities with deltas but NO base spec: `act1-campaign` (archived `2026-11-04-m3-act1chain` ADDED 2, open m3-act1run/m3-andariel ADDED 2), `primitives` (open m1-liveproof), `d2d-mesh` (open d2dmesh), `stream-scene` (open stream-grid). A base spec for `act1-campaign` is owed when the archived delta is folded in.
- No REMOVED or RENAMED delta changes a base requirement that is still present (the single REMOVED row is APPLIED: its target is gone from the base).
- Hand verdicts that rest on a quoted comparison: soak-gates (Gate 1 deaths/belt, Resource slopes, Gate 2/3, G2 meaning), fidelity-gate (Verdict classes, Twenty percent tolerance), merc-policy Route guard, backend-contract acceptance target. Other NOT APPLIED-IN FORCE rows are V only for "the requirement name is in no base spec" (script grep over all 17 specs).

Precedence used for SUPERSEDED: ADR 0015 and the "Current truth" block of `docs/decisions.md` (verified against ADR 0015 on 2026-11-04) win over older text. Current truth: Gate 0 to 3 = d2d ladder, Gate A = game.exe acceptance, G1/G2 retired names (ADR 0013), M3 = G-B, d2d container / stream-grid / d2dmesh deferred or parked.

## Summary counts per capability and verdict

- act1-campaign: NOT APPLIED-IN FORCE=2, NOT APPLIED-OPEN=2
- backend-contract: APPLIED=27, NOT APPLIED-IN FORCE=7, SUPERSEDED=3, UNSURE=1
- bot-loop: APPLIED=6
- combat-loop: APPLIED=9
- d2d-farm: APPLIED=8
- d2d-mesh: NOT APPLIED-OPEN=13
- dashboard-pub: APPLIED=3, NOT APPLIED-IN FORCE=2
- early-game-policy: APPLIED=11, NOT APPLIED-OPEN=2, SUPERSEDED=1
- fidelity-gate: APPLIED=5, CONFLICT=2, NOT APPLIED-IN FORCE=7, SUPERSEDED=1, UNSURE=3
- lab-rules: APPLIED=3, NOT APPLIED-IN FORCE=3
- licence-policy: APPLIED=7
- merc-policy: APPLIED=2, NOT APPLIED-IN FORCE=9, NOT APPLIED-OPEN=4, UNSURE=1
- primitives: NOT APPLIED-OPEN=3
- process-hygiene: APPLIED=9
- public-split: APPLIED=10
- replay: APPLIED=7
- soak-gates: APPLIED=21, CONFLICT=1, NOT APPLIED-IN FORCE=10, SUPERSEDED=3
- stream-scene: NOT APPLIED-OPEN=16
- town-basics: APPLIED=17, NOT APPLIED-IN FORCE=6
- upstream-pr: APPLIED=14, SUPERSEDED=2
- TOTAL: APPLIED=159, CONFLICT=3, NOT APPLIED-IN FORCE=46, NOT APPLIED-OPEN=40, SUPERSEDED=10, UNSURE=5

A verdict ending in `?` is a heuristic not hand-checked. Rows with `HAND:` in the evidence were read by a person.

### act1-campaign

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-11-04-m3-act1chain | archived | ADDED | Act 1 quest order | NOT APPLIED-IN FORCE(no base cap) | openspec/changes/archive/2026-11-04-m3-act1chain/specs/act1-campaign/spec.md:8 |
| 2 | 2026-11-04-m3-act1chain | archived | ADDED | Quest evidence on d2d | NOT APPLIED-IN FORCE(no base cap) | openspec/changes/archive/2026-11-04-m3-act1chain/specs/act1-campaign/spec.md:15 |
| 3 | m3-act1run | open:deferred | ADDED | Unattended Act 1 clear | NOT APPLIED-OPEN | openspec/changes/m3-act1run/specs/act1-campaign/spec.md:8 |
| 4 | m3-andariel | open:deferred | ADDED | Andariel fight rules | NOT APPLIED-OPEN | openspec/changes/m3-andariel/specs/act1-campaign/spec.md:8; HAND: card deferred; Current truth maps M3 to goal G-B (still wanted) |

### backend-contract

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-enterportal | archived | ADDED | The d2d backend enters a town portal with the client's portal command | APPLIED | openspec/changes/archive/2026-10-31-enterportal/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:236 ratio 1.00 |
| 2 | 2026-10-31-fix-missile | archived | ADDED | The d2d state source lists missiles | SUPERSEDED(by 2026-10-31-fix-missile2) | openspec/changes/archive/2026-10-31-fix-missile/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:283 ratio 0.66; base vs later ratio 1.00; HAND: same requirement re-added by fix-missile2 |
| 3 | 2026-10-31-fix-missile2 | archived | MODIFIED | The d2d state source lists missiles | APPLIED | openspec/changes/archive/2026-10-31-fix-missile2/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:283 ratio 1.00 |
| 4 | 2026-10-31-fix-nav | archived | ADDED | A bounced step away from a cave mouth changes direction | APPLIED | openspec/changes/archive/2026-10-31-fix-nav/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:265 ratio 1.00 |
| 5 | 2026-10-31-fix-nav | archived | ADDED | Touched waypoints reach the planner | APPLIED | openspec/changes/archive/2026-10-31-fix-nav/specs/backend-contract/spec.md:16; base openspec/specs/backend-contract/spec.md:276 ratio 1.00 |
| 6 | 2026-10-31-fix-obj | archived | ADDED | Field objects are listed and operable on d2d | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-fix-obj/specs/backend-contract/spec.md:5 |
| 7 | 2026-10-31-fix-obj | archived | ADDED | Waits step the game under lockstep | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-fix-obj/specs/backend-contract/spec.md:16 |
| 8 | 2026-10-31-fix-portal | archived | ADDED | The portal verb picks a source with charges | APPLIED | openspec/changes/archive/2026-10-31-fix-portal/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:218 ratio 1.00 |
| 9 | 2026-10-31-fix-portal | archived | ADDED | HasTP counts charges | APPLIED | openspec/changes/archive/2026-10-31-fix-portal/specs/backend-contract/spec.md:16; base openspec/specs/backend-contract/spec.md:229 ratio 1.00 |
| 10 | 2026-10-31-fix-portal2 | archived | ADDED | The d2d portal verb never clicks with a dead context | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-fix-portal2/specs/backend-contract/spec.md:5 |
| 11 | 2026-10-31-fix-portal2 | archived | ADDED | The d2d portal verb clicks only when the player is idle | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-fix-portal2/specs/backend-contract/spec.md:12 |
| 12 | 2026-10-31-fix-portal2 | archived | ADDED | The d2d portal verb succeeds only on a portal object | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-fix-portal2/specs/backend-contract/spec.md:23 |
| 13 | 2026-10-31-fix-wp | archived | ADDED | The waypoint unit is found on d2d | APPLIED | openspec/changes/archive/2026-10-31-fix-wp/specs/backend-contract/spec.md:5; base openspec/specs/backend-contract/spec.md:247 ratio 1.00 |
| 14 | 2026-10-31-fix-wp | archived | ADDED | Waypoint rows are clicked at their panel centres | APPLIED | openspec/changes/archive/2026-10-31-fix-wp/specs/backend-contract/spec.md:16; base openspec/specs/backend-contract/spec.md:258 ratio 1.00 |
| 15 | 2026-10-31-spec-backfill-d2dcombat | archived | ADDED | d2d belt keys drink by belt column | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dcombat/specs/backend-contract/spec.md:3; base openspec/specs/backend-contract/spec.md:57 ratio 1.00 |
| 16 | 2026-10-31-spec-backfill-d2dcombat | archived | ADDED | A d2d cast targets the monster at the cell | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dcombat/specs/backend-contract/spec.md:10; base openspec/specs/backend-contract/spec.md:64 ratio 1.00 |
| 17 | 2026-10-31-spec-backfill-d2dcombat | archived | ADDED | Combat skills are bound as pseudo hotkeys on d2d | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dcombat/specs/backend-contract/spec.md:25; base openspec/specs/backend-contract/spec.md:79 ratio 1.00 |
| 18 | 2026-10-31-spec-backfill-d2dcombat | archived | ADDED | The d2d snapshot gets a viewport for combat | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dcombat/specs/backend-contract/spec.md:32; base openspec/specs/backend-contract/spec.md:86 ratio 1.00 |
| 19 | 2026-10-31-spec-backfill-d2dcombat | archived | ADDED | The combat clock reads state after every sleep | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dcombat/specs/backend-contract/spec.md:39; base openspec/specs/backend-contract/spec.md:93 ratio 1.00 |
| 20 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | The d2d link wait outlives the verb timeout | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:3; base openspec/specs/backend-contract/spec.md:168 ratio 1.00 |
| 21 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | The d2d backend waits for links by default | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:10; base openspec/specs/backend-contract/spec.md:175 ratio 1.00 |
| 22 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | A missing layout verb is a warning | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:17; base openspec/specs/backend-contract/spec.md:182 ratio 1.00 |
| 23 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | Arrival diagnostics are logged | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:24; base openspec/specs/backend-contract/spec.md:189 ratio 1.00 |
| 24 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | The liveness beat grows only with game progress | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:31; base openspec/specs/backend-contract/spec.md:196 ratio 1.00 |
| 25 | 2026-10-31-spec-backfill-d2dexit | archived | ADDED | Unknown d2d fields are one log line per snapshot | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dexit/specs/backend-contract/spec.md:46; base openspec/specs/backend-contract/spec.md:211 ratio 1.00 |
| 26 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | The d2d town source offers every seam the town reads | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:3; base openspec/specs/backend-contract/spec.md:104 ratio 1.00 |
| 27 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Vendor stock becomes stored item units | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:10; base openspec/specs/backend-contract/spec.md:111 ratio 1.00 |
| 28 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Item owner is the vendor NPC or the player | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:17; base openspec/specs/backend-contract/spec.md:118 ratio 1.00 |
| 29 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Item stats are synthesised from the dump extras | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:24; base openspec/specs/backend-contract/spec.md:125 ratio 1.00 |
| 30 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Dialog entries come from the menu verb | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:35; base openspec/specs/backend-contract/spec.md:136 ratio 1.00 |
| 31 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Named objects get the class the town searches for | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:42; base openspec/specs/backend-contract/spec.md:143 ratio 1.00 |
| 32 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Clicks on d2d hover first | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:49; base openspec/specs/backend-contract/spec.md:150 ratio 1.00 |
| 33 | 2026-10-31-spec-backfill-d2dtown | archived | ADDED | Town interactions walk to the target | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dtown/specs/backend-contract/spec.md:60; base openspec/specs/backend-contract/spec.md:161 ratio 1.00 |
| 34 | 2026-10-31-waits-audit | archived | ADDED | Deadlines of game waits are game time under lockstep | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-waits-audit/specs/backend-contract/spec.md:5 |
| 35 | 2026-10-31-waits-audit | archived | ADDED | One call installs the lockstep-aware waits | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-10-31-waits-audit/specs/backend-contract/spec.md:16 |
| 36 | d2d-default-backend | open:proposed | MODIFIED | Original game remains the acceptance target | SUPERSEDED(by ADR 0015) | openspec/changes/d2d-default-backend/specs/backend-contract/spec.md:3; base openspec/specs/backend-contract/spec.md:50 ratio 0.45; HAND: delta says original-game confirmation only for replicated-soak metrics; ADR 0015 makes game.exe Gate A the acceptance target; base backend-contract:50 already matches ADR 0015 better |
| 37 | d2d-default-backend | open:proposed | ADDED | Game.exe is frozen | SUPERSEDED(by ADR 0015) | openspec/changes/d2d-default-backend/specs/backend-contract/spec.md:16; HAND: delta: no Game.exe work until the d2d container 24 h soak; ADR 0015 items 4 and 5, docs/decisions.md Current truth lift the freeze for Gate A fixes and defer the container |
| 38 | d2d-default-backend | open:proposed | ADDED | Leaving d2d is break glass | UNSURE | openspec/changes/d2d-default-backend/specs/backend-contract/spec.md:23; HAND: ADR 0015 amends ADR 0009 items 1 and 5 only; index line says exit stays break glass; the 16-agent-hour trigger is not restated in Current truth |

### bot-loop

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-kitgate | archived | ADDED | Recovery without the kit stays near town | APPLIED | openspec/changes/archive/2026-10-31-kitgate/specs/bot-loop/spec.md:3; base openspec/specs/bot-loop/spec.md:53 ratio 1.00 |
| 2 | 2026-10-31-nowall | archived | ADDED | Campaign stall escalation | APPLIED | openspec/changes/archive/2026-10-31-nowall/specs/bot-loop/spec.md:3; base openspec/specs/bot-loop/spec.md:68 ratio 1.00 |
| 3 | 2026-10-31-spec-backfill-botloop | archived | ADDED | Restarts counts every bot-started game | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-botloop/specs/bot-loop/spec.md:8; base openspec/specs/bot-loop/spec.md:8 ratio 1.00 |
| 4 | 2026-10-31-spec-backfill-botloop | archived | ADDED | Per-game memory is forgotten on every new game | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-botloop/specs/bot-loop/spec.md:27; base openspec/specs/bot-loop/spec.md:27 ratio 1.00 |
| 5 | 2026-10-31-spec-backfill-botloop | archived | ADDED | Monster classification by flag bits | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-botloop/specs/bot-loop/spec.md:38; base openspec/specs/bot-loop/spec.md:38 ratio 1.00 |
| 6 | 2026-10-31-townwedge | archived | ADDED | A failed walk does not condemn a route for good | APPLIED | openspec/changes/archive/2026-10-31-townwedge/specs/bot-loop/spec.md:3; base openspec/specs/bot-loop/spec.md:83 ratio 1.00 |

### combat-loop

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-combatspin | archived | ADDED | A skipped move costs one poll tick | APPLIED | openspec/changes/archive/2026-10-31-combatspin/specs/combat-loop/spec.md:3; base openspec/specs/combat-loop/spec.md:97 ratio 1.00 |
| 2 | 2026-10-31-fix-missile | archived | ADDED | The cast gap covers the cast animation | APPLIED | openspec/changes/archive/2026-10-31-fix-missile/specs/combat-loop/spec.md:5; base openspec/specs/combat-loop/spec.md:108 ratio 1.00 |
| 3 | 2026-10-31-spec-backfill-combat | archived | ADDED | A failed walk costs one poll tick | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:8; base openspec/specs/combat-loop/spec.md:8 ratio 1.00 |
| 4 | 2026-10-31-spec-backfill-combat | archived | ADDED | No retreat while pinned | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:19; base openspec/specs/combat-loop/spec.md:19 ratio 1.00 |
| 5 | 2026-10-31-spec-backfill-combat | archived | ADDED | Fight-loop no-progress watchdog | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:30; base openspec/specs/combat-loop/spec.md:30 ratio 1.00 |
| 6 | 2026-10-31-spec-backfill-combat | archived | ADDED | Champion group is avoided at low level | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:41; base openspec/specs/combat-loop/spec.md:41 ratio 1.00 |
| 7 | 2026-10-31-spec-backfill-combat | archived | ADDED | Champion caution raises the potion lines | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:52; base openspec/specs/combat-loop/spec.md:52 ratio 1.00 |
| 8 | 2026-10-31-spec-backfill-combat | archived | ADDED | Champion death is labelled as a champion | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-combat/specs/combat-loop/spec.md:67; base openspec/specs/combat-loop/spec.md:67 ratio 1.00 |
| 9 | 2026-10-31-spec-backfill-rejuvuse | archived | ADDED | Rejuvenation is drunk at a higher line against a boss pack | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-rejuvuse/specs/combat-loop/spec.md:3; base openspec/specs/combat-loop/spec.md:82 ratio 1.00 |

### d2d-farm

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-d2dfarm | archived | ADDED | Node registration and labels | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:3; base openspec/specs/d2d-farm/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-d2dfarm | archived | ADDED | Capacity probe and admission | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:14; base openspec/specs/d2d-farm/spec.md:19 ratio 0.99 |
| 3 | 2026-10-30-d2dfarm | archived | ADDED | Pause and resume | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:25; base openspec/specs/d2d-farm/spec.md:30 ratio 1.00 |
| 4 | 2026-10-30-d2dfarm | archived | ADDED | Queue and dispatch by label | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:32; base openspec/specs/d2d-farm/spec.md:37 ratio 1.00 |
| 5 | 2026-10-30-d2dfarm | archived | ADDED | Result collection and one verdict | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:43; base openspec/specs/d2d-farm/spec.md:48 ratio 1.00 |
| 6 | 2026-10-30-d2dfarm | archived | ADDED | Idle-only nodes | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:50; base openspec/specs/d2d-farm/spec.md:55 ratio 1.00 |
| 7 | 2026-10-30-d2dfarm | archived | ADDED | Safety | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:61; base openspec/specs/d2d-farm/spec.md:66 ratio 1.00 |
| 8 | 2026-10-30-d2dfarm | archived | ADDED | Cross-host determinism as qualification | APPLIED | openspec/changes/archive/2026-10-30-d2dfarm/specs/d2d-farm/spec.md:68; base openspec/specs/d2d-farm/spec.md:73 ratio 1.00 |

### d2d-mesh

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | d2dmesh | open:deferred | ADDED | Join with a token | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:3 |
| 2 | d2dmesh | open:deferred | ADDED | Membership convergence | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:14 |
| 3 | d2dmesh | open:deferred | ADDED | Node record fields | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:21 |
| 4 | d2dmesh | open:deferred | ADDED | Any-node job management | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:28 |
| 5 | d2dmesh | open:deferred | ADDED | Job leases | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:35 |
| 6 | d2dmesh | open:deferred | ADDED | Partition tolerance | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:46 |
| 7 | d2dmesh | open:deferred | ADDED | Authentication and encryption | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:53 |
| 8 | d2dmesh | open:deferred | ADDED | One listening port per node | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:60 |
| 9 | d2dmesh | open:deferred | ADDED | Admission rules | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:67 |
| 10 | d2dmesh | open:deferred | ADDED | Safety of what a node runs | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:78 |
| 11 | d2dmesh | open:deferred | ADDED | Result collection | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:89 |
| 12 | d2dmesh | open:deferred | ADDED | Public-repo hygiene | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:96 |
| 13 | d2dmesh | open:deferred | ADDED | Version skew tolerance | NOT APPLIED-OPEN | openspec/changes/d2dmesh/specs/d2d-mesh/spec.md:103 |

### dashboard-pub

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-spec-backfill-viewer | archived | ADDED | Pub backend field follows the badge | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-viewer/specs/dashboard-pub/spec.md:8; base openspec/specs/dashboard-pub/spec.md:8 ratio 1.00 |
| 2 | 2026-10-31-spec-backfill-viewer | archived | ADDED | Pub engine field and status row | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-viewer/specs/dashboard-pub/spec.md:19; base openspec/specs/dashboard-pub/spec.md:19 ratio 1.00 |
| 3 | 2026-10-31-spec-backfill-viewer | archived | ADDED | Stream image reconnects | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-viewer/specs/dashboard-pub/spec.md:30; base openspec/specs/dashboard-pub/spec.md:30 ratio 1.00 |
| 4 | 2026-11-04-mapitem | archived | ADDED | The stream map is contiguous across levels the game draws together | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-mapitem/specs/dashboard-pub/spec.md:5 |
| 5 | 2026-11-04-mapitem | archived | ADDED | A crossing does not blank or rebuild the map | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-mapitem/specs/dashboard-pub/spec.md:20 |

### early-game-policy

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-basicsfix | archived | ADDED | Identify scrolls only for likely valuables | APPLIED | openspec/changes/archive/2026-10-31-basicsfix/specs/early-game-policy/spec.md:3; base openspec/specs/early-game-policy/spec.md:50 ratio 1.00 |
| 2 | 2026-10-31-basicsfix | archived | ADDED | A failed repair is not retried until something changes | APPLIED | openspec/changes/archive/2026-10-31-basicsfix/specs/early-game-policy/spec.md:18; base openspec/specs/early-game-policy/spec.md:65 ratio 1.00 |
| 3 | 2026-10-31-basicsfix | archived | ADDED | Bank the surplus before the shop | APPLIED | openspec/changes/archive/2026-10-31-basicsfix/specs/early-game-policy/spec.md:25; base openspec/specs/early-game-policy/spec.md:72 ratio 1.00 |
| 4 | 2026-10-31-basicsfix | archived | ADDED | Potions before the first town portal scroll | SUPERSEDED(by 2026-10-31-m1-audittests) | openspec/changes/archive/2026-10-31-basicsfix/specs/early-game-policy/spec.md:36; base openspec/specs/early-game-policy/spec.md:83 ratio 0.74; base vs later ratio 1.00; HAND: later delta, base early-game-policy:83 |
| 5 | 2026-10-31-m1-audittests | archived | MODIFIED | Potions before the first town portal scroll | APPLIED | openspec/changes/archive/2026-10-31-m1-audittests/specs/early-game-policy/spec.md:5; base openspec/specs/early-game-policy/spec.md:83 ratio 1.00 |
| 6 | 2026-10-31-spec-backfill-d2dlevel | archived | ADDED | The Den floor does not yield far below the floor | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dlevel/specs/early-game-policy/spec.md:3; base openspec/specs/early-game-policy/spec.md:106 ratio 1.00 |
| 7 | 2026-10-31-spec-backfill-d2dlevel | archived | ADDED | The Den floor yields only one level below with a kit | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-d2dlevel/specs/early-game-policy/spec.md:10; base openspec/specs/early-game-policy/spec.md:113 ratio 1.00 |
| 8 | 2026-10-31-spec-backfill-equiplive | archived | ADDED | Every equip slot click is logged | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-equiplive/specs/early-game-policy/spec.md:3; base openspec/specs/early-game-policy/spec.md:124 ratio 1.00 |
| 9 | 2026-10-31-spec-backfill-equiplive | archived | ADDED | A refused item is skipped until the character changes | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-equiplive/specs/early-game-policy/spec.md:10; base openspec/specs/early-game-policy/spec.md:131 ratio 1.00 |
| 10 | 2026-10-31-spec-backfill-equiplive | archived | ADDED | A failed equip returns the item and says why | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-equiplive/specs/early-game-policy/spec.md:21; base openspec/specs/early-game-policy/spec.md:142 ratio 1.00 |
| 11 | 2026-10-31-spec-backfill-rejuvuse | archived | ADDED | Full Rejuvenation is preferred at the cap | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-rejuvuse/specs/early-game-policy/spec.md:3; base openspec/specs/early-game-policy/spec.md:149 ratio 1.00 |
| 12 | 2026-10-31-spec-backfill-rejuvuse | archived | ADDED | A pack rejuvenation is belted without a rejuvenation column | APPLIED | openspec/changes/archive/2026-10-31-spec-backfill-rejuvuse/specs/early-game-policy/spec.md:14; base openspec/specs/early-game-policy/spec.md:160 ratio 1.00 |
| 13 | m2-roam | open:deferred | ADDED | Leave an empty area | NOT APPLIED-OPEN | openspec/changes/m2-roam/specs/early-game-policy/spec.md:3 |
| 14 | m2-tripwp | open:deferred | ADDED | Leave town by waypoint | NOT APPLIED-OPEN | openspec/changes/m2-tripwp/specs/early-game-policy/spec.md:3 |

### fidelity-gate

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-gatedelta | archived | MODIFIED | Twenty percent tolerance | SUPERSEDED(by 2026-11-02-abtool) | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:3; base openspec/specs/fidelity-gate/spec.md:8 ratio 0.52; base vs later ratio 0.83; HAND: abtool restores deaths/h and deaths/1000 kills to the relative tolerance that gatedelta removed |
| 2 | 2026-10-30-gatedelta | archived | ADDED | Death budget | CONFLICT | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:12; HAND: gatedelta: at most 1 death per 6 h exposure (user 2026-10-30); base death-rule spec and ADR 0005 judge deaths by streak, recovery share and rate; abtool lists deaths/h as relative-tolerance metric again |
| 3 | 2026-10-30-gatedelta | archived | ADDED | Death rate cap and Poisson bound | CONFLICT | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:23; HAND: gatedelta: lower bound of death rate <= 0.15/h plus Poisson test; same disagreement as Death budget; base death-rule/Death rate (FAIL-3) is the live rule |
| 4 | 2026-10-30-gatedelta | archived | ADDED | Proxy metrics | UNSURE | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:30; HAND: text not hand-read; gatedelta belongs to the same retired gate v2 as Death budget |
| 5 | 2026-10-30-gatedelta | archived | ADDED | Proxy definitions | UNSURE | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:37; HAND: as Proxy metrics |
| 6 | 2026-10-30-gatedelta | archived | ADDED | Missing proxy data | UNSURE | openspec/changes/archive/2026-10-30-gatedelta/specs/fidelity-gate/spec.md:44; HAND: as Proxy metrics |
| 7 | 2026-10-31-m2-abdesign | archived | ADDED | A/B protocol for bot changes | APPLIED | openspec/changes/archive/2026-10-31-m2-abdesign/specs/fidelity-gate/spec.md:3; base openspec/specs/fidelity-gate/spec.md:79 ratio 1.00 |
| 8 | 2026-10-31-m2-abdesign | archived | ADDED | A/B sample size and interleaving | APPLIED | openspec/changes/archive/2026-10-31-m2-abdesign/specs/fidelity-gate/spec.md:14; base openspec/specs/fidelity-gate/spec.md:90 ratio 1.00 |
| 9 | 2026-10-31-m2-earlybase | archived | ADDED | Early baseline fields | APPLIED | openspec/changes/archive/2026-10-31-m2-earlybase/specs/fidelity-gate/spec.md:3; base openspec/specs/fidelity-gate/spec.md:97 ratio 1.00 |
| 10 | 2026-10-31-m2-soakexit | archived | ADDED | Verdict exit codes | APPLIED | openspec/changes/archive/2026-10-31-m2-soakexit/specs/fidelity-gate/spec.md:3; base openspec/specs/fidelity-gate/spec.md:57 ratio 1.00 |
| 11 | 2026-10-31-m2-soakexit | archived | ADDED | Excluded seeds | APPLIED | openspec/changes/archive/2026-10-31-m2-soakexit/specs/fidelity-gate/spec.md:18; base openspec/specs/fidelity-gate/spec.md:72 ratio 1.00 |
| 12 | 2026-11-02-abtool | archived | MODIFIED | Verdict classes | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-abtool/specs/fidelity-gate/spec.md:3; base openspec/specs/fidelity-gate/spec.md:15 ratio 0.48; HAND: base fidelity-gate:15 has the pre-abtool text without the A:B ratio interval (read) |
| 13 | 2026-11-02-abtool | archived | MODIFIED | Twenty percent tolerance | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-abtool/specs/fidelity-gate/spec.md:14; base openspec/specs/fidelity-gate/spec.md:8 ratio 0.83; HAND: base fidelity-gate:8 lacks 'An A:B report MAY state a tighter margin' and the Named margin scenario (read) |
| 14 | 2026-11-02-abtool | archived | ADDED | Game.exe arm sample and regimes | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-abtool/specs/fidelity-gate/spec.md:27 |
| 15 | 2026-11-02-abtool | archived | ADDED | Pair setup on both backends | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-abtool/specs/fidelity-gate/spec.md:34 |
| 16 | 2026-11-04-d2d-replicated-soak | archived | ADDED | Replicated soak is a fidelity check | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-d2d-replicated-soak/specs/fidelity-gate/spec.md:3 |
| 17 | 2026-11-04-d2d-replicated-soak | archived | ADDED | Soak FAIL attribution | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-d2d-replicated-soak/specs/fidelity-gate/spec.md:10 |
| 18 | 2026-11-04-m2-latebase | archived | ADDED | Late baseline | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-m2-latebase/specs/fidelity-gate/spec.md:3 |

### lab-rules

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-compute | archived | MODIFIED | One d2d at a time | APPLIED | openspec/changes/archive/2026-10-30-compute/specs/lab-rules/spec.md:3; base openspec/specs/lab-rules/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-softguard | archived | MODIFIED | Safety limits for lab runs | APPLIED | openspec/changes/archive/2026-10-30-softguard/specs/lab-rules/spec.md:3; base openspec/specs/lab-rules/spec.md:69 ratio 1.00 |
| 3 | 2026-10-30-softguard | archived | ADDED | Big run warning | APPLIED | openspec/changes/archive/2026-10-30-softguard/specs/lab-rules/spec.md:20; base openspec/specs/lab-rules/spec.md:84 ratio 1.00 |
| 4 | 2026-11-01-memrules | archived | ADDED | Memory access | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-memrules/specs/lab-rules/spec.md:3 |
| 5 | 2026-11-01-memrules | archived | ADDED | Crutch rule | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-memrules/specs/lab-rules/spec.md:14 |
| 6 | 2026-11-01-memrules | archived | ADDED | Cheated runs and the writes line | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-memrules/specs/lab-rules/spec.md:29 |

### licence-policy

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-specdocs | archived | ADDED | Repository licence | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:3; base openspec/specs/licence-policy/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-specdocs | archived | ADDED | SPDX header on new source files | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:14; base openspec/specs/licence-policy/spec.md:19 ratio 1.00 |
| 3 | 2026-10-30-specdocs | archived | ADDED | Third-party code needs a compatible licence | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:25; base openspec/specs/licence-policy/spec.md:30 ratio 1.00 |
| 4 | 2026-10-30-specdocs | archived | ADDED | Redivivus numbers only | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:36; base openspec/specs/licence-policy/spec.md:41 ratio 1.00 |
| 5 | 2026-10-30-specdocs | archived | ADDED | Stdlib-only dependencies | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:43; base openspec/specs/licence-policy/spec.md:48 ratio 1.00 |
| 6 | 2026-10-30-specdocs | archived | ADDED | Fork patches stay downstream | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:50; base openspec/specs/licence-policy/spec.md:55 ratio 1.00 |
| 7 | 2026-10-30-specdocs | archived | ADDED | Distribution needs the corresponding source | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/licence-policy/spec.md:57; base openspec/specs/licence-policy/spec.md:62 ratio 1.00 |

### merc-policy

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-m3-routeguard | archived | MODIFIED | Route guard | APPLIED | openspec/changes/archive/2026-10-31-m3-routeguard/specs/merc-policy/spec.md:3; base openspec/specs/merc-policy/spec.md:36 ratio 1.00 |
| 2 | 2026-10-31-m3-routeguard | archived | MODIFIED | Andariel floor | APPLIED | openspec/changes/archive/2026-10-31-m3-routeguard/specs/merc-policy/spec.md:26; base openspec/specs/merc-policy/spec.md:59 ratio 1.00 |
| 3 | 2026-11-02-mercpolicy | archived | ADDED | Spending order | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:3 |
| 4 | 2026-11-02-mercpolicy | archived | ADDED | Revive price is read, not computed | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:14 |
| 5 | 2026-11-02-mercpolicy | archived | ADDED | Act 2 Desert Mercenary swap | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:21 |
| 6 | 2026-11-02-mercpolicy | archived | ADDED | Gold farm while the merc is missing | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:32 |
| 7 | 2026-11-02-mercpolicy | archived | ADDED | Merc policy on by default | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:43 |
| 8 | 2026-11-02-mercpolicy | archived | MODIFIED | Blood Raven kit gate | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:52; base openspec/specs/merc-policy/spec.md:22 ratio 0.69; HAND: ratio 0.69 against base merc-policy:22; text not diffed line by line |
| 9 | 2026-11-02-mercpolicy | archived | MODIFIED | Revive and stop rule | UNSURE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:63; base openspec/specs/merc-policy/spec.md:29 ratio 0.23; base vs later ratio 0.81; HAND: ratio 0.23 vs base merc-policy:29, but open m3-merc MODIFIES the same name at ratio 0.81 to base; decide which text is the base |
| 10 | 2026-11-02-mercpolicy | archived | MODIFIED | Route guard | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:78; base openspec/specs/merc-policy/spec.md:36 ratio 0.64; HAND: NUMBER drift: base merc-policy:37 says Monastery road 16 (no merc) / 14 (merc), delta says 14 / 14; code default route.monasteryMinLevel 14 (internal/env/tuning_route.go:37) (V) |
| 11 | 2026-11-02-mercpolicy | archived | MODIFIED | Merc gear and potions | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-02-mercpolicy/specs/merc-policy/spec.md:101; base openspec/specs/merc-policy/spec.md:82 ratio 0.59; HAND: ratio 0.59 against base merc-policy:82; text not diffed line by line |
| 12 | 2026-11-04-m3-readiness-model | archived | ADDED | Readiness model | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-m3-readiness-model/specs/merc-policy/spec.md:3 |
| 13 | m3-merc | open:proposed | ADDED | Merc uptime in Act 1 | NOT APPLIED-OPEN | openspec/changes/m3-merc/specs/merc-policy/spec.md:3 |
| 14 | m3-merc | open:proposed | MODIFIED | Revive and stop rule | NOT APPLIED-OPEN | openspec/changes/m3-merc/specs/merc-policy/spec.md:16; base openspec/specs/merc-policy/spec.md:29 ratio 0.81 |
| 15 | m3-merc | open:proposed | ADDED | Blood Raven bailout | NOT APPLIED-OPEN | openspec/changes/m3-merc/specs/merc-policy/spec.md:29 |
| 16 | m3-merc | open:proposed | ADDED | Keeper and throwaway mercs | NOT APPLIED-OPEN | openspec/changes/m3-merc/specs/merc-policy/spec.md:40 |

### primitives

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | m1-liveproof | open:in-progress | ADDED | Primitive drills | NOT APPLIED-OPEN | openspec/changes/m1-liveproof/specs/primitives/spec.md:8 |
| 2 | m1-liveproof | open:in-progress | ADDED | Clean UI around every drill | NOT APPLIED-OPEN | openspec/changes/m1-liveproof/specs/primitives/spec.md:19 |
| 3 | m1-liveproof | open:in-progress | ADDED | High criticality needs a live pass | NOT APPLIED-OPEN | openspec/changes/m1-liveproof/specs/primitives/spec.md:26 |

### process-hygiene

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-specdocs | archived | ADDED | Long processes start through the registry wrapper | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:3; base openspec/specs/process-hygiene/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-specdocs | archived | ADDED | Group kill on exit and TTL | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:14; base openspec/specs/process-hygiene/spec.md:19 ratio 1.00 |
| 3 | 2026-10-30-specdocs | archived | ADDED | Reap by record with pid and starttime check | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:25; base openspec/specs/process-hygiene/spec.md:30 ratio 1.00 |
| 4 | 2026-10-30-specdocs | archived | ADDED | Orphans are listed, never killed | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:36; base openspec/specs/process-hygiene/spec.md:41 ratio 1.00 |
| 5 | 2026-10-30-specdocs | archived | ADDED | Orphan stops need the user's approval | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:47; base openspec/specs/process-hygiene/spec.md:52 ratio 1.00 |
| 6 | 2026-10-30-specdocs | archived | ADDED | No pattern kills | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:54; base openspec/specs/process-hygiene/spec.md:59 ratio 1.00 |
| 7 | 2026-10-30-specdocs | archived | ADDED | Waiters are bounded | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:65; base openspec/specs/process-hygiene/spec.md:70 ratio 1.00 |
| 8 | 2026-10-30-specdocs | archived | ADDED | Agents confirm cleanup in the report | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:72; base openspec/specs/process-hygiene/spec.md:77 ratio 1.00 |
| 9 | 2026-10-30-specdocs | archived | ADDED | Dispatcher janitor duty | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/process-hygiene/spec.md:83; base openspec/specs/process-hygiene/spec.md:88 ratio 1.00 |

### public-split

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-specdocs | archived | ADDED | Classes decide what is public | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:3; base openspec/specs/public-split/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-specdocs | archived | ADDED | No scrub-needed entries at go-live | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:10; base openspec/specs/public-split/spec.md:15 ratio 1.00 |
| 3 | 2026-10-30-specdocs | archived | ADDED | No private names in committed files | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:17; base openspec/specs/public-split/spec.md:22 ratio 1.00 |
| 4 | 2026-10-30-specdocs | archived | ADDED | Host config from the environment with role names | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:28; base openspec/specs/public-split/spec.md:33 ratio 1.00 |
| 5 | 2026-10-30-specdocs | archived | ADDED | Leakcheck warns before go-live | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:39; base openspec/specs/public-split/spec.md:44 ratio 1.00 |
| 6 | 2026-10-30-specdocs | archived | ADDED | Leakcheck fails from go-live | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:46; base openspec/specs/public-split/spec.md:51 ratio 1.00 |
| 7 | 2026-10-30-specdocs | archived | ADDED | Blizzard-derived data is never committed | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:53; base openspec/specs/public-split/spec.md:58 ratio 1.00 |
| 8 | 2026-10-30-specdocs | archived | ADDED | Cut preconditions | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:64; base openspec/specs/public-split/spec.md:69 ratio 1.00 |
| 9 | 2026-10-30-specdocs | archived | ADDED | Cut result and verification | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:75; base openspec/specs/public-split/spec.md:80 ratio 1.00 |
| 10 | 2026-10-30-specdocs | archived | ADDED | Ops material stays private | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/public-split/spec.md:86; base openspec/specs/public-split/spec.md:91 ratio 1.00 |

### replay

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-replay | archived | ADDED | Replay file content | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:8; base openspec/specs/replay/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-replay | archived | ADDED | No game data or secrets in a replay | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:15; base openspec/specs/replay/spec.md:15 ratio 1.00 |
| 3 | 2026-10-30-replay | archived | ADDED | Passive recording | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:26; base openspec/specs/replay/spec.md:26 ratio 1.00 |
| 4 | 2026-10-30-replay | archived | ADDED | Lockstep playback verifies checkpoints | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:37; base openspec/specs/replay/spec.md:37 ratio 1.00 |
| 5 | 2026-10-30-replay | archived | ADDED | Checkpoint hash | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:56; base openspec/specs/replay/spec.md:56 ratio 1.00 |
| 6 | 2026-10-30-replay | archived | ADDED | Render from a replay | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:63; base openspec/specs/replay/spec.md:63 ratio 1.00 |
| 7 | 2026-10-30-replay | archived | ADDED | Size | APPLIED | openspec/changes/archive/2026-10-30-replay/specs/replay/spec.md:70; base openspec/specs/replay/spec.md:70 ratio 1.00 |

### soak-gates

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-softguard | archived | MODIFIED | Start guard | APPLIED | openspec/changes/archive/2026-10-30-softguard/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:36 ratio 0.98 |
| 2 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate ladder on d2d | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:90 ratio 1.00 |
| 3 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate 1 stability and save semantics | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:14; base openspec/specs/soak-gates/spec.md:101 ratio 1.00 |
| 4 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate 1 deaths and progression | SUPERSEDED(by 2026-11-03-specdelta) | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:21; base openspec/specs/soak-gates/spec.md:108 ratio 0.64; base vs later ratio 0.51; HAND: two MODIFIED rounds followed; base soak-gates:108 is the 2026-11-01 text |
| 5 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate 1 belt and portal | SUPERSEDED(by 2026-11-03-specdelta) | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:28; base openspec/specs/soak-gates/spec.md:119 ratio 0.52; base vs later ratio 0.26; HAND: specdelta MODIFIED it; base soak-gates:119 is still the older text |
| 6 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate 1 route guard respected | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:39; base openspec/specs/soak-gates/spec.md:130 ratio 1.00 |
| 7 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Injected fault recovery on d2d | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:46; base openspec/specs/soak-gates/spec.md:137 ratio 1.00 |
| 8 | 2026-10-31-d2d-gate-ladder | archived | ADDED | Gate 2 and Gate 3 on d2d | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:57; base openspec/specs/soak-gates/spec.md:148 ratio 1.00 |
| 9 | 2026-10-31-d2d-gate-ladder | archived | MODIFIED | G2 meaning and floors | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:66; base openspec/specs/soak-gates/spec.md:8 ratio 1.00 |
| 10 | 2026-10-31-d2d-gate-ladder | archived | MODIFIED | Start guard | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:73; base openspec/specs/soak-gates/spec.md:36 ratio 1.00 |
| 11 | 2026-10-31-d2d-gate-ladder | archived | REMOVED | G1 stability | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:86 |
| 12 | 2026-10-31-d2d-gate-ladder | archived | REMOVED | G1 progression | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:89 |
| 13 | 2026-10-31-d2d-gate-ladder | archived | REMOVED | G1 belt | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:92 |
| 14 | 2026-10-31-d2d-gate-ladder | archived | REMOVED | G1 kill recovery | APPLIED | openspec/changes/archive/2026-10-31-d2d-gate-ladder/specs/soak-gates/spec.md:95 |
| 15 | 2026-10-31-fix-restart | archived | ADDED | A game-lost recovery counts as a restart | APPLIED | openspec/changes/archive/2026-10-31-fix-restart/specs/soak-gates/spec.md:5; base openspec/specs/soak-gates/spec.md:155 ratio 1.00 |
| 16 | 2026-10-31-fix-restart | archived | ADDED | xpRisk is XP since the last save on d2d | APPLIED | openspec/changes/archive/2026-10-31-fix-restart/specs/soak-gates/spec.md:16; base openspec/specs/soak-gates/spec.md:166 ratio 1.00 |
| 17 | 2026-10-31-gen-invariants | archived | ADDED | Per-seed invariants | APPLIED | openspec/changes/archive/2026-10-31-gen-invariants/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:54 ratio 1.00 |
| 18 | 2026-10-31-gen-invariants | archived | ADDED | Death and exit invariants | APPLIED | openspec/changes/archive/2026-10-31-gen-invariants/specs/soak-gates/spec.md:14; base openspec/specs/soak-gates/spec.md:65 ratio 1.00 |
| 19 | 2026-10-31-gen-invariants | archived | ADDED | Invariants verdict | APPLIED | openspec/changes/archive/2026-10-31-gen-invariants/specs/soak-gates/spec.md:21; base openspec/specs/soak-gates/spec.md:72 ratio 1.00 |
| 20 | 2026-10-31-gen-invariants | archived | ADDED | Holdout seeds | APPLIED | openspec/changes/archive/2026-10-31-gen-invariants/specs/soak-gates/spec.md:28; base openspec/specs/soak-gates/spec.md:79 ratio 1.00 |
| 21 | 2026-10-31-soak-minutes-virtual | archived | ADDED | Gate 1 minutes are virtual minutes | APPLIED | openspec/changes/archive/2026-10-31-soak-minutes-virtual/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:177 ratio 1.00 |
| 22 | 2026-10-31-soak-minutes-virtual | archived | MODIFIED | Gate 1 deaths and progression | APPLIED | openspec/changes/archive/2026-10-31-soak-minutes-virtual/specs/soak-gates/spec.md:32; base openspec/specs/soak-gates/spec.md:108 ratio 1.00 |
| 23 | 2026-10-31-soak-minutes-virtual | archived | MODIFIED | Gate 1 belt and portal | APPLIED | openspec/changes/archive/2026-10-31-soak-minutes-virtual/specs/soak-gates/spec.md:43; base openspec/specs/soak-gates/spec.md:119 ratio 1.00 |
| 24 | 2026-11-01-baselinefree | archived | MODIFIED | Gate 1 deaths and progression | CONFLICT | openspec/changes/archive/2026-11-01-baselinefree/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:108 ratio 0.42; base vs later ratio 0.51; HAND: baselinefree: FAIL by any band verdict; specdelta (2026-11-03) replaces the same requirement with d2d baseline plus INCONCLUSIVE note and drops the band clause, while the 3 band requirements baselinefree ADDED stay unapplied; base soak-gates:108 is neither |
| 25 | 2026-11-01-baselinefree | archived | ADDED | Gate 1 XP per clvl band | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-baselinefree/specs/soak-gates/spec.md:16; HAND: orphaned if specdelta text stands (no requirement cites bands); see Needs the user |
| 26 | 2026-11-01-baselinefree | archived | ADDED | Band baseline sample | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-baselinefree/specs/soak-gates/spec.md:27; HAND: as Gate 1 XP per clvl band; code d2dsoakstat BandCVFloor 0.14, z 1.96 (internal/d2dsoakstat/bands.go:122-123) (V) |
| 27 | 2026-11-01-baselinefree | archived | ADDED | Band tolerance | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-01-baselinefree/specs/soak-gates/spec.md:34; HAND: as Gate 1 XP per clvl band; 1.96 and 0.14 match bands.go:122-123 (V) |
| 28 | 2026-11-03-gate3gaps | archived | MODIFIED | Resource slopes | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-gate3gaps/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:22 ratio 0.46; HAND: mem.csv sampling clause absent from base soak-gates:22 (read) |
| 29 | 2026-11-03-gate3gaps | archived | MODIFIED | Gate 2 and Gate 3 on d2d | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-gate3gaps/specs/soak-gates/spec.md:26; base openspec/specs/soak-gates/spec.md:148 ratio 0.38; HAND: planned-recycle clause absent from base soak-gates:148 (read) |
| 30 | 2026-11-03-gate3gaps | archived | ADDED | Full bag and disk guard | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-gate3gaps/specs/soak-gates/spec.md:43 |
| 31 | 2026-11-03-gate3gaps | archived | ADDED | Gate 0.5 smoke | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-gate3gaps/specs/soak-gates/spec.md:54 |
| 32 | 2026-11-03-specdelta | archived | MODIFIED | Gate 1 deaths and progression | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-specdelta/specs/soak-gates/spec.md:5; base openspec/specs/soak-gates/spec.md:108 ratio 0.51; HAND: newest text for this requirement (user decision 2026-11-03); base soak-gates:108 holds the 2026-11-01 wording (read) |
| 33 | 2026-11-03-specdelta | archived | MODIFIED | Gate 1 belt and portal | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-03-specdelta/specs/soak-gates/spec.md:20; base openspec/specs/soak-gates/spec.md:119 ratio 0.26; HAND: funded-window and stashGold wording absent from base soak-gates:119 (read) |
| 34 | 2026-11-04-andarielspec | archived | MODIFIED | G2 meaning and floors | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-andarielspec/specs/soak-gates/spec.md:3; base openspec/specs/soak-gates/spec.md:8 ratio 0.20; HAND: base soak-gates:9 lacks the Andariel hold rules and 5 scenarios (read); also STALE name G2 vs ADR 0015 |
| 35 | d2d-container-soak | open:proposed | ADDED | d2d container one-slot 24 h soak | SUPERSEDED(by ADR 0015) | openspec/changes/d2d-container-soak/specs/soak-gates/spec.md:3; HAND: container deferred until Gate A passes (ADR 0015 item 5; Current truth 'Deferred or parked') |

### stream-scene

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | stream-grid | open:deferred | ADDED | Holding card without gameplay | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:3 |
| 2 | stream-grid | open:deferred | ADDED | Recent merges in plain words | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:14 |
| 3 | stream-grid | open:deferred | ADDED | Current work from sanitized sources | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:21 |
| 4 | stream-grid | open:deferred | ADDED | Next gameplay ETA | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:28 |
| 5 | stream-grid | open:deferred | ADDED | Time since last gameplay | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:47 |
| 6 | stream-grid | open:deferred | ADDED | Rotating info cards | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:58 |
| 7 | stream-grid | open:deferred | ADDED | Grid layout by source count | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:69 |
| 8 | stream-grid | open:deferred | ADDED | Sources join and leave automatically | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:92 |
| 9 | stream-grid | open:deferred | ADDED | Tile contents and backend badge | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:103 |
| 10 | stream-grid | open:deferred | ADDED | TESTING label | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:118 |
| 11 | stream-grid | open:deferred | ADDED | Source health | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:125 |
| 12 | stream-grid | open:deferred | ADDED | Stream-safe privacy | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:140 |
| 13 | stream-grid | open:deferred | ADDED | Stream off by default and user go | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:151 |
| 14 | stream-grid | open:deferred | ADDED | No new exposure | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:162 |
| 15 | stream-grid | open:deferred | ADDED | Light scene page | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:169 |
| 16 | stream-grid | open:deferred | ADDED | Fixture mode | NOT APPLIED-OPEN | openspec/changes/stream-grid/specs/stream-scene/spec.md:180 |

### town-basics

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-31-enterportal | archived | ADDED | ToTown never reads a scroll while an open portal is in reach | APPLIED | openspec/changes/archive/2026-10-31-enterportal/specs/town-basics/spec.md:5; base openspec/specs/town-basics/spec.md:113 ratio 1.00 |
| 2 | 2026-10-31-fix-bank | archived | ADDED | Gold deposit works on d2d | APPLIED | openspec/changes/archive/2026-10-31-fix-bank/specs/town-basics/spec.md:5; base openspec/specs/town-basics/spec.md:128 ratio 1.00 |
| 3 | 2026-10-31-fix-bank | archived | ADDED | Repair is not attempted without gold | APPLIED | openspec/changes/archive/2026-10-31-fix-bank/specs/town-basics/spec.md:20; base openspec/specs/town-basics/spec.md:143 ratio 1.00 |
| 4 | 2026-10-31-fix-belt | archived | ADDED | Belt fill retries a vanished item id | APPLIED | openspec/changes/archive/2026-10-31-fix-belt/specs/town-basics/spec.md:3; base openspec/specs/town-basics/spec.md:85 ratio 1.00 |
| 5 | 2026-10-31-fix-belt | archived | ADDED | Belt filled before leaving town | APPLIED | openspec/changes/archive/2026-10-31-fix-belt/specs/town-basics/spec.md:10; base openspec/specs/town-basics/spec.md:92 ratio 1.00 |
| 6 | 2026-10-31-fix-belt | archived | ADDED | Belt column holds one kind | APPLIED | openspec/changes/archive/2026-10-31-fix-belt/specs/town-basics/spec.md:17; base openspec/specs/town-basics/spec.md:99 ratio 1.00 |
| 7 | 2026-10-31-fix-belt | archived | ADDED | Kit counts pack potions | APPLIED | openspec/changes/archive/2026-10-31-fix-belt/specs/town-basics/spec.md:24; base openspec/specs/town-basics/spec.md:106 ratio 1.00 |
| 8 | 2026-10-31-fix-belt2 | archived | ADDED | A kind without room is not a full belt | APPLIED | openspec/changes/archive/2026-10-31-fix-belt2/specs/town-basics/spec.md:3; base openspec/specs/town-basics/spec.md:184 ratio 1.00 |
| 9 | 2026-10-31-fix-loot | archived | ADDED | Recovery pickup is wired | APPLIED | openspec/changes/archive/2026-10-31-fix-loot/specs/town-basics/spec.md:8; base openspec/specs/town-basics/spec.md:74 ratio 1.00 |
| 10 | 2026-10-31-fix-pickup | archived | ADDED | Ground potion goes where the game puts it | APPLIED | openspec/changes/archive/2026-10-31-fix-pickup/specs/town-basics/spec.md:3; base openspec/specs/town-basics/spec.md:165 ratio 1.00 |
| 11 | 2026-10-31-fix-price | archived | ADDED | Prices follow game.exe's formulas | APPLIED | openspec/changes/archive/2026-10-31-fix-price/specs/town-basics/spec.md:5; base openspec/specs/town-basics/spec.md:150 ratio 1.00 |
| 12 | 2026-10-31-m1-audittests | archived | ADDED | ID scrolls only on demand | APPLIED | openspec/changes/archive/2026-10-31-m1-audittests/specs/town-basics/spec.md:8; base openspec/specs/town-basics/spec.md:8 ratio 1.00 |
| 13 | 2026-10-31-m1-audittests | archived | ADDED | Purchase order when poor | APPLIED | openspec/changes/archive/2026-10-31-m1-audittests/specs/town-basics/spec.md:19; base openspec/specs/town-basics/spec.md:19 ratio 1.00 |
| 14 | 2026-10-31-m1-audittests | archived | ADDED | Bank before the shop | APPLIED | openspec/changes/archive/2026-10-31-m1-audittests/specs/town-basics/spec.md:34; base openspec/specs/town-basics/spec.md:34 ratio 1.00 |
| 15 | 2026-10-31-m1-audittests | archived | ADDED | Repair failure latch | APPLIED | openspec/changes/archive/2026-10-31-m1-audittests/specs/town-basics/spec.md:41; base openspec/specs/town-basics/spec.md:41 ratio 1.00 |
| 16 | 2026-10-31-m1-loot | archived | ADDED | Charms, runes and gems on a sell visit | APPLIED | openspec/changes/archive/2026-10-31-m1-loot/specs/town-basics/spec.md:8; base openspec/specs/town-basics/spec.md:48 ratio 1.00 |
| 17 | 2026-10-31-m1-stashcap | archived | ADDED | Stash gold cap | APPLIED | openspec/changes/archive/2026-10-31-m1-stashcap/specs/town-basics/spec.md:8; base openspec/specs/town-basics/spec.md:63 ratio 1.00 |
| 18 | 2026-11-04-goldfloor | archived | ADDED | Trips must buy something | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-goldfloor/specs/town-basics/spec.md:8 |
| 19 | 2026-11-04-goldfloor | archived | ADDED | No scroll rebuy while poor | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-goldfloor/specs/town-basics/spec.md:19 |
| 20 | 2026-11-04-goldfloor | archived | ADDED | Early mana cap | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-goldfloor/specs/town-basics/spec.md:26 |
| 21 | 2026-11-04-m1-trips | archived | ADDED | Sell trips only when worth it | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-m1-trips/specs/town-basics/spec.md:8 |
| 22 | 2026-11-04-m1-trips | archived | ADDED | Gold trip above the keep line | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-m1-trips/specs/town-basics/spec.md:15 |
| 23 | 2026-11-04-m1-trips | archived | ADDED | Pack-full threshold | NOT APPLIED-IN FORCE | openspec/changes/archive/2026-11-04-m1-trips/specs/town-basics/spec.md:22 |

### upstream-pr

| # | delta source | state | op | requirement | verdict | evidence |
|---|---|---|---|---|---|---|
| 1 | 2026-10-30-specdocs | archived | ADDED | Differential proof against game.exe | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:3; base openspec/specs/upstream-pr/spec.md:8 ratio 1.00 |
| 2 | 2026-10-30-specdocs | archived | ADDED | Address-anchored claims without unverified links | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:18; base openspec/specs/upstream-pr/spec.md:23 ratio 1.00 |
| 3 | 2026-10-30-specdocs | archived | ADDED | pr_guard clean | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:26; base openspec/specs/upstream-pr/spec.md:30 ratio 1.00 |
| 4 | 2026-10-30-specdocs | archived | ADDED | Statistics are corroboration only | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:33; base openspec/specs/upstream-pr/spec.md:37 ratio 1.00 |
| 5 | 2026-10-30-specdocs | archived | ADDED | One claim per PR | SUPERSEDED(by 2026-10-31-upstream-bundle) | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:44; base openspec/specs/upstream-pr/spec.md:48 ratio 0.13; base vs later ratio 1.00; HAND: later delta, base upstream-pr:48 |
| 6 | 2026-10-30-specdocs | archived | ADDED | Nothing else moved | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:51; base openspec/specs/upstream-pr/spec.md:63 ratio 1.00 |
| 7 | 2026-10-30-specdocs | archived | ADDED | Independent adversarial verification | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:62; base openspec/specs/upstream-pr/spec.md:74 ratio 1.00 |
| 8 | 2026-10-30-specdocs | archived | ADDED | Limits disclosed in the PR body | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:73; base openspec/specs/upstream-pr/spec.md:85 ratio 1.00 |
| 9 | 2026-10-30-specdocs | archived | ADDED | Evidence pack and go/no-go before asking | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:80; base openspec/specs/upstream-pr/spec.md:92 ratio 1.00 |
| 10 | 2026-10-30-specdocs | archived | ADDED | PR authorship and body format | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:91; base openspec/specs/upstream-pr/spec.md:103 ratio 1.00 |
| 11 | 2026-10-30-specdocs | archived | ADDED | Nothing posted without the user's go | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:102; base openspec/specs/upstream-pr/spec.md:114 ratio 1.00 |
| 12 | 2026-10-30-specdocs | archived | ADDED | PR spacing and bundled follow-ups | SUPERSEDED(by 2026-10-31-upstream-bundle) | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:109; base openspec/specs/upstream-pr/spec.md:121 ratio 0.09; base vs later ratio 1.00; HAND: later delta, base upstream-pr:121 |
| 13 | 2026-10-30-specdocs | archived | ADDED | No game data in an upstream PR | APPLIED | openspec/changes/archive/2026-10-30-specdocs/specs/upstream-pr/spec.md:120; base openspec/specs/upstream-pr/spec.md:132 ratio 1.00 |
| 14 | 2026-10-31-spec-process | archived | ADDED | Every upstream post is logged | APPLIED | openspec/changes/archive/2026-10-31-spec-process/specs/upstream-pr/spec.md:3; base openspec/specs/upstream-pr/spec.md:139 ratio 1.00 |
| 15 | 2026-10-31-upstream-bundle | archived | MODIFIED | One claim per PR | APPLIED | openspec/changes/archive/2026-10-31-upstream-bundle/specs/upstream-pr/spec.md:3; base openspec/specs/upstream-pr/spec.md:48 ratio 1.00 |
| 16 | 2026-10-31-upstream-bundle | archived | MODIFIED | PR spacing and bundled follow-ups | APPLIED | openspec/changes/archive/2026-10-31-upstream-bundle/specs/upstream-pr/spec.md:18; base openspec/specs/upstream-pr/spec.md:121 ratio 1.00 |


## Base-spec flags

### Stale-term hits (G1/G2, milestone, Mn, Game.exe, frozen, container)

- openspec/specs/backend-contract/spec.md:8 Decision code stays backend-agnostic: Game.exe
- openspec/specs/backend-contract/spec.md:22 Input client: Game.exe
- openspec/specs/backend-contract/spec.md:104 The d2d town source offers every seam the town reads: Game.exe
- openspec/specs/backend-contract/spec.md:258 Waypoint rows are clicked at their panel centres: game.exe
- openspec/specs/combat-loop/spec.md:30 Fight-loop no-progress watchdog: Frozen
- openspec/specs/early-game-policy/spec.md:22 No vendor gear below the early gear cap: G1, g1
- openspec/specs/fidelity-gate/spec.md:29 Same character, same setup: m1
- openspec/specs/fidelity-gate/spec.md:43 What d2d cannot validate: G1, G2, Game.exe
- openspec/specs/fidelity-gate/spec.md:57 Verdict exit codes: m2
- openspec/specs/fidelity-gate/spec.md:72 Excluded seeds: m2
- openspec/specs/fidelity-gate/spec.md:79 A/B protocol for bot changes: m2
- openspec/specs/fidelity-gate/spec.md:97 Early baseline fields: m2
- openspec/specs/lab-rules/spec.md:69 Safety limits for lab runs: Game.exe
- openspec/specs/licence-policy/spec.md:62 Distribution needs the corresponding source: container
- openspec/specs/merc-policy/spec.md:59 Andariel floor: m3
- openspec/specs/soak-gates/spec.md:8 G2 meaning and floors: G2, Gate 2
- openspec/specs/soak-gates/spec.md:54 Per-seed invariants: Frozen
- openspec/specs/soak-gates/spec.md:90 Gate ladder on d2d: Gate 2
- openspec/specs/soak-gates/spec.md:101 Gate 1 stability and save semantics: G1
- openspec/specs/soak-gates/spec.md:108 Gate 1 deaths and progression: G1
- openspec/specs/soak-gates/spec.md:119 Gate 1 belt and portal: G1
- openspec/specs/soak-gates/spec.md:130 Gate 1 route guard respected: G1
- openspec/specs/soak-gates/spec.md:137 Injected fault recovery on d2d: G1
- openspec/specs/soak-gates/spec.md:148 Gate 2 and Gate 3 on d2d: Gate 2, container
- openspec/specs/soak-gates/spec.md:155 A game-lost recovery counts as a restart: Game.exe
- openspec/specs/soak-gates/spec.md:166 xpRisk is XP since the last save on d2d: Game.exe
- openspec/specs/town-basics/spec.md:8 ID scrolls only on demand: G1
- openspec/specs/town-basics/spec.md:63 Stash gold cap: G1, m1
- openspec/specs/town-basics/spec.md:150 Prices follow game.exe's formulas: game.exe
- openspec/specs/upstream-pr/spec.md:8 Differential proof against game.exe: game.exe
- openspec/specs/upstream-pr/spec.md:23 Address-anchored claims without unverified links: game.exe
- openspec/specs/upstream-pr/spec.md:48 One claim per PR: game.exe

### Dead references (backticked or plain repo paths that do not exist)

- openspec/specs/upstream-pr/spec.md:30 pr_guard clean -> tools/pr_guard.py (missing)
- openspec/specs/upstream-pr/spec.md:63 Nothing else moved -> tools/presubmit.py (missing)

### Duplicate names across base specs


### Near-duplicate bodies across specs (ratio >= 0.8)


### Numbers per base requirement (for hand-checking against code)

- openspec/specs/backend-contract/spec.md:8 Decision code stays backend-agnostic: 1 10 2026 30
- openspec/specs/backend-contract/spec.md:22 Input client: 2 47012
- openspec/specs/backend-contract/spec.md:36 Per-instance sockets and paths: 2
- openspec/specs/backend-contract/spec.md:43 Backend gaps are named: 1.2
- openspec/specs/backend-contract/spec.md:50 Original game remains the acceptance target: 10 11 2026 30
- openspec/specs/backend-contract/spec.md:57 d2d belt keys drink by belt column: 0 1 3 4
- openspec/specs/backend-contract/spec.md:64 A d2d cast targets the monster at the cell: 0.5 1 5
- openspec/specs/backend-contract/spec.md:86 The d2d snapshot gets a viewport for combat: 0.5 16 284 400 8
- openspec/specs/backend-contract/spec.md:93 The combat clock reads state after every sleep: 0
- openspec/specs/backend-contract/spec.md:111 Vendor stock becomes stored item units: 1
- openspec/specs/backend-contract/spec.md:125 Item stats are synthesised from the dump extras: 0 70 72 73
- openspec/specs/backend-contract/spec.md:136 Dialog entries come from the menu verb: 0 1
- openspec/specs/backend-contract/spec.md:143 Named objects get the class the town searches for: 1 103 109 2 3 4 40 5 75
- openspec/specs/backend-contract/spec.md:150 Clicks on d2d hover first: 120 ms 3 350 ms
- openspec/specs/backend-contract/spec.md:161 Town interactions walk to the target: 15 s
- openspec/specs/backend-contract/spec.md:168 The d2d link wait outlives the verb timeout: 10 s
- openspec/specs/backend-contract/spec.md:175 The d2d backend waits for links by default: 0 32 s 90 s
- openspec/specs/backend-contract/spec.md:218 The portal verb picks a source with charges: 0 70
- openspec/specs/backend-contract/spec.md:229 HasTP counts charges: 0
- openspec/specs/backend-contract/spec.md:236 The d2d backend enters a town portal with the client's portal command: 2000
- openspec/specs/backend-contract/spec.md:247 The waypoint unit is found on d2d: 1 295
- openspec/specs/backend-contract/spec.md:258 Waypoint rows are clicked at their panel centres: 0 112 135 171 176 207 230 240 243 280 304 316 352 368 388 424 77 8
- openspec/specs/backend-contract/spec.md:265 A bounced step away from a cave mouth changes direction: 16 20 6
- openspec/specs/backend-contract/spec.md:276 Touched waypoints reach the planner: 15 23
- openspec/specs/backend-contract/spec.md:283 The d2d state source lists missiles: 0
- openspec/specs/bot-loop/spec.md:8 Restarts counts every bot-started game: 3
- openspec/specs/bot-loop/spec.md:27 Per-game memory is forgotten on every new game: 0
- openspec/specs/bot-loop/spec.md:53 Recovery without the kit stays near town: 0 1 13 17 18 19 2 3 8 9
- openspec/specs/bot-loop/spec.md:68 Campaign stall escalation: 0 30 5
- openspec/specs/bot-loop/spec.md:83 A failed walk does not condemn a route for good: 0 300
- openspec/specs/combat-loop/spec.md:19 No retreat while pinned: 2 8 s
- openspec/specs/combat-loop/spec.md:30 Fight-loop no-progress watchdog: 40
- openspec/specs/combat-loop/spec.md:41 Champion group is avoided at low level: 3 8
- openspec/specs/combat-loop/spec.md:52 Champion caution raises the potion lines: 0 10 15 25 50 55
- openspec/specs/combat-loop/spec.md:82 Rejuvenation is drunk at a higher line against a boss pack: 0 1 35 50 70
- openspec/specs/combat-loop/spec.md:108 The cast gap covers the cast animation: 1.02 1.04 1.6 1.75 450 ms 560 ms 650 ms
- openspec/specs/dashboard-pub/spec.md:8 Pub backend field follows the badge: 40
- openspec/specs/dashboard-pub/spec.md:30 Stream image reconnects: 10 1500 ms 5000 ms
- openspec/specs/death-rule/spec.md:8 Death is normal: 0 1 10 10.2 3 5
- openspec/specs/death-rule/spec.md:15 Death streak (FAIL-1): 10.2 3 30 4 5 60
- openspec/specs/death-rule/spec.md:22 Recovery stuck (FAIL-2): 10 10.2 5 60
- openspec/specs/death-rule/spec.md:29 Death rate (FAIL-3): 10.2 1000 15 150 16 30 37 5
- openspec/specs/death-rule/spec.md:36 Warnings: 10.2 2 5 60
- openspec/specs/death-rule/spec.md:43 Repeat-site action: 10.2 11.2 12 180 2 30 6 h
- openspec/specs/death-rule/spec.md:50 Bug tag: 1 10.2
- openspec/specs/death-rule/spec.md:57 Gold carry and corpse: 10.2 10.3 10.6 11 11.3 300 50 x 850
- openspec/specs/early-game-policy/spec.md:8 Den of Evil first: 1 10 10.1 2 2026 30 5
- openspec/specs/early-game-policy/spec.md:22 No vendor gear below the early gear cap: 14 7
- openspec/specs/early-game-policy/spec.md:29 Rejuvenation potion cap: 3
- openspec/specs/early-game-policy/spec.md:36 Level where XP is: 11.5 12 3 7885
- openspec/specs/early-game-policy/spec.md:43 Belt before the field: 11.2 4
- openspec/specs/early-game-policy/spec.md:50 Identify scrolls only for likely valuables: 0 1 2 3 7
- openspec/specs/early-game-policy/spec.md:65 A failed repair is not retried until something changes: 0 10 100 5
- openspec/specs/early-game-policy/spec.md:72 Bank the surplus before the shop: 1 11 190 300 50 x
- openspec/specs/early-game-policy/spec.md:83 Potions before the first town portal scroll: 0 1 10 2 2026 31 4
- openspec/specs/early-game-policy/spec.md:106 The Den floor does not yield far below the floor: 1 12 s 3 4 5
- openspec/specs/early-game-policy/spec.md:113 The Den floor yields only one level below with a kit: 2
- openspec/specs/early-game-policy/spec.md:124 Every equip slot click is logged: 3
- openspec/specs/early-game-policy/spec.md:142 A failed equip returns the item and says why: 1
- openspec/specs/early-game-policy/spec.md:149 Full Rejuvenation is preferred at the cap: 0 3 6
- openspec/specs/early-game-policy/spec.md:160 A pack rejuvenation is belted without a rejuvenation column: 31
- openspec/specs/fidelity-gate/spec.md:8 Twenty percent tolerance: 0.5 1.0 1000 11 20 20 % 3
- openspec/specs/fidelity-gate/spec.md:15 Verdict classes: 3 5 95
- openspec/specs/fidelity-gate/spec.md:22 Minimum sample: 10 15 16 h 2 2 h 3 5 8
- openspec/specs/fidelity-gate/spec.md:29 Same character, same setup: 1 11 14 3
- openspec/specs/fidelity-gate/spec.md:36 Known gaps are not bugs: 14 250 4 450
- openspec/specs/fidelity-gate/spec.md:43 What d2d cannot validate: 20 4
- openspec/specs/fidelity-gate/spec.md:50 Fast d2d runs use the lab fork: 1 10 100 15 380 45
- openspec/specs/fidelity-gate/spec.md:57 Verdict exit codes: 0 10 2 2026 3 30
- openspec/specs/fidelity-gate/spec.md:72 Excluded seeds: 10 2026 30
- openspec/specs/fidelity-gate/spec.md:79 A/B protocol for bot changes: 10 2 2026 31 4 7 8 9 95
- openspec/specs/fidelity-gate/spec.md:90 A/B sample size and interleaving: 0.84 1.96 2 8
- openspec/specs/fidelity-gate/spec.md:97 Early baseline fields: 1 2 5 7
- openspec/specs/lab-rules/spec.md:8 One d2d at a time: 0 10 2026 30 4
- openspec/specs/lab-rules/spec.md:41 Fork on the forge the same day: 10 2026 30
- openspec/specs/lab-rules/spec.md:48 GPL separation: 10 2026 3 30
- openspec/specs/lab-rules/spec.md:69 Safety limits for lab runs: 1 10 15 25 % 40 % 45 5
- openspec/specs/lab-rules/spec.md:84 Big run warning: 10 2 2026 30 5
- openspec/specs/licence-policy/spec.md:8 Repository licence: 0010 10 2026 3.0 30
- openspec/specs/licence-policy/spec.md:19 SPDX header on new source files: 0010 3.0
- openspec/specs/licence-policy/spec.md:30 Third-party code needs a compatible licence: 0010
- openspec/specs/licence-policy/spec.md:41 Redivivus numbers only: 0010
- openspec/specs/licence-policy/spec.md:48 Stdlib-only dependencies: 0010
- openspec/specs/licence-policy/spec.md:55 Fork patches stay downstream: 0003 0010 10 2026 30
- openspec/specs/licence-policy/spec.md:62 Distribution needs the corresponding source: 0010
- openspec/specs/merc-policy/spec.md:8 Free Rogue first: 1 10 10.6 11.3 3 5
- openspec/specs/merc-policy/spec.md:15 Paid Rogue only as fallback: 0 2
- openspec/specs/merc-policy/spec.md:22 Blood Raven kit gate: 1 10.1 1477431 2 250 3 4 8 9
- openspec/specs/merc-policy/spec.md:29 Revive and stop rule: 000 15 2 x 5 50
- openspec/specs/merc-policy/spec.md:36 Route guard: 0 0011 14 16 20 25 26 37 7
- openspec/specs/merc-policy/spec.md:59 Andariel floor: 0 0011 1 15 17 2
- openspec/specs/merc-policy/spec.md:82 Merc gear and potions: 10 40 7 8
- openspec/specs/process-hygiene/spec.md:19 Group kill on exit and TTL: 5 s
- openspec/specs/process-hygiene/spec.md:52 Orphan stops need the user's approval: 10 2026 30
- openspec/specs/public-split/spec.md:8 Classes decide what is public: 1 10 2026 30
- openspec/specs/public-split/spec.md:15 No scrub-needed entries at go-live: 1 5
- openspec/specs/public-split/spec.md:22 No private names in committed files: 1 7
- openspec/specs/public-split/spec.md:33 Host config from the environment with role names: 2
- openspec/specs/public-split/spec.md:44 Leakcheck warns before go-live: 0 7
- openspec/specs/public-split/spec.md:51 Leakcheck fails from go-live: 1 3 5
- openspec/specs/public-split/spec.md:58 Blizzard-derived data is never committed: 0010
- openspec/specs/public-split/spec.md:69 Cut preconditions: 0 1 2 5
- openspec/specs/public-split/spec.md:80 Cut result and verification: 5
- openspec/specs/public-split/spec.md:91 Ops material stays private: 2 4 7
- openspec/specs/replay/spec.md:8 Replay file content: 10 2026 30
- openspec/specs/replay/spec.md:15 No game data or secrets in a replay: 10 2026 30
- openspec/specs/replay/spec.md:37 Lockstep playback verifies checkpoints: 5
- openspec/specs/replay/spec.md:56 Checkpoint hash: 0
- openspec/specs/replay/spec.md:70 Size: 1 10 2026 30
- openspec/specs/soak-gates/spec.md:8 G2 meaning and floors: 0013 10 14 16 2 2026 24 h 3 30 5 8 h
- openspec/specs/soak-gates/spec.md:15 Soak progress floors: 10 1000 24 h 3 90
- openspec/specs/soak-gates/spec.md:22 Resource slopes: 10 2 3 40 5
- openspec/specs/soak-gates/spec.md:29 Heap runaway alert: 1024 16 4 48.8 5 8192
- openspec/specs/soak-gates/spec.md:36 Start guard: 1 10 2 28
- openspec/specs/soak-gates/spec.md:47 Verdict: 0 3
- openspec/specs/soak-gates/spec.md:54 Per-seed invariants: 15 30 min
- openspec/specs/soak-gates/spec.md:65 Death and exit invariants: 1 10 1000 15 150 3 30 min 5 60 min 95
- openspec/specs/soak-gates/spec.md:79 Holdout seeds: 1 16 23 3 5
- openspec/specs/soak-gates/spec.md:90 Gate ladder on d2d: 0 0013 1 2 2 h 24 h 3 8 h
- openspec/specs/soak-gates/spec.md:101 Gate 1 stability and save semantics: 1 2 h
- openspec/specs/soak-gates/spec.md:108 Gate 1 deaths and progression: 0009 01 1 10 11 2 2026 3
- openspec/specs/soak-gates/spec.md:119 Gate 1 belt and portal: 01 1 11 2026 4 5
- openspec/specs/soak-gates/spec.md:130 Gate 1 route guard respected: 1 30 s 5
- openspec/specs/soak-gates/spec.md:137 Injected fault recovery on d2d: 0013 1 3 6 60 7
- openspec/specs/soak-gates/spec.md:148 Gate 2 and Gate 3 on d2d: 1 10 2 24 h 3 8 h
- openspec/specs/soak-gates/spec.md:166 xpRisk is XP since the last save on d2d: 0
- openspec/specs/soak-gates/spec.md:177 Gate 1 minutes are virtual minutes: 01 1 10 11 2 2026 3 5
- openspec/specs/town-basics/spec.md:8 ID scrolls only on demand: 10 100 2 2026 3 31 7
- openspec/specs/town-basics/spec.md:19 Purchase order when poor: 1 10 2 2026 3 31
- openspec/specs/town-basics/spec.md:34 Bank before the shop: 11 300 50 x
- openspec/specs/town-basics/spec.md:41 Repair failure latch: 10 2
- openspec/specs/town-basics/spec.md:48 Charms, runes and gems on a sell visit: 0 10 2026 3 31 6
- openspec/specs/town-basics/spec.md:63 Stash gold cap: 000 100 2 500
- openspec/specs/town-basics/spec.md:99 Belt column holds one kind: 0 4
- openspec/specs/town-basics/spec.md:113 ToTown never reads a scroll while an open portal is in reach: 3
- openspec/specs/town-basics/spec.md:150 Prices follow game.exe's formulas: 1 2 8
- openspec/specs/upstream-pr/spec.md:8 Differential proof against game.exe: 1 10 100000 2026 30
- openspec/specs/upstream-pr/spec.md:23 Address-anchored claims without unverified links: 2
- openspec/specs/upstream-pr/spec.md:30 pr_guard clean: 0 2
- openspec/specs/upstream-pr/spec.md:37 Statistics are corroboration only: 3
- openspec/specs/upstream-pr/spec.md:48 One claim per PR: 10 2026 31 4
- openspec/specs/upstream-pr/spec.md:63 Nothing else moved: 4
- openspec/specs/upstream-pr/spec.md:74 Independent adversarial verification: 5
- openspec/specs/upstream-pr/spec.md:85 Limits disclosed in the PR body: 6
- openspec/specs/upstream-pr/spec.md:92 Evidence pack and go/no-go before asking: 10 2026 30 7
- openspec/specs/upstream-pr/spec.md:103 PR authorship and body format: 3 5 8
- openspec/specs/upstream-pr/spec.md:121 PR spacing and bundled follow-ups: 10 2026 31 8
- openspec/specs/upstream-pr/spec.md:139 Every upstream post is logged: 10 134 135 136 2026 31

## Open cards with deltas

- d2d-container-soak: status=proposed merge=
- d2d-default-backend: status=proposed merge=
- d2dmesh: status=deferred merge=
- m1-liveproof: status=in-progress merge=
- m2-roam: status=deferred merge=
- m2-tripwp: status=deferred merge=
- m3-act1run: status=deferred merge=
- m3-andariel: status=deferred merge=
- m3-merc: status=proposed merge=
- stream-grid: status=deferred merge=""

## Base-spec findings (hand triage of the flags above)

STALE vs ADR 0015 / Current truth (V unless marked):
- `soak-gates` Purpose (line 4) names "G1 (2 h), G2 (8 h) and 24 h soaks"; G1/G2 are retired names (ADR 0013, Current truth). Requirement "G2 meaning and floors" (line 8) carries the retired name and "Gate 2 on d2d, 24 h Gate 3". It also says a "stability and economy soak" until Andariel work; ADR 0015 now defines done as Act 1 cleared on game.exe (Gate A), so the progression wording needs the user (see below). The "Gate 1 ..." requirement names (lines 101 to 130, 177) are the live d2d rung names (Gate 1 = 2 h), not stale; their `Source:` lines cite "G1 criteria" (S, harmless).
- `fidelity-gate:43` "What d2d cannot validate" says these topics "stay original-game soak topics (G1, G2)": the Game.exe G1/G2 soaks were retired; the matching home is Gate A (ADR 0015). `fidelity-gate` Purpose and `backend-contract:50` talk of the original game as the acceptance target: consistent with ADR 0015, no change.
- `early-game-policy:23` cites "G1 attempts 7 to 14" and `town-basics` lines 8 and 63 cite G1: historic evidence, not a gate, leave (S, text not re-read in full).
- `backend-contract:8` "Decision code stays backend-agnostic" and `:50` cite ADR 0009 whose items 1 and 5 ADR 0015 amends (U: line 8 text was not re-read for the amended items).
- `merc-policy:37` Route guard Monastery floor 16 (no merc) vs code default 14: see NUMBER.
- `licence-policy:62` "container" requirement: still valid text; the container is deferred (Current truth), no change.

DEAD REFERENCE (V, `ls tools` finds neither): `upstream-pr:30` `tools/pr_guard.py`, `upstream-pr:63` `tools/presubmit.py`. They may live in the private ops repo (U). No other repo path in a base spec is missing (script check covers docs/, internal/, cmd/, tools/, openspec/, deploy/, scripts/ prefixes only; bare doc names like `docs/soak24.md` are in that check, bare `ADR`/section citations are not).

DUPLICATE: none (V: no requirement name appears in two base specs; no pair of requirements across specs has text ratio >= 0.8).

NUMBER vs CODE (V, both cited):
| spec | number | code | match |
|---|---|---|---|
| soak-gates:16 | avg XP below 1000/h fails (m24) | cmd/d2soak/main.go:90 `MinXPPerHour = 1000` | yes |
| soak-gates:16 | no-gain window 90 min | cmd/d2soak/progress.go:86 `NoGain: 90 * time.Minute` | yes, but code also counts XP >= 500 as progress (progress.go:59, 247): spec lists only clvl, quest step, gold: DRIFT (spec incomplete) |
| soak-gates:29 | heap alert 8192 MB or +1024 MB in 5 min | cmd/d2soak/status.go:35 | yes |
| merc-policy:37 | Monastery road no merc 16 / merc 14 | internal/env/tuning_route.go:37-40 defaults 14 / 14; mercpolicy delta says 14 / 14 | spec STALE, delta and code agree |
| merc-policy:37, 59 | Forgotten Tower 16 / 14; Andariel 17 / 15 | tuning_route.go:41-44 Forgotten Tower 16 / 14; Andariel `DefaultAndarielMinLevel*` not opened (U) | Forgotten Tower yes |
| band rule (baselinefree delta, unapplied) | CV floor 0.14, z 1.96 | internal/d2dsoakstat/bands.go:122-123 | yes |
| soak-gates:119 / specdelta | stash-aware funded window, price 30 | internal/d2dsoakstat/invariants.go:162 `TotMax >= PotionPrice` (value of PotionPrice not opened, U) | wording of specdelta matches code shape |
Not checked (U): fidelity-gate numbers (20 %, 5 seeds, 2 h), death-rule numbers (10.2 section citations), town-basics 300/50, lab-rules 25/40 percent limits, gate3gaps 95 percent / 60 lines / 5 GB / 20 GB.

Approval status of numbers (V where the delta says it):
- User-approved, dated in the delta text: soak-gates Gate 1 belt/deaths (user 2026-11-01 and 2026-11-03), G2 floors (plan-v2 approved 2026-10-30; andarielspec user decision 2026-11-04), virtual minutes (user 2026-11-01), gatedelta death budget (user 2026-10-30, now contested).
- Unclear approval: gate3gaps thresholds (`U threshold`, spec text: invPct 95 over 60 lines, 5 GB disk, 20 GB logs, Gate 0.5 smoke 45 wall minutes); baselinefree band rule (cites docs/research/baseline-audit.md, no user date in the quoted lines); Route guard 14/14 (code comment cites "merc decision 2026-11-02", card result not opened: U); progress floors 1000 XP/h, 90 min (spec says `U until the first 24 h run calibrates it`).

## Needs the user

1. CONFLICT, death judging: gatedelta (2026-10-30, user) adds "at most one death per full 6 h" and a 0.15/h Poisson cap to fidelity-gate (not in base); ADR 0005 and base `death-rule` judge deaths by streak, recovery share and rate "not by a fixed count"; abtool (2026-11-02) puts deaths/h back under the 20 percent tolerance. Which stands? Proposed: drop gatedelta's Death budget, Death rate cap and the 3 Proxy rows (DROPPED/SUPERSEDED by death-rule).
2. CONFLICT, Gate 1 deaths and progression: base soak-gates:108 (2026-11-01 wording), baselinefree (band FAIL), specdelta (2026-11-03, baseline file + INCONCLUSIVE pass note). Proposed: apply specdelta; then either fold the 3 band requirements in and cite them, or drop them. Need the user's call.
3. UNSURE, merc-policy "Revive and stop rule": mercpolicy MODIFIED (ratio 0.23 to base) and open m3-merc MODIFIED (ratio 0.81 to base): fold mercpolicy first, then re-check m3-merc's block against it.
4. SUPERSEDED calls to confirm: d2d-default-backend "Game.exe is frozen" and "Original game remains the acceptance target" (ADR 0015); d2d-container-soak delta (container deferred). Also "Leaving d2d is break glass": ADR 0015 does not restate it, keep or drop?
5. Number changes with unclear approval: Monastery road floor 16 to 14 (mercpolicy delta; code already 14); gate3gaps thresholds (all U); soak-gates:16 floors still `U`. 
6. "G2 meaning and floors" wording: the andarielspec delta keeps "stability and economy soak, not a progression soak", which Current truth does not mention; ask whether Gate 2 keeps that label under ADR 0015 or is renamed.
7. d2d-default-backend is `proposed` and unmerged here although the brief calls it archived: confirm before the apply job folds or discards it.
8. Overlap: archived m3-act1chain (ADDED, act1-campaign) and open m3-act1run / m3-andariel (ADDED, same capability): which requirement names are final once act1-campaign gets a base spec?

## Proposed order of edits (smallest first)

1. No change: bot-loop, combat-loop, d2d-farm, licence-policy, process-hygiene, public-split, replay (all deltas APPLIED).
2. early-game-policy: nothing in force to fold (2 open m2 cards stay pending); no edit.
3. upstream-pr: two SUPERSEDED rows need no edit; fix the 2 dead tool references (decide: private ops repo note or removal).
4. lab-rules: fold memrules (Memory access, Crutch rule, Cheated runs and the writes line); optional G1/G2 wording check.
5. dashboard-pub: fold mapitem (2 requirements).
6. town-basics: fold goldfloor (3) and m1-trips (3).
7. backend-contract: fold fix-obj (2), fix-portal2 (3), waits-audit (2); decide "Leaving d2d" row; leave d2d-default-backend pending the user.
8. merc-policy: fold mercpolicy (5 ADDED, 4 MODIFIED incl. Route guard 14/14) and m3-readiness-model, resolve item 3 above; m3-merc stays open.
9. fidelity-gate: fold abtool (2 MODIFIED, 2 ADDED), d2d-replicated-soak (2), m2-latebase (1); resolve item 1; replace "(G1, G2)" in line 43 with Gate A wording.
10. soak-gates (largest): apply in date order d2d-gate-ladder (already in), baselinefree (per item 2), gate3gaps (2 MODIFIED, 2 ADDED), specdelta (2 MODIFIED), andarielspec (1 MODIFIED); rewrite Purpose and the G2 title per ADR 0015; add the `no-gain` XP-progress clause to the progress floors if the user wants spec and code equal.
11. New base spec: act1-campaign from m3-act1chain after item 8 of Needs the user.
12. Leave pending: m1-liveproof (primitives), m2-roam, m2-tripwp, m3-act1run, m3-andariel, m3-merc, d2dmesh, stream-grid.
