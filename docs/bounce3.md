# bounce3: the warp arrival bounce, root cause and fix

> status: current (TASK card bounce3). Supersedes the cause sections of [passagebounce.md](passagebounce.md), [fixdenbounce.md](fixdenbounce.md) and the kick part of [hourfix.md](hourfix.md). Tags: V = verified (how is said), S = secondary, U = unverified.

## Symptom (V, sandbox container log 2026-11-12 19:06Z to 19:48Z)
The hero ping-pongs Cold Plains (3) and Cave Level 1 (9): 41 `arrival bounce` lines, stays 0.5 to 3.2 s, `leftFrom == arrivedAt`. Counters on devp4: 99 arrivals, 34 bounces. The breaker then skipped Cold Plains itself (`area skipped Cold Plains 2880 s border-loop`).

## Root cause (V)
d2d's warp-click rule plus the bot's priming click.
- d2d (fork at sandbox-warp f8ea366c, V read in source): `walk_to` in `components/game/world/commands.cpp` marks a warp as taken when a FRESH move's target (a mouse click or `cmd move`) lies within 2 cells of the warp's centre (cell + 0.5); `World::use_warp` (`travel.cpp`) then fires as soon as the player stands within the same 2 cells. A fresh arrival stands 0.63 to 1.71 cells from the return warp (V, repro below), so it fires at once.
- The bot (V, code and log): after every level change `d2dinput.Controller.primeToward` sends a priming click ("kick") before the first `cmd move`. Since arrivalsettle it lands 110 px from the feet toward the move target, which is only 1.2 to 1.6 cells (V: inverted from the 24 logged kick pixels); the untargeted `prime()` (InteractNPC, UsePortal) still clicked the feet (400,300). Both land inside the 2-cell radius of the warp the hero just came through.
- Classification of the 41 bounces (V, `tools/bounce3` analysis of the log): 24 had a kick as the only order before the leave (c, stale input: our own click), 9 had no logged order (the untraced feet kick of `prime()`, S: the code path is the only click sent there), 8 were deliberate `nav exit` back, each after an accidental entry by the previous bounce (`exploring crossed an area border (target area N)`, or the to-town walk that needs the neighbour). None was a goal flip (b) and none was d2d firing a warp without a command (a).

## Reproduction, end to end (V, 2026-11-12, worker host)
`tools/bounce3/repro.sh d2dbot:devp4 SEED ./kickrepro -sock SOCK` runs a throwaway headless d2d of the play image (own HOME, `--network none`, game data read-only; never the sandbox) and drives it through the bot's own `d2dinput.Controller`: take a warp with `debug enter`, then `Controller.Move` 4 cells straight away from the return warp.
- origin/main controller: 6 of 12 arrivals bounced (seeds 3, 7, 11; every arrival 0.63 cells from its warp: Cave->Cold Plains and Den->Blood Moor).
- this branch: 0 of 12; every kick lands 3.04 to 3.28 cells from the warp.
`tools/bounce3/repro_kick.py` is the same in Python with `--safe` for the new placement.

## Fix
1. `d2dinput.Controller.kickSafe`: the priming click lands at least `warpTakeR + kickWarpMargin` (2 + 1 cells) from every warp of the level (`debug warps`, read-only, one call per level change): along the target direction (no target: away from the nearest warp) pushed out from 110 px to the play-area edge, then rotated in 15 degree steps. The feet click stays where no warp is near (town NPCs). The `nav.trace kick` line carries `warpDist`. Tests: `TestKickLeavesTheReturnWarpAlone` (the old placement is the witness: 1.68 cells), `TestPrimeAsksForWarps`.
2. Breaker: a bounce counts for the pair and the DEEPER side (more hops from the act's town, `townDepth`) takes the escalating skip (3, 6, 12, 24, 48 min); the old transit rule only decides without depths. A `bounce breaker` line says the pair, the side and why. Test `TestBounceBetweenAPlainAndItsCaveSkipsTheCave` (failed on main: Cold Plains banned).
3. Minute line: `arrivals`, `bounces`, `bouncesPer100`.
Class independent: no class or build data involved.

## Not done, and why
- "Do not pick a goal whose first action leaves the area just arrived in": not added. All 8 deliberate leaves followed an accidental entry, where going back is right (V, the 41 bounces above).
- Residual (U, from d2d source): a `cmd pickup` or `cmd interact` whose target lies within 2 cells of a warp also walks through it. The mouth gate covers world clicks only.
- Fidelity (U): whether game.exe takes a warp for a ground click near it (rather than a hover on the tile unit) is not checked; if not, d2d's 2-cell rule is a deviation for docs/d2d-deviations.md.

## Timeline of earlier attempts
- passagebounce (#609): breaker keyed by the bounced area. Treated the symptom; banned transit areas.
- arrivalsettle: kick moved off the feet, 110 px toward the target. Still inside 2 cells (not known then: the radius was not read).
- hourfix (#627/#634): bounce skip takes the cave when the other side is a transit. Cold Plains was not a transit in the live plan, so it was banned.
- hourfix2 (unmerged, agent/hourfix2): a 240 px kick outside towns. Direction-blind; superseded by kickSafe (conflicts in `primeToward`).
- devp4 d2d arrival spot (`Level::warp_spot`): moved the arrival nearer the game's spot; does not change the 2-cell rule, so it could not help.
