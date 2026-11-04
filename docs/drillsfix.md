# drillsfix: stash (DS) and tome (TM) drill repairs

Evidence: the first live run (d2d, slot 3): DS.1 PASS, DS.2 FAIL, DS.4 BLOCKED, DS.3 BLOCKED, TM.a BLOCKED.

## What changed (offline, fake-lab tests; nothing run live)

- DS.2 / DS.4 / DS.1: the chore plan is now `potions gold stash` (const `stashPlan`). The `withdraw` step is enabled by
  `plan.Potions && plan.Gold` (V: internal/town/chores.go, step list), the drill had no `gold`, so it never withdrew. The fake
  lab now only withdraws when both words are in the plan.
- DS.4: the rejuvenation goes into the pack with `debug give <code> pack` (V: docs/forkcheats.md; new `lab.GivePack`)
  instead of ground give plus pickup, which put it into the belt. The fake lab models that belt behaviour.
- DS.3: unchanged, BLOCKED (needs `TripLab.RunnerRounds` and a way to make the shop unable to sell).
- TM.a: setup gold is `lab.TomeGold()` = tbk price + 13 x tsc price + `MinGoldReserve` (data.BuyPrice at vendor mult 1024), not
  1000. The drill also sets `TPScrollTarget` to 13 (S: shop.go caps a tome at target 5 by default, so gold alone gives 5
  charges) and restores it afterwards. Cleanup now returns to town (`Goto` level 1) even on failure.
- TM.b / TM.c: unchanged, BLOCKED: need a verb/option to make the producer use the stock `items` verb for one look, and a fault
  injection verb that makes `dump items` fail for one look.

## Rerun (live, d2d; V: flag names read from cmd/d2drill/main.go)

`d2drill` refuses to start outside `tools/d2dslot.sh` (it needs `D2D_SOCK` and `D2D_SLOT_DIR`; the slot comes from the
slot tool, not from a `D2D_SLOT=3` prefix). Use the run script of docs/livecheckdrills.md, which wraps it; the flags it
passes are `-group` (DS or TM), `-only` (comma list of drill ids), `-repeat` and `-label`:

    docs/probes/livedrills2/2026-11-04-run.sh LABEL GROUP ONLY REPEAT
    docs/probes/livedrills2/2026-11-04-run.sh drillsfix1 DS DS.1,DS.2,DS.4 2
    docs/probes/livedrills2/2026-11-04-run.sh drillsfix2 TM TM.a 1

TM.a needs a longer limit than the script's `--ttl 200` (copy the script, use `--ttl 2400 -t 2300`; U).
