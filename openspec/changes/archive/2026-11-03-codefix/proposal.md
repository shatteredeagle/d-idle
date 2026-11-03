## Why

The independent review docs/research/codereview1.md found side effects in the Gate 1 belt fixes: the widened walk-home list also drove NearTown (F1), stash gold broke the futile-trip guard (F2), the mana reserve applied to a complete kit (F3), the chain walk was not limited to the route (F4), recovery capped the visit the withdrawal pays for (F5).

## What Changes

Fixes for F1 to F5 and F8, each with a test that fails when the fix is reverted. No knob or spec change. See docs/research/codefix.md.

## Impact

cmd/d2drill/portal_copy.go (verbatim copy refresh), internal/resource/town.go, types.go, internal/bot/resource.go, goal.go, kitgate.go, internal/town/trip.go, shop.go, chores.go, basicsfix.go, town.go, cmd/d2bot/d2dwire.go and their tests.

Overlap (declared): these source files belong to other (merged) cards or to no card in docs/OWNERS; each edit is a few lines and no signature other than the new `ChainHome` input field, `ChainHomeArea` and `manaReserve(s)` changes. AREAREACH (agent/areareach) owns internal/bot/task.go and the planner Pick code, which this card does not touch.
