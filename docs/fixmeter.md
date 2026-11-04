# fixmeter: combat totals on the d2d backend

> status: offline fix, live proof pending (card [fixmeter](archive/openspec-index.md#fixmeter)). Closes the P28 "Meter totals BLOCKED" note of DRILLS5 ([primitives.md](primitives.md)).

## What was wrong

`d2dcombat.New` built a `Meter`, `d2dcombat.Attach` (the bot and the drill harness) did not: `Rig.Meter` was nil, so no drill could read totals. The Meter also only knew the player's life and mana.

## What the Meter reports on d2d

`rig.Meter.Totals()` and `Totals.String()`:

| Field | Source | State |
|---|---|---|
| `Damage` (taken) | sum of drops of the player's life between samples (`state`, life cur) | derived; healing and potions are not netted, as in `combat.FightTelemetry` |
| `ManaSpent`, `Deaths`, `MinLife`, `Samples` | player `state` | derived (existing) |
| `DamageDealt` | sum of drops of hostile monsters' life between samples (`monsters` hp cur/max, via the snapshot), capped at the last life read alive when the monster dies | derived, all sources (the player, minions, traps, missiles of other monsters), not attributable to the player alone; overkill is not counted; monster regeneration is not subtracted |
| `Kills` | hostile monster seen alive, then seen in death mode; one that vanishes from the table is not a kill (the same rule as `combat.noteKills`) | derived |
| hits, misses | the d2d state carries no swing result | not available: printed `n/a`, there is no field to read a fake 0 from |
| `DealtOK` | false until a snapshot with monsters was sampled, or when no monster source is wired | when false, dealt and kills print `n/a` |

Friends (town NPCs, mercenary, summons) are filtered with `combat.IsFriendly`, as the Fighter does.

## Wiring

- `Attach` builds the Meter around the backend clock (`e.Clock`) and installs it as the combat Env's clock, so every combat sleep samples (one `state` verb and one snapshot read). It starts OFF: it forwards `Now`/`Sleep` and reads nothing, so the bot and the soak keep their per-sleep cost. A drill or probe calls `rig.Meter.Enable()` (it also forgets the last sample); call `Reset()` after a cheat that sets life.
- `New` (cmd/d2dfight) is unchanged: the Meter is the clock, with no monster source, so dealt and kills are `n/a` there unless `Mons` is set.
- Soak report: not changed. It sums the `fight ...` telemetry lines of `combat.FightTelemetry` (kills, damage taken), which already work on both backends; this Meter is for drills and probes. The report has no damage-dealt, hits or misses column, so there is nothing to print as `n/a` there.

## VERIFIED vs UNVERIFIED

- VERIFIED (offline): the totals of a scripted fight (`TestMeterFakeFight`), n/a when unavailable, the off Meter only forwards, `Attach` wiring (internal/d2dcombat/meter_test.go).
- VERIFIED LIVE earlier: `monsters` carries hp cur/max (docs/d2d-backend.md) and the player's life drops are read the same way by `New`'s Meter (DRILLS5 D5).
- UNVERIFIED: that monster hp from `monsters` is whole points that fall by the real damage (the table says "V" for the field, not for a damaged monster under fire); the sample interval (one per sleep) misses damage between samples only if life is restored in between; that a dying monster's mode shows in the snapshot before it leaves the table.

## Live drill recipe (a DRILL agent, offline lab rules apply)

1. Attach a rig as `cmd/d2drill/live.go` does; `rig.Meter.Enable()`.
2. `debug spawn` a pack (for example 20 zombies, as D5.p28) near the character, record `monsters` hp for each id (ground truth: sum of start hp).
3. `rig.Fighter.Clear` until the pack is dead or a timeout; note `res.Kills`.
4. Compare: `Totals.Kills` against `res.Kills` and the count of spawned ids seen in death mode; `DamageDealt` against the sum of (first hp read minus last alive hp read) per id (expect equal up to the sample granularity, no more than the spawn hp sum); `Damage` against the player's life drops from `state`.
5. Pass: kills equal, dealt within the spawn hp sum and above 0, taken matches; any mismatch: record it in the drill README and fix the derivation here. Tag the result VERIFIED LIVE only then.
