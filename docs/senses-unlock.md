# Senses unlock ladder (TASK-149, sensesunlock)

The seven senses tiers of `docs/research/info-unlocks.md` as entries of the shared ladder (`docs/unlock-ladder.md`, engine `internal/unlock`). Tags: V offline test, U not run live.

## Table (`data/unlocks/ladder.json`, group `senses`)

| tier | id | opens on | price |
|---|---|---|---|
| 1 minimap | senses_map | first Codex area first, or clvl 2 | |
| 2 enemy dots | senses_dots | first Codex monster first | |
| 3 bars and names | senses_bars | first Codex monster first (per kind: `LadderUnlocker.Met`) | |
| 4 elite markers | senses_elites | first Codex champion first, or a first of Corpsefire, Bishibosh, Rakanishu, Blood Raven, Treehead, Griswold, Countess | |
| 5 loot markers | senses_loot | buy, or first Codex drop first | 40 |
| 6 danger rating | senses_danger | buy | 80 |
| 7 hero overlays | senses_hero | clvl 12, or the Andariel first | |

Prices are U (a first guess, to be tuned with the tally rows; the info-unlocks note leaves the number to the ledger registry). The superunique list is S (standard D2 superuniques; only ids that exist in `internal/decide/firsts_act1.json`). The table file is `data/unlocks/ladder.json`, not the `internal/watch/senses_tiers.json` the card named: the ladder is shared, so the senses rows live beside the others.

## Behaviour

- `watch.LadderUnlocker{Engine, Cap}` is a `watch.Unlocker`: tier n is allowed when its entry is open. Tiers open independently (tier 4 can precede tier 2 if a boss is the first monster).
- Precedence with knob `senses.tier`: 0 means no panel (no `Senses` is mounted). Without a ladder the knob is `FixedTier(n)` as before. With the ladder, `Cap` (0 = none) is a maximum on top of it, meant for an explicit override; the knob's default 1 is NOT passed as a cap (it would pin the ladder to tier 1), so wiring passes `Cap` only when the tuner value was set by hand (U: the tuner cannot tell yet; see "Not done").
- `GET /api/senses` returns `tiers_unlocked`, the data of the open tiers (`map` and `hero` with tier 1, `units` with tier 2) and `locked` (id, tier, title, teaser, price; no spoiler text) plus `next` (the lowest locked tier). A locked tier's data keys are absent from the bytes. Tier 3 to 7 have no data keys yet (TASK-151, TASK-152 and later cards add them); the gate is ready for them.
- Reveal and return screen: the generic ladder reveal (Chronicle line `Unlocked: <title>. New sense: ...`, section "Unlocked since you left"). Opening a tier raises no question.
- Tiers 5 and 6 are bought with `POST /api/unlocks/buy {id}`; the ledger books a `spend` entry (key `unlock/<profile>/<id>`).

## Verified

V offline (`go test ./internal/watch ./internal/unlock`): golden JSON of the reply at 4 ladder states (`internal/watch/testdata/senses-ladder-step0..3.json`, tiers 0 to 4), locked list and next, Cap as a maximum, every trigger of the table, buying tiers 5 and 6 (short refused, accepted, ledger balance, second buy refused), the whole path from funStream Codex firsts for Sorceress and Paladin. U live: no game run; the tier 2 `units` come from `senses.Snapshot.Unit`, which nothing fills yet (TASK-151).

## Not done

- `cmd/d2bot` still mounts `FixedTier(l.sensesTier)`. Wiring: open the engine (`unlock.Shipped`, `unlock.Open(stateDir, profile, ...)`), pass it as `FunConfig.Unlock` and mount `Lock: watch.LadderUnlocker{Engine: eng}`.
- Acceptance #4 of the card (tier 5 and 6 spend rows in the currency registry): the registry (`internal/currency/registry.json`) is not owned here; the spends are booked with upgrade `unlock` and the price in the ladder. The tallystrip owner decides whether they become registry rows.
