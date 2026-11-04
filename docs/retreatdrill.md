# retreatdrill: the P29 retreat action executed live (designed, offline-tested, not run)

Why: P29 was only ever recorded (D5.p29 "Safety would choose chicken/drink-rejuv", DRILLS5; D-portalmelee records `chicken`, DRILLS12: "the chicken action itself is not run by the harness"). codefix F1 (RetreatOut vs LeaveGame in `internal/resource/abort.go`) changed the abort path offline only (V: tests). This group makes the bot's own `Fighter.Safety` decide and `Fighter.Act` EXECUTE, and reads the effect.

## Group `RT` (internal/drill/retreat.go, instance side cmd/d2drill/retreat_lab.go)

| id | setup (cheats, ledgered, all before the Act) | expected decision | PASS numeric |
|---|---|---|---|
| RT.drink | Blood Moor, level cleared, max life 300 (`debug stat 7/6`), pack cleared, belt as the new character has it (healing potions), 1 zombie spawned until 1 within 3 subtiles, life set to 20 percent | `drink-*` | decision `drink*`, Act no error, belt healing count -1 and life above the decision value within 50 ticks, life floor above 0 |
| RT.portal | same but belt cleared and exactly one scroll (`d11OneScroll`), life 34 percent | `town-portal` | scroll count -1 and portal object listed within 50 ticks, life floor above 0 |
| RT.leave | belt cleared, pack cleared, no scroll, life 10 percent | `chicken` | decision `chicken`, Act no error, snapshot no longer in game within 200 ticks |

After the Act the lab only steps (`Tick 1`) and reads (snapshot, `PView`): no gameplay write. Tick bounds (50, 200) are U: first guesses, tune on the first run. Numbers (20, 34, 10 percent; 300 life) are setup data in `RetreatSpec`; the decision thresholds are read from the bot's own tune, the drill never copies them. The monster is `zombie1` as in the D11 drills (the existing `d11Mon`, V by DRILLS12: it does not scatter).

Verdicts: Blocked when setup is not reached (no adjacent monster, no potion, spawn failed) or the leave hook is missing; Fail when the decision differs, Act errors, the life floor is 0 (died before the effect) or no effect inside the bound; Pass otherwise. Offline: `go test -p 1 ./internal/drill -run Retreat` (V, 3 tests: verdict table per variant, effect-not-seen fails, specs are data).

## Command (hidden slot, d2d lab binary with the debug verbs; the coordinator or a d2-live agent runs it)

`D2D_AGENT=retreatdrill D2D_BIN=<fork>/bin/d2d.lab-cheats3 tools/agentrun.sh retreatdrill --ttl 1200 -- tools/d2dslot.sh run -n 0 -t 1200 -- <built>/d2drill -group RT -label retreatdrill1` (repeat 2 times, the default). Build: `go build -o <built>/d2drill ./cmd/d2drill`. Same pattern as docs/fixcrowd.md (D-portalmelee).

Required verbs, all existing (S, docs/forkcheats.md and used by d11_lab.go): `debug goto`, `debug kill`, `debug stat 6|7`, `debug spawn zombie1 1 near`, `debug belt clear`, `debug clearinv`, `debug give tsc`. Nothing missing for drink and portal.

## RT.leave: not reachable yet

`Fighter.Leave` is nil in the d2drill rig (d2dcombat.Attach does not set it), so `Act(Chicken)` would send the unverified esc-and-click placeholder (combat/fighter.go `leave`, tagged UNVERIFIED there). The lab returns Blocked with that reason instead of running it. To make it reachable: set `rig.Fighter.Leave` to the same `flow.LeaveGame` the bot wires (cmd/d2bot/d2dwire.go: `be.Cycler.LeaveGame`) and rejoin afterwards (D10/D9 harness restart; the runner must re-enter the game before the next drill) (U: not designed here, needs the cycler in the d2drill backend). The codefix F1 split (RetreatOut vs LeaveGame in a chain field) is a resource-policy decision, not reached through `Fighter.Act`; a drill of it needs a chain level (Black Marsh, areas 4 to 6) with no potions, also U.

## Risks

- A spawned zombie hits during the setup ticks; the life write is the last cheat, then one snapshot, so the gap to the decision is a few ticks (V in D11: same order). If life is hit to 0 first, RT.drink fails as "died" (a harness race, rerun).
- Drink: the starting belt must hold healing potions (S: ArmBelt notes four columns of healing for a new character); a prior drill that drank them gives Blocked "no healing potion".
- Drink decision may be `drink-rejuv` or `drink-life` depending on the belt; both pass by prefix.
- Portal: the cast can be cut by a hit (D-castcut record, DRILLS11), so a Fail may be the known cut, not a retreat bug; read the log.
- Tick-to-effect of the leave path (menus, seconds) is U.

## Tags

V: offline tests, build and vet of both packages. S: verb list, decision names (`drink-*`, `town-portal`, `chicken` come from D5.p29 and D-portalmelee live runs). U: tick bounds, the percent values giving exactly these decisions at maxlife 300 on this build, the whole leave variant. Not verified live.
