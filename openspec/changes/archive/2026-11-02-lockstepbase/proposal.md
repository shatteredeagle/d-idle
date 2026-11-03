## Why

The user approved (2026-11-02) a paired experiment: baseline lanes with the clock hook at k=2 and `D2BOT_TIMESCALE=2` against the k=1 lanes of set `kit`. The baseline tooling had no notion of a scaled lane: no hook install and restore, no environment forwarding to the bot, no game-time conversion, no record of the regime.

## What Changes

- Public repo: `gclock.StartNote` and one log line in `cmd/d2bot` (`gclock: timescale K`, only when K is not 1) that the hooked driver checks; runbook section 10 in `docs/process/baseline-runs.md`.
- Ops repo (private, not in this card's diff): `baseline-run.sh --timescale K`, hook install and restore, watchdog, game-time series, regime flag, tests.
- Finding: the pin of set `kit` (`b023979a`) has no `internal/gclock`, so a scaled lane needs a newer pin.

## Impact

Docs and a one-line log addition to the bot; no behaviour change at K=1 (the line is not printed). No spec delta, no game, no hook source or addresses in this repository.

Overlap (declared): `internal/gclock` and `docs/timescale.md` belong to the archived card `botclock`; this card adds one exported function and its test, nothing else there.
