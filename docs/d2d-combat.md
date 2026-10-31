# Combat on d2d (agent D2DCOMBAT, 2026-10-30)

The bot's real combat layer (`internal/combat`: cast-first loop, kiting, potions, target choice, ignore rules) runs on the d2d
backend through `internal/d2dcombat`. Nothing in `internal/combat` or `internal/nav` changed; Game.exe paths are untouched.
Tags: V = observed in the live runs below (private fork, the worker host), S = from code reading, U = not verified.

## Pieces

| Where | What |
|---|---|
| `internal/d2dctl/combat_verbs.go` | Thin verbs: `CastUnit`, `CastCell`, `Melee`, `Drink(slot)`, `SetSkill(id, left)` (`debug skill left|right`), `NearestAlive`, `Monster.Alive`. Fake support for `debug skill`. |
| `internal/d2dcombat` `Rig` | `New(Config)` builds env (reader, input, clock), `nav.Navigator` (+`UseWorld`), `combat.Fighter`, and sets `nav.Yield` to `bot.PackYield` around `Fighter.Clear(Yield: true)` (the hook cmd/d2bot installs for Game.exe, minus its pinned-evade guard). `BindSkills()` replaces `EnsureHotkeys`. |
| `d2dcombat.Input` | `d2dinput.Controller` plus: `Key("1".."4")` -> `cmd belt <col-1>` (potions); `CastAt(hotkey, w)` -> `cmd skill <id> x y <unit>` with the unit found by one `monsters` call (nearest living within 1 cell); counters. |
| `d2dcombat.Meter` | The env clock: `Pacer.Sleep` (lockstep `step`) plus one `state` read, summing damage, mana spent, deaths, min life. |
| `viewReader` | Fills `Snapshot.Viewport` from the player position (d2dsrc leaves it zero; see below). |
| `cmd/d2dfight` | Smoke tool (below). |

Wiring decisions:
- Hotkeys are pseudo keys `sk<id>`: `BindSkills` maps every skill the character has and the catalog knows to one, installs it in
  `Controller.Skills` and in `Fighter.SetHotkeys`. No popup clicking, no F-keys, no `cmd select` needed.
- Potions: combat presses `1..4`; the wrapper turns that into the belt verb. Belt contents come from d2dsrc items (ContBelt, S).
- Time: everything sleeps through the Meter/Pacer, so combat's cast gap (`CastGapMs` 450), watchdogs and per-target limits are in
  game time. A cast = `cmd skill` applied at the next tick; the actual cast animation is d2d's, so the effective period is
  `max(CastGapMs, animation)` exactly as on the original (U: not compared frame by frame).

## Differences from Game.exe worth knowing

1. `Snapshot.Viewport` must exist: combat's `onScreen`/`screenOf` project with it. d2dsrc does not fill it, so `viewReader` computes the
   Game.exe standing-player value (VX = 16(px-py)-400, VY = 8(px+py)-284). U that d2d draws its player at that pixel
   (`state` has `cam=` in cells; compare when needed). Before this fix the first live run cast 0 times in 130 approaches
   ("watchdog cannot cast (no hostile on screen)") because every target looked off screen (V).
2. Line of sight: `env.World` (a `*d2map.Tracker`) is nil on d2d, so combat's `los` returns true (no wall test). Nav's own d2dworld grid
   is not given to combat. Walls between caster and target are therefore invisible to the shot decision (S).
3. Resists: `UnitStats` is unsupported on d2dsrc, so `refineResists` falls back to the static monstats tables (already a handled
   path, S). Immunities of champions (MonUMod) are not applied (U).
4. Monster hostility is not read (d2dsrc), friendliness comes from `combat.IsFriendly` class tables; outdoors there are no town NPCs.
5. Hold skills (Zeal etc.) and click modifiers are unsupported (`ErrUnsupported`, gaps G1/G5 of docs/d2d-backend.md): melee classes
   fall back to plain `Click`. The Sorceress path (cast-first, Frost Nova) needs none of them.
6. Shared state: the shared fork socket `~/d2d-lab/fork/d2d.sock` is used by several agents at once. Run your own d2d on a private socket
   and home (see the checklist) or you will talk to somebody else's character.

## Smoke tool

```
GOOS=linux go build -o /tmp/dfight/d2dfight ./cmd/d2dfight && scp /tmp/dfight/d2dfight worker:d2d-lab/fork/bin/
# private socket and home, own lockstep d2d (class 4 = Sorceress in --start-class UI order: Barbarian, Necromancer, Paladin, Amazon, Sorceress, ...)
d2d --data ~/d2d-lab/data --headless --devctl ~/d2d-lab/fork/dfight.sock --no-video --toggle autoloot=off,autoparty=off \
    --lockstep --seed 7 --no-save --start-screen ingame --start-class 4 --start-name Fighty      # niced, flock /tmp/d2d-lab.lock
d2dfight -sock ~/d2d-lab/fork/dfight.sock -tables ~/d2lod-live/combat/data -clvl 12 -goto -route 2,3 -wander 3
```

Flags: `-clvl N` (fork cheats on the throwaway: level, 11 skill points on Fire Bolt / Ice Bolt / Frost Nova / Static Field via
`debug points`, life and mana via `debug stat 6..9`), `-goto` (enter levels with `debug goto`; use it when the d2d build has no
`debug layout` verb, otherwise nav `ExitTo`), `-wander N` (walks per level with nav.GoTo, where the combat Yield clears the way),
`-clear`, `-radius`, `-wall`. Output: per-leg result, per-Clear `ClearResult`, then a summary of kills, damage, casts per skill id,
mana, deaths, virtual versus wall time.

## Result (V, private fork, seed 7, throwaway Sorceress "Fighty" cheated to clvl 12, `--no-save`, load 20 to 30)

`d2dfight -clvl 12 -goto -wander 3 -route 2,3,2` (Blood Moor, Cold Plains, Blood Moor), run `out10`:
- **51 kills, 1 boss (Doom Flame the Tainted, Cold Plains), 0 deaths, 0 potions** (no belt: fresh character, so the potion path is only unit-tested).
- 176 casts, all Fire Bolt (skill 36); about 1.5 casts/s while fighting (castsPerSec 1.2 to 1.6), so the effective period is about 0.65 s
  (CastGapMs 450 plus the d2d cast animation and the loop's reads; U: not compared with Game.exe).
- Damage taken 24 (life min 136 of 160), mana spent 240 (Meter sums; regeneration makes this approximate).
- Virtual 3m30 versus wall 7m59 (load average 20 to 30, about 10 verbs per observation plus one `monsters` per cast and one `state` per sleep).
- nav's combat Yield fired ("nav stalled by monsters, yielding") and cleared the corridor packs that stalled d2dwalk; no `-clear` cheat.
- The run ended with a client timeout on the last wander (the wall limit of 8 min, `pacer err` i/o timeout); the character was alive.
- Earlier runs: 0 casts in 130 approaches before `viewReader` (Viewport); a first run with class index 5 was a Druid (the `--start-class`
  index is the UI order: Barbarian 0, Necromancer 1, Paladin 2, Amazon 3, Sorceress 4, Druid 5, Assassin 6, V).
- Not exercised: kiting, Frost Nova, potions, melee, death handling, buffs (only Fire Bolt was bound by the point plan 6/3/1/1; `BindSkills` found only skills with points).
- `ExitTo` needs the `debug layout` verb; the shared fork build (branch d2dtown-vendor) lacked it, hence `-goto`. `debug goto` also leaves the kick quirk (handled by d2dctl auto kick).
- Incident: one early run (class 3 tool bug, shared socket) connected to another agent's d2d on `~/d2d-lab/fork/d2d.sock` (character "Townz") and applied the -clvl 12 cheat (level, life and mana stats) to it. Use your own socket as above.
