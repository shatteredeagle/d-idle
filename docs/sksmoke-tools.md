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

    tools/live.sh sk combatlive sk --skill "Fire Bolt" --key f1 --level 13 --monster zombie1 --hits 40

Flags: `--skill`, `--key` (required: the hotkey the skill is already bound to; the tool never opens the popup),
`--level` (picks the table row), `--monster` (monstats id, default nearest alive monster), `--hits` (40),
`--casts` (200), `--wait` (900ms between cast and second read), `--timeout` (5m).

- Per sample: read the target's life (stat 6, 256ths, via `UnitStats` of the monster unit), `CastAt(key, monster
  position)`, wait, read again. Delta in points is one sample. Reads only; no game memory is written.
- Kills (life <= 0 afterwards) are counted and discarded (overkill hides the hit); unchanged life counts as a miss.
  Charged Bolt: the delta is the whole cast (several bolts), so compare against bolts x table (design file), not the
  per-bolt row; the built-in per-bolt table gives no valid verdict for it (U).
- Output: one JSON line: `hits mean min max casts kills misses expMin expMax expMean meanDev outRange pass samples note`.
- `pass` = mean within 10 percent of the table mean, every hit inside [min-0.5, max+0.5], at least 40 hits. The
  thresholds are the design file's proposal (U). The table in sk.go is hand computed from skills.txt (S).
- Open questions (U): that `UnitStats` of a server-table monster returns real life (unit table is V; monster life
  stat is V only for town NPCs, see d2.MonsterInfo); that 900 ms covers the missile flight; that CastAt (key plus
  right click at the unit's subtile position) hits a moving target.

## Tests

`go test -p 1 ./cmd/d2cheat ./cmd/combatlive` (synthetic save, d2/fake reader with a fake controller; no game).

## Live checklist (needs the main agent to start i2)

1. Stop i2. `d2cheat --reason ... skilllevel Combi "Fire Bolt" N` (and Fire Ball, Meteor, Lightning 0).
2. Start i2, load Combi, bind Fire Bolt to F1, stand near one isolated zombie.
3. `d2cheat --reason ... runstart SK-N`, then `combatlive sk ...`, then `d2cheat ledger --since-run SK-N`.
4. Repeat for levels 1, 4, 8, 13, 20 and for Charged Bolt.
