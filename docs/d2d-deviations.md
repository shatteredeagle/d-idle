# d2d deviations register

status: draft 2026-11-04 (job shimregister); checker: `python3 -I tools/shims/check.py` (not in CI yet)

## Policy

1. game.exe 1.14d is the spec; d2d is the end product and is measured against it.
2. A d2d behaviour that differs from game.exe is a d2d bug, to be fixed upstream.
3. Until then the bot may carry a SHIM: temporary, listed here, with an upstream removal path.
4. A MODEL row is a legitimate switch (a capability or wiring difference that is not a bug).
5. Adapters (`internal/d2d*`, `cmd/d2d*`, `cmd/d2bot/d2d*.go`, launch/setup/drill tooling) may know d2d; decision code (combat, bot, town, nav, resource, merc) may not, except through a row here.
6. A new d2d-specific branch outside the adapters without a row fails `tools/shims/check.py`.

Tags: V verified (how), S secondary, U unverified. The PR numbers #146..#195 of docs/research/openpr-log.md cover monster/missile/curse/armor work; none of them is a cast-timing, mana, belt or waypoint-arrival fix (V: grepped the log). `fix/lightning-sequence` and `fix/cast-ticks` have branch names and commits in docs/research/cast-trace-gameexe.md and docs/research/upstream-prs.md (#128 for cast-ticks); a PR number for lightning-sequence is not in the repo (U).

## Register

Key = `file:identifier` as the checker sees it. "Last verified" = date the row was read against the code (2026-11-04), not a live run.

| id | area | game.exe rule | d2d behaviour | code (keys) | kind | upstream PR / status | removal condition | last verified |
|---|---|---|---|---|---|---|---|---|
| R01 | cast length, Lightning / Chain Lightning | anim SQ with seqtrans SC: 19 steps at the cast rate (V: docs/research/cast-trace-gameexe.md, trace of 006216e0) | plays them as plain SC casts, ignores seqtrans (13 ticks, not 19) (V-d2d per castgap.go comment) | `internal/combat/castgap.go:d2d` `internal/combat/castgap.go:D2D` `internal/combat/castlog.go:D2D` | SHIM | branch fix/lightning-sequence 2036fd3f; PR number not in repo (U) | lab binary has fix/lightning-sequence: drop the `d2d` test in castTicks (comment there says so) | 2026-11-04 |
| R02 | cast gap carrier | gap = cast length + margin | Fighter.D2D flag selects R01 and R03 | `internal/combat/fighter.go:D2D` `cmd/d2bot/behaviors.go:D2D` | SHIM | with R01, R03 | delete the field when R01 and R03 are gone | 2026-11-04 |
| R03 | mana payment timing | paid at the action frame (8 ticks, Fire Bolt L1, no FCR), refunded when the cast is cut first (V read: docs/research/sk-mana-trace.md, traces D1, D3) | pays 1 tick after the command, also for a cut cast (V-d2d DRILLS13 M1, DRILLS15 M2: 7 of 7 paid) | `internal/resource/manamodel.go:BookD2D` `internal/resource/manamodel.go:BookGameExe` `internal/combat/hard.go:D2D` `internal/combat/hard.go:BookD2D` `internal/combat/hard.go:BookGameExe` | SHIM | none found in the log (U); candidate for a new upstream PR | d2d pays at the action frame with refund: delete BookD2D and the branch | 2026-11-04 |
| R04 | belt fill on pickup/purchase (test sim only) | a potion clicked on the ground goes to the belt first, no modifier (V: docs/research/belt-pickup-trace.md P7) | lands in the pack, never in the belt | `internal/town/sim/sim.go:NoAutoBelt` | SHIM | none found (U) | d2d auto-belts like game.exe: delete the flag | 2026-11-04 |
| R05 | arrival inside a warp trigger | not observed as a problem (U) | arrival can land inside a cave mouth's trigger area (docs/d2d-loop.md, S) | `internal/bot/config.go:MouthGuard` `internal/nav/nav.go:mouthGuard` `internal/nav/walk.go:SetMouthGuard` `internal/nav/walk.go:mouthGuard` | SHIM | none found (U) | d2d spawns outside the trigger area; drop the config key (default off already) | 2026-11-04 |
| R06 | waypoint coordinates, acts 3 to 5 towns | act-absolute positions | level-relative; the bot adds the level origin (V live DRILLS4 per act2hints.go comment) | `internal/nav/act2hints.go:knownWaypointRel` `internal/nav/waypoint.go:knownWaypointRel` | SHIM | none found (U) | d2dworld reports act-absolute positions: move the table into the adapter or delete it | 2026-11-04 |
| R07 | backend selection (cmd wiring) | n/a | `--backend d2d` picks the d2d environment | `cmd/d2bot/main.go:d2d` `cmd/d2bot/behaviors.go:d2d` | MODEL | n/a | none (this is the seam) | 2026-11-04 |

## Seen only in comments or data (not checked by the script)

| id | where | what | kind (proposed) |
|---|---|---|---|
| C01 | `internal/town/beltfill.go` (~232) | d2d has no click modifiers: ErrUnsupported falls back to the drag path | MODEL (capability, handled via an error) |
| C02 | `internal/bot/observe.go` (~125, 253), `goal.go` | d2d shows no menus: recycles and saved exp counted without a menu poll | MODEL |
| C03 | `internal/bot/task.go` (~160), `loop.go` (~837) | env clock is virtual under d2d lockstep, wall on game.exe | MODEL |
| C04 | `internal/env/tuning_nav.go` (~123) | arrival cooldown knob; game.exe has none; d2d dropped its own in #153 (S, string only) | SHIM already deprecated, default 0 |
| C05 | `internal/env/tuning_combat.go` castPerSkill description | Lightning cast gap 760 vs 650 ms: part of R01 | SHIM (R01) |
| C06 | `internal/resource/tuning.go` recoveryXPPick | gold per kill is the same everywhere "(d2d)" (U whether game.exe agrees) | UNSURE |
| C07 | `internal/town/trip.go` (~255) WalkingInteractor | d2d verb walks to the target itself | MODEL |

## Unregistered (UNSURE)

- Hit recovery cuts, potion-over-time and other behaviours named in the brief: no branch on the backend found in decision code with the checker's identifiers (U: they may be handled inside the adapters, or by a name this script does not know). Not claimed absent.
- `internal/nav/roam.go:22` "U: not measured on d2d" and `internal/combat/portal.go:13` (tome quantity stat 70 only when qty > 0): prose only, probably adapter-side behaviour (UNSURE).
- `internal/merc/*`: mentions d2d only in comments; not classified.
