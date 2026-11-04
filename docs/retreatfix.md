# retreatfix: RT.drink setup life and RT.portal bound

Fixes from docs/retreatlive.md (V: live run, d2d).

- RT.drink sets life to the midpoint of `ChickenPct` and `DrinkLifePct` (default 25 and 55 gives 40), derived from `env.CombatTuning`, never a literal. Test: `TestRetreatDrinkLifeBetweenKnobs` (default, wide and tight knob sets).
- RT.portal effect bound is 10 ticks (`portalMaxTicks`); live measured 1 tick.
- RT.leave stays Blocked (Fighter.Leave not wired). No bot change.

## Live rerun (not yet done, U)

Same recipe as docs/probes/retreat/2026-11-04-RT-run.sh (retreatlive branch): build `go build -o /tmp/retreatfix-d2drill ./cmd/d2drill`, then on the worker host
`D2D_AGENT=retreatfix D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3 tools/agentrun.sh retreatfix --ttl 420 -- tools/d2dslot.sh run -n 3 -t 400 -- /tmp/retreatfix-d2drill -group RT -only RT.drink -repeat 2 -label retreatfix -out /tmp/retreatfix-out -bin $HOME/d2d-lab/fork/bin/d2d.lab-cheats3`
and again with `-only RT.portal`.

Expected: RT.drink PASS, decision drink-life, belt 4 to 3, life up, identical in both runs; RT.portal PASS, effect at 1 tick, scroll 1 to 0.
