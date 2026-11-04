# SK drill tools (card sksmoke-a1)

Two offline-built tools for the live Fire Bolt / Charged Bolt damage drill in
`docs/probes/drills/2026-11-03-SK-design.txt`. Status: unit tests only. NOTHING here has run against a game (U).

## d2cheat skilllevel

    tools/live.sh multi d2cheat --reason "SK setup: skill level 13" skilllevel Combi "Fire Bolt" 13

- Edits the hard points of one skill in the character's `.d2s` (SetSkillLevel in internal/cheat/save.go through
  `KindSaveEdit`): the game must be stopped, a backup is made, the ledger gets an entry (params `skills=ID:LEVEL`,
  `skillname`). The save edit itself is V from the existing saveedit tests; the live effect on a loaded character is U.
- The skill is resolved by name from the install's tables (data.SkillByName, skills.txt key or display name), so no
  skill id is typed. The skill's class must match the save's class; errors for an unknown skill or class come before
  anything is written (no ledger entry, nothing attempted).
- `--reason` is required (missing reason is refused and ledgered). Characters that are not on the throwaway
  allow-list (docs/cheats.md) are refused by cheat.Env.Do and the refusal is ledgered.
- Level 0 to 255. Skill points are not touched (the level is set directly).
- Run it before `d2cheat runstart SK-<n>` so `ledger --since-run` shows 0 writes after start.

## combatlive sk

    tools/live.sh sk combatlive sk --skill "Fire Bolt" --key f1 --level 13 --monster zombie1 --kills 10 --timeout 4m

Flags: `--skill`, `--key` (required: the hotkey the skill is already bound to; the tool never opens the popup),
`--level` (picks the table row), `--monster` (monstats id, default nearest alive monster), `--kills` (10 usable),
`--casts` (200), `--wait` (900ms between cast and second read), `--timeout` (5m; keep it below the job's max).

### Why "bolts to kill" (VERIFIED LIVE 2026-11-03 for the scale; the rest U)

- The monster life the reader sees (stat 6 of a unit from the table `internal/d2` reads) is a 0..128 PERCENT scale
  (V live, docs/probes/drills/2026-11-03-SK-live.txt), so a per-hit delta is not damage in points.
- Exact read searched offline: none exists. `d2.Unit.Life/MaxLife` are the same stat; `mem` has no monster-life
  reader; the d2d `monsters` verb and `debug mon set` give exact hp points (V, docs/forkdbg.md) but only on the d2d
  backend, not on game.exe, which the drill targets. docs/d2-1.14d-memory.md 2.5 calls the table internal/d2 reads
  "server side" but notes (R) it is the client's table; docs/cheats.md shows the real server unit is a separate heap copy.
- Therefore: cast at one monster until it dies (life from the percent read only tells hurt / unhurt / dead), count
  the casts that hurt it (n, the killing cast included), take its max life ML from monstats (midpoint of MinHP..MaxHP of
  the character's difficulty, U: the rolled value, champion/unique modifiers and area scaling are not known).
- Bounds per kill: the sum of the n hits is >= ML (so mean >= ML/n, the last hit overkills) and the first n-1 hits
  sum to < ML (so their mean < ML/(n-1)). Kills with n = 1 only prove damage >= ML and are counted (`oneShot`), not used.
  This is an ESTIMATE bracket, not a measurement of single hits: no per-hit range or spread, regeneration ignored.
  Use a weak monster whose HP is fixed or a narrow range, with skill levels where n is 3 or more.
- Charged Bolt: several bolts per cast, so n counts casts, not bolts; the per-bolt table gives no valid verdict (U).
- Output: progress lines on stderr (one per cast: target, life before/after, hit/miss/kill, counters), so a killed run
  still leaves output; then one JSON line: `skill level casts misses oneShot kills[{class id hits maxLife lo hi}]
  meanLo meanHi expMin expMax expMean pass note trace`.
- `pass` (interim, U): at least 10 usable kills and the table mean inside [meanLo, meanHi]. The table in sk.go is hand
  computed from skills.txt (S).
- Open questions (U): that 900 ms covers the missile flight; that CastAt (key plus right click at the unit's subtile
  position) hits a moving target.

### Which read would make the original drill valid (not built, U)

An exact life of one monster in points, before and after each cast. Two routes, both need work outside this card:
1. game.exe: find the embedded game server's own monster UnitAny (type 1, same class and id as the client unit), the
   way `serverPlayer` finds the player's (internal/cheat/memory.go: scan private rw mappings, match type, class, id,
   stat list). Its stat 6 should be 24.8 points. A new scanner in internal/cheat or internal/mem plus one live check.
2. d2d backend: `monsters` already gives hp `[cur,max]` in points (V). Run the drill on d2d with the existing SkillDebug
   (`mon set` for a known life, resist 0): needs no fork change, but then it tests d2d's damage, not game.exe's.
No d2d or fork change is needed for 1 or 2 as described (S); the fork is not edited here.

## Tests

`go test -p 1 ./cmd/d2cheat ./internal/combat/cmd/combatlive` (synthetic save, d2/fake reader with a fake controller; no game).

## Live checklist (needs the main agent to start i2)

1. Stop i2. `d2cheat --reason ... skilllevel Combi "Fire Bolt" N` (and Fire Ball, Meteor, Lightning 0).
2. Start i2, load Combi, bind Fire Bolt to F1, stand near one isolated zombie.
3. `d2cheat --reason ... runstart SK-N`, then `combatlive sk ...`, then `d2cheat ledger --since-run SK-N`.
4. Repeat for levels 1, 4, 8, 13, 20 and for Charged Bolt.
