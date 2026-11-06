# Same-save fidelity probe: design

> status: design, 2026-11-06, card `fidelity-matrix` (docs only; nothing built, nothing run). The strongest cheap check of "d2d equals game.exe for what the bot uses": load ONE character save into both and compare the DETERMINISTIC quantities side by side. Matrix and the rows this closes: [fidelity-matrix.md](fidelity-matrix.md).
> Tags: **V** read in this repo this session (file named), **S** secondary, **U** unverified. Classic 1.14d only. Boundaries of the probe run itself (user brief 2026-11-06): game.exe side = hidden instance i1..i5 on a private Xvfb, read-only memory reads through `internal/mem`; d2d side = a slot from `tools/d2dslot.sh`, the `dump` verb and the ungated read-only debug verbs; the fork's `--lab-verbs` stays OFF; no cheats, no memory writes, no ledger entries in the measured part. Phase 2 (vendors) needs mouse/keyboard input on game.exe; that is flagged below and is a user question (matrix open question 4).

## 1. What "same save" means and whether d2d loads a real `.d2s`

- **d2d loads a game.exe-made save: V** (`docs/d2d-lan-oracle.md`: the joiner was "a COPY of an i1 character", a game.exe character, placed as `d2d/save/Palli.d2s` under `XDG_CONFIG_HOME`, selected at char select, then "in 127.0.0.1's game: act 1 ... 5 of our 5 items match the host's"). `docs/d2d-fork.md` also shows the plain path: `--start-screen charselect --start-name NAME`, `key Return`, then `save`/restart round trips (V). Open: whether every 1.14d item type and a merc block round-trips (U); upstream lists `.d2s` layouts as "unverified" in its gap map (S, `docs/research/d2d-fidelity-riskmap.md` section 1: 13 unverified, file formats).
- **Same map seed from the same file: V by read, U by run.** The d2s map id at 0xAB plus the active-difficulty bit pins the seed in both games (D, `docs/research/seed-study.md`; d2d `game_seed()` honours it, V). That the layout is then equal is the oracle result of `docs/research/seedspawn-trace.md` (5 areas, V). The live proof (card SEEDPIN) was never run (U, seed-study.md P3-0).
- **Third reference: the file itself.** `internal/cheat/d2s` and `saves.Decode` parse the stat block, skills, quest/waypoint words (`docs/saves.md` V) and `cmd/d2savecheck` verifies; so every field has THREE values: file, game.exe memory, d2d. A field where game.exe == file != d2d is a d2d bug; file != both is a parser limit.
- The host rewrites the save on exit and on join: always work on COPIES (V, d2d-lan-oracle.md), recompute the checksum if a file is edited (V, seed-study.md).

## 2. What each side can show (read-only)

### game.exe (hidden instance, `internal/mem`, `internal/d2`)

V live (`docs/mem.md`, character Testy clvl 1, 2026-10-28): UnitAny layout, merged stat list (Ex list at +0x48: life/mana/stamina in 24.8 fixed point), `Info` first/left/right skill, inventory list and item fields (quality, flags, ilvl, body location, page), UI panel flags, difficulty byte, player name. Level and area, monsters/NPC list with class and position, objects: the bot's `d2.Snapshot` reads them every tick (V, `internal/d2/snapshot.go`; `cmd/d2snap` records fixtures, V).
U or partial: skill levels INCLUDING bonuses from items (`Skill.Level` is base points only; +bonus via stat 107, V mem.md Q4 but class/tab bonus not added), per-item stat lists (affix values; used by pickit `stats` rules, live-checked in drills but not diffed, U), vendor stock via the NPC's item list (the town code reads it: V `docs/d2d-town.md` for d2d; for game.exe U in the memory layer), merc stats (hire-reader traces, `docs/research/hirereader-trace.md`, S), damage/range numbers shown by the skill UI (not read today: U, would need new offsets).

### d2d (slot, devctl socket, `internal/d2dsrc`, `cmd/d2dsnap`)

V: `dump [player,stats,skills,items,quests,ui,merc,time]` returns one JSON line (docs/d2d-fork.md); `dump stats` carries a `gear` sum for ids 16..25 (docs/forkdbg.md); `d2dsnap` builds the same `d2.Snapshot` through `d2dsrc` (cmd/d2dsnap). Ungated read-only debug verbs (V, `docs/lab-verbs.md`): players, level, warps, portals, presets, objects, corpses, layout, nearby, grid, blocked, attack, passives, itemat, missiles, tomb, pets, bare `states`, `quest list|show`; plus ping, state, info, clock, dump, screenshot, step, lockstep, cmd, input verbs, save. `debug attack` prints wsm/ias/fcr/fhr fields (V, d2d-fidelity.md 8). NOT available without `--lab-verbs`: `debug give|equip|stat|goto|spawn...` (not needed). Vendor stock: a `dump vendor` patch existed in a local fork branch (d2d-town.md V); whether it is in the current lab binary is U and must be checked in slice 2.

## 3. Comparable versus random

| Quantity | Comparable? | How | Matrix rows |
|---|---|---|---|
| base stats str/dex/vit/ene, points left, level, XP, gold, stash gold | yes, exact | file vs mem vs dump | ST01 ST07 ST09 |
| life/mana/stamina max and current, regen per second | yes (formula) | derived sheet; regen by two reads N seconds apart in town | ST01 ST04 |
| defence, block %, to-hit/AR, resists (with penalty), speed fields | yes | stat sheet; d2d `gear` sums vs game.exe merged list | ST02 ST03 ST06 |
| skill list, base levels, bonus levels, selected skills | yes | `Info` list vs `dump skills` | SK13 SK14 |
| per-skill mana cost, range, damage, cast frames | yes if read (skill UI) or computed from the same level; otherwise already oracle-covered | formula check: compute from level with the oracle, compare to d2d value; game.exe memory has no cost field (U) | SK02 SK04 SK06 |
| inventory/equipment: code, position, quality, ilvl, sockets, ethereal, identified, each affix value | yes (they are in the file) | item key = container + x + y; unit ids differ and are ignored | IT02 IT04 |
| durability and charges | yes | stat 72/73 | IT08 |
| vendor stock, per-item quote | stock lists maybe (seed rule U), quotes yes (formula; `price_model.py` equals game.exe on 100,000 cases V) | open the window on both, diff list and quotes; if lists differ use the quote on identical items | VN01-VN05 |
| merc: class, level, life, damage, resists, skills, revive/hire quote | yes | hireling block vs `dump merc` | ME01-ME03 ME06 |
| town layout, NPC class + position, waypoint object positions, portal | yes | Snapshot Monsters/Objects in the town; rooms via `debug layout` vs whisper-style dumps (INSTRUMENTATION, not in this run) | VN07 MA03 |
| waypoint mask, quest words, difficulty unlocks | yes, exact | file vs memory vs `dump quests` | PR05 PR06 |
| first-load monster/object list of the loaded rooms | positions and classes yes at first load (seed pinned); counts depend on which rooms are up (U) | `monsters` vs Snapshot Monsters, same save in a field level | MA03 MA04 MA05 |
| anything rolled during play: hits, drops, damage, XP, deaths, vendor refresh timing | NO (RNG streams differ by design, tierb-design-20261102.md V) | oracle (C) or statistical A:B (D) | rest |

## 4. Harness shape

One Go tool, standard library only (build rule), new directory so no existing owner is touched:

- `cmd/d2fidprobe/`: flags `-scenarios FILE`, `-class`, `-gameexe-instance iN`, `-d2d-sock PATH`, `-out DIR`, `-phase 1|2`.
- `internal/fidprobe/`: `Sheet` (normalised struct of the quantities above, every field `value, source tag, readable bool`), readers `FromFile` (d2s), `FromGameExe` (`d2.Reader` over `mem.OpenGame()`), `FromD2D` (`d2dctl` + `d2dsrc`), `Diff(a,b) []Cell`, `Expected` (a table of known, explained differences keyed by matrix row id and field, e.g. units of fixed point), `Report` (Markdown table plus JSON).
- Result classes per field: EQUAL, KNOWN (explained by an `Expected` row), MISMATCH, UNREADABLE (one side cannot read it; counted as coverage, not hidden). A scenario passes with 0 MISMATCH and every UNREADABLE listed in the report.
- Class and build are parameters (AGENTS.md): scenario files name class and build; skills are resolved by name through the data tables, never by numeric id in Go.

Run order per scenario (all read-only): copy the `.d2s` to the instance save dir and to a d2d slot dir; start game.exe on it (OOG login to the character, main-agent/ops step) and d2d on its copy; wait until both are in town and stable (two Snapshot reads equal 1 s apart); read each side three times; diff; write the table. The tool never sends input in phase 1.

## 5. Scenario catalogue (14; the saves are fixtures made earlier by the character factory or by real play)

Fixture creation is a SETUP step outside the measured probe (docs/research/character-factory.md V: levels, stats, quests, waypoints can be set; creating ITEMS on the original is not possible, so item-rich saves must come from played characters or the live Gate characters, V charfactory).

| # | Scenario | Purpose |
|---|---|---|
| 1 | fresh Sorceress clvl 1, Normal, Rogue Encampment, starting kit | base sheet, starting items, NPC list, waypoint mask |
| 2 | same for a second class (Paladin) | build-flexibility rule: not Sorceress-only |
| 3 | clvl 6 Sorceress, Charged Bolt 4, Frozen Armor 1, Frost Nova 1, tome, belt of potions | skill points, derived stats at a real plan state |
| 4 | clvl 6 with common vendor gear (cap, quilted, buckler, short sword of X) | defence, requirements, magic affixes |
| 5 | clvl 9 poor (0 gold) and clvl 9 rich (20,000 gold) | quote and purse behaviour |
| 6 | clvl 12 with the Rogue merc alive | merc stats, auras, gear |
| 7 | clvl 12 with the merc dead | revive quote |
| 8 | clvl 12 standing in Stony Field (saved there) | first-load room/monster lists on a pinned seed (MA04) |
| 9 | clvl 16 with the Gate 2 kit | the real test state: skills, belt, tomes, charms |
| 10 | item zoo clvl 20: magic, rare, set, unique, ethereal, socketed, runes, gems (from a played character) | IT04 affix readout |
| 11 | Den of Evil done, Blood Raven done, Cain rescued, all Act 1 waypoints | quest words and waypoint mask |
| 12 | Andariel dead, in Lut Gholein | act 2 town layout (the known tile mismatch, d2d-lan-oracle.md), waypoint list |
| 13 | Nightmare clvl 30 Sorceress | difficulty penalties, NM level formula |
| 14 | hardcore-flag character, softcore control | flag handling only; skip if no fixture |

## 6. Pass criteria

- Integers (stats, levels, positions in subtiles, counts, item codes, masks): exact.
- Fixed-point or float fields (life/mana 24.8, regen, speed): relative 0.5 %, after converting both to the same unit; the unit conversion lives in `Expected`, one line per field.
- Sets (items by container/x/y, waypoints, quests, skills, NPCs by class): exact set equality, with order ignored.
- Strings (names, descriptions): not compared; codes are.
- A mismatch becomes a matrix row update and (for a d2d deviation) an oracle task; it is never silently added to `Expected`.
- Success of the whole design: scenarios 1 to 11 at 0 MISMATCH and fewer than 5 % UNREADABLE fields, otherwise the list of UNREADABLE fields is the next reader-work list.

## 7. Where it runs and the cost

- d2d side on the worker host, one slot (`tools/d2dslot.sh run -- ...`, niced, under `tools/agentrun.sh`), game.exe side on one hidden instance (i1..i5) via `tools/live.sh` (ptrace setcap through the ops wrapper, instance flock). The tool needs both at once on the same host (about 2 cores); capacity check `tools/capacity.sh --add 2` first; one scenario about 3 min, the full catalogue about 45 min. Never i0, never another agent's instance, game windows muted (`-w -ns`, `tools/mute-d2.sh`); CLEAN UP with `tools/game.sh stop <id>`, `tools/agentreap.sh <agent>`.
- Only multi/OOG tooling may start an instance (public AGENTS.md); the first run therefore needs a coordinator step to start and log in the instance.
- Every probe record carries the lines `writes after start: 0` and `lab verbs: off` (CRUTCH RULE wording; there is no gate here, the record is just auditable). The ledger stays empty because nothing is written.
- Unit tests offline: `mem.Fake` plus recorded `d2snap`/`d2dsnap` JSON fixtures for the diff and the normaliser (V, both tools already emit them).

## 8. Phase 2: vendors, needs input

Vendor windows open only by talking to the NPC (walk, click the NPC, click Trade): mouse/keyboard input on the hidden game.exe, still no memory writes. Compare per vendor (Akara, Charsi, Gheed, Kashya; Cain is not a vendor before the rescue): stock list (code, quality, ilvl, affixes), prices of every stock item, plus the Sell quote of three known inventory items and the Repair quote of a damaged one. Stock identity depends on whether stock generation is seeded the same on both sides (U, VN04); if the lists differ the comparison falls back to quotes on items both sides hold and to the 100,000-case price oracle (V) for the rest. Restore the character by loading the save copy afterwards.

## 9. Limits and likely first findings

- Probe only sees state at rest; combat outcomes need C and D. A passing probe does not prove 1:1; it proves the deterministic inputs the bot reads are equal.
- Expected early findings (S/I, from existing docs): gear-share truncation in stat 17..25 (playerdef-trace D3), vendor tab compaction (adapter, not a bug), sell/buy prices (VN01/VN02 known), corpse count, d2d autoloot (off by toggle).
- Risk: Wine process memory reads may race during level load; the three-read stability rule handles it.

## 10. Implementation status (card `fidprobe-core`, TASK-55, 2026-11-06)

Built offline, no live run yet. Tags: V = proven by the offline tests named, U = needs the first live run.

- **Engine, `internal/fidprobe` (V, `go test ./internal/fidprobe`).** Both sides are flattened from a `d2.Snapshot` (plus quest words and waypoints) into a `Sheet` of `group.name` fields (int, float or sorted set). `Diff` classifies every field EQUAL, KNOWN, MISMATCH or UNREADABLE. A field absent on one side, or reported unknown by `d2dsrc.Report`, is UNREADABLE and never EQUAL. A scenario group with no field on either side is listed as `group:NAME` UNREADABLE.
- **Rules are data (V).** `rules.json` (first match wins: `rel` 0.5 % for life, mana and stamina, `exact` for everything else; sets are always compared by equality), `known.json` (match, matrix row, reason; turns a MISMATCH into KNOWN, never hides UNREADABLE), `scenarios.json` (the 14 scenarios: id, name, save role, class, difficulty, comparable groups). One shipped known entry: `skills.bonus` row SK14 (class and tab bonuses not added on the game.exe side, S).
- **Tool, `cmd/d2fidprobe` (V for replay, U for live).** Calls the existing readers as libraries, no subprocess: `mem.OpenInstance` + `d2.Reader` (Snapshot, FindServerPlayer, ReadProgress) and `d2dsrc` (Read, Progress), the code `d2snap` and `d2dsnap` run. `json:FILE` replays a recorded document (d2dsnap output or plain Snapshot). Each side is read `-reads` times (default 3) and must agree, else the run fails as unstable. The loaded class and difficulty must match the scenario. Exit 0 = no MISMATCH, 3 = MISMATCH, 1 = read error, 2 = usage. The report ends with `writes after start: 0` and `lab verbs: off`.
- **Not done:** the third reference (the `.d2s` file through `saves.Decode`; the tool only checks that `<role>.d2s` exists), merc and vendor fields (the Snapshot carries none: scenario groups `merc` show as UNREADABLE until a reader exists), item affix values (the Snapshot has item base, quality, ilvl and flag letters only), unit conversion rows in `known.json` (none needed so far).
- Run recipe and live checklist: [../probes/fidprobe.md](../probes/fidprobe.md).
