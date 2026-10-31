# Merc wiring (agent MERCWIRE)

Policy: [merc.md](merc.md) (`internal/merc`, pure, unchanged except that it is now called). This file is what connects it to the running bot, what is verified, and the plan for the original game (Game.exe).
Tags: V verified (how), S secondary source, U unverified.

## What is wired (d2d backend only)

Switch: knob `merc.enabled` (group `merc.*`, default 0, NoHandicap; `PUT .../knobs {"overrides":{"merc.enabled":1}}` or config.json `"tuning": {"merc.enabled": 1}`).
OFF (default): no seam is wrapped, `task.MercAlive` stays nil, no goroutine, no log line, no API body but `{"enabled":false}`. V by test `TestKnobOffLeavesSeamsAndHookAlone` (cmd/d2bot) that a seam built with the knob off returns the inner Townsman object itself and leaves the hook nil. Game.exe (`--backend mem`, live.go) is not touched at all: nothing in this change is reachable from it.

ON (d2d, `runD2D`):
| Piece | Where | What it does |
|---|---|---|
| Reader | `internal/d2dmerc.Reader` | `merc.Reader` over `d2dsrc.Source.Read`: Known, Owned (dump `merc.present`), Alive (`merc.dead`), Class (only code `RG` maps, V d2dsrc), LifePct. Level 0 (not in the dump), Aura "", no stats, no gear. `Offers()` always errors (no hire-window reader on d2d). Stock d2d (no `dump` verb) gives `Known=false`, so the policy fails closed. V by `go test ./internal/d2dmerc` on the recorded dump fixtures (`internal/d2dsrc/testdata/dump_head_*.json`) served through a unix socket and on the d2dctl fake. |
| Level estimate | `cmd/d2bot/mercwire.go levelEstimate` | the revive price needs the merc level and the dump has none: use the player's level (U; an overestimate makes the revive stricter: level^2/2 x 15). |
| Seam wrapper | `installMercSeam` (called from `init()` on `d2dSeams`) | wraps `Seams.Town`: `mercTown` embeds the real Townsman; `Chores` runs `ReviveStep` after the real chores, `NeedsTrip` ticks the once-a-minute merc line. A nil Town seam is wrapped around `NotYetTown`. d2dbackend.go is NOT edited. Caveat: an `init()` in a file sorted AFTER `mercwire.go` that assigns `d2dSeams.Town` would replace the wrapper; then call `installMercSeam(&d2dSeams)` after it. |
| Revive decision | `mercWire.ReviveStep` -> `merc.ShouldRevive` | in town only, merc Known, owned, dead; Ctx: gold carried+stash, kit from the catalog (a town-portal item by name, a healing or rejuvenation potion in belt or pack; without a catalog: no kit, so "kit"), revives and deaths in the last hour (deaths counted from alive -> dead transitions seen by the reader), knobs from `merc.*`. Logs `merc: revived price=.. level=..` or `merc: revive skipped (reason) price= gold=`. |
| Reviver (act) | `d2dmerc.Reviver` | hireling NPC (Kashya, Greiz, Asheara, Tyrael, Qual-Kehk) menu, line "resurrect", verifies by gold delta or merc code change, else `ErrNotRevived`. **U on every step after the menu opens**: no merc ever died in the d2d runs (docs/d2d-probes.md section 4); the line exists while the merc is dead (S, D2D-src). Tested only against the fake (no resurrect line -> `ErrNoReviveLine`). |
| Route-guard input | `task.MercAlive = mercWire.AliveHook()` | the Monastery, the NAS and Andariel guards read `task.MercAlive()` (internal/bot/routeguard.go, task.go). The hook answers (alive, known) only for an owned merc (`merc.AliveHook`), so a character without a merc never raises the Andariel floor. The guard already lifted on a living merc through `HasMerc: d2.FindMerc(s).Alive` (the dump merc is in the snapshot, V code reading); the hook adds "known dead" (floor +2 below 1000 gold). State is cached 5 s (each read is a full d2d snapshot). V by `TestHookAndTTL`. |
| Dashboard/API | `mercMount()` -> `/api/v1/instances/{id}/merc/` | JSON `{enabled, known, owned, alive, class, lifePct, level, deathsLastHour, revivesLastHour, lastDecision}`. **One line is missing in d2dbackend.go** (not mine to edit): add `"merc": mercMount()` to the `Mounts` map of `server.Instance` in `runD2D`. |
| Sidecar line | `mercWire.Tick` | `merc: seen class=.. alive=.. life=.. level=.. aura=-` at most once a minute while a merc is owned (slog line of the bot). V by `TestSidecarLineOncePerMinute`. |
| Field death | `merc.MercDied` | NOT wired: there is no seam into the fighter to "leave the area". Only the `merc: died` line is logged. Follow-up for the combat agent. |

## Free Rogue claim (BRKILL stub)
- The stub already exists: `task.MercClaimer` <- `bot.TaskDeps.ClaimMerc func(ctx) error` (nil: logs "claim step skipped (no backend)").
- This change adds `Claimer` (cmd/d2bot/mercwire.go: `ClaimFreeRogue(ctx) (granted bool, err error)`), `claimDep(claimer, reader, difficulty, log)` which returns a `TaskDeps.ClaimMerc` (policy check `merc.ShouldClaimFreeRogue`, then the claimer; nil claimer -> `task.ErrClaimUnsupported`; merc already owned -> logged, no call), and the d2d implementation `d2dmerc.Claimer{C}` over `d2dtown.ClaimRogue` (merged on master, V-d2d: grant on opening Kashya's menu, docs/d2d-probes.md section 1). Tests: `TestClaimDep` (fake claimer) and `TestActsAgainstFake`.
- NOT connected: `runD2D` builds `bot.Deps` without `Task`, so no task step runs on d2d yet; whoever adds `Task` deps there sets `ClaimMerc: claimDep(d2dmerc.Claimer{C: client}, reader, diff, log)` (client = `be.Client`). Claim BEFORE any hire (the reward is forfeited by any merc, even dead).

## Verified vs not
| Claim | Status |
|---|---|
| knob off = nothing changes (seam passthrough, hook nil) | V offline test |
| reader fields from the dump (alive, dead, none, unknown code, stock d2d unknown) | V offline on the recorded dump fixtures (the fixtures are the repo's synthetic-but-format-true dumps, not a new live recording) |
| revive decision, kit, hour counters, hook, status JSON, once-a-minute line, claim policy | V offline unit tests with fakes |
| `cmd resurrect` / the "resurrect" menu click really revives on d2d | U (never seen, no merc died) |
| hire window rows, merc level, aura, gear on d2d | not readable (dump has none), `Offers()` errors |
| live d2d run with the knob on | NOT done (no game started in this task) |

Checklist for the d2d run (short, private fork only, `flock /tmp/d2d-lab.lock`, `pgrep d2d` first, `nice -n 10`, `--no-save --toggle autoloot=off,autoparty=off`): start d2bot with `--backend d2d` and `merc.enabled=1`, a character with a merc (claim at Kashya after `debug quest`), read `.../merc/` and the `merc: seen` line, `debug kill` the merc if the fork has such a verb or let it die, then walk to town and watch `merc: revive ...` plus the gold delta. Stop d2d afterwards.

## Plan for the ORIGINAL game (Game.exe 1.14d), U unless stated

Today `d2.FindMerc(snapshot)` finds a merc-class unit (271, 338, 359, 561) in the server unit table; owner words read 0/0 for the hired Rogue (V docs/friendly.md), life from stats 6/7 (U for a hurt merc), level stat 12 (U, never read live). `MercAlive`/`HasMerc` therefore work for a living merc next to the player and nothing else. A `game.go` reader implementing `merc.Reader` needs, in this order:

1. **Unit**: merc-class monster in the client unit table (S: SME). Match by class plus proximity (single player: the only hireling-class unit is ours). Owner words 0/0 (V friendly.md); the owning record is probably a client-side merc record fed by packet 0x81 (S: merc type, owner, id, seed), address U.
2. **Dead merc**: S: stays as a corpse unit in the area, freed when the area is left; in town after a field death no unit exists, only the client's "dead merc known" flag (address U). Signal to use first: the Kashya/Greiz menu shows the "Resurrect %s" line only while that flag is set (S D2D-src); read it from the menu text, not memory. Owned-but-absent merc = known-dead only if the bot saw him alive this run (`town.SetMercKnown`).
3. **Fields**: level = stat 12, experience = stat 13, life = stat 6 / max (all S, never read live); name from the 0x81 seed through the hireling name table (U). Gear = the merc unit's pInventory, body locations helm 1, armor 3, hand 4/5 (S; which hand is weapon/shield U). Aura: no memory source known; persist the aura from the hire row at hire time (recommended by the SME) and keep it in bot state.
4. **Minimal probe on Fresha (i5, never Soaky, no stash gold)**, one short run, read-only first:
   a. Town monster dump (class, mode, stats 6/7/12/13, owner words, state list). Read-only.
   b. Reach clvl 8+ only via the normal bot; claim the free Rogue after Blood Raven BEFORE any hire (a hire forfeits it). Do not cheat levels on Fresha: it is a gate character (docs/cheats.md); the one allowed exception is `d2cheat gold` with a reason for the revive price test, after which Fresha is retired from gates.
   c. With the Rogue: dump the same fields standing, after a hit, and read `pInventory` of the unit.
   d. Put one cheap identified helm on him (panel/click path of docs/merc-farm.md item 6) and re-read `pInventory`: confirms body location index.
   e. Let him die in the field: dump the unit table in the field (corpse? mode?), after a town portal (gone?), and open Kashya's menu: record the "Resurrect" line text, the quoted price, then pay it: gold delta against `merc.ReviveCost(level)`, carried-vs-stash order, unit reappearing.
   f. Record all of it as a fixture (JSON of the reads) so `internal/d2` gets an offline test; then implement `merc.Reader` over `internal/mem` and flip nothing: `merc.enabled` stays 0 on Game.exe until a soak with it on passes.
5. Not known (SME and docs): client merc record address, the owner semantics, the aura source, the dead-merc flag address. Nothing in docs/sme covers a client-side merc layout.
