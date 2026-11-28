# Survival slice S-D: bot.zoneModel (card survsd, 2026-11-16)

> status: current (TASK-552). Offline: nothing here ran against a live game. Tags: V verified (how), S secondary (a doc), U unverified. Design: [design-2026-11-14.md](design-2026-11-14.md) sections 5.3, 10, 12; gate: [ab-harness.md](ab-harness.md). The knob DEFAULT STAYS 0: the shipped default flips only after the A/B passes and the user says go.

## 1. What the knobs do

| Knob | Values | Meaning |
|---|---|---|
| `bot.zoneModel` | 0 (default), 1 | 0: the plan tables choose the leveling zone, exactly as before. The shadow lines (`zone shadow:`, `zone choice:`) keep running. 1: the survival controller (`survival.Ledger.Choose`) chooses among the plan's zones by margin and expected value. |
| `bot.zoneDial` | 0 cautious, 1 balanced (default), 2 bold | the enter and stay lines of the margin (`internal/power/profiles.json` dials). Read only with `bot.zoneModel` 1. The knob is a number because every knob is; the ids are the table `zoneDialIDs` in `internal/bot/zonemodel.go`. |

## 2. How the choice is made (V: code read, tests below)

- `PickOptions.Model` (`internal/bot/plan.go`) is the hook `Plan.Pick` asks first. It is shown the eligible steps (acts, kit, exclusions already applied); the plan tables stay the candidate list. The model names an area (`ok`), asks for a fresh game (`fresh`) or declines.
- Declining is the fallback, never a stall: a controller error (`zone model: error, the plan tables decide`, one line per distinct error), no candidate the profile covers, an area the plan did not offer, or no hook (no game tables) all leave the plan rule to decide.
- Candidates: the eligible steps plus the zones cleared in this game, marked `Depleted`. The controller's own rule decides when a depleted best zone means a fresh game (`FreshRatio`, `FreshMinutes` in `internal/survival/params.json`, U).
- The log line is the controller's `zone choice:` line (numbers allowed in the log); with the model on the shadow `zone choice:` is not repeated.
- Hysteresis: the controller's stay line, switch margin and ledger step-down apply, and the goal keeps a pick for the dwell `zoneRecheck` (3 min, design 11, U) before it asks again; the pick is also re-asked at once on a level change, a cleared or banned zone, as before.

## 3. Boss floors

The margin of the hero against a boss (`power.BossPack`, `Model.BossWord`: the boss alone as one melee monster, minions not in the pack, U) can replace the Andariel class floor, boss by boss; Andariel is the first and only boss wired (the other floors stay legacy until their data is shown; add a boss by calling `zoneBossFloor` for its key).

- Margin data exists (`danger.Table.FindBoss`, a readable hero): word "safe" makes the level floors met. "risky" or "too hard" keep the legacy floor and switch on the below-recommended warning (`AndarielWarn`): warn, never wall. The margin never raises a floor.
- No margin data, an explicit `boss.andarielMinLevel*` knob, or `bot.zoneModel` 0: the legacy floor, unchanged.

## 4. Merc and fresh games

- Merc dead policy: with the model on, `mercWanted` returns nothing for a "revive" while the latest verdict (younger than 12 min) says the merc is dead and `survival.Ledger.Revive` judged it not worth it (the zone kills him, the kit comes first, or he adds too little). Log: `merc: left dead by the zone model`. Hire is untouched. Reviving is itself off by default (`bot.mercReviveUnverified`), so today this only matters for builds that turn it on.
- Fresh game as a zone tool: when the verdict is `Fresh` (best zone cleared and a new game beats walking to a poorer one) and there is a Cycler and a cleared zone, `Pick` reports nothing eligible; the existing path (`freshGameToLevel`, then the one guarded `startFreshGame` with its in-flight and debounce guard, `freshAllowed` no-gain gap and the thrash cap) starts the game. No new call site.
- Farm loop (`docs/farmloop.md`): untouched. The controller's depleted notion is the planner's cleared set (`loop.cleared`); the farm reset still acts on coverage inside a zone.

## 5. Words only

`/api/survival` (the glance block) carries the margin word (safe, risky, too hard) and the why sentence, and the new field `model: true` when the block is the zone in effect instead of a shadow. `Runner.ZoneMargin()` gives the same word and sentence to the Status line. No number leaves the log.

## 6. Tests (V: run on this branch)

`internal/bot/zonemodel_test.go` (Sorceress and Paladin): model arm picks a lower zone than legacy; error, unknown area, knob off fall back; fresh verdict and depleted candidates; dwell; merc left dead and dial table; boss floor (safe lowers, risky and too hard warn, no data and knob keep legacy); knob defaults. `internal/survival/zonedial_test.go` (both builds): the dial changes the choice and never inverts; no zone flapping over a plateau. `internal/power/boss_test.go` (both builds): boss word ordering. `cmd/d2bot/powerwire_test.go`: no tables, no hooks.

## 7. A/B

`tools/survival/abrun.sh` refuses a binary without the string `bot.zoneModel`; the built `d2bot` now contains it (V: `strings` on a build of this branch). The run itself is the A/B card's job ([ab-harness.md](ab-harness.md) section 4 flip rule); nothing was run here. Not done: per-boss floors beyond Andariel; the `zone.why` in the Status line is words from the controller, the UI sentence of Status is the watch card's.

## 8. Supply cap and the caster FAIL (card zone5, 2026-11-28)

Offline; nothing here ran against a game. `bot.zoneModel` stays 0.

- **What was wrong (I, from [holdout 5](../probes/survival-holdout5-2026-11-21.md) section 4 and the code).** `survival.Economy` prices a zone at `60 / (kill seconds + moveSecondsPerKill)` kills a minute for the whole stay. It never asks how many monsters the zone holds. Burial Grounds holds 38 units, the Crypt 29 and the Mausoleum 26 (V: `internal/danger/areaunits.json`, 12 seeds), against 198 in the Cold Plains and 153 in the Stony Field. At the model's kill rate those three are empty within about four minutes, yet their prior (150 to 165 xp/min) was held for the 20-minute stay, so they were the caster's first pick on every seed and realized 4 to 11. The Paladin margin arm spent its minutes in areas 4 and 10 (V holdout 5 section 3).
- **The rule.** `supplyMinutes` (global and per archetype row in `internal/survival/params.json`; 0 = off): with the controller deciding, the kill rate behind a zone's prior is capped at `units / supplyMinutes`. The capped prior is what the blend, the yield-starved rule, the unvisited discount and the `(model N)` part of the `zone choice:` line use. `Candidate.Units` comes from the danger table (`Density.Units`); 0 = unknown, no cap. The shadow line is unchanged.
- **Rows (data, no class in Go).** `caster` 20 (= `stayMinutes`, I, U until a screen). New rows `summoner`, `ranged_physical` and `hybrid_tank_caster` carry the same value: these archetypes had no row and no A/B pair (Necromancer summoner; Amazon bow and javelin; Hammerdin from the hammer on), and the Act 1 route of `docs/sme/play-act-route.md` (S: Cold Plains, then Stony Field, Dark Wood, Black Marsh) names fields, not graveyards. `bruiser` (Paladin until the hammer, Barbarian) is left as measured: the Paladin PASS was run without the cap. Druid and Assassin builds resolve to `caster`.
- **Tests (V: run on this branch, fail without the rows).** `internal/survival/zone5_test.go`: a 30-minute choose-and-observe loop over the five Act 1 zones with the holdout 5 priors and the measured units; the Sorceress with the shipped rows opens in a field and spends no minute in a graveyard (12 of 30 without the rows) and earns more; the Paladin run is identical with and without; the cap formula for both builds; every power archetype has a survival row.
- **Not isolated offline (U).** Holdout 5 names two more suspects on the three worst seeds: `k` at the 3.00 clamp and the merc left dead 15 to 26 times a run. Whether the graveyard opening caused them (slow, mana-starved kills there feed the hero-wide kill-time ratio) cannot be told from the report. Diagnostic added: a deciding `zone choice:` line with `k` other than 1 ends in `k parts: zone Z hero H`, so the next screen says which factor is pinned.
- **Lab checklist (not run).** Worker host, night window, capacity checked first. Three fresh Sorceress seeds, both arms, the setup of holdout 5 section 1 (speed 3, horizon 1.5 virtual hours, clvl 12, 500 gold, wand):
  1. Build `cmd/d2bot` from this branch into its own directory; name the lab d2d with `D2D_BIN`.
  2. `for s in 331 333 335; do for arm in legacy margin; do AB_CLASS=4 tools/survival/abrun.sh $arm $s 1.5; done; done` (or the same seeds through `tools/survival/wave.sh`).
  3. `go run ./tools/survival/abreport -base legacy -arm margin -death-slack 1 -chicken-slack 1 -xp-floor 0.95 -horizon 90 <AB_DIR>`.
  4. In each margin log: the first `zone choice:` must not be Burial Grounds, Crypt or Mausoleum; count `k parts:` lines with `hero 3.00` against those with `zone 3.00`; count `zone revive: merc_left_dead`. Every log must say `writes after start: 0`.
  5. PASS at 3 seeds is a screen only; the flip needs the 8-seed gate of [ab-harness.md](ab-harness.md) and three more Paladin pairs.
