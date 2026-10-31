> status: partly stale (2026-10-30). Andariel's level floor is the knob `boss.andarielMinLevel` (docs/knobs.md, generated from code), not the 22 quoted in older passages below.

# Boss scripts: internal/boss (agent BOSSRUNS)

Offline only. Code: `internal/boss` (stdlib only, imports nothing from the repo). Tests: `go test -p 1 -count=1 ./internal/boss` (fake world with
scripted monsters, 16 tests). Tags: V verified, S secondary (Kolbot, docs/archive/prior-art/02), U unverified. **Every live behaviour here is U** until section 4 ran.
Shape follows docs/progression/bosses.md section 1; this package is the executor and the Act 1 data.

## 1. Pieces

| file | content |
|---|---|
| `types.go` | `Pt`, `Mon`, `Danger`, `Preset`, `View` (one read), `World` (View, Presets, Now, Sleep), `Actions` (12 calls), `LootSpec`, `Err*` |
| `script.go` | `Script` (data), `Match`, `Anchor`/`PresetRef`/`Variant`, `StandPoint`, `Lure`, `Kite`, `Hazard`/`Response`, `Hooks` |
| `hazard.go` | builtin hazards: `PoisonStatus`, `PoisonCloud`, `RaisedAdds`, `Skeletons`, `CrowdedAndHurt`, `NoPotionLowLife` |
| `run.go` | `Run(ctx, script, World, Actions, Opts) (Result, error)` |
| `act1.go` | the four scripts, `Lookup(name)`, `Runner.KillBoss(ctx, name)` (same signature as `task.Actions.KillBoss`) |

Run sequence: level floor check (`MinLevel[difficulty]`, else `ErrTooWeak`) -> approach (Travel to the WP when not on the chain, then `ExitTo` per route area,
re-reading the area after each hop; works from town, mid-route or inside the arena) -> locate (look; walk to the preset stand point; look; walk to the Kolbot hint;
`Explore` until the boss is in the list; look; else `ErrBossNotFound`) -> fight loop every 250 ms -> wait 2 s -> optional clear around the corpse -> `Loot` -> optional `ToTown`.
Fight loop order: chicken line, escape (low life and no potions: town portal, or leave game when `NoTP`/no TP), own drink line, hazards (first one that fires and is off cooldown),
lure (walk back to the arrival point and wait until the boss is close), kite (step to the next stand point when the boss is within `Range`), attack (`Actions.Attack` slice of 3 s, buffs hook once).
A boss missing 3 polls in a row counts as dead only when last seen at <= 10% life, else `ErrBossLost`. A boss found already dead goes straight to loot (idempotent re-run).
Stand points marked `Rel` are offsets from where the boss was first seen, so they survive a different level origin; Kolbot's absolute coordinates are only `Anchor.Hint`
(cross-check note in `Result.Notes` when the preset-derived stand is more than 60 subtiles from it, then a last-resort walk target).

## 2. The Act 1 scripts (all numbers U unless stated)

| boss | WP -> route | match | anchor | hazards and rules | loot |
|---|---|---|---|---|---|
| Blood Raven | Cold Plains 3 -> Burial Grounds 17 | class 267 + name | none (boss list, then Explore) | (no lure, V live; outer clear around the arrival point, retreat to the entry below 40% life, 2nd retreat = town portal), raised skeletons >= 3 within 14 (match by name "skeleton"/minion flag) clear first, >= 6 adds clear, no-potion escape < 30%; floor clvl 6/40/60 | any tier |
| The Countess | Black Marsh 6 -> 20, 21, 22, 23, 24, 25 | name "The Countess"; UniqueNo via `WithUniqueNo(db.SuperUniqueByName(..).ID)` | object preset 580 (Kolbot SuperChest, S): chest X 12565 -> stand (12578,11043), X 12526 -> (12548,11083); hint (12578,11043) | route life < 40% after any hop escapes (no WP inside), adds >= 5 clear, clear 20 around corpse; floor 14/42/67; her chest is never opened (loot module must not open chests) | runes, keys, gems, unique/set/rare |
| The Smith | Outer Cloister 27 -> Barracks 28 | class 402 + name | none | crowded (>= 6 hostiles within 15) and life < 50% retreats to arrival and drinks; floor 8/40/60 | quest item "Horadric Malus" always, else any |
| Andariel | Catacombs 2 (35) -> 36 -> 37 | class 156 + name | boss list, hint (22549,9520) | antidote when poisoned and one exists (20 s cooldown); one before the first hit when poison resist < 75; poison cloud within 6 -> step 10 away; kite Range 10 with rel sides (-14,+14)/(+14,+14) every 1.5 s; chicken < 12%; floor 22/45/65 (script default per difficulty; Normal is overridden by the knob `boss.andarielMinLevel`, see docs/knobs.md, internal/boss/run.go:781) | any tier |

Not covered here: Cow level, the Andariel -> Warriv talk (task layer step `Talk`), Duriel and later bosses (framework supports `NoTP`, `NeedsTeleport`, `Lure`, `Kite`).

## 3. Interface proposal for INTEGRATE (adapter methods)

Wire `boss.Runner{W, A, Opts}` and let the bot's `task.Actions.KillBoss(ctx, name)` call `runner.KillBoss(ctx, name)`. The task steps already travel to the arena first;
`Run` also accepts being on the route or in town. Adapter methods:

| `boss.World` / `boss.Actions` | maps to |
|---|---|
| `View` | `d2.Snapshot` -> `View` (Monsters: `Unit.Class/ID/Mode/X/Y`, `Monster.UniqueNo/Name/LifePct/IsMinion`, Hostile = combat `Hostiles` rule; `PoisonRes` stat 45; `Potions`, `Antidotes` from belt/pack; `HasTP` = `Fighter.HasPortal`) |
| `Presets` | nav/d2map presets of the current level (`d2map.Preset{Kind,ID,Pos}`) |
| `Travel` / `ExitTo` / `GoTo` / `Explore` | `nav.ToArea` / `ExitTo` / `GoTo` / `Explore` (stop callback checked between steps; NEW if Explore has no early stop) |
| `Attack(id, slice)` | `Fighter.Kill` under `context.WithTimeout(slice)`; deadline -> nil; `ErrTownPortal` -> `boss.ErrEscaped`, `ErrChicken` -> `boss.ErrChicken`, `ErrDead` -> `boss.ErrDead`; `ErrTargetGone`/`GaveUp` -> plain error (counts as a failure, 5 in a row end the run) |
| `ClearAround` | `Fighter.Clear(ClearOpts{Radius, Origin, Timeout})` |
| `DrinkHealing` / `Chicken` / `ToTown` | `Fighter.Act(DrinkLife/DrinkRejuv)` / `Fighter.Leave` / `town.ToTown` |
| `Loot` | loot module: pick up items within `Radius` of `Center`: names in `Quest` always, tiers in `Kinds` by loot policy |
| `Antidote` | NEW: belt/pack antidote use (combat `Act` has no antidote action) |

Error to task `Kind` (via `task.Wrap`): `ErrTooWeak`, `ErrAborted` -> KindDanger; `ErrBossNotFound`, `ErrBossLost` -> KindNotFound; `ErrTimeout` -> KindTimeout; `ErrDead` -> KindDeath;
`ErrEscaped`, `ErrChicken` -> KindDanger (the executor's retry then goes through Chores). Events: combat already emits `EvKill` with the boss flag; `Hooks.OnKill` is for extras.
`Hooks.BeforeAttack` is where the bot casts buffs / Static Field once; `Hooks.OnPhase` for logging.

## 4. New bot capabilities needed (none exist yet)

1. Poison state and ground hazards: `View.Poisoned` (U, state list or life-curve guess) and `Danger{Kind: "poison-cloud"}` from missiles/objects (U, class ids unknown). Without them the
   poison hazards never fire; the rest still works (pre-fight antidote, combat's own kiting).
2. Antidote potion drinking and counting (`yps` base id; belt column classification in `combat.BeltOf` covers healing/mana/rejuv only).
3. Unique number and name of monsters (`MonsterInfo.UniqueNo/Name` offsets are U in internal/d2); Countess and Smith need one of them (class alone is shared with normal monsters).
4. Loot by radius and by name with a no-chest rule (Countess chests, `Loot` in adapter).
5. `Explore` with an early-stop callback.

## 5. Live checks for the main agent (run in this order, Normal, one instance, nothing destructive)

Each: area, anchor, expected value, then what to record. A failed expectation means edit the script data in `act1.go`, not the runner.

1. **Area ids** (levels.txt): Cold Plains 3, Burial Grounds 17, Black Marsh 6, Forgotten Tower 20, Tower Cellar 1..5 = 21..25, Outer Cloister 27, Barracks 28, Catacombs 2/3/4 = 35/36/37. Expect each `View.Area` after the ExitTo chain to equal the next id.
2. **Class ids** (monstats via `db`): Blood Raven 267, Andariel 156, Smith 402, Countess name "The Countess" and her `SuperUniqueByName` id. In the first game dump `Monster.Class/UniqueNo/Name/Flags` of the boss unit; expect Name set and Flags boss bit.
3. **Blood Raven** (clvl >= 6, Cold Plains WP): after `ExitTo(17)` read monsters for 4 s; expect her within the area, mode 1/2, LifePct 100. Record her position relative to the arrival point (to tune Lure) and whether raised skeletons carry `Minion` or a name containing "skeleton". Expect a kill, `Result.Phases` locate/lure/attack/loot, no death.
4. **Countess** (clvl >= 14): at Tower Cellar 5, `Presets` must contain object 580 (check Kind value, expected 2); expect chest X 12565 or 12526 (else the note "preset ... hint" appears and the Kolbot variants are wrong, record the real X and the boss spawn). Expect the boss within ~40 subtiles of the stand point; each of the five ExitTo hops must land in 21..25 in order. Verify no chest was opened and runes were picked (`Result.Loot` > 0 on a drop).
5. **Smith** (clvl >= 8, Tools of the Trade active): enter Barracks from the Outer Cloister WP; expect a unit with Name "The Smith" and class 402; after the kill the Horadric Malus is picked (`Items["Horadric Malus"]` becomes 1). Record the crowd size near the arrival point (hazard threshold 6 within 15).
6. **Andariel** (clvl >= `boss.andarielMinLevel`, docs/knobs.md; the script's own 22 is only the fallback): Catacombs 4; record her spawn versus Kolbot (22549,9520) (the run writes a note when a preset stand is far from the hint; with `Anchor.Boss` log `St.Spawn`); expect her to stay put until aggro. Check poison: does the character show `Poisoned`; does an antidote clear it; do clouds appear as missiles/objects. Expect the Rel sides to be walkable and >= 10 from her.
7. **Escape paths**: with 0 potions and life forced low in a spare game, expect `ErrEscaped` + town; on a script with `NoTP` expect `ErrChicken` + game left.

Dead ends / open: Blood Raven moves (S: Kolbot walks a fixed path), so `Anchor.Boss` plus Explore may need a graveyard sweep list; Kolbot's Andariel side points are absolute and
were converted to boss-relative guesses (U); Normal-only data, Nightmare/Hell floors are copied from `danger-table.json`.

## 6. INTEGRATE2: adapter, capabilities and the first live runs (2026-10-28, i4, Booti, V = verified live, U = not)
Code: `cmd/d2bot/boss.go` (`bossGame` = `boss.World`, `bossActs` = `boss.Actions`, `newBossRunner`, `bossKill`, `mapBossErr`), hook `bot.TaskDeps.Boss`
(used by `taskActs.KillBoss`; without it the old Explore + Kill + PickUp path stays). Tests: `cmd/d2bot/boss_test.go`, `internal/bot/task_test.go`.
- Mapping as in section 3: `Attack` = `Fighter.Kill` under a slice timeout (slice end is not an error), `ErrTownPortal`/`ErrChicken`/`ErrDead` become `boss.Err*`, and
  `mapBossErr` turns the run's errors into `bot.ErrPortalOpened` (escape), `bot.ErrLeftGame` (chicken) or task kinds (TooWeak/Aborted/LeftArena Danger, NotFound/Lost NotFound,
  Timeout, Dead Death). Every run logs `boss run` (phases, notes, loot, error) and every unique/champion unit logs `boss view: unique` (flags, class, unique number, name, position, life).
- New capabilities (section 4): (1) poison: `d2.Player.States` (`StateSet`, bit n = states.txt state n) read through `UnitStates`; `View.Poisoned`. **V: the array is behind a pointer at
  StatList+0x58** (+0x5C is a code address in Game.exe); bit 100 `healthpot` switched on right after a healing potion was drunk, bit 102 `just_portaled` on arrival through a portal, bit 105
  `alignment` is always on. Poison (state 2) itself was not seen yet (U). Poison clouds: `View.Dangers` kind `poison-cloud` from missiles whose missiles.txt name is in `poisonCloudMissiles`
  (U: Andariel's real clouds). (2) antidote: `combat.PotionAntidote` (code `yps`), `Fighter.Antidote`, belt count in `View.Antidotes` (belt only; the pack is not used). (3) monster name and unique number:
  **V: `MonsterData+0x2C` is a POINTER to the wide name** (read inline it gave pointer bytes as CJK text); with the fix Blood Raven reads flags 8, class 267, name "Blood Raven",
  UniqueNo 0 (0 is every random unique's value, so the adapter reports -1 unless the name is Bishibosh; the Countess is matched by name, her `SuperUniqueByName` id is also set). (4) loot by name and radius:
  `loot.PickOpts.Names` (takes named items whatever the pickit says) and the existing Center/Radius; only ground items are ever clicked, so chests stay shut. (5) `Explore` already had the early-stop callback (`done`).
- **Blood Raven live (checks 1 to 3, partly):** Cold Plains (3) -> Burial Grounds (17) by the task layer (nav `ExitTo`) V. She is the monster preset **805 at (5568,4954)** (kind 1), exactly where she
  stood, in several games of the character V; `Anchor.Preset` uses it and the script walks there first. The client lists units only within about 60 subtiles (listed at 58, absent at 95), so "wait
  for her to show" failed from some arrival points. She does NOT come to the player (stationary until aggro), and the client unloads her when the player walks away: the lure phase made every run end
  in "boss vanished alive", so the Blood Raven script has no lure now, and the runner walks back to a vanished boss's last position (3 times) before `ErrBossLost`.
  She hits in a crowd: raised skeletons in the dozens around her at 5568,4954. A clvl 6 and a clvl 12 Booti (starter gear, no mana potions) died or fled (emergency portal worked, V) and a
  clvl 24 Booti under logged invincibility (cheat ledger, reason on record) dealt about 10 percent damage per 5 minutes with the Lightning profile: the combat profile, not the script, is the limit.
- Not run (time): checks 4 (Countess), 5 (Smith), 6 (Andariel, poison and antidote), 7 (escape paths on a script with `NoTP`). Area ids Forgotten Tower to Catacombs and the other class ids stay U.

## 7. BOSSWIRE (2026-10-29): Countess, Smith, Andariel prepared for live checks (offline work; V offline = tests or real tables, U = live not done)

### 7.1 What changed offline (all in `internal/boss`, `internal/task`; tests `wire_test.go`, `act1_real_test.go`, `TestBossTravelGatesLevelAndPotions`)
- **Potion floor** `Script.MinPotions` (Countess 2, Smith 1, Andariel 3): `approach` refuses with `ErrAborted` (-> KindDanger -> chores) before walking anywhere. Not applied when already in the arena (the fight loop has its own escape).
  The task layer now has the same floor on the Smith and Countess travel steps (`gatePotions`, inserts Chores like Andariel's), so an escaped run restocks before it walks back.
- **Route failure** (`routeFail`): a failed `Travel`/`ExitTo`, or the total approach+locate time exceeding `Script.Timeout`, while the character stands on a route area or in the arena, opens a town portal (best effort, the original error is returned,
  a note says so). Death after a hop is `ErrDead` at once. A failure at the waypoint area itself does nothing (nothing to abort).
- **Tolerant calls**: a failed `Presets` read is a note, not the end of the run (falls through to hint and Explore); a failed `Antidote` is a note; a failed `DrinkHealing` is a note and, below `EscapePct`, leaves the fight (town portal, or leave game without TP) instead of ending the run raw.
- **Quest loot** (Smith's Horadric Malus): `Loot` returning 0 with `Quest` names is retried once after 2 s, a note says "still missing" if it fails again (the task step `StepPickUp` then gets its own chance).
- Hostile checks: the package still has none of its own; `Mon.Hostile` is filled by the adapter from `combat.IsFriendly` (test `TestSmithIgnoresFriendlyCrowd`).
- **V (real tables, the worker host /tmp/d2mpq, `go test -run Real ./internal/boss` with the install present, 2026-10-29)**: area ids 3 Cold Plains, 6 Black Marsh, 17 Burial Grounds, 20 Forgotten Tower, 21..25 Tower Cellar Level 1..5, 27 Outer Cloister, 28 Barracks, 35/36/37 Catacombs Level 2/3/4;
  class 156 Andariel, 267 Blood Raven, 402 Smith (key "smith", name "The Smith"); super unique "The Countess" = no. 6, base class corruptrogue3.
  **Finding: object 580 (`presetCountessChest`, Kolbot SuperChest) is NOT in the parsed objects.txt (573 rows, max class id 572).** The chest-based stand point is therefore suspect (U); the fallback chain (Kolbot hint -> Explore) is what will probably carry the run. Live check C2 must dump the real presets.
- **V by docs/nav.md (live, earlier)**: Black Marsh -> Forgotten Tower is a tile exit and works (9 s); Outer Cloister -> Barracks is a border gap and works (21 s); Barracks -> Catacombs 1-4 never walked (nav does not fight, packs blocked a character that could not fight); Tower Cellar 1..5 and Catacombs 2..4 hops never walked.

### 7.2 Known gaps (not fixable inside these packages)
1. `ExitTo`, `GoTo`, `Explore` of the adapter do not fight. Whether the bot's own watcher fights during them is U (observe in B3/C3). If it does not, the Barracks and the NAS hops need an escort (INTEGRATE).
2. Chores buy no antidotes (grep: none in `internal/task`/`internal/bot`), so Andariel's pre-fight antidote only fires when the belt already holds `yps`. The task layer cannot know the antidote count (`World` has none).
3. Andariel's poison state (state 2) and her cloud missile names are U (docs/boss.md section 6); until seen, the poison hazards do not fire.
4. The Smith may only exist while Tools of the Trade is active (S, the game wiki; U for 1.14d). The Charsi greeting step is `Optional`; if B2 shows no Smith without it, make that talk mandatory.
5. No tool runs a single boss script. Campaign order makes the Smith run first at clvl 8..13 (Den and Blood Raven finished), the Countess at 14..21, Andariel at `boss.andarielMinLevel` (docs/knobs.md; was 22 when this was written). Request: a `cmd/bossrun` (owner of `cmd/`) that builds `bossGame` + `newBossRunner` from `cmd/d2bot/boss.go`, runs `KillBoss(name)` once from wherever the character stands and prints `Result`. Everything below works with the campaign too, only slower.

### 7.3 Common prerequisites and stop rules (every check)
- Instance and character: a throwaway on a d2-live instance, Normal, offline, one boss per game, standing in town, `tools/devhealth.sh` clean, no `/tmp/d2lod-live.notice`. Gear and levels may be set only through `d2cheat` (ledger, reason on record): clvl >= floor, gold for potions. Quest state cannot be cheated; `Killed` is per game.
- Potions: at least 6 healing in the belt (3 columns), a TP tome with >= 3 scrolls (HasTP), 2 antidotes for Andariel. Waypoints: Black Marsh (Countess), Outer Cloister (Smith), Catacombs 2 (Andariel; else the walk is the check of the hops 28..35 and takes far longer).
- Log lines to collect from the d2bot log: `boss run` (boss, killed, loot, phases, notes, elapsed, err), `boss view: unique` (flags, class, unique number, name, position, life), `boss presets` (level, monster presets; objects are NOT listed there, use `d2mapdump dump` for those).
- Stop at once (Esc, leave the character in town) when: life under 25% twice in a row, three consecutive hops fail, the run exceeds its time limit, or the character dies. Do not retry a failed check more than once per game; record the failure and move on.
- Afterwards: leave the game, `tools/game.sh stop <id>` (AGENTS.md), say what was done to the character.

### 7.4 Checks (order: C, B, A; each independent)
Notation: expect = pass condition; record = values to copy into this section; fix = what to edit when the expectation fails.

**C. The Countess (clvl >= 14, Normal, potions >= 2)**
- C1 Approach: start in town with the WP active. Run the script (or the campaign). expect: `travel 6` lands in area 6; six hops land in 20, 21, 22, 23, 24, 25 in that order (V ids); life stays above 40% after each hop (else `ErrEscaped` and a town portal, which is itself check 7 below). record: seconds per hop, any `ErrNoExit`/`ErrStuck`, monsters met on each level, whether the bot fought them. fix: ids -> `act1.go` constants; slow stairs -> nav, not here.
- C2 Anchor: on arrival in 25 run `d2mapdump dump` (or read `boss presets`). expect either (a) object 580 listed with X 12565 or 12526 (then `Variants` stand points are right), or (b) no 580 (likely, see 7.1). record: ALL object presets (kind 2) with id and position, all monster presets (kind 1), the player arrival point, and the Countess' first-seen position and whether the `boss view: unique` line shows name "The Countess" and flags with the boss bit. fix: set `presetCountessChest` / `Variants` from the dump, or switch the anchor to the monster preset id of her spawn (kind 1) like Blood Raven's 805; keep `Hint` (12578,11043) only if her position is within 60 subtiles of it.
- C3 Locate: expect the note "preset ... not found" when (b), then `goto 12578,11043`, then she is seen (unit list reach is about 60 subtiles) within the 16 + 8 polls, otherwise Explore finds her in under 90 s. record: phase list, elapsed to first sight, `Result.Notes`.
- C4 Fight: expect phases `locate,attack` (plus `hazard:adds` when her escort is >= 5 within 12), a kill, no death, `Result.Phases` ending `loot`. record: kill time, potions used, minimum life, escort size, whether she moves (stays put until aggro?).
- C5 Loot and chest: expect `Result.Loot` > 0 on a rune/key/gem/unique/set/rare drop; her chest stays closed (look at the screenshot with `tools/shot.sh`, `loot` never clicks objects); runes picked inside 30 subtiles. record: items, the list of drops left on the ground.
- Pass: C1 to C5 as expected, or the failed ones recorded with the fix applied in `act1.go` data.

**B. The Smith (clvl >= 8, Normal, potions >= 1, Charsi talked to)**
- B1 Quest gate: before talking to Charsi enter the Barracks once with nothing to fight (clvl cheat ok) and list monsters 10 s. expect: no "The Smith" if the S claim holds. record: yes/no (decides gap 4). Then talk to Charsi ("quest" option) and re-enter.
- B2 Approach: `travel 27`, `exit 28` (V once, 21 s). expect area 28 after the hop; arrival point recorded. record: crowd size within 15 of arrival (hazard threshold 6), classes of the first pack (Barracks corridor packs are the known risk, nav docs).
- B3 Locate: no preset, no hint: the boss list, then Explore (<= 90 s). expect "The Smith" with class 402, mode 1/2, LifePct 100. record: his position relative to arrival, elapsed, whether Explore was needed, whether the watcher fought en route.
- B4 Fight: expect `locate,attack`, `hazard:crowded` only when life < 50% with >= 6 hostiles within 15 (then retreat to arrival and drink). record: kill time, potions, hazards fired, minimum life.
- B5 Quest drop: expect `Result.Loot` >= 1 and `Items["Horadric Malus"] == 1` after the run; a "quest item ... not picked up; retrying once" note is acceptable. fix when the Malus lies on the ground unpicked: `LootSpec.Quest` name vs the real item name (record the exact display name in `boss view`/loot log).

**A. Andariel (clvl >= `boss.andarielMinLevel`, Normal, potions >= 3, 2 antidotes in the belt, poison resist recorded)**
- A1 Approach: from the Catacombs 2 WP expect `exit 36`, `exit 37`, then area 37 (V ids). record per-hop time and monsters. Without the WP use the long walk once and record it.
- A2 Locate: expect she is NOT in the list on arrival (she is far), then Hint walk to (22549,9520) or Explore until seen. record her real spawn position, the arrival point, distance, the `Anchor.Hint` error in notes if any, and whether GoTo to the hint passed through monsters without the bot fighting (gap 1).
- A3 Aggro and poison: before the first hit expect the pre-fight antidote (`antidote` call, belt count -1) because poison resist < 75. During the fight record: does `View.Poisoned` become true (state 2 in the state array), does the antidote clear it, do clouds show as missiles (class ids and missiles.txt names for `poisonCloudMissiles`). expect no run abort on antidote errors (notes only).
- A4 Kite: expect phase `kite` every >= 1.5 s while she is within 10, the relative points `(+-14,+14)` from her first-seen position walkable and >= 10 from her. record: walkability, times she reached melee, damage taken, life minimum. fix: sides -> `Stand` data in `act1.go`.
- A5 Kill and loot: expect kill, `loot` phase, `Result.Loot` >= 0 (any tier), run time under the 12 min `Timeout`. record the duration and potions used.
- Chicken line 12%: not forced (see check 7).

**7. Escape paths (one game, spare character, after the three above)**: with 0 potions in the belt and life brought low in the arena by the boss itself (no cheats): expect `ErrEscaped` and a town portal; on a script with `NoTP` expect `ErrChicken` and the game left. Cheats on life are allowed only through `d2cheat` with a ledger reason.

### 7.5 Reporting back
Add the results as section 8 here: per check pass/fail, the recorded values, and the `act1.go` edits (data only). Mark every confirmed number V with the date and the character.

## 9. bossrun (cmd/bossrun): run one boss script live

`tools/live.sh <agent> bossrun [flags] BOSS` with BOSS = `countess|smith|andariel|bloodraven` (or the exact script name). Instance from `D2BOT_INSTANCE` (live.sh sets it).
- Flags: `--dry` (read-only: opens the game, fake input, prints the script gates and the character's state, runs nothing), `--timeout 15m` (hard limit, context deadline over the whole run), `--gamedir`, `--cache`.
- Starts from wherever the character stands (the script walks from there; stand in town at the WP for the checks in 7.4). Logs at debug level on stderr (`boss run`-style detail is the runner's own notes plus `boss view: unique` / `boss presets` lines from the adapter); a summary (`killed`, `loot`, `phases`, `elapsed`, `note:` lines, `err:`) goes to stdout.
- Exit codes: 0 killed, 1 ended without a kill, 2 usage, 3 setup failed (no game, no tables), 4 timed out, 5 died.
- It copies `cmd/d2bot/boss.go` (package main cannot be imported) as `cmd/bossrun/adapter.go`, plus the ToTown helper in `town.go`; re-copy when d2bot's adapter changes. Not repeated from d2bot: class setup (`applyClass`), leveler, task layer, potion gates of the task layer (the script's own `MinPotions` still applies). Hotkeys are ensured once at start, and combat's `Leave` is wired to `launch.Flow.LeaveGame`, so a `NoTP` chicken leaves the game.
- Status: V offline (`go test ./cmd/bossrun`: name resolution, exit-code mapping, summary output). U: everything live (nothing was run against a game).

## 8. BOSSLIVE (2026-10-29, i2, Combi clvl 30 Sorceress, Normal; V = verified live, U = not)
Combi had no waypoints (walked from town, ~8 min to Tower Cellar 5), no armor, str 10: cheats (ledger): gold 15000, vit 120 then vit 500 / ene 150 (life 969).
- **C1 Approach: V.** Hops 20, 21, 22, 23, 24, 25 all land in the expected areas; roughly 20 s per hop on the tower levels (22 at 00:48:59, 23 at :49:19, 24 at :49:44, 25 at :50:04). Monsters met: possessed/champion Dark Archers (22), fanatic/champion Ghosts (23). The bot's own watcher fought during hops (kills logged), so gap 1 is partly closed.
- **C2 Anchor: V.** Object 580 is NOT in the presets (30 presets listed, no objects shown by `boss presets`); the Countess has a MONSTER preset: `836@12534,11040` (others 833@12520,11099, 834@12603,11043). She stood at (12542,11036) in game 1 and (12555,11017) in game 2 (she moves a little). `act1.go` now anchors on `presetCountessSpawn = 836` (kind NPC/monster) instead of the chest; the Kolbot hint (12578,11043) is 36-40 from her (inside the 60 slop) but the nav got "stuck at 12585,11037 going to 12578,11043 after 6 recoveries" (hint point not walkable).
- **C3 Locate: V.** Run 1: preset not found note, hint walk stuck 7 short, she was listed (name "The Countess", uniqueNo 6, class 45, flags 10, area 25) once in view; area 25 arrival at about (12546,9521), her spawn 1500 position units south (y 9521 to 11036) (arrival to first sight about 30 s). Escort: Ghosts, Devilkin, Stone Thirst (flags 8).
- **C4 Fight: partly.** Run 1 (209 life): escort in hand, life 52 after one drink, `hazard:adds`, combat left the game ("left the game to survive"), exit 1. Run 2 (969 life, 2 minor potions): killed about a dozen escort monsters, kited, Countess down to 85% in 5 s of melee, then death at about 01:03:40: `Combat.Attack` tried its own Town Portal and failed 5 times with `combat: pointer is not over the portal scroll` (internal/combat/fighter.go openPortal, hovered item id mismatch; cause U, probably the inventory cell/scroll vs tome pick; owner of combat should check). Minor potions do not out-heal her; the runner only escaped on `potions == 0`.
  Fix here: escape also when life < EscapePct/2 with potions left (`TestCriticalLifeEscapesEvenWithPotions`).
  Gear matters: Normal Countess kills a 969-life naked sorc in seconds. Min floor should include a defence check (U).
- **C5 Loot: not reached.** Death cost the gold (gold 0 afterwards, cheat cannot raise a missing stat: needs a save edit).
- **A (Andariel): not run** (no Catacombs WP, no potions or gold left, character stripped; the Countess death is the blocker).
- Character state after: alive in town (respawned), 0 gold, no potions, corpse with the short staff left in Tower Cellar 5.
- Bring-up for the next check: save-edit gold (game stopped), waypoints via `d2cheat waypoints`, real armor, potions; then `tools/live.sh --bg --max 900 <agent> bossrun --timeout 14m countess`.

- **openPortal fix (PORTALNAV):** the 5x "pointer is not over the portal scroll" cause was the click point lacking the (screen-640, screen-480)/2 panel offset plus a hover check against the world selection (V by code reading, live U). `Fighter.openPortal` now uses `panel.NewLayout`, presses `i` only when the inventory UI flag is off, verifies the cast (portal object 59 near the player or the scroll gone), retries 3 times, and `Fight` leaves the game (`ErrChicken`) when the portal still fails.

## 10. BOSSLIVE2 (2026-10-29, i2, Combi clvl 30 Sorceress, Normal, all runs under `bossrun --god` = logged god-mode cheat; V = verified live, U = not)
Setup (ledger, actor BOSSLIVE2): save edit gold 20000, waypoints (39 bits), maxlife 969 after the earlier death left life 0 / gold 0 (game left to the main menu with `oog leave`, edited, `oog "start Combi"`, about 30 s). `townlive chores potions scrolls` filled the 4-slot belt (2 minor heal, 2 minor mana). No real gear (str 10: armor cannot be worn), so every fight below is god-mode: the escape logic (potions, EscapePct, NoTP chicken) is NOT exercised.
New: `bossrun --god REASON` (keeps god mode until the portal home; stopping god inside the Barracks crowd killed Combi once), `--nokite` (experiment).
- **Countess C1 (approach from town WP): V.** WP Black Marsh, hops 20..25 in about 2.5 min from town to area 25, no stuck, no ErrNoExit. C2/C3 as in section 8 (monster preset 836@12534,11040; first sight of her at (12555,11017) from player (12534,11042) 8 s after the crowd arrived; escort Blood Clan, Dark Archers, champions/uniques "Dread Jade the Quick", "Seethe Web").
- **C4 Fight: V, with a bug found and fixed.** Run 1: 10 minutes, phases `approach,locate,attack,hazard:adds` repeating, Countess at 100 -> 75% only, timeout (exit 4). Cause: the character stood pinned on one tile in the escort crowd while combat's kite (elite in range) and missile dodge tried to move every slice and never cast; nav "stuck / stalled by monsters, yielding" loop. Run 2 with `--nokite` (KiteEnabled=0, DodgeRadius=0): she died within 10 s of the start of the run (75% -> 0), `killed=true elapsed=1m35s`, phases `locate,hazard:adds,loot`.
  Fix (adapter in `cmd/bossrun/adapter.go` and the same 20 lines in `cmd/d2bot/boss.go`): after two consecutive `Attack` slices that ended on the starting tile, the next slice runs with kite and dodge off (restored afterwards). U live (the fix itself was not re-run on the Countess; the nokite run is the V evidence).
- **C5 Loot: V.** `loot=18`: El and Nef runes (inside the 30 subtile radius), plus pickit "any"-style junk (Hard Leather Armor unidentified, Light Crossbow, arrows, minor potions); gold pickups failed 7 times ("item left the ground without arriving in the pack" / "pointer never hovered") but the gold was not needed. Her chest was left closed (screenshot: chest standing beside the player).
- **Smith B2: V.** `travel 27` via the Outer Cloister WP and `exit 28`: area 28 reached in about 30 s from town. The Barracks arrival is a dense champion crowd (Champion Black Rogue x2 in 27, Champion Devilkin Shamans x4, Champion Death Clan x2 in 28).
- **Smith B3, run 1: not found.** `boss not found in the arena` after the 90 s Explore budget (about 16 explore legs, Barracks not fully explored). Combi had NO quest state (word 3 Tools of the Trade = 0, Den word 0 = 0, `d2qmem`: only Sisters started). Charsi's menu is only Talk / Trade/Repair / Cancel (V screenshot, no "quest" row). Result: B1 not decided yet, see run 2 below. Added `Script.ExploreBudget` (Smith 4 min).
- Side finding: `bossrun --god` ending a failed run stopped god mode in the crowd and Combi died (exit 1); fixed by portalling home first.
- **Smith B1/B3, runs 4 to 9: NOT found, cause undecided.** Cheat (ledger, reason "Smith only spawns with Tools of the Trade active"): Den of Evil done (word 1 = 0x8001) and Tools of the Trade started (word 3 = 0x000c) via `d2cheat quest set`, game restarted; later also Charsi's Talk menu (now with a "Tools of the Trade" row; picked with `townlive pickline 154 1:4 2:4`, word 3 stayed 0x000c, a fresh game afterwards). In every game the Barracks (area 28) showed no "The Smith"; first runs 2-4 min of Explore. Barracks object presets near the arrival: 15@15135,4705, 16@15135,4740, 29@(15136,4725 / 15131,4747 / 15144,4747), 29@15138,4669; monster presets 754@15119,4389, 802@15120,4635, 803@15173,4468 / 15218,4538 / 15142,4623, 818@15143,4416 / 15121,4751 / 15147,4767, 836@15074,4459 (none identified as the Smith). U whether the Explore reached the Smith's room: with the pinned problem below the character spent most of 2-4 min stuck in packs (explore leg count 13-16, nothing north of y 4389). Next step for whoever continues: find the Smith's object/monster preset in the Barracks with `d2mapdump dump` after a longer manual walk, or hand the real quest flow (Charsi imbue row after the Malus) to the task layer; do not assume the quest word controls the spawn.
- **Pinned in crowds (cause of the Barracks stalls too).** Nav "stalled by monsters, yielding" calls `Fighter.Clear` (PackYield default 20 s): inside it combat dodges and kites again and again without casting while pinned. `unpinned` (adapter, `cmd/bossrun` and `cmd/d2bot/boss.go`; counts calls that end within 6 subtiles, three in a row switch kite/dodge off for the next call) plus `YieldDeps.Timeout = 4 s` in bossrun made Explore reach y 4389 in 40 s once (run 7/8) but another run (9) still lost 4 minutes. `cmd/d2bot` (the real bot) builds its own PackYield: it still needs `Timeout: 4s` and the `unpinned` wrapper around its Clear closure (not done: cmd/d2bot setup is not in these directories). The root fix belongs to combat (do not dodge/kite when the move leaves the position unchanged).
- **Andariel A1 gate: V.** Combi with 2 healing potions in the belt: `Andariel needs 3 potions, have 2`, ErrAborted before any walking (phases `approach`, 0 s). The count is belt only (a third potion in the pack is not counted; Combi has a 4-slot belt, the town chores keep 2 mana potions). Not run past the gate (Combi cannot hold 3 healing potions without a belt): A2 to A5, the antidote and poison checks and check 7 (escape paths) stay U.
- Cheat/ledger notes: god mode stopped by SIGTERM (a killed run) leaves max life 8191: `d2cheat restore` fixes it (V). Death costs the gold and the save-edit cycle restores it; `bossrun --god` now portals home before stopping god (V once, "arrived in town"; failed once after a 45 s timeout in a crowd, Combi died, the tome was empty).
- The worker host load was 28-33 for most of this session (other agents); the abort rule (> 26) was exceeded for a few short runs, noted here.

## 11. BOSSFIX (2026-10-29, offline only; V offline = tests/real tables, U = live not done)
- **Pinned evade (combat):** `Fighter.evade` wraps every kite, dodge, missile sidestep and melee back-off; two evasive moves in a row that end within 2 subtiles of the start switch evading off for 8 s (`pinMoves/pinHold`, `mayEvade`) so the fighter casts in place. State is on the Fighter (Kill/Clear are called in short slices, a per-fight counter reset each slice). Test `TestPinnedKiteFightsInPlace`. `unpinned` in `cmd/bossrun` and `cmd/d2bot` stays as belt and braces; `cmd/d2bot/behaviors.go` packYield now also has `Timeout: 4s` and the `unpinned` wrapper (outside the brief's file list, 10 lines).
- **Cursed retreat (review-day M4):** below `CurseRetreatPct` with Amplify/Iron Maiden the engage step walks away from hostiles within `KiteDistance` (bounded 2x `KiteMaxSteps`, off while pinned) instead of swinging; Safety still drinks/portals. `TestCursedBacksOff`.
- **Weak pack (M5):** `weakHostiles` now requires every monster to be read as normal kind (`MonKindNormal`), not elite/boss/affixed, base level <= clvl+1, and at most 8 in range.
- **Smith location:** monstats `smith` = class 402 (V real table), superuniques row "The Smith" hcIdx 20, Rarity 0 (never random), Malus = objects.txt id 108 (V). Superunique monster presets are 734 + hcIdx (S, recalled Kolbot ids Bishibosh 734, Rakanishu 737) = 754 for the Smith, and 754@15119,4389 IS in the BOSSLIVE2 Barracks preset dump (V live list, U that it is the Smith). The script now anchors on preset 754 (then object 108) before Explore (`Anchor.Also`). Also: preset 836 occurs in both the Tower Cellar 5 and the Barracks, so it is a generic place_* id; the Countess anchor on 836 is not an identification (she was merely next to it). Quest dependence unresolved (U); do not trust the S claim.
- **Potion floor:** `View.BeltSlots` (adapter: `combat.BeltCapacity`, belts.txt numboxes via the equipped belt, 4 without one) and `View.PotionFloor`: slots unknown -> unchanged; <= 4 slots -> min(want, max(1, slots/2)) (half is left to the mana potions the chores keep; plain min(3, 4) would still fail Combi's 2 heal + 2 mana); else min(want, slots).
- **Live checks:** (1) Countess without `--nokite`/`unpinned` (set `Tune.Combat` defaults, grep log for "pinned: evasive moves go nowhere"); (2) Smith: `bossrun smith` from the Outer Cloister WP, look for log "preset 754/..." resolved and a "boss view: unique" at/near (15119,4389); if the Smith is absent there, dump presets after standing at that tile and report the quest word; (3) Andariel gate with 2 heal + 2 mana in a 4-slot belt passes; (4) Iron Maiden: no live source known.
