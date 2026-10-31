# navsoak2: second hidden soak, same seed, on current main (2026-10-31)

> status: handoff for the analysis agent. Soak started 2026-10-31 02:58:35 local (worker host clock), hard end 06:58:35 (240 wall minutes; registry TTL 15000 s), slot 1, agent `navsoak2`.

## Built from
origin/main d82bd47 (bot binary `bin/d2bot.navsoak2`, sha256 prefix a5d1b39d237c, d2d `d2d.lab6`). Proof, run on origin/main:
- KITGATE: `git grep -n recoveryKitGate origin/main -- internal/env` printed nothing (knob lives in internal/resource); found in `docs/knobs.md:320` (`resource.recoveryKitGate`, default 1) and `internal/bot/kitgate.go:34`.
- M1LOOT: `git grep -n tpScrollPotions origin/main -- internal/env` hits `internal/env/tuning_town.go:113` (also `potionsFirst`, line 110).
- nav fix: `git merge-base --is-ancestor 85b429d origin/main` true.

## What it is
Same recipe as docs/probes/navsoak/README.md (branch agent/navsoak, cfde63b): fresh uncheated Sorceress, seed 5, `--backend d2d --managed --d2d-class 4 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --goals campaign`, `D2D_RUN=on`, nice 10, no viewer. Differences: character `Navy2`, instance `navsoak2`, supervisor `bin/navsoak2-sup.sh` (up to 3 restarts with `--d2d-class -1 --clean-start`), started with `D2D_AGENT=navsoak2 bin/d2dslot.sh run -w 600 -t 15000 -- bash bin/navsoak2-sup.sh`.

## Where (worker host, relative to `~/d2d-lab/navsoak2/`)
`logs/supervisor.log`, `logs/bot-0.log` (then `bot-1.log`..`bot-3.log` after restarts; last line `exit <code>`), `data/` (saves), `bin/`.

## Analysis (read-only)
```
cd ~/d2d-lab/navsoak2
cat logs/supervisor.log
bin/agentreap.sh --list
grep -c 'combat no progress, holding' logs/bot-*.log
grep '"msg":"minute"' logs/bot-0.log | tail -3     # clvl, kills, deaths, potions, gold, area
grep '"msg":"pace"' logs/bot-0.log | tail -1
grep -E 'died|death|failsafe|panic|runner loop made no progress' logs/bot-*.log | tail
# A/B against ~/d2d-lab/navsoak/logs/bot-0.log (same fields): compare minute lines by "virtual" time, not wall time
```

## Stop
`tools/agentreap.sh navsoak2` (worker host; takes supervisor, bot and d2d down, frees slot 1). Never touch the `navsoak` entry.

## A/B note
navsoak (279a163) has nav fix, town basics and champion caution but NOT KITGATE or M1LOOT. Seed 5 is the same, so early behaviour should match until the first decision that the new changes alter; the first divergence point is meaningful. Compare: the death at clvl 7 (Burial Grounds, no kit) at the same virtual time or not; recovery area choices (Blood Moor and Cold Plains only when the kit is missing); scroll and potion purchases (potions first, then the first town portal scroll); gem sale and charm keeps. Lockstep is deterministic only per binary and load is not part of the state, but treat equal-minute comparisons as approximate.
