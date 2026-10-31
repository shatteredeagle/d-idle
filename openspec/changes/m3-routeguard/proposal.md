## Why

The route guard (merc-policy spec) blocked the Monastery road and the Forgotten Tower below clvl 20 without a merc, and the Andariel floor was 15 (+2 with a dead merc); the 9 road deaths were at clvl 11-12 with a 2-potion kit and the d2-sme disputed the 20 (docs/sme/audit-bot-vs-human.md A1). Grill 0003 settled interim numbers in [ADR 0011](../../../docs/adr/0011-readiness-power-vs-danger.md) (accepted 2026-10-31); this card sets the knob defaults to them and carries the MODIFIED deltas.

## What Changes

- internal/env/tuning_route.go, tuning_boss.go: Monastery road and Forgotten Tower floor 16 with no merc and 14 with a living merc (new with-merc knobs), `route.towerNeedsMerc` default 0, Andariel floor 17 with no merc and 15 with a living merc; the Andariel +2 dead-merc rule is removed.
- Andariel kit gate gains at least 2 antidote potions; the supply kit gate stays on every guard.
- docs/knobs.md regenerated; internal/bot and internal/task floor tests updated (TestPickNeverEmptyInAct1 and the route guard tests).
- Delta: merc-policy MODIFIED `Route guard` and `Andariel floor` with the ADR 0011 numbers.
- Interim only: the floors become a safety net once card m3-readiness-model is calibrated (model at most 2 clvl below them).

## Dependencies

After xpgap is archived (it owned docs/knobs.md; check changecheck). Does not wait for the readiness model. Gates and triggers from ADR 0011: (1) m3-andariel and m3-act1run confirm or move the floors; (2) road floor trigger: if road deaths fail the death rule (ADR 0005, soak-gates) at 16 in m3-act1run, raise it to 18 and re-grill, if m3-act1run shows no road deaths try 14; (3) no unattended Andariel attempt before m3-andariel's throwaway fight has won once.

## Open questions (parked by grill 0003)

- Max-life floor for Andariel: waits for m3-andariel's life-per-phase numbers.
