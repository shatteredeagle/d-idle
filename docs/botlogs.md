# Countable guard lines

Soak extractors (tools/ops gate judge, d2dsoak) count guard events by `msg=` in the bot log (slog text format). One line per event, never repeated for the same event; behaviour is unchanged by the logging. Status: offline tests only (internal/bot/botlogs_test.go), U live.

| msg | when (exactly once) | attrs |
|---|---|---|
| `explore-giveup` | the second failed explore in a quiet area with no kill in between; the area is then marked cleared. Quest and boss runs never emit. The older human line `area abandoned: exploring failed with no kill` still follows. | `area`, `failures` (2), `kills` (kills this game so far) |
| `tp-debounce-hold` | a run of false portal-source looks ended in a true look before `tpDebounceSpan` (2 virtual minutes): the debounce kept the verdict `true`. Written when the run ends. | `verdict` (true), `raw` (true: the look that ended the run), `looks` (false looks in the run), `held_s` (virtual seconds from first false look to the true one) |
| `tp-debounce-flip` | the false run reached the span and the verdict became false. Written on that look. | `verdict` (false), `raw` (false), `looks`, `held_s` (>= 120) |
| `kitgate-trip` | a finished town trip whose reason holds `kit: ` (counted by `logTrip`, next to `trip budget: trip done`). | `level`, `missing` (text after `kit: `), `gold` (after the trip), `trip` (number since the kit was last complete) |
| `kitgate-skip` | from KitGateLevel (clvl 8) up: the cap `kitTripCap` of kit trips is reached with the kit still incomplete; once per stretch, reset by a complete kit. Never below clvl 8, where the old hold applies. | `trips`, `missing`, `gold`, `level` |
| `inv-tp-dump` | unchanged (docs/hastpdump.md). | |
| `town: resupply budget hit` | a Chores call ran out of `choreBudgetS` inside a buy loop (potions or scrolls); once per Chores call, the purchases so far stay (docs/townshop.md). | `budgetS`, `spent`, `bought` |

Counting: `grep -c 'msg=explore-giveup '`. A hold line is a debounce that saved a false verdict; a flip line is a verdict change that the debounce let through. Not every false run is logged: a run still open when the log ends has no line.

Merc claim (cmd/d2bot/mercwire.go `claimDep`): `merc: claimed free rogue` is followed by a read-back of the dump (merc.present); `merc: claim verified` when the merc is seen, `merc: claim not seen after 4 tries` when not (the step errors and retries, at most 3 times). Mercfix card, root cause in docs/research/gate2-analysis-20261102.md fix 1.

Merc policy lines (card mercpolicy, formats in internal/merc/logs.go, tested in internal/merc/logs_test.go): `merc: revived price=<paid> level=<n> estimate=<formula>` (price is the gold the revive took, the read value; estimate is the S formula pre-check, compare the two on the next live revive); `merc: hire window row=<i> name=<n> class=<id> level=<n> aura=<a> price=<gold> gold=<purse> clvl=<n>` (one line per row, once per difficulty, read-only: the live probe that answers the hire price/level/aura question); `merc: strip slot=<slot> item=<name> cells=<n>` (before a replacing hire); `merc: hired class=<id> aura=<a> price=<paid> gold_before=<g> gold_after=<g>`; `merc: hire skipped (<reason>)` and `merc: hire window unreadable (<err>)`; `merc: gold farm on|off need=<gold> gold=<purse> clvl=<n>` (the early-areas gold farm while the merc is missing, once per change).

Merc field rules (card mercwire, cmd/d2bot/mercwire.go and internal/combat/mercfield.go; only with `merc.enabled` 1): `merc: waiting for the merc why=portal|waypoint` then `merc: wait over waited_ms=<n> near=<bool>` (after a portal return or waypoint hop, at most 6 s; near=false means the wait timed out); `merc: down in a crowd, leaving action=TownPortal|CrowdRetreat crowd=<n>` (at most once a minute); `merc: gave healing potion col=<1..4> merc_life=<pct>`; `merc: potion failed col=<n> err=<e>`; `merc: potion wanted (no verb)` (at most once a minute, the input has no `Belt(col, merc)` verb); `merc: first-visit flag not saved (<err>)` (the state file could not be written).
