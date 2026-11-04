# Kill credit and hits per cast (change killcredit)

Offline only, not verified live. Logging only: `TestCastLogDoesNotChangeActions` still passes (V offline).

| Field | Line | Source | Tag |
|---|---|---|---|
| `kills` | `cast result` | kills noted while that cast was the pending one (cast window = modelled length + slack) | S: window attribution, not the hit that killed |
| `since_cast_ms` | `combat kill` | ms since the last cast, next to `last_skill` | S |
| `targets` | `cast result` | monsters in the nova radius counted before the cast (crowd nova only) | S: counted, not confirmed hits |

Report (`d2dsoak report`): kills per skill = `last_skill` count plus "inside the cast window"; casts/kill per skill = its casts over its credited kills (n/a without credit lines); hits/cast = mean `targets` where logged.

## Still n/a, and why

- Hits per cast for missile skills: the combat code reads no missile or damage events, only the player and monster tables. Exact edit needed: a per-tick read of the missile table (or each target's HP drop) in `internal/combat/fight.go`, matched to the pending cast in `castlog.go` (`pendCast`), logged as `hits` on `cast result`. Needs a live offset check first (U).
- Kill credit by the hit that killed: same missing signal. Until then kills go to the last cast, and `since_cast_ms` shows how stale that credit can be (a bolt in flight may kill after the next cast starts).

Edits outside owns: `internal/combat/castfirst.go` (one call, `castLogTargets`), `internal/combat/fight.go` (one call, `castLogKill`).
