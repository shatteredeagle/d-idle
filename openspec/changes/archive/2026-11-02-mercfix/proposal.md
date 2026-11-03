## Why

Gate 2 (20261102) logged `merc: claimed free rogue` at clvl 7 and the route guards read `merc=none` for the rest of the 8 h run: Andariel and the Monastery stayed closed, the character sat at clvl 15 to 16 for 960 virtual minutes (docs/research/gate2-analysis-20261102.md fix 1).

Root cause (V: the run's bot log, 08:31:09 to 08:31:17): `task.MercAlive` is set only by the merc policy (`merc.enabled=1`, default 0; 0 lines `merc policy on`), so the guards used the unit table, where the Rogue is listed only near the player: `merc=alive` once in town, `merc=none` 5 s later. The re-fired claim got `granted no merc (burial 5 -> 5)`: the reward was spent and the Rogue gone or dead (U which).

## What Changes

- cmd/d2bot/d2dwire.go + mercwire.go `guardMercHook`: the route guards read the dump's merc section (merc.present, alive) also when the merc policy is off, so a merc that is owned and alive but not in the unit table counts as present; unknown (read error, no section) leaves the unit table in charge.
- `claimDep`: after a grant, read the merc back (4 tries, 500 ms): log `merc: claim verified` or `merc: claim not seen after 4 tries` and return an error so the step retries (max 3 unseen claims, then stop). An owned merc (dead or alive) makes the policy skip the claim, ending the pointless Kashya loop.
- docs/botlogs.md: one paragraph for the lines.
- Not done (needs the user): a dead Rogue still cannot be replaced for free; reviving needs `merc.enabled=1` (revive U on every game) or a paid hire, both deliberate policy choices.

## Overlap (declared)

cmd/d2bot/d2dwire.go is touched by one line group (hook install) next to the claim wiring.
