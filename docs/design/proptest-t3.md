# Property tests on the pure deciders (T3)

> status: current (TASK-980, card proptest1). Roadmap row T3, docs/research/fable-logic-review-2026-11-25.md section 5.3.

Random inputs from a fixed-seed `math/rand/v2` PCG generator (one seed per build), standard library only, no clock, no
wall-clock ceiling. Each property runs for two builds: a caster-shaped input (`sorc`: small life pool, large mana pool,
mana need 2, count-rule cast guard) and a melee-shaped one (`pal`: large life, small mana, mana need 0, damage-term cast
guard, larger potion floor). The builds are input shapes only; none of the three deciders takes a class.

| package | file | properties |
|---|---|---|
| internal/combat | decide_prop_test.go | every action is feasible (a drink names a ready potion in the belt, a portal needs a scroll, a retreat a way out, nothing in town or dead); at or below the chicken line the answer is always a rejuvenation, a healing potion out of a fight, or chicken (a reflex always wins); no exposed portal cast in a fight; one more potion never turns a non-chicken into chicken; deterministic |
| internal/supply | solve_prop_test.go | books (spent at most the budget, spent is the sum of line costs, no negative buy, deterministic: invariant T8); the verdict matches the lines (ready covers targets, ready-lean covers mins, short-gold or blocked names an open min line); one more healing, mana, rejuvenation, scroll or gold never worsens the verdict |
| internal/arbiter | decide_prop_test.go | over random runs of 200 ticks (random sources, reports, area changes, refusals): player and reflex intents are never refused; every proposal and every live refusal intent is chosen or rejected, and nothing else is; a refusal whose Until condition holds, or whose TTL ran out, never comes back; nothing is chosen only when nothing is admissible; no admissible candidate of a higher class loses (a reflex always wins, the lease never holds against a higher class); a candidate outranked continuously for StarveSec is flagged starved exactly once per episode and never earlier; deterministic |

Verified (V, offline): each property was checked against a one-line mutation of the decider it guards (sort order
reversed, starve limit shifted, Until check removed, player intents refused, cast guard removed, readiness check
dropped, chicken-line drink allowed in a fight, overspend by one unit, ready-lean folded into ready); every mutation
failed its property. Larger runs (300 000 combat inputs, 100 000 supply cases) found nothing.

Finding (not a bug today): `supply.Solve` is monotone in gold only for a policy that spends every min and one step
before any target step. A custom `Policy.Priority` with a target step before a min step can trade a min line for a
target as gold grows (more stash gold turned ready-lean into short-gold). `DefaultPriority` has that shape and no
caller passes another order, so the generator keeps mins before targets. A check in Solve or in the policy loader
would make the rule explicit.

Next: the admission function of P4 once admit1 lands (one package per PR, card row T3).
